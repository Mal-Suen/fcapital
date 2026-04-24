package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Mal-Suen/fcapital/internal/core/ai"
	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
	appcontext "github.com/Mal-Suen/fcapital/internal/core/context"
	"github.com/Mal-Suen/fcapital/internal/core/orchestrator"
	"github.com/Mal-Suen/fcapital/internal/core/orchestrator/phases"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// aiCmd represents the ai command
var aiCmd = &cobra.Command{
	Use:   "ai-scan",
	Short: "AI驱动的渗透测试扫描",
	Long: `使用AI驱动的自动化渗透测试。

该命令会:
1. 收集系统信息和工具状态
2. 初始化AI会话
3. 按阶段执行渗透测试
4. 每个阶段结束后AI分析结果并决定下一步

示例:
  fcapital ai-scan -t example.com
  fcapital ai-scan -t example.com --auto-continue
  fcapital ai-scan -t example.com --provider deepseek`,
	Run: runAIScan,
}

var (
	aiTarget     string
	autoContinue bool
	aiProvider   string
	aiModel      string
	skipPhases   []string
)

func init() {
	rootCmd.AddCommand(aiCmd)

	aiCmd.Flags().StringVarP(&aiTarget, "target", "t", "", "目标域名或IP")
	aiCmd.Flags().BoolVar(&autoContinue, "auto-continue", false, "AI决策后自动继续")
	aiCmd.Flags().StringVar(&aiProvider, "provider", "deepseek", "AI提供者 (openai/deepseek/ollama)")
	aiCmd.Flags().StringVar(&aiModel, "model", "", "AI模型名称")
	aiCmd.Flags().StringSliceVar(&skipPhases, "skip", []string{}, "要跳过的阶段")

	aiCmd.MarkFlagRequired("target")
}

func runAIScan(cmd *cobra.Command, args []string) {
	printBanner()

	// 1. 初始化上下文管理器
	ctxMgr := appcontext.NewManager()
	if err := ctxMgr.Initialize(); err != nil {
		fmt.Printf("❌ 初始化上下文失败: %v\n", err)
		return
	}

	fmt.Println("✅ 系统信息收集完成")
	fmt.Printf("   OS: %s %s (%s)\n", ctxMgr.GetContext().SystemInfo.OS, ctxMgr.GetContext().SystemInfo.OSVersion, ctxMgr.GetContext().SystemInfo.Arch)
	fmt.Printf("   可用包管理器: %s\n", strings.Join(ctxMgr.GetContext().SystemInfo.PackageManagers, ", "))

	// 2. 初始化调度器
	sched := scheduler.New()

	// 注册工具
	registerTools(sched)

	// 检查工具状态
	fmt.Println("\n📋 检查工具状态...")
	toolStatus := sched.GetToolStatus()
	ready := 0
	missing := 0
	for _, status := range toolStatus {
		if status.Status == "ready" {
			ready++
		} else if status.Status == "missing" {
			missing++
		}
	}
	fmt.Printf("   ✅ 就绪: %d | ❌ 缺失: %d\n", ready, missing)

	// 更新上下文中的工具信息
	tools := make([]appcontext.ToolInfo, 0)
	for name, status := range toolStatus {
		tools = append(tools, appcontext.ToolInfo{
			Name:    name,
			Version: status.Version,
			Path:    status.Path,
			Status:  status.Status,
		})
	}
	ctxMgr.SetTools(tools)

	// 3. 初始化AI引擎
	var aiEngine *ai.Engine
	apiKey := getAPIKey(aiProvider)

	if apiKey != "" || aiProvider == "ollama" {
		provider := createProvider(aiProvider, apiKey, aiModel)
		aiEngine = ai.NewEngine(provider)
		fmt.Printf("\n🤖 AI引擎初始化完成 (Provider: %s)\n", aiProvider)
	} else {
		fmt.Printf("\n⚠️  未配置 %s API密钥，将使用预设工作流\n", aiProvider)
		fmt.Printf("   设置环境变量 %s_API_KEY 以启用AI功能\n", strings.ToUpper(aiProvider))
	}

	// 4. 初始化编排器
	orch := orchestrator.New(
		orchestrator.WithAIEngine(aiEngine),
		orchestrator.WithContextManager(ctxMgr),
		orchestrator.WithScheduler(sched),
	)

	// 注册阶段
	orch.RegisterPhase(phases.NewReconPhase(sched))
	orch.RegisterPhase(phases.NewDiscoveryPhase(sched))
	orch.RegisterPhase(phases.NewVerificationPhase(sched))
	orch.RegisterPhase(phases.NewReportPhase(sched))

	// 5. 运行工作流
	fmt.Printf("\n🎯 目标: %s\n", aiTarget)
	fmt.Println("🚀 开始AI驱动的渗透测试...\n")

	opts := &orchestrator.RunOptions{
		AutoContinue:    autoContinue,
		ConfirmCritical: true,
		SkipPhases:      skipPhases,
	}

	result, err := orch.Run(context.Background(), aiTarget, opts)
	if err != nil {
		fmt.Printf("\n❌ 执行失败: %v\n", err)
		return
	}

	// 6. 输出结果
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 测试结果摘要")
	fmt.Println(strings.Repeat("=", 50))

	for phaseID, phaseResult := range result.PhaseResults {
		fmt.Printf("\n📌 %s (%s)\n", phaseResult.PhaseName, phaseID)
		fmt.Printf("   状态: %s\n", phaseResult.Status)
		if phaseResult.Error != "" {
			fmt.Printf("   错误: %s\n", phaseResult.Error)
		}
		if len(phaseResult.Findings) > 0 {
			fmt.Println("   发现:")
			for key, val := range phaseResult.Findings {
				fmt.Printf("   - %s: %v\n", key, truncateString(fmt.Sprintf("%v", val), 100))
			}
		}
	}

	// 7. Token使用统计
	if aiEngine != nil {
		fmt.Printf("\n💰 Token使用: %d\n", aiEngine.GetTokenUsage())
	}

	fmt.Println("\n✅ 测试完成!")
}

