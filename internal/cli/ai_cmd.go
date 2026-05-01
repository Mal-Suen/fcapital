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
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
	"github.com/Mal-Suen/fcapital/internal/core/script"
	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/Mal-Suen/fcapital/internal/pkg/logger"
	"github.com/Mal-Suen/fcapital/internal/pkg/toolcheck"
	"github.com/spf13/cobra"
)

// aiCmd represents the AI-driven scan command
var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI驱动的渗透测试",
	Long: `使用 AI 驱动的渗透测试，自动分析结果并推荐下一步操作。

AI 模式会：
- 自动判断任务类型（标准工具 vs AI生成脚本）
- 分析每个阶段的执行结果
- 智能推荐下一步操作
- 支持会话持久化和恢复

示例:
  fcapital ai -t example.com
  fcapital ai -t example.com --auto-confirm
  fcapital ai -t example.com --provider deepseek`,
	Run: runAI,
}

var (
	aiTarget      string
	aiAutoConfirm bool
	aiProvider    string
	aiModel       string
	aiLogDir      string
	aiSession     string
)

func init() {
	aiCmd.Flags().StringVarP(&aiTarget, "target", "t", "", "目标域名或IP")
	aiCmd.Flags().BoolVar(&aiAutoConfirm, "auto-confirm", false, "自动确认脚本执行")
	aiCmd.Flags().StringVar(&aiProvider, "provider", "openai", "AI提供者 (openai/deepseek/ollama)")
	aiCmd.Flags().StringVar(&aiModel, "model", "", "AI模型名称")
	aiCmd.Flags().StringVar(&aiLogDir, "log-dir", "", "日志目录 (默认 ~/.fcapital/sessions)")
	aiCmd.Flags().StringVar(&aiSession, "session", "", "恢复会话ID")
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

func runAI(cmd *cobra.Command, args []string) {
	printBanner()

	// Initialize logger
	log := logger.NewLogger(aiLogDir)

	// Check if resuming a session
	if aiSession != "" {
		sessionLog, err := log.LoadSession(aiSession)
		if err != nil {
			fmt.Printf("❌ 加载会话失败: %v\n", err)
			return
		}

		fmt.Printf("📋 会话ID: %s\n", sessionLog.ID)
		fmt.Printf("📍 目标: %s\n", sessionLog.Target)
		fmt.Printf("📊 状态: %s\n", sessionLog.Status)
		fmt.Printf("📁 日志文件: %s\n\n", log.GetLogPath())

		if len(sessionLog.History) > 0 {
			fmt.Println("📜 已完成的操作:")
			for i, h := range sessionLog.History {
				fmt.Printf("  [%d] %s - %s: %s\n", i+1, h.Action, h.Tool, h.Summary)
			}
			fmt.Println()
		}

		setupSignalHandler(log, sessionLog.ID)
		runAISession("", log, sessionLog.NextAction)
		return
	}

	// New session
	if aiTarget == "" {
		fmt.Println("❌ 请指定目标: fcapital ai -t <target>")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  fcapital ai -t example.com")
		fmt.Println("  fcapital ai -t 192.168.1.1")
		return
	}

	sessionLog := log.NewSession(aiTarget)
	fmt.Printf("📋 会话ID: %s\n", sessionLog.ID)
	fmt.Printf("📁 日志文件: %s\n\n", log.GetLogPath())

	setupSignalHandler(log, sessionLog.ID)
	runAISession("", log, "")
}

func setupSignalHandler(log *logger.Logger, sessionID string) {
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
		fmt.Printf("   fcapital ai --session %s\n", sessionID)
		os.Exit(0)
	}()
}

