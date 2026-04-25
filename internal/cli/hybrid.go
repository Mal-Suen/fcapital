package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
	"github.com/Mal-Suen/fcapital/internal/core/dispatcher"
	"github.com/Mal-Suen/fcapital/internal/core/merger"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
	"github.com/Mal-Suen/fcapital/internal/core/script"
	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/Mal-Suen/fcapital/internal/pkg/logger"
	"github.com/Mal-Suen/fcapital/internal/pkg/toolcheck"
	"github.com/spf13/cobra"
)

// hybridCmd represents the hybrid mode command
var hybridCmd = &cobra.Command{
	Use:   "hybrid",
	Short: "混合模式执行",
	Long: `使用混合模式执行渗透测试任务。

混合模式会自动判断任务类型：
- 标准任务：使用成熟工具（nmap/nuclei/sqlmap等）
- 非标准任务：AI生成辅助脚本
- 混合任务：工具+脚本组合

示例:
  fcapital hybrid run "port scan" -t example.com
  fcapital hybrid run "custom poc for CVE-2024-xxxx" -t example.com
  fcapital hybrid analyze "waf bypass for example.com"`,
}

var (
	hybridTarget      string
	hybridAutoConfirm bool
	hybridLanguage    string
	hybridProvider    string
	hybridModel       string
	hybridSession     string // 恢复会话ID
	hybridLogDir      string // 日志目录
)

func init() {
	rootCmd.AddCommand(hybridCmd)

	hybridCmd.AddCommand(hybridRunCmd)
	hybridCmd.AddCommand(hybridAnalyzeCmd)
	hybridCmd.AddCommand(hybridGenerateCmd)
	hybridCmd.AddCommand(hybridListCmd)
	hybridCmd.AddCommand(hybridResumeCmd)

	// Global flags for hybrid command
	hybridCmd.PersistentFlags().StringVar(&hybridProvider, "provider", "openai", "AI提供者 (openai/deepseek/ollama)")
	hybridCmd.PersistentFlags().StringVar(&hybridModel, "model", "", "AI模型名称")
	hybridCmd.PersistentFlags().StringVar(&hybridLogDir, "log-dir", "", "日志目录 (默认 ~/.fcapital/sessions)")

	hybridRunCmd.Flags().StringVarP(&hybridTarget, "target", "t", "", "目标域名或IP")
	hybridRunCmd.Flags().BoolVar(&hybridAutoConfirm, "auto-confirm", false, "自动确认脚本执行")
	hybridRunCmd.Flags().StringVar(&hybridLanguage, "language", "python", "脚本语言")

	hybridResumeCmd.Flags().BoolVar(&hybridAutoConfirm, "auto-confirm", false, "自动确认脚本执行")
	hybridResumeCmd.Flags().StringVar(&hybridLanguage, "language", "python", "脚本语言")

	hybridGenerateCmd.Flags().StringVar(&hybridLanguage, "language", "python", "脚本语言")
}

// Recommendation AI 建议
type Recommendation struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tool        string `json:"tool"`
	Command     string `json:"command"`
	Priority    int    `json:"priority"`
	RiskLevel   string `json:"risk_level"`
}

// SessionState 会话状态
type SessionState struct {
	Target       string
	CurrentPhase string
	Results      []PhaseResult
	History      []HistoryEntry
}

// PhaseResult 阶段结果
type PhaseResult struct {
	Phase   string `json:"phase"`
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Summary string `json:"summary"`
}

// HistoryEntry 历史记录
type HistoryEntry struct {
	Action  string `json:"action"`
	Tool    string `json:"tool"`
	Result  string `json:"result"`
	Summary string `json:"summary"`
}

// hybridRunCmd runs a task in hybrid mode
var hybridRunCmd = &cobra.Command{
	Use:   "run [task]",
	Short: "运行混合模式任务",
	Long: `运行混合模式任务，自动选择最佳执行方式。

示例:
  fcapital hybrid run "port scan" -t example.com
  fcapital hybrid run "waf bypass" -t example.com --auto-confirm`,
	Args: cobra.MinimumNArgs(1),
	Run:  runHybridTask,
}

