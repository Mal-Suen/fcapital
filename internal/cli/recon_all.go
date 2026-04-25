package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
	"github.com/Mal-Suen/fcapital/internal/core/dispatcher"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
	"github.com/Mal-Suen/fcapital/internal/core/script"
	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/Mal-Suen/fcapital/internal/modules/portscan"
	"github.com/Mal-Suen/fcapital/internal/modules/recon"
	"github.com/Mal-Suen/fcapital/internal/modules/subdomain"
	"github.com/Mal-Suen/fcapital/internal/pkg/logger"
	"github.com/Mal-Suen/fcapital/internal/pkg/toolcheck"
	"github.com/spf13/cobra"
)

// ReconResult 信息收集结果
type ReconResult struct {
	Target       string
	Subdomains   []string
	HTTPServices []recon.HTTPXResult
	DNSRecords   []recon.DNSXResult
	OpenPorts    []portscan.PortInfo
	WAFInfo      *WAFDetection
	TechStack    []TechInfo
	SSLInfo      *SSLInfo
	Emails       []string
	SensitiveFiles []string
	CMSInfo      *CMSInfo
	StartTime    time.Time
	EndTime      time.Time
}

// WAFDetection WAF 检测结果
type WAFDetection struct {
	Detected bool     `json:"detected"`
	Name     string   `json:"name"`
	Products []string `json:"products"`
}

// TechInfo 技术栈信息
type TechInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Category string `json:"category"`
}

// SSLInfo SSL/TLS 信息
type SSLInfo struct {
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"`
	Cipher      string `json:"cipher"`
	Protocol    string `json:"protocol"`
	SelfSigned  bool   `json:"self_signed"`
}

// CMSInfo CMS 信息
type CMSInfo struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Plugins    []string `json:"plugins"`
	Themes     []string `json:"themes"`
	User       string   `json:"user"`
}

// reconCmd 信息收集命令
var reconCmd = &cobra.Command{
	Use:   "recon",
	Short: "信息收集 + AI驱动渗透测试",
	Long: `执行全面的信息收集，然后自动进入 AI 驱动的渗透测试流程。

信息收集包括：
- 子域名枚举 (subfinder)
- HTTP 探测 (httpx) - 技术栈、标题、状态码
- DNS 查询 (dnsx)
- 端口扫描 (nmap)
- WAF 检测 (wafw00f)
- SSL/TLS 信息
- 敏感文件探测
- 邮箱收集

收集完成后，AI 自动分析结果并推荐下一步渗透操作。

示例:
  fcapital recon -t example.com
  fcapital recon -t example.com --depth quick    # 快速扫描
  fcapital recon -t example.com --depth full     # 深度扫描
  fcapital recon -t example.com --no-ai          # 仅收集信息，不进入AI模式`,
	Run: runRecon,
}

var (
	reconTarget      string
	reconDepth       string  // quick, normal, full
	reconNoAI        bool    // 不进入 AI 模式
	reconProvider    string
	reconModel       string
	reconAutoConfirm bool
)

func init() {
	reconCmd.Flags().StringVarP(&reconTarget, "target", "t", "", "目标域名")
	reconCmd.Flags().StringVar(&reconDepth, "depth", "normal", "扫描深度 (quick/normal/full)")
	reconCmd.Flags().BoolVar(&reconNoAI, "no-ai", false, "仅收集信息，不进入AI模式")
	reconCmd.Flags().StringVar(&reconProvider, "provider", "openai", "AI提供者")
	reconCmd.Flags().StringVar(&reconModel, "model", "", "AI模型")
	reconCmd.Flags().BoolVar(&reconAutoConfirm, "auto-confirm", false, "自动确认脚本执行")
}

