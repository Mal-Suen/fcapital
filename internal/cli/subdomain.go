package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/Mal-Suen/fcapital/internal/modules/subdomain"
	"github.com/spf13/cobra"
)

var subdomainCmd = &cobra.Command{
	Use:   "subdomain",
	Short: "Subdomain enumeration module",
	Long:  `Enumerate subdomains using various techniques.`,
}

var subdomainPassiveCmd = &cobra.Command{
	Use:   "passive",
	Short: "Passive subdomain enumeration using subfinder",
	Long:  `Discover subdomains from passive sources.`,
	Run: func(cmd *cobra.Command, args []string) {
		domain, _ := cmd.Flags().GetString("domain")
		if domain == "" {
			fmt.Println("[!] Error: domain is required. Use -d or --domain")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("subfinder"); err != nil || !tool.IsReady() {
			red.Println("[!] subfinder is not installed. Run 'fcapital deps install subfinder'")
			return
		}

		fmt.Printf("[*] Running passive subdomain enumeration on %s...\n\n", domain)

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// 执行枚举
		runner, err := subdomain.NewSubfinderRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize subfinder: %v\n", err)
			return
		}

		// 设置选项，包含超时
		opts := &subdomain.SubfinderOptions{
			Timeout: 30 * time.Second,
			Silent:  true,
		}

		results, err := runner.Enumerate(ctx, domain, opts)
		if err != nil {
			red.Printf("[!] Subdomain enumeration failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			yellow.Println("[*] No subdomains found")
			return
		}

		for _, subdomain := range results {
			green.Printf("[+] %s\n", subdomain)
		}

		fmt.Println()
		green.Printf("[+] Found %d subdomain(s)\n", len(results))
	},
}

func init() {
	subdomainPassiveCmd.Flags().StringP("domain", "d", "", "Target domain")
	subdomainPassiveCmd.Flags().StringP("output", "o", "", "Output file")
	subdomainCmd.AddCommand(subdomainPassiveCmd)
}