func runHybridTask(cmd *cobra.Command, args []string) {
	task := strings.Join(args, " ")
	printBanner()

	// Initialize logger
	log := logger.NewLogger(hybridLogDir)

	// Create new session
	sessionLog := log.NewSession(hybridTarget)
	fmt.Printf("📋 会话ID: %s\n", sessionLog.ID)
	fmt.Printf("📁 日志文件: %s\n\n", log.GetLogPath())

	// Setup signal handler for graceful interrupt
	sigChan := make(chan os.Signal, 1)
	if runtime.GOOS == "windows" {
		// Windows only supports os.Interrupt
		signal.Notify(sigChan, os.Interrupt)
	} else {
		// Unix-like systems support both
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	}
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  收到中断信号，正在保存会话...")
		log.Interrupt()
		fmt.Printf("✅ 会话已保存，可使用以下命令恢复:\n")
		fmt.Printf("   fcapital hybrid resume %s\n", sessionLog.ID)
		os.Exit(0)
	}()

	// Run the session
	runSession(task, log, "")
}

// hybridResumeCmd resumes an interrupted session
var hybridResumeCmd = &cobra.Command{
	Use:   "resume [session-id]",
	Short: "恢复中断的会话",
	Long: `恢复之前中断的渗透测试会话。

示例:
  fcapital hybrid resume session_20240101_120000
  fcapital hybrid resume session_20240101_120000 --auto-confirm`,
	Args: cobra.ExactArgs(1),
	Run:  runHybridResume,
}

func runHybridResume(cmd *cobra.Command, args []string) {
	sessionID := args[0]
	printBanner()

	// Initialize logger
	log := logger.NewLogger(hybridLogDir)

	// Load existing session
	sessionLog, err := log.LoadSession(sessionID)
	if err != nil {
		fmt.Printf("❌ 加载会话失败: %v\n", err)
		return
	}

	fmt.Printf("📋 会话ID: %s\n", sessionLog.ID)
	fmt.Printf("📍 目标: %s\n", sessionLog.Target)
	fmt.Printf("📊 状态: %s\n", sessionLog.Status)
	fmt.Printf("📁 日志文件: %s\n\n", log.GetLogPath())

	// Show previous results
	if len(sessionLog.History) > 0 {
		fmt.Println("📜 已完成的操作:")
		for i, h := range sessionLog.History {
			fmt.Printf("  [%d] %s - %s: %s\n", i+1, h.Action, h.Tool, h.Summary)
		}
		fmt.Println()
	}

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	if runtime.GOOS == "windows" {
		signal.Notify(sigChan, os.Interrupt)
	} else {
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	}
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  收到中断信号，正在保存会话...")
		log.Interrupt()
		fmt.Printf("✅ 会话已保存，可使用以下命令恢复:\n")
		fmt.Printf("   fcapital hybrid resume %s\n", sessionLog.ID)
		os.Exit(0)
	}()

	// Resume session - start from AI recommendations
	runSession("", log, sessionLog.NextAction)
}

// hybridListCmd lists all sessions
var hybridListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有会话",
	Long:  `列出所有渗透测试会话，包括已完成和中断的。`,
	Run:   runHybridList,
}

func runHybridList(cmd *cobra.Command, args []string) {
	printBanner()

	sessions, err := logger.ListSessions(hybridLogDir)
	if err != nil {
		fmt.Printf("❌ 读取会话列表失败: %v\n", err)
		return
	}

	if len(sessions) == 0 {
		fmt.Println("📭 没有找到任何会话")
		return
	}

	fmt.Println("📋 会话列表:\n")
	fmt.Printf("%-30s %-20s %-10s %-15s %s\n", "会话ID", "开始时间", "状态", "目标", "操作数")
	fmt.Println(strings.Repeat("-", 90))

	for _, s := range sessions {
		statusIcon := "✅"
		if s.Status == "interrupted" {
			statusIcon = "⚠️ "
		} else if s.Status == "running" {
			statusIcon = "🔄"
		}
		fmt.Printf("%-30s %-20s %-10s %-15s %d\n",
			s.ID,
			s.StartTime.Format("2006-01-02 15:04:05"),
			statusIcon+" "+s.Status,
			truncate(s.Target, 15),
			len(s.History),
		)
	}

	fmt.Println("\n💡 使用 'fcapital hybrid resume <session-id>' 恢复会话")
}