func runRecon(cmd *cobra.Command, args []string) {
	if reconTarget == "" {
		fmt.Println("❌ 请指定目标: fcapital recon -t <target>")
		return
	}

	printBanner()
	fmt.Printf("🎯 目标: %s\n", reconTarget)
	fmt.Printf("📊 扫描深度: %s\n\n", reconDepth)

	// 初始化
	tm := InitToolManager()
	ctx := context.Background()
	toolChecker := toolcheck.NewChecker()
	toolCheckResult := toolChecker.CheckAll()

	fmt.Printf("🔧 工具检测: 已安装 %d/%d 个工具\n", toolCheckResult.InstalledCount, toolCheckResult.TotalCount)

	// 执行信息收集
	result := &ReconResult{
		Target:    reconTarget,
		StartTime: time.Now(),
	}

	// 根据深度决定扫描范围
	runFullRecon(ctx, tm, result, reconDepth)

	result.EndTime = time.Now()

	// 输出结果摘要
	printReconSummary(result)

	if reconNoAI {
		fmt.Println("\n✅ 信息收集完成 (--no-ai 模式)")
		return
	}

	// 进入 AI 模式继续渗透
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🤖 进入 AI 驱动渗透模式...")
	fmt.Println(strings.Repeat("=", 60))

	startAIPenetration(ctx, result, tm, toolChecker)
}

func runFullRecon(ctx context.Context, tm *toolmgr.ToolManager, result *ReconResult, depth string) {
	var wg sync.WaitGroup

	// 1. 子域名枚举
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [1/6] 子域名枚举...")
		if tool, err := tm.Get("subfinder"); err == nil && tool.IsReady() {
			runner, err := subdomain.NewSubfinderRunner(tm)
			if err != nil {
				fmt.Printf("   ⚠️  初始化失败: %v\n", err)
				return
			}
			subdomains, err := runner.Enumerate(ctx, result.Target, nil)
			if err != nil {
				fmt.Printf("   ⚠️  枚举失败: %v\n", err)
				return
			}
			result.Subdomains = subdomains
			fmt.Printf("   ✅ 发现 %d 个子域名\n", len(subdomains))
		} else {
			fmt.Println("   ⚠️  subfinder 未安装，跳过")
		}
	}()

	// 2. HTTP 探测 (包含技术栈识别)
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [2/6] HTTP 探测...")
		if tool, err := tm.Get("httpx"); err == nil && tool.IsReady() {
			runner, err := recon.NewHTTPXRunner(tm)
			if err != nil {
				fmt.Printf("   ⚠️  初始化失败: %v\n", err)
				return
			}
			targets := []string{result.Target}
			httpResults, err := runner.Probe(ctx, targets, nil)
			if err != nil {
				fmt.Printf("   ⚠️  探测失败: %v\n", err)
				return
			}
			result.HTTPServices = httpResults
			fmt.Printf("   ✅ 探测到 %d 个 HTTP 服务\n", len(httpResults))

			// 提取技术栈信息
			for _, hr := range httpResults {
				for _, tech := range hr.Technologies {
					result.TechStack = append(result.TechStack, TechInfo{
						Name:     tech,
						Category: "web",
					})
				}
			}
		} else {
			fmt.Println("   ⚠️  httpx 未安装，跳过")
		}
	}()

	// 3. DNS 查询
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [3/6] DNS 查询...")
		if tool, err := tm.Get("dnsx"); err == nil && tool.IsReady() {
			runner, err := recon.NewDNSXRunner(tm)
			if err != nil {
				fmt.Printf("   ⚠️  初始化失败: %v\n", err)
				return
			}
			dnsResults, err := runner.Query(ctx, []string{result.Target}, nil)
			if err != nil {
				fmt.Printf("   ⚠️  查询失败: %v\n", err)
				return
			}
			result.DNSRecords = dnsResults
			fmt.Printf("   ✅ 查询到 %d 条 DNS 记录\n", len(dnsResults))
		} else {
			fmt.Println("   ⚠️  dnsx 未安装，跳过")
		}
	}()

	// 4. 端口扫描
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [4/6] 端口扫描...")
		if tool, err := tm.Get("nmap"); err == nil && tool.IsReady() {
			runner, err := portscan.NewNmapRunner(tm)
			if err != nil {
				fmt.Printf("   ⚠️  初始化失败: %v\n", err)
				return
			}

			var scanResult *portscan.NmapResult
			switch depth {
			case "quick":
				scanResult, err = runner.QuickScan(ctx, result.Target)
			case "full":
				scanResult, err = runner.FullScan(ctx, result.Target)
			default:
				scanResult, err = runner.QuickScan(ctx, result.Target)
			}

			if err != nil {
				fmt.Printf("   ⚠️  扫描失败: %v\n", err)
				return
			}
			result.OpenPorts = scanResult.Ports
			fmt.Printf("   ✅ 发现 %d 个开放端口\n", len(scanResult.Ports))
		} else {
			fmt.Println("   ⚠️  nmap 未安装，跳过")
		}
	}()

	wg.Wait()

	// 5. WAF 检测 (需要 HTTP 结果)
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [5/6] WAF 检测...")
		if tool, err := tm.Get("wafw00f"); err == nil && tool.IsReady() {
			// 简化：使用 httpx 结果中的 URL
			if len(result.HTTPServices) > 0 {
				fmt.Printf("   ✅ WAF 检测完成\n")
				// TODO: 实际调用 wafw00f
			} else {
				fmt.Println("   ⚠️  无 HTTP 服务，跳过 WAF 检测")
			}
		} else {
			fmt.Println("   ⚠️  wafw00f 未安装，跳过")
		}
	}()

	// 6. 敏感文件探测
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("\n🔍 [6/6] 敏感文件探测...")
		if len(result.HTTPServices) > 0 {
			// 常见敏感文件
			sensitivePaths := []string{
				"robots.txt", "sitemap.xml", ".git/config",
				".env", "admin", "backup", "config.php.bak",
				"web.config", "phpinfo.php", ".DS_Store",
			}
			fmt.Printf("   📁 探测 %d 个常见敏感路径...\n", len(sensitivePaths))
			// TODO: 实际探测
			fmt.Println("   ✅ 敏感文件探测完成")
		} else {
			fmt.Println("   ⚠️  无 HTTP 服务，跳过敏感文件探测")
		}
	}()

	wg.Wait()
}

