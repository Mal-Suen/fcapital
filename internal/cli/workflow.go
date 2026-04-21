package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/Mal-Suen/fcapital/internal/core/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:     "workflow",
	Aliases: []string{"wf"},
	Short:   "Run automated penetration testing workflows",
	Long: `Execute predefined workflows that chain multiple modules together.

Available workflows:
  full    - Complete penetration test: subdomain → http probe → port scan → dir scan → vuln scan
  recon   - Quick reconnaissance: subdomain → http probe
  webapp  - Web application scan: http probe → dir scan → vuln scan
  vuln    - Vulnerability scan: http probe → nuclei scan

Examples:
  fcapital workflow run full -t example.com
  fcapital workflow run recon -t example.com -o ./results
  fcapital workflow list`,
}

var workflowRunCmd = &cobra.Command{
	Use:   "run <workflow> -t <target>",
	Short: "Execute a workflow against a target",
	Long:  `Run a predefined workflow with all its steps against the specified target.`,
	Args:  cobra.ExactArgs(1),
	Run:   runWorkflow,
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available workflows",
	Long:  `Show all predefined workflows and their descriptions.`,
	Run:   runWorkflowList,
}

var (
	workflowTarget   string
	workflowOutput   string
	workflowTimeout  int
	workflowReport   bool
)

func init() {
	workflowRunCmd.Flags().StringVarP(&workflowTarget, "target", "t", "", "Target domain or IP (required)")
	workflowRunCmd.Flags().StringVarP(&workflowOutput, "output", "o", "", "Output directory (default: ~/.fcapital/results/<target>_<timestamp>)")
	workflowRunCmd.Flags().IntVarP(&workflowTimeout, "timeout", "T", 60, "Total workflow timeout in minutes")
	workflowRunCmd.Flags().BoolVarP(&workflowReport, "report", "r", true, "Generate HTML/JSON/Markdown reports")
	workflowRunCmd.MarkFlagRequired("target")

	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowListCmd)
}

func runWorkflow(cmd *cobra.Command, args []string) {
	workflowName := args[0]
	target := workflowTarget

	if target == "" {
		red.Println("[!] Target is required. Use -t <target>")
		return
	}

	// 初始化工具管理器
	tm := InitToolManager()

	// 创建工作流引擎
	engine := workflow.NewEngine(tm)

	// 注册处理器
	registerHandlers(engine, tm)

	// 检查工作流是否存在
	wf, ok := engine.GetWorkflow(workflowName)
	if !ok {
		red.Printf("[!] Unknown workflow: %s\n", workflowName)
		yellow.Println("\nAvailable workflows:")
		for _, w := range engine.ListWorkflows() {
			fmt.Printf("  - %s: %s\n", w.Name, w.Description)
		}
		return
	}

	// 设置输出目录
	outputDir := workflowOutput
	if outputDir == "" {
		homeDir, _ := os.UserHomeDir()
		timestamp := time.Now().Format("20060102_150405")
		outputDir = filepath.Join(homeDir, ".fcapital", "results", fmt.Sprintf("%s_%s", target, timestamp))
	}

	// 打印开始信息
	cyan.Println("\n" + strings.Repeat("=", 60))
	cyan.Printf("  fcapital workflow: %s\n", wf.Name)
	cyan.Printf("  Target: %s\n", target)
	cyan.Printf("  Output: %s\n", outputDir)
	cyan.Println(strings.Repeat("=", 60))
	fmt.Println()

	// 打印工作流步骤
	fmt.Println("[*] Workflow steps:")
	for i, step := range wf.Steps {
		deps := ""
		if len(step.DependsOn) > 0 {
			deps = fmt.Sprintf(" (depends: %v)", step.DependsOn)
		}
		fmt.Printf("    %d. %s [%s/%s]%s\n", i+1, step.Name, step.Module, step.Action, deps)
	}
	fmt.Println()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(workflowTimeout)*time.Minute)
	defer cancel()

	// 执行工作流
	result, err := engine.Execute(ctx, workflowName, target, outputDir)
	if err != nil {
		red.Printf("[!] Workflow failed: %v\n", err)
		return
	}

	// 生成报告
	if workflowReport {
		fmt.Println("\n[*] Generating reports...")
		reportGen := workflow.NewReportGenerator()
		if err := reportGen.GenerateAll(result, outputDir); err != nil {
			red.Printf("[!] Failed to generate reports: %v\n", err)
		} else {
			green.Printf("[+] Reports saved to: %s\n", outputDir)
		}
	}

	// 打印摘要
	printWorkflowSummary(result)

	// 打印结果位置
	fmt.Println()
	green.Printf("[+] Results saved to: %s\n", outputDir)
	fmt.Printf("    - result.json (full results)\n")
	fmt.Printf("    - report.html (HTML report)\n")
	fmt.Printf("    - report.json (JSON report)\n")
	fmt.Printf("    - report.md (Markdown report)\n")
}