// runSession runs the interactive session loop
func runSession(initialTask string, log *logger.Logger, nextAction string) {
	sessionLog := log.GetSession()

	// Initialize components
	sched := scheduler.New()
	registerTools(sched)
	tm := toolmgr.NewToolManager()

	// Initialize AI provider
	apiKey := getAPIKey(hybridProvider)
	var provider providers.Provider
	if apiKey != "" || hybridProvider == "ollama" {
		provider = createProvider(hybridProvider, apiKey, hybridModel)
	}

	// Debug: show provider info
	if provider != nil {
		fmt.Printf("🔧 AI 提供者: %s\n", provider.Name())
		providerUpper := strings.ToUpper(hybridProvider)
		baseURL := os.Getenv(fmt.Sprintf("%s_BASE_URL", providerUpper))
		if baseURL == "" {
			baseURL = os.Getenv(fmt.Sprintf("%s_BASEURL", providerUpper))
		}
		fmt.Printf("🔧 API Base URL: %s\n", baseURL)
		model := hybridModel
		if model == "" {
			model = os.Getenv("AI_MODEL")
		}
		fmt.Printf("🔧 AI Model: %s\n", model)
	}

	if provider == nil {
		fmt.Println("❌ 交互式模式需要AI支持，请配置API密钥")
		fmt.Println()
		fmt.Println("📝 配置方法:")
		fmt.Println("   1. 在项目根目录创建 .env 文件:")
		fmt.Println("      OPENAI_API_KEY=your-api-key")
		fmt.Println("      OPENAI_BASE_URL=https://api.openai.com/v2  # 可选")
		fmt.Println("      AI_MODEL=gpt-4  # 可选")
		fmt.Println()
		fmt.Println("   2. 或设置环境变量:")
		fmt.Println("      export OPENAI_API_KEY=your-api-key  # Linux/macOS")
		fmt.Println("      set OPENAI_API_KEY=your-api-key     # Windows CMD")
		fmt.Println("      $env:OPENAI_API_KEY=\"your-key\"      # Windows PowerShell")
		fmt.Println()
		fmt.Println("   3. 或使用其他提供者:")
		fmt.Println("      --provider deepseek  # DeepSeek API")
		fmt.Println("      --provider ollama    # 本地 Ollama (无需API密钥)")
		fmt.Println()
		fmt.Println("💡 .env 文件搜索路径:")
		fmt.Println("   - 当前工作目录")
		fmt.Println("   - 可执行文件所在目录及其上级目录")
		fmt.Println("   - ~/.fcapital/.env")
		return
	}

	// Initialize script generator
	gen := script.NewGenerator(provider)

	// Initialize dispatcher
	disp := dispatcher.NewDispatcher(
		dispatcher.WithScheduler(sched),
		dispatcher.WithGenerator(gen),
		dispatcher.WithToolManager(tm),
	)

	// Initialize tool checker and check available tools
	toolChecker := toolcheck.NewChecker()
	toolCheckResult := toolChecker.CheckAll()
	
	fmt.Printf("\n🔧 工具检测: 已安装 %d/%d 个工具\n", toolCheckResult.InstalledCount, toolCheckResult.TotalCount)
	if len(toolCheckResult.Available) > 0 {
		fmt.Printf("   可用: %s\n", toolChecker.FormatAvailableTools(toolCheckResult))
	}

	// Initialize session state from log
	session := &SessionState{
		Target:       sessionLog.Target,
		CurrentPhase: sessionLog.CurrentPhase,
		Results:      convertResults(sessionLog.Results),
		History:      convertHistory(sessionLog.History),
	}

	ctx := context.Background()

	// Execute initial task if provided
	if initialTask != "" {
		fmt.Printf("🎯 初始任务: %s\n", initialTask)
		fmt.Printf("📍 目标: %s\n\n", session.Target)

		execResult := executeTaskWithLog(ctx, disp, gen, initialTask, session.Target, session, log, toolChecker)
		if execResult == nil {
			return
		}
	} else {
		fmt.Printf("📍 目标: %s\n", session.Target)
		fmt.Printf("📊 当前阶段: %s\n\n", session.CurrentPhase)

		if len(session.History) > 0 {
			fmt.Println("📜 已完成的操作:")
			for i, h := range session.History {
				fmt.Printf("  [%d] %s - %s\n", i+1, h.Action, h.Summary)
			}
			fmt.Println()
		}
	}

	// Interactive loop
	reader := bufio.NewReader(os.Stdin)
	for {
		// 1. Format results for AI
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("📊 执行结果摘要")
		fmt.Println(strings.Repeat("=", 60))
		printSessionSummary(session)

		// 2. Send to AI for analysis (with tool availability info)
		fmt.Println("\n🤖 AI 正在分析结果...")
		recommendations := getAIRecommendations(ctx, provider, session, toolCheckResult)
		if len(recommendations) == 0 {
			fmt.Println("❌ AI 分析失败，无法生成建议")
			log.Interrupt()
			return
		}

		// 3. Display recommendations with tool availability
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("📝 AI 建议的下一步操作")
		fmt.Println(strings.Repeat("=", 60))
		for _, rec := range recommendations {
			riskIcon := getRiskIcon(rec.RiskLevel)
			toolStatus := ""
			if rec.Tool != "" && !toolChecker.IsToolAvailable(rec.Tool) {
				toolStatus = " ⚠️[未安装]"
			}
			fmt.Printf("\n[%d] %s %s%s\n", rec.ID, riskIcon, rec.Title, toolStatus)
			fmt.Printf("    %s\n", rec.Description)
			if rec.Tool != "" {
				fmt.Printf("    工具: %s\n", rec.Tool)
			}
			fmt.Printf("    风险: %s │ 优先级: %s\n", rec.RiskLevel, getPriorityText(rec.Priority))
		}
		fmt.Printf("\n[0] 📄 结束测试，生成报告\n")
		fmt.Printf("\n请选择下一步操作 [0-%d]: ", len(recommendations))

		// 4. Get user choice
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice := 0
		fmt.Sscanf(input, "%d", &choice)

		if choice == 0 {
			fmt.Println("\n📄 正在生成测试报告...")
			generateReport(session)
			log.Complete()
			fmt.Println("✅ 测试完成!")
			fmt.Printf("📁 会话日志: %s\n", log.GetLogPath())
			return
		}

		if choice < 1 || choice > len(recommendations) {
			fmt.Println("❌ 无效选择，请重新输入")
			continue
		}

		// 5. Execute selected recommendation
		selected := recommendations[choice-1]
		fmt.Printf("\n▶️ 执行: %s\n", selected.Title)

		// Check if tool is available
		if selected.Tool != "" && !toolChecker.IsToolAvailable(selected.Tool) {
			// Tool not installed - handle installation
			installed := handleToolInstallation(selected.Tool, reader, provider, ctx)
			if !installed {
				fmt.Println("❌ 工具安装失败，跳过此操作")
				continue
			}
			// Re-check tools after installation
			toolCheckResult = toolChecker.CheckAll()
		}

		// Save next action for recovery
		log.SetNextAction(selected.Title)

		nextTask := selected.Tool
		if nextTask == "" {
			nextTask = selected.Title
		}

		executeTaskWithLog(ctx, disp, gen, nextTask, session.Target, session, log, toolChecker)
	}
}