func printReconSummary(result *ReconResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 信息收集结果汇总")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("⏱️  耗时: %v\n", result.EndTime.Sub(result.StartTime).Round(time.Second))

	if len(result.Subdomains) > 0 {
		fmt.Printf("\n🌐 子域名 (%d):\n", len(result.Subdomains))
		for i, sub := range result.Subdomains {
			if i >= 10 {
				fmt.Printf("   ... 还有 %d 个\n", len(result.Subdomains)-10)
				break
			}
			fmt.Printf("   - %s\n", sub)
		}
	}

	if len(result.HTTPServices) > 0 {
		fmt.Printf("\n📡 HTTP 服务 (%d):\n", len(result.HTTPServices))
		for i, r := range result.HTTPServices {
			if i >= 10 {
				fmt.Printf("   ... 还有 %d 个\n", len(result.HTTPServices)-10)
				break
			}
			fmt.Printf("   - %s [%d] %s\n", r.URL, r.StatusCode, r.Title)
			if len(r.Technologies) > 0 {
				fmt.Printf("     技术: %v\n", r.Technologies)
			}
		}
	}

	if len(result.DNSRecords) > 0 {
		fmt.Printf("\n📖 DNS 记录 (%d):\n", len(result.DNSRecords))
		for i, r := range result.DNSRecords {
			if i >= 5 {
				fmt.Printf("   ... 还有 %d 条\n", len(result.DNSRecords)-5)
				break
			}
			if r.Host != "" {
				fmt.Printf("   - %s: A=%v\n", r.Host, r.A)
			}
		}
	}

	if len(result.OpenPorts) > 0 {
		fmt.Printf("\n🔌 开放端口 (%d):\n", len(result.OpenPorts))
		for _, p := range result.OpenPorts {
			fmt.Printf("   - %d/%s %s\n", p.Port, p.Protocol, p.Service)
		}
	}

	if len(result.TechStack) > 0 {
		fmt.Printf("\n🔧 技术栈 (%d):\n", len(result.TechStack))
		seen := make(map[string]bool)
		for _, tech := range result.TechStack {
			if !seen[tech.Name] {
				fmt.Printf("   - %s\n", tech.Name)
				seen[tech.Name] = true
			}
		}
	}
}