func runAISession(initialTask string, log *logger.Logger, nextAction string) {
	sessionLog := log.GetSession()

	// Initialize components
	sched := scheduler.New()
	registerTools(sched)
	tm := toolmgr.NewToolManager()

	// Initialize AI provider
	apiKey := getAPIKey(aiProvider)
	var provider providers.Provider
	if apiKey != "" || aiProvider == "ollama" {
		provider = createProvider(aiProvider, apiKey, aiModel)
	}

	if provider != nil {
		fmt.Printf("🔧 AI 提供者: %s\n", provider.Name())
		providerUpper := strings.ToUpper(aiProvider)
		baseURL := os.Getenv(fmt.Sprintf("%s_BASE_URL", providerUpper))
		if baseURL == "" {
			baseURL = os.Getenv(fmt.Sprintf("%s_BASEURL", providerUpper))
		}
		fmt.Printf("🔧 API Base URL: %s\n", baseURL)
		model := aiModel
		if model == "" {
			model = os.Getenv("AI_MODEL")
		}
		fmt.Printf("🔧 AI Model: %s\n", model)
	}

	if provider == nil {
		fmt.Println("❌ AI 模式需要配置 API 密钥")
		fmt.Println()
		fmt.Println("📝 配置方法:")
		fmt.Println("   1. 在项目根目录创建 .env 文件:")
		fmt.Println("      OPENAI_API_KEY=your-api-key")
		fmt.Println("      OPENAI_BASE_URL=https://api.openai.com/v2  # 可选")
		fmt.Println("      AI_MODEL=gpt-4  # 可选")
		fmt.Println()
		fmt.Println("   2. 或使用其他提供者:")
		fmt.Println("      --provider deepseek  # DeepSeek API")
		fmt.Println("      --provider ollama    # 本地 Ollama (无需API密钥)")
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

	// Initialize tool checker
	toolChecker := toolcheck.NewChecker()
	toolCheckResult := toolChecker.CheckAll()

	fmt.Printf("\n🔧 工具检测: 已安装 %d/%d 个工具\n", toolCheckResult.InstalledCount, toolCheckResult.TotalCount)
	if len(toolCheckResult.Available) > 0 {
		fmt.Printf("   可用: %s\n", toolChecker.FormatAvailableTools(toolCheckResult))
	}

	// Initialize session state
	session := &SessionState{
		Target:       sessionLog.Target,
		CurrentPhase: sessionLog.CurrentPhase,
		Results:      convertResults(sessionLog.Results),
		History:      convertHistory(sessionLog.History),
	}

	ctx := context.Background()

	// Execute initial task
	if initialTask != "" {
		fmt.Printf("🎯 初始任务: %s\n", initialTask)
		fmt.Printf("📍 目标: %s\n\n", session.Target)
		executeTaskWithLog(ctx, disp, gen, initialTask, session.Target, session, log, toolChecker)
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
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("📊 执行结果摘要")
		fmt.Println(strings.Repeat("=", 60))
		printSessionSummary(session)

		fmt.Println("\n🤖 AI 正在分析结果...")
		recommendations := getAIRecommendations(ctx, provider, session, toolCheckResult)
		if len(recommendations) == 0 {
			fmt.Println("❌ AI 分析失败，无法生成建议")
			log.Interrupt()
			return
		}

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

		selected := recommendations[choice-1]
		fmt.Printf("\n▶️ 执行: %s\n", selected.Title)

		if selected.Tool != "" && !toolChecker.IsToolAvailable(selected.Tool) {
			installed := handleToolInstallation(selected.Tool, reader, provider, ctx)
			if !installed {
				fmt.Println("❌ 工具安装失败，跳过此操作")
				continue
			}
			toolCheckResult = toolChecker.CheckAll()
		}

		log.SetNextAction(selected.Title)

		nextTask := selected.Tool
		if nextTask == "" {
			nextTask = selected.Title
		}

		executeTaskWithLog(ctx, disp, gen, nextTask, session.Target, session, log, toolChecker)
	}
}

func executeTaskWithLog(ctx context.Context, disp *dispatcher.Dispatcher, gen *script.Generator, task string, target string, session *SessionState, log *logger.Logger, toolChecker *toolcheck.Checker) *dispatcher.ExecutionResult {
	contextData := map[string]interface{}{
		"target": target,
	}

	result, err := disp.Dispatch(ctx, task, contextData)
	if err != nil {
		fmt.Printf("❌ 调度失败: %v\n", err)
		log.Log(logger.LevelError, session.CurrentPhase, "", fmt.Sprintf("调度失败: %v", err), false, "")
		return nil
	}

	fmt.Println("\n📊 调度决策:")
	fmt.Printf("   场景类型: %s\n", result.ScenarioType)
	if result.ToolName != "" {
		fmt.Printf("   使用工具: %s\n", result.ToolName)
		if toolPath := disp.GetToolPath(result.ToolName); toolPath != "" {
			fmt.Printf("   工具路径: %s\n", toolPath)
		}
	}
	fmt.Printf("   决策理由: %s\n\n", result.Reasoning)

	log.Log(logger.LevelInfo, session.CurrentPhase, result.ToolName, result.Reasoning, true, "")

	var execResult *dispatcher.ExecutionResult
	switch result.ScenarioType {
	case dispatcher.ScenarioStandard:
		execResult, err = disp.ExecuteStandard(ctx, result.ToolName, []string{target})
	case dispatcher.ScenarioNonStandard:
		execResult, err = disp.ExecuteNonStandard(ctx, result.ScriptTask, contextData, aiAutoConfirm)
	case dispatcher.ScenarioMixed:
		execResult, err = disp.ExecuteMixed(ctx, result.ToolName, result.ScriptTask, []string{target}, contextData, aiAutoConfirm)
	}

	if err != nil {
		fmt.Printf("❌ 执行失败: %v\n", err)
		log.Log(logger.LevelError, session.CurrentPhase, result.ToolName, fmt.Sprintf("执行失败: %v", err), false, "")
		return nil
	}

	summary := extractSummary(execResult.Output, result.ToolName)

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

	log.RecordResult(session.CurrentPhase, result.ToolName, execResult.Success, execResult.Output, summary)
	log.RecordHistory(task, result.ToolName, boolToStatus(execResult.Success), summary)

	return execResult
}

func handleToolInstallation(toolName string, reader *bufio.Reader, provider providers.Provider, ctx context.Context) bool {
	fmt.Printf("\n⚠️  工具 '%s' 未安装\n", toolName)

	instructions := toolcheck.GetInstallInstructions(toolName)
	fmt.Printf("\n📦 安装说明:\n%s\n", instructions)

	fmt.Printf("\n尝试自动安装? (y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		fmt.Printf("\n🔧 正在尝试自动安装 %s...\n", toolName)
		success, msg := toolcheck.TryAutoInstall(toolName)
		fmt.Printf("%s\n", msg)
		return success
	}

	fmt.Printf("\n请手动安装后按 Enter 继续，或输入 'skip' 跳过: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "skip" {
		return false
	}

	checker := toolcheck.NewChecker()
	if checker.IsToolAvailable(toolName) {
		fmt.Printf("✅ 检测到 %s 已安装\n", toolName)
		return true
	}

	fmt.Printf("❌ 仍未检测到 %s\n", toolName)
	return false
}

func getAIRecommendations(ctx context.Context, provider providers.Provider, session *SessionState, toolCheckResult *toolcheck.CheckResult) []Recommendation {
	contextJSON, _ := json.Marshal(map[string]interface{}{
		"target":  session.Target,
		"phase":   session.CurrentPhase,
		"results": session.Results,
	})

	// 构建已安装工具列表
	var availableTools []string
	var missingTools []string
	if toolCheckResult != nil {
		for _, tool := range toolCheckResult.Available {
			availableTools = append(availableTools, tool.Name)
		}
		for _, tool := range toolCheckResult.Missing {
			missingTools = append(missingTools, tool.Name)
		}
	}

	prompt := fmt.Sprintf(`你是一个渗透测试专家。根据以下测试结果，分析目标并推荐最佳的下一步操作。

目标: %s
当前阶段: %s

## 当前系统环境
- 操作系统: %s
- 已安装工具 (%d个): %s
- 未安装工具: %s

测试结果:
%s

重要说明:
1. 推荐最适合当前任务的工具，不受限于已安装工具
2. 如果推荐的最佳工具未安装，系统会尝试自动安装
3. 在 description 中简要说明为什么推荐这个工具
4. 不要推荐与已执行操作重复的工具

请以JSON数组格式返回3-5个下一步建议，每个建议包含:
- id: 序号 (1-5)
- title: 建议标题 (简短)
- description: 详细描述（包含推荐理由）
- tool: 推荐工具名称 (推荐最佳工具，不限制已安装)
- priority: 优先级 (1-5, 5最高)
- risk_level: 风险等级 (low/medium/high)

只返回JSON数组，不要其他内容。`, session.Target, session.CurrentPhase, runtime.GOOS, len(availableTools), strings.Join(availableTools, ", "), strings.Join(missingTools, ", "), string(contextJSON))

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

	var recommendations []Recommendation
	content := resp.Content

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

func printSessionSummary(session *SessionState) {
	for i, result := range session.Results {
		status := "❌"
		if result.Success {
			status = "✅"
		}
		fmt.Printf("%s [%d] %s: %s\n", status, i+1, result.Tool, result.Summary)
	}
}

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

func boolToStatus(b bool) string {
	if b {
		return "✅ 成功"
	}
	return "❌ 失败"
}