// executeTaskWithLog executes a task with logging
func executeTaskWithLog(ctx context.Context, disp *dispatcher.Dispatcher, gen *script.Generator, task string, target string, session *SessionState, log *logger.Logger, toolChecker *toolcheck.Checker) *dispatcher.ExecutionResult {
	contextData := map[string]interface{}{
		"target":             target,
		"preferred_language": hybridLanguage,
	}

	// Dispatch decision
	result, err := disp.Dispatch(ctx, task, contextData)
	if err != nil {
		fmt.Printf("❌ 调度失败: %v\n", err)
		log.Log(logger.LevelError, session.CurrentPhase, "", fmt.Sprintf("调度失败: %v", err), false, "")
		return nil
	}

	// Display decision
	fmt.Println("\n📊 调度决策:")
	fmt.Printf("   场景类型: %s\n", result.ScenarioType)
	if result.ToolName != "" {
		fmt.Printf("   使用工具: %s\n", result.ToolName)
		if toolPath := disp.GetToolPath(result.ToolName); toolPath != "" {
			fmt.Printf("   工具路径: %s\n", toolPath)
		}
	}
	fmt.Printf("   决策理由: %s\n\n", result.Reasoning)

	// Log dispatch decision
	log.Log(logger.LevelInfo, session.CurrentPhase, result.ToolName, result.Reasoning, true, "")

	// Execute based on scenario type
	var execResult *dispatcher.ExecutionResult
	switch result.ScenarioType {
	case dispatcher.ScenarioStandard:
		execResult, err = disp.ExecuteStandard(ctx, result.ToolName, []string{target})

	case dispatcher.ScenarioNonStandard:
		execResult, err = disp.ExecuteNonStandard(ctx, result.ScriptTask, contextData, hybridAutoConfirm)

	case dispatcher.ScenarioMixed:
		execResult, err = disp.ExecuteMixed(ctx, result.ToolName, result.ScriptTask, []string{target}, contextData, hybridAutoConfirm)
	}

	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		log.Log(logger.LevelError, session.CurrentPhase, result.ToolName, fmt.Sprintf("执行失败: %v", err), false, "")
		return nil
	}

	// Extract summary
	summary := extractSummary(execResult.Output, result.ToolName)

	// Record result
	phaseResult := PhaseResult{
		Phase:   session.CurrentPhase,
		Tool:    result.ToolName,
		Success: execResult.Success,
		Output:  execResult.Output,
		Summary: summary,
	}
	session.Results = append(session.Results, phaseResult)

	session.History = append(session.History, HistoryEntry{
		Action:  task,
		Tool:    result.ToolName,
		Result:  boolToStatus(execResult.Success),
		Summary: summary,
	})

	// Log the result
	log.RecordResult(session.CurrentPhase, result.ToolName, execResult.Success, execResult.Output, summary)
	log.RecordHistory(task, result.ToolName, boolToStatus(execResult.Success), summary)

	return execResult
}