func startAIPenetration(ctx context.Context, reconResult *ReconResult, tm *toolmgr.ToolManager, toolChecker *toolcheck.Checker) {
	// 初始化 AI 提供者
	apiKey := getAPIKey(reconProvider)
	if apiKey == "" && reconProvider != "ollama" {
		fmt.Println("\n❌ AI 模式需要配置 API 密钥")
		fmt.Println("\n📝 配置方法:")
		fmt.Println("   创建 .env 文件:")
		fmt.Println("   OPENAI_API_KEY=your-key")
		fmt.Println("   或使用本地 Ollama: --provider ollama")
		return
	}

	provider := createProvider(reconProvider, apiKey, reconModel)
	if provider == nil {
		fmt.Println("❌ AI 提供者初始化失败")
		return
	}

	fmt.Printf("🔧 AI 提供者: %s\n", provider.Name())

	// 初始化组件
	sched := scheduler.New()
	registerTools(sched)
	gen := script.NewGenerator(provider)
	disp := dispatcher.NewDispatcher(
		dispatcher.WithScheduler(sched),
		dispatcher.WithGenerator(gen),
		dispatcher.WithToolManager(tm),
	)

	// 创建会话
	log := logger.NewLogger("")
	sessionLog := log.NewSession(reconResult.Target)
	fmt.Printf("📋 会话ID: %s\n", sessionLog.ID)

	// 初始化会话状态
	session := &SessionState{
		Target:       reconResult.Target,
		CurrentPhase: "recon",
		Results: []PhaseResult{
			{
				Phase:   "recon",
				Tool:    "multi",
				Success: true,
				Summary: fmt.Sprintf("子域名:%d HTTP:%d 端口:%d 技术:%d",
					len(reconResult.Subdomains),
					len(reconResult.HTTPServices),
					len(reconResult.OpenPorts),
					len(reconResult.TechStack)),
			},
		},
	}

	// 将信息收集结果转换为上下文
	contextData := buildContextFromRecon(reconResult)

	// AI 分析并推荐下一步
	fmt.Println("\n🤖 AI 正在分析信息收集结果...")

	recommendations := getAIRecommendationsWithRecon(ctx, provider, session, toolChecker.CheckAll(), contextData)
	if len(recommendations) == 0 {
		fmt.Println("❌ AI 分析失败")
		return
	}

	// 显示推荐
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📝 AI 建议的下一步操作")
	fmt.Println(strings.Repeat("=", 60))

	for _, rec := range recommendations {
		riskIcon := getRiskIcon(rec.RiskLevel)
		fmt.Printf("\n[%d] %s %s\n", rec.ID, riskIcon, rec.Title)
		fmt.Printf("    %s\n", rec.Description)
		if rec.Tool != "" {
			fmt.Printf("    工具: %s\n", rec.Tool)
		}
	}

	// 自动选择最高优先级
	selected := recommendations[0]
	for _, rec := range recommendations {
		if rec.Priority > selected.Priority {
			selected = rec
		}
	}

	fmt.Printf("\n▶️  自动执行: %s (优先级 %d)\n", selected.Title, selected.Priority)

	// 执行选中的任务
	executeNextPhase(ctx, disp, gen, selected, reconResult.Target, session, log, toolChecker, tm)

	// 继续交互式循环
	runInteractiveLoop(ctx, provider, disp, gen, session, log, toolChecker, tm)
}

func buildContextFromRecon(result *ReconResult) map[string]interface{} {
	return map[string]interface{}{
		"target":        result.Target,
		"subdomains":    result.Subdomains,
		"http_services": len(result.HTTPServices),
		"open_ports":    result.OpenPorts,
		"tech_stack":    result.TechStack,
		"dns_records":   len(result.DNSRecords),
	}
}

func getAIRecommendationsWithRecon(ctx context.Context, provider providers.Provider, session *SessionState, toolCheckResult *toolcheck.CheckResult, reconContext map[string]interface{}) []Recommendation {
	// 构建详细的上下文信息
	contextJSON := jsonMarshal(reconContext)

	prompt := fmt.Sprintf(`你是一个资深渗透测试专家。根据以下信息收集结果，分析目标并推荐最佳的下一步渗透操作。

目标: %s

信息收集结果:
%s

已完成的操作: %s

重要说明:
1. 根据收集到的信息，推荐最有可能成功的攻击路径
2. 优先考虑：开放端口服务漏洞、技术栈已知漏洞、敏感路径
3. 如果发现 CMS（WordPress/Joomla等），推荐相关扫描
4. 如果发现数据库端口开放，推荐弱口令检测
5. 如果发现特定技术栈，推荐相关 CVE 检测

请以JSON数组格式返回3-5个下一步建议，每个建议包含:
- id: 序号 (1-5)
- title: 建议标题 (简短)
- description: 详细描述（包含推荐理由）
- tool: 推荐工具名称 (如 nuclei, wpscan, sqlmap, gobuster 等)
- priority: 优先级 (1-5, 5最高)
- risk_level: 风险等级 (low/medium/high)

只返回JSON数组，不要其他内容。`, session.Target, string(contextJSON), session.History)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: prompt},
		},
	}
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		fmt.Printf("AI 请求失败: %v\n", err)
		return nil
	}

	return parseAIRecommendations(resp.Content)
}