func runWorkflowList(cmd *cobra.Command, args []string) {
	tm := GetToolManager()
	engine := workflow.NewEngine(tm)

	fmt.Println("\n[*] Available workflows:")
	fmt.Println()

	for _, wf := range engine.ListWorkflows() {
		cyan.Printf("  %s\n", wf.Name)
		fmt.Printf("    %s\n", wf.Description)
		fmt.Printf("    Steps: ")
		for i, step := range wf.Steps {
			if i > 0 {
				fmt.Printf(" → ")
			}
			fmt.Printf("%s", step.Name)
		}
		fmt.Println("\n")
	}
}

func registerHandlers(engine *workflow.Engine, tm *toolmgr.ToolManager) {
	engine.RegisterHandler("recon", workflow.NewReconHandler(tm))
	engine.RegisterHandler("subdomain", workflow.NewSubdomainHandler(tm))
	engine.RegisterHandler("portscan", workflow.NewPortscanHandler(tm))
	engine.RegisterHandler("webscan", workflow.NewWebscanHandler(tm))
	engine.RegisterHandler("vulnscan", workflow.NewVulnscanHandler(tm))
}

func printWorkflowSummary(result *workflow.WorkflowResult) {
	fmt.Println()
	cyan.Println("═══════════════════════════════════════════════════════════")
	cyan.Println("                      SCAN SUMMARY                          ")
	cyan.Println("═══════════════════════════════════════════════════════════")

	// 状态
	statusIcon := "✅"
	if result.Status == "partial" {
		statusIcon = "⚠️"
	} else if result.Status == "failed" {
		statusIcon = "❌"
	}
	fmt.Printf("\n  Status: %s %s\n", statusIcon, result.Status)
	fmt.Printf("  Duration: %s\n", result.Duration)

	// 摘要统计
	if result.Summary != nil {
		fmt.Println("\n  ┌─────────────────────────────────────────┐")
		fmt.Printf("  │ %-20s │ %-18s │\n", "Metric", "Count")
		fmt.Println("  ├─────────────────────────────────────────┤")
		fmt.Printf("  │ %-20s │ %18d │\n", "Subdomains", result.Summary.Subdomains)
		fmt.Printf("  │ %-20s │ %18d │\n", "Alive Hosts", result.Summary.AliveHosts)
		fmt.Printf("  │ %-20s │ %18d │\n", "Open Ports", result.Summary.OpenPorts)
		fmt.Printf("  │ %-20s │ %18d │\n", "Directories", result.Summary.Directories)
		fmt.Printf("  │ %-20s │ %18d │\n", "Vulnerabilities", result.Summary.Vulnerabilities)
		fmt.Println("  └─────────────────────────────────────────┘")
	}

	// 步骤状态
	fmt.Println("\n  Step Results:")
	for stepID, stepResult := range result.Steps {
		icon := "✅"
		if stepResult.Status == "failed" {
			icon = "❌"
		} else if stepResult.Status == "skipped" {
			icon = "⏭️"
		}
		fmt.Printf("    %s %s (%s)\n", icon, stepID, stepResult.Duration)
		if stepResult.Error != "" {
			red.Printf("       Error: %s\n", stepResult.Error)
		}
	}

	// 严重漏洞
	if result.Summary != nil && len(result.Summary.CriticalVulns) > 0 {
		red.Println("\n  ⚠️  CRITICAL VULNERABILITIES FOUND:")
		for _, vuln := range result.Summary.CriticalVulns {
			fmt.Printf("    - [%s] %s on %s\n", vuln.Severity, vuln.Name, vuln.Host)
		}
	}

	fmt.Println()
	cyan.Println("═══════════════════════════════════════════════════════════")
}