// handleToolInstallation handles tool installation when tool is not available
func handleToolInstallation(toolName string, reader *bufio.Reader, provider providers.Provider, ctx context.Context) bool {
	fmt.Printf("\n⚠️  工具 '%s' 未安装\n", toolName)
	
	// Show installation instructions
	instructions := toolcheck.GetInstallInstructions(toolName)
	fmt.Printf("\n📦 安装说明:\n%s\n", instructions)
	
	// Try auto install
	fmt.Printf("\n尝试自动安装? (y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "y" || input == "yes" {
		fmt.Printf("\n🔧 正在尝试自动安装 %s...\n", toolName)
		success, msg := toolcheck.TryAutoInstall(toolName)
		fmt.Printf("%s\n", msg)
		
		if success {
			return true
		}
	}
	
	// Ask AI for alternative approach
	fmt.Printf("\n是否让 AI 推荐替代方案? (y/n): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "y" || input == "yes" {
		fmt.Printf("\n🤖 正在请求 AI 推荐替代方案...\n")
		alternative := getAIAlternativeTool(ctx, provider, toolName)
		if alternative != "" {
			fmt.Printf("\n💡 AI 建议: %s\n", alternative)
			return false
		}
	}
	
	// Manual installation guide
	fmt.Printf("\n请手动安装后按 Enter 继续，或输入 'skip' 跳过: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input == "skip" {
		return false
	}
	
	// Re-check if tool is now available
	checker := toolcheck.NewChecker()
	if checker.IsToolAvailable(toolName) {
		fmt.Printf("✅ 检测到 %s 已安装\n", toolName)
		return true
	}
	
	fmt.Printf("❌ 仍未检测到 %s\n", toolName)
	return false
}