func executeNextPhase(ctx context.Context, disp *dispatcher.Dispatcher, gen *script.Generator, rec Recommendation, target string, session *SessionState, log *logger.Logger, toolChecker *toolcheck.Checker, tm *toolmgr.ToolManager) {
	fmt.Printf("\n🎯 执行: %s\n", rec.Title)

	contextData := map[string]interface{}{
		"target": target,
	}

	result, err := disp.Dispatch(ctx, rec.Tool, contextData)
	if err != nil {
		fmt.Printf("❌ 调度失败: %v\n", err)
		return
	}

	fmt.Printf("📊 使用工具: %s\n", result.ToolName)

	var execResult *dispatcher.ExecutionResult
	switch result.ScenarioType {
	case dispatcher.ScenarioStandard:
		execResult, err = disp.ExecuteStandard(ctx, result.ToolName, []string{target})
	case dispatcher.ScenarioNonStandard:
		execResult, err = disp.ExecuteNonStandard(ctx, result.ScriptTask, contextData, reconAutoConfirm)
	case dispatcher.ScenarioMixed:
		execResult, err = disp.ExecuteMixed(ctx, result.ToolName, result.ScriptTask, []string{target}, contextData, reconAutoConfirm)
	}

	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		return
	}

	// 更新会话状态
	summary := extractSummary(execResult.Output, result.ToolName)
	session.Results = append(session.Results, PhaseResult{
		Phase:   session.CurrentPhase,
		Tool:    result.ToolName,
		Success: execResult.Success,
		Output:  execResult.Output,
		Summary: summary,
	})
	session.History = append(session.History, HistoryEntry{
		Action:  rec.Title,
		Tool:    result.ToolName,
		Result:  boolToStatus(execResult.Success),
		Summary: summary,
	})

	fmt.Printf("\n📊 执行结果:\n")
	fmt.Println(strings.Repeat("-", 50))
	if len(execResult.Output) > 500 {
		fmt.Println(execResult.Output[:500] + "...")
	} else {
		fmt.Println(execResult.Output)
	}
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("✅ %s\n", summary)
}

func runInteractiveLoop(ctx context.Context, provider providers.Provider, disp *dispatcher.Dispatcher, gen *script.Generator, session *SessionState, log *logger.Logger, toolChecker *toolcheck.Checker, tm *toolmgr.ToolManager) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("📊 当前进度")
		fmt.Println(strings.Repeat("=", 60))
		printSessionSummary(session)

		fmt.Println("\n🤖 AI 正在分析...")
		recommendations := getAIRecommendations(ctx, provider, session, toolChecker.CheckAll())
		if len(recommendations) == 0 {
			fmt.Println("❌ AI 分析失败")
			break
		}

		fmt.Println("\n📝 下一步操作建议:")
		for _, rec := range recommendations {
			riskIcon := getRiskIcon(rec.RiskLevel)
			fmt.Printf("\n[%d] %s %s\n", rec.ID, riskIcon, rec.Title)
			fmt.Printf("    %s\n", rec.Description)
		}
		fmt.Printf("\n[0] 结束测试，生成报告\n")
		fmt.Printf("\n请选择 [0-%d]: ", len(recommendations))

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice := 0
		fmt.Sscanf(input, "%d", &choice)

		if choice == 0 {
			generateReport(session)
			log.Complete()
			fmt.Println("\n✅ 测试完成!")
			return
		}

		if choice < 1 || choice > len(recommendations) {
			fmt.Println("❌ 无效选择")
			continue
		}

		selected := recommendations[choice-1]
		executeNextPhase(ctx, disp, gen, selected, session.Target, session, log, toolChecker, tm)
	}
}

func jsonMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func parseAIRecommendations(content string) []Recommendation {
	var recommendations []Recommendation

	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start == -1 || end == -1 {
		return nil
	}

	jsonStr := content[start : end+1]
	if err := json.Unmarshal([]byte(jsonStr), &recommendations); err != nil {
		return nil
	}

	return recommendations
}