// aiChatCmd represents the ai chat command
var aiChatCmd = &cobra.Command{
	Use:   "ai-chat",
	Short: "与AI助手交互",
	Long: `与AI助手进行交互式对话。

示例:
  fcapital ai-chat
  fcapital ai-chat --provider deepseek`,
	Run: runAIChat,
}

func init() {
	rootCmd.AddCommand(aiChatCmd)

	aiChatCmd.Flags().StringVar(&aiProvider, "provider", "deepseek", "AI提供者")
	aiChatCmd.Flags().StringVar(&aiModel, "model", "", "AI模型名称")
}

func runAIChat(cmd *cobra.Command, args []string) {
	printBanner()

	// 初始化AI引擎
	apiKey := getAPIKey(aiProvider)
	if apiKey == "" && aiProvider != "ollama" {
		fmt.Printf("❌ 未配置 %s API密钥\n", aiProvider)
		fmt.Printf("   设置环境变量 %s_API_KEY\n", strings.ToUpper(aiProvider))
		return
	}

	provider := createProvider(aiProvider, apiKey, aiModel)
	engine := ai.NewEngine(provider)

	// 初始化上下文
	ctxMgr := appcontext.NewManager()
	ctxMgr.Initialize()

	fmt.Printf("🤖 AI助手已就绪 (Provider: %s)\n", aiProvider)
	fmt.Println("输入消息与AI对话，输入 'quit' 或 'exit' 退出")
	fmt.Println(strings.Repeat("-", 50))

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n你: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" || input == "exit" {
			fmt.Println("\n再见!")
			break
		}

		if input == "" {
			continue
		}

		// 发送消息给AI
		ctxMgr.AddMessage("user", input)

		req := &providers.ChatRequest{
			Messages: convertMessages(ctxMgr.GetConversation()),
		}

		fmt.Print("\nAI: ")
		resp, err := engine.Chat(context.Background(), req)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			continue
		}

		fmt.Println(resp.Content)
		ctxMgr.AddMessage("assistant", resp.Content)
	}
}

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "管理测试上下文",
	Long: `管理渗透测试上下文信息。

子命令:
  show    显示当前上下文
  clear   清除上下文
  export  导出上下文到文件`,
}

var contextShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前上下文",
	Run: func(cmd *cobra.Command, args []string) {
		ctxMgr := appcontext.NewManager()
		if err := ctxMgr.Initialize(); err != nil {
			fmt.Printf("❌ 初始化失败: %v\n", err)
			return
		}

		json, err := ctxMgr.ToJSON()
		if err != nil {
			fmt.Printf("❌ 序列化失败: %v\n", err)
			return
		}

		fmt.Println(json)
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextShowCmd)
}

// Helper functions

func getAPIKey(provider string) string {
	providerUpper := strings.ToUpper(provider)

	// 1. 优先检查环境变量（.env 文件已加载到环境变量）
	// 支持多种命名格式: DEEPSEEK_API_KEY, DEEPSEEK_APIKEY, DEEPSEEK_KEY
	envKeys := []string{
		fmt.Sprintf("%s_API_KEY", providerUpper),
		fmt.Sprintf("%s_APIKEY", providerUpper),
		fmt.Sprintf("%s_KEY", providerUpper),
	}

	for _, key := range envKeys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}

	// 2. 检查 viper 配置文件
	if viper.IsSet(fmt.Sprintf("ai.%s.api_key", provider)) {
		return viper.GetString(fmt.Sprintf("ai.%s.api_key", provider))
	}

	// 3. 检查通用 AI API 密钥配置
	if viper.IsSet("ai.api_key") {
		return viper.GetString("ai.api_key")
	}

	return ""
}

func createProvider(providerType, apiKey, model string) providers.Provider {
	// 优先从环境变量读取 baseURL
	providerUpper := strings.ToUpper(providerType)
	baseURL := os.Getenv(fmt.Sprintf("%s_BASE_URL", providerUpper))
	if baseURL == "" {
		baseURL = os.Getenv(fmt.Sprintf("%s_BASEURL", providerUpper))
	}
	// 其次从 viper 配置文件读取
	if baseURL == "" {
		baseURL = viper.GetString(fmt.Sprintf("ai.%s.base_url", providerType))
	}

	// 如果 model 为空，从环境变量读取
	if model == "" {
		model = strings.TrimSpace(os.Getenv("AI_MODEL"))
	}
	// 其次从 viper 配置文件读取
	if model == "" {
		model = viper.GetString(fmt.Sprintf("ai.%s.model", providerType))
	}
	model = strings.TrimSpace(model)

	switch providerType {
	case "openai":
		return providers.NewOpenAIProvider(apiKey, model, baseURL)
	case "deepseek":
		return providers.NewDeepSeekProvider(apiKey, model, baseURL)
	case "ollama":
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return providers.NewOllamaProvider(baseURL, model)
	default:
		return providers.NewDeepSeekProvider(apiKey, model, baseURL)
	}
}