// getAIAlternativeTool asks AI for alternative tool
func getAIAlternativeTool(ctx context.Context, provider providers.Provider, missingTool string) string {
	prompt := fmt.Sprintf(`工具 '%s' 不可用，请推荐一个替代工具或方法。

要求:
1. 推荐一个功能相似的工具
2. 说明为什么这个工具可以作为替代
3. 如果没有合适的替代，说明如何手动完成相同任务

请用简洁的中文回答，不超过200字。`, missingTool)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: prompt},
		},
	}
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return ""
	}
	
	return resp.Content
}

// convertResults converts logger.PhaseResult to SessionState.PhaseResult
func convertResults(results []logger.PhaseResult) []PhaseResult {
	var converted []PhaseResult
	for _, r := range results {
		converted = append(converted, PhaseResult{
			Phase:   r.Phase,
			Tool:    r.Tool,
			Success: r.Success,
			Output:  r.Output,
			Summary: r.Summary,
		})
	}
	return converted
}

// convertHistory converts logger.HistoryEntry to SessionState.HistoryEntry
func convertHistory(history []logger.HistoryEntry) []HistoryEntry {
	var converted []HistoryEntry
	for _, h := range history {
		converted = append(converted, HistoryEntry{
			Action:  h.Action,
			Tool:    h.Tool,
			Result:  h.Result,
			Summary: h.Summary,
		})
	}
	return converted
}

// truncate truncates a string
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// printSessionSummary prints a summary of the session
func printSessionSummary(session *SessionState) {
	for i, result := range session.Results {
		status := "❌"
		if result.Success {
			status = "✅"
		}
		fmt.Printf("%s [%d] %s: %s\n", status, i+1, result.Tool, result.Summary)
	}
}

// getAIRecommendations gets AI recommendations based on current results
func getAIRecommendations(ctx context.Context, provider providers.Provider, session *SessionState, toolCheckResult *toolcheck.CheckResult) []Recommendation {
	// Build context for AI
	contextJSON, _ := json.Marshal(map[string]interface{}{
		"target":  session.Target,
		"phase":   session.CurrentPhase,
		"results": session.Results,
	})

	// Format available tools for AI (仅供参考，不限制推荐)
	availableTools := ""
	if toolCheckResult != nil {
		var toolNames []string
		for _, tool := range toolCheckResult.Available {
			toolNames = append(toolNames, tool.Name)
		}
		availableTools = fmt.Sprintf("\n本地已安装工具: %s", strings.Join(toolNames, ", "))
	}

	prompt := fmt.Sprintf(`你是一个渗透测试专家。根据以下测试结果，分析目标并推荐最佳的下一步操作。

目标: %s
当前阶段: %s
%s

测试结果:
%s

重要说明:
1. 推荐最适合当前任务的工具，不受本地已安装工具限制
2. 如果推荐的最佳工具未安装，系统会自动处理安装
3. 在 description 中简要说明为什么推荐这个工具
4. 不要推荐与已执行操作重复的工具

请以JSON数组格式返回3-5个下一步建议，每个建议包含:
- id: 序号 (1-5)
- title: 建议标题 (简短)
- description: 详细描述（包含推荐理由）
- tool: 推荐工具名称 (如 nmap, nuclei, wpscan, gobuster 等)
- priority: 优先级 (1-5, 5最高)
- risk_level: 风险等级 (low/medium/high)

只返回JSON数组，不要其他内容。示例格式:
[
  {"id": 1, "title": "WordPress漏洞扫描", "description": "使用wpscan检测WordPress插件和主题漏洞，这是最专业的WordPress扫描工具", "tool": "wpscan", "priority": 5, "risk_level": "low"},
  {"id": 2, "title": "目录扫描", "description": "使用ffuf进行目录爆破，速度快且支持多种模式", "tool": "ffuf", "priority": 3, "risk_level": "low"}
]`, session.Target, session.CurrentPhase, availableTools, string(contextJSON))

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

	// Parse AI response
	var recommendations []Recommendation
	content := resp.Content

	// Extract JSON array from response
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start == -1 || end == -1 {
		fmt.Printf("AI 返回格式错误: %s\n", content)
		return nil
	}

	jsonStr := content[start : end+1]
	if err := json.Unmarshal([]byte(jsonStr), &recommendations); err != nil {
		fmt.Printf("解析 AI 响应失败: %v\n", err)
		return nil
	}

	return recommendations
}

