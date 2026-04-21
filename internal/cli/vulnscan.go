package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourname/fcapital/internal/modules/vulnscan"
)

var vulnscanCmd = &cobra.Command{
	Use:   "vulnscan",
	Short: "Vulnerability scanning module",
	Long:  `Scan for vulnerabilities using various tools.`,
}

var vulnscanNucleiCmd = &cobra.Command{
	Use:   "nuclei",
	Short: "Vulnerability scan using nuclei",
	Long:  `Scan for vulnerabilities using nuclei templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		tags, _ := cmd.Flags().GetString("tags")
		severity, _ := cmd.Flags().GetString("severity")

		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("nuclei"); err != nil || !tool.IsReady() {
			red.Println("[!] nuclei is not installed. Run 'fcapital deps install nuclei'")
			return
		}

		fmt.Printf("[*] Running nuclei scan on %s...\n\n", target)

		// 构建选项
		opts := &vulnscan.NucleiOptions{}
		if tags != "" {
			opts.Tags = splitByComma(tags)
		}
		if severity != "" {
			opts.Severity = splitByComma(severity)
		}

		// 执行扫描
		runner, err := vulnscan.NewNucleiRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize nuclei: %v\n", err)
			return
		}

		results, err := runner.Scan(context.Background(), target, opts)
		if err != nil {
			red.Printf("[!] Nuclei scan failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			green.Println("[+] No vulnerabilities found")
			return
		}

		for _, r := range results {
			// 根据严重级别着色
			switch r.Severity {
			case "critical":
				red.Printf("[CRITICAL] %s\n", r.TemplateName)
			case "high":
				red.Printf("[HIGH] %s\n", r.TemplateName)
			case "medium":
				yellow.Printf("[MEDIUM] %s\n", r.TemplateName)
			case "low":
				fmt.Printf("[LOW] %s\n", r.TemplateName)
			default:
				fmt.Printf("[INFO] %s\n", r.TemplateName)
			}
			fmt.Printf("    Template: %s\n", r.TemplateID)
			fmt.Printf("    URL: %s\n", r.MatchedAt)
			fmt.Println()
		}

		// 统计
		critical := countBySeverity(results, "critical")
		high := countBySeverity(results, "high")
		medium := countBySeverity(results, "medium")
		low := countBySeverity(results, "low")

		fmt.Printf("Summary: %d critical, %d high, %d medium, %d low\n", critical, high, medium, low)
	},
}

var vulnscanSQLMapCmd = &cobra.Command{
	Use:   "sqlmap",
	Short: "SQL injection scan using sqlmap",
	Long:  `Test for SQL injection vulnerabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		data, _ := cmd.Flags().GetString("data")
		cookie, _ := cmd.Flags().GetString("cookie")

		if url == "" {
			fmt.Println("[!] Error: URL is required. Use -u or --url")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("sqlmap"); err != nil || !tool.IsReady() {
			red.Println("[!] sqlmap is not installed. Run 'fcapital deps install sqlmap'")
			return
		}

		fmt.Printf("[*] Running SQL injection scan on %s...\n\n", url)

		// 构建选项
		opts := &vulnscan.SQLMapOptions{
			Batch: true,
		}
		if data != "" {
			opts.Data = data
		}
		if cookie != "" {
			opts.Cookie = cookie
		}

		// 执行扫描
		runner, err := vulnscan.NewSQLMapRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize sqlmap: %v\n", err)
			return
		}

		result, err := runner.Scan(context.Background(), url, opts)
		if err != nil {
			red.Printf("[!] SQLMap scan failed: %v\n", err)
			return
		}

		// 输出结果
		if !result.Vulnerable {
			green.Println("[+] No SQL injection found")
			return
		}

		red.Println("[!] SQL Injection vulnerability found!")
		fmt.Printf("    Parameter: %s\n", result.Parameter)
		fmt.Printf("    Type: %s\n", result.InjectionType)
		fmt.Printf("    DBMS: %s\n", result.DBMS)
	},
}

func countBySeverity(results []vulnscan.NucleiResult, severity string) int {
	count := 0
	for _, r := range results {
		if r.Severity == severity {
			count++
		}
	}
	return count
}

func init() {
	// Nuclei scan
	vulnscanNucleiCmd.Flags().StringP("target", "t", "", "Target URL")
	vulnscanNucleiCmd.Flags().StringP("tags", "", "", "Template tags (comma-separated)")
	vulnscanNucleiCmd.Flags().StringP("severity", "s", "", "Severity filter (critical,high,medium,low)")
	vulnscanCmd.AddCommand(vulnscanNucleiCmd)

	// SQLMap scan
	vulnscanSQLMapCmd.Flags().StringP("url", "u", "", "Target URL")
	vulnscanSQLMapCmd.Flags().StringP("data", "", "", "POST data")
	vulnscanSQLMapCmd.Flags().StringP("cookie", "", "", "Cookie header")
	vulnscanCmd.AddCommand(vulnscanSQLMapCmd)
}