func convertMessages(msgs []appcontext.Message) []providers.Message {
	result := make([]providers.Message, len(msgs))
	for i, m := range msgs {
		result[i] = providers.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}

func registerTools(s *scheduler.Scheduler) {
	// Register common tools
	tools := []*scheduler.ToolDefinition{
		{
			Name:         "nmap",
			Description:  "Network Security Scanner",
			Category:     "port_scan",
			Capabilities: []string{"port_scan", "service_detection", "os_fingerprinting"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "winget", Package: "Insecure.Nmap"},
				{Type: "choco", Package: "nmap"},
				{Type: "apt", Package: "nmap"},
				{Type: "brew", Package: "nmap"},
			},
			VerifyCmd: "nmap --version",
			Fallbacks: []string{"masscan", "rustscan"},
		},
		{
			Name:         "subfinder",
			Description:  "Subdomain Discovery Tool",
			Category:     "recon",
			Capabilities: []string{"subdomain_enum_passive"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "go", Package: "github.com/projectdiscovery/subfinder/v2/cmd/subfinder"},
			},
			VerifyCmd: "subfinder -version",
			Fallbacks: []string{"amass"},
		},
		{
			Name:         "httpx",
			Description:  "HTTP Toolkit",
			Category:     "recon",
			Capabilities: []string{"http_probe", "technology_detection"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "go", Package: "github.com/projectdiscovery/httpx/cmd/httpx"},
			},
			VerifyCmd: "httpx -version",
			Fallbacks: []string{"httprobe"},
		},
		{
			Name:         "nuclei",
			Description:  "Vulnerability Scanner",
			Category:     "vuln_scan",
			Capabilities: []string{"vulnerability_scan", "cve_scan"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "go", Package: "github.com/projectdiscovery/nuclei/v3/cmd/nuclei"},
			},
			VerifyCmd: "nuclei -version",
			Fallbacks: []string{"nikto"},
		},
		{
			Name:         "sqlmap",
			Description:  "Automatic SQL Injection Tool",
			Category:     "vuln_scan",
			Capabilities: []string{"sql_injection"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "pip", Package: "sqlmap"},
				{Type: "pip3", Package: "sqlmap"},
			},
			VerifyCmd: "sqlmap --version",
			Fallbacks: []string{},
		},
		{
			Name:         "gobuster",
			Description:  "Directory/File/DNS Busting Tool",
			Category:     "web_scan",
			Capabilities: []string{"directory_bruteforce", "dns_bruteforce"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "go", Package: "github.com/OJ/gobuster/v3"},
			},
			VerifyCmd: "gobuster version",
			Fallbacks: []string{"dirsearch", "ffuf"},
		},
		{
			Name:         "ffuf",
			Description:  "Fast Web Fuzzer",
			Category:     "web_scan",
			Capabilities: []string{"directory_bruteforce", "fuzzing"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "go", Package: "github.com/ffuf/ffuf/v2"},
			},
			VerifyCmd: "ffuf -V",
			Fallbacks: []string{"gobuster", "dirsearch"},
		},
		{
			Name:         "dirsearch",
			Description:  "Web Path Scanner",
			Category:     "web_scan",
			Capabilities: []string{"directory_bruteforce"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "pip", Package: "dirsearch"},
				{Type: "pip3", Package: "dirsearch"},
			},
			VerifyCmd: "dirsearch --version",
			Fallbacks: []string{"gobuster", "ffuf"},
		},
		{
			Name:         "wpscan",
			Description:  "WordPress Security Scanner",
			Category:     "web_scan",
			Capabilities: []string{"wordpress_scan", "plugin_detection", "user_enumeration"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "gem", Package: "wpscan"},
				{Type: "apt", Package: "wpscan"},
				{Type: "brew", Package: "wpscan"},
			},
			VerifyCmd: "wpscan --version",
			Fallbacks: []string{},
		},
		{
			Name:         "nikto",
			Description:  "Web Server Scanner",
			Category:     "web_scan",
			Capabilities: []string{"web_server_scan", "vulnerability_scan"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "apt", Package: "nikto"},
				{Type: "brew", Package: "nikto"},
			},
			VerifyCmd: "nikto -Version",
			Fallbacks: []string{"nuclei"},
		},
		{
			Name:         "whatweb",
			Description:  "Web Technology Fingerprinter",
			Category:     "recon",
			Capabilities: []string{"technology_detection", "web_fingerprint"},
			InstallMethods: []scheduler.InstallMethod{
				{Type: "apt", Package: "whatweb"},
				{Type: "brew", Package: "whatweb"},
			},
			VerifyCmd: "whatweb --version",
			Fallbacks: []string{"httpx"},
		},
	}

	for _, tool := range tools {
		s.RegisterTool(tool)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