// extractSummary extracts a brief summary from tool output
func extractSummary(output string, tool string) string {
	lines := strings.Split(output, "\n")
	summary := ""

	switch tool {
	case "nmap":
		for _, line := range lines {
			if strings.Contains(line, "open") {
				summary += strings.TrimSpace(line) + "; "
			}
		}
		if len(summary) > 100 {
			summary = summary[:100] + "..."
		}

	case "httpx":
		// Extract key info from JSON
		var result map[string]interface{}
		for _, line := range lines {
			if strings.HasPrefix(line, "{") {
				if err := json.Unmarshal([]byte(line), &result); err == nil {
					if title, ok := result["title"].(string); ok {
						summary = fmt.Sprintf("Title: %s, ", title)
					}
					if status, ok := result["status_code"].(float64); ok {
						summary += fmt.Sprintf("Status: %d, ", int(status))
					}
					if tech, ok := result["tech"].([]interface{}); ok {
						techs := make([]string, 0)
						for _, t := range tech[:min(3, len(tech))] {
							techs = append(techs, fmt.Sprintf("%v", t))
						}
						summary += fmt.Sprintf("Tech: %s", strings.Join(techs, ", "))
					}
					break
				}
			}
		}

	case "subfinder":
		count := 0
		for _, line := range lines {
			if strings.Contains(line, ".") && !strings.HasPrefix(line, "[") {
				count++
			}
		}
		summary = fmt.Sprintf("发现 %d 个子域名", count)

	default:
		// Generic summary
		if len(output) > 100 {
			summary = output[:100] + "..."
		} else {
			summary = output
		}
	}

	if summary == "" {
		summary = "执行完成"
	}

	return summary
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getRiskIcon returns icon for risk level
func getRiskIcon(level string) string {
	switch level {
	case "low":
		return "🟢"
	case "medium":
		return "🟡"
	case "high":
		return "🔴"
	default:
		return "⚪"
	}
}

// getPriorityText returns text for priority
func getPriorityText(priority int) string {
	if priority >= 5 {
		return "最高"
	} else if priority >= 4 {
		return "高"
	} else if priority >= 3 {
		return "中"
	} else if priority >= 2 {
		return "低"
	}
	return "最低"
}

// generateReport generates a test report
func generateReport(session *SessionState) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📄 渗透测试报告")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("目标: %s\n", session.Target)
	fmt.Printf("执行操作数: %d\n", len(session.History))
	fmt.Println("\n操作历史:")
	for i, h := range session.History {
		fmt.Printf("  [%d] %s - %s: %s\n", i+1, h.Action, h.Tool, h.Summary)
	}
	fmt.Println(strings.Repeat("=", 60))
}

// hybridAnalyzeCmd analyzes a task without executing
var hybridAnalyzeCmd = &cobra.Command{
	Use:   "analyze [task]",
	Short: "分析任务类型",
	Long: `分析任务类型，显示调度决策但不执行。

示例:
  fcapital hybrid analyze "port scan"
  fcapital hybrid analyze "custom poc for CVE-2024-xxxx"`,
	Args: cobra.MinimumNArgs(1),
	Run:  runHybridAnalyze,
}

func runHybridAnalyze(cmd *cobra.Command, args []string) {
	task := strings.Join(args, " ")
	printBanner()

	// Initialize dispatcher
	disp := dispatcher.NewDispatcher()

	// Analyze
	ctx := context.Background()
	result, err := disp.Dispatch(ctx, task, map[string]interface{}{})
	if err != nil {
		fmt.Printf("❌ 分析失败: %v\n", err)
		return
	}

	// Display analysis
	fmt.Printf("🎯 任务: %s\n\n", task)
	fmt.Println("📊 分析结果:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("场景类型: %s\n", result.ScenarioType)

	switch result.ScenarioType {
	case dispatcher.ScenarioStandard:
		fmt.Printf("推荐工具: %s\n", result.ToolName)
		fmt.Println("说明: 这是一个标准任务，有成熟的工具可用")

	case dispatcher.ScenarioNonStandard:
		fmt.Printf("推荐语言: %s\n", result.ScriptLanguage)
		fmt.Println("说明: 这是一个非标准任务，需要AI生成辅助脚本")

	case dispatcher.ScenarioMixed:
		fmt.Printf("推荐工具: %s\n", result.ToolName)
		fmt.Printf("辅助脚本: %s\n", result.ScriptLanguage)
		fmt.Println("说明: 这是一个混合任务，需要工具和脚本配合")
	}

	fmt.Printf("\n决策理由: %s\n", result.Reasoning)
}

// hybridGenerateCmd generates a script without executing
var hybridGenerateCmd = &cobra.Command{
	Use:   "generate [task]",
	Short: "生成辅助脚本",
	Long: `生成辅助脚本但不执行。

示例:
  fcapital hybrid generate "custom poc for CVE-2024-xxxx"
  fcapital hybrid generate "waf bypass script" --language python`,
	Args: cobra.MinimumNArgs(1),
	Run:  runHybridGenerate,
}

func runHybridGenerate(cmd *cobra.Command, args []string) {
	task := strings.Join(args, " ")
	printBanner()

	// Initialize AI provider
	apiKey := getAPIKey(hybridProvider)
	if apiKey == "" && hybridProvider != "ollama" {
		fmt.Printf("❌ 脚本生成需要AI支持，请配置 %s API密钥\n", hybridProvider)
		return
	}

	provider := createProvider(hybridProvider, apiKey, hybridModel)
	gen := script.NewGenerator(provider)

	// Generate script
	fmt.Printf("🎯 任务: %s\n", task)
	fmt.Printf("📝 语言: %s\n\n", hybridLanguage)

	ctx := context.Background()
	req := &script.GenerateRequest{
		TaskDescription: task,
		Language:        hybridLanguage,
		Context: map[string]interface{}{
			"target": hybridTarget,
		},
	}

	result, err := gen.Generate(ctx, req)
	if err != nil {
		fmt.Printf("❌ 生成失败: %v\n", err)
		return
	}

	// Display result
	fmt.Println("✅ 脚本生成成功")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("语言: %s\n", result.Language)
	fmt.Printf("安全评分: %d/100\n", result.SafetyScore)

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  警告:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	fmt.Println("\n📝 代码:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(result.Code)
	fmt.Println(strings.Repeat("-", 50))

	if result.Explanation != "" {
		fmt.Printf("\n💡 说明: %s\n", result.Explanation)
	}
}

// mergeCmd represents the merge command
var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "合并多个结果",
	Long: `合并多个工具/脚本的执行结果。

示例:
  fcapital merge results1.json results2.json
  fcapital merge --dedup highest results1.json results2.json`,
	Args: cobra.MinimumNArgs(2),
	Run:  runMerge,
}

var mergeDedup string

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().StringVar(&mergeDedup, "dedup", "highest", "去重策略 (first/highest/merge)")
}

func runMerge(cmd *cobra.Command, args []string) {
	printBanner()
	fmt.Println("📋 合并结果...")

	// Create merger with dedup strategy
	var strategy merger.DeduplicationStrategy
	switch mergeDedup {
	case "first":
		strategy = merger.DedupKeepFirst
	case "highest":
		strategy = merger.DedupKeepHighestSeverity
	case "merge":
		strategy = merger.DedupMerge
	default:
		strategy = merger.DedupKeepHighestSeverity
	}

	_ = merger.NewMerger(merger.WithDedupStrategy(strategy))

	// TODO: Load results from files and merge
	fmt.Println("⚠️  结果合并功能开发中...")
}

// Helper functions

func boolToStatus(b bool) string {
	if b {
		return "✅ 成功"
	}
	return "❌ 失败"
}
