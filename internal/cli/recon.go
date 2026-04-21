package cli

import (
	"context"
	"fmt"

	"github.com/Mal-Suen/fcapital/internal/modules/recon"
	"github.com/spf13/cobra"
)

var reconCmd = &cobra.Command{
	Use:   "recon",
	Short: "Information gathering module",
	Long:  `Perform information gathering on targets (HTTP probe, DNS query, etc.).`,
}

var reconHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "HTTP probe using httpx",
	Long:  `Probe HTTP/HTTPS endpoints and gather information.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("httpx"); err != nil || !tool.IsReady() {
			red.Println("[!] httpx is not installed. Run 'fcapital deps install httpx'")
			return
		}

		fmt.Printf("[*] Running HTTP probe on %s...\n\n", target)

		// 执行扫描
		runner, err := recon.NewHTTPXRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize httpx: %v\n", err)
			return
		}

		results, err := runner.Probe(context.Background(), []string{target}, nil)
		if err != nil {
			red.Printf("[!] HTTP probe failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			yellow.Println("[*] No results found")
			return
		}

		for _, r := range results {
			green.Printf("[+] %s\n", r.URL)
			if r.Title != "" {
				fmt.Printf("    Title: %s\n", r.Title)
			}
			if r.StatusCode > 0 {
				fmt.Printf("    Status: %d\n", r.StatusCode)
			}
			if r.WebServer != "" {
				fmt.Printf("    Server: %s\n", r.WebServer)
			}
			if len(r.Technologies) > 0 {
				fmt.Printf("    Tech: %v\n", r.Technologies)
			}
			fmt.Println()
		}

		green.Printf("[+] Found %d HTTP endpoint(s)\n", len(results))
	},
}

var reconDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS query using dnsx",
	Long:  `Query DNS records for a domain.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("dnsx"); err != nil || !tool.IsReady() {
			red.Println("[!] dnsx is not installed. Run 'fcapital deps install dnsx'")
			return
		}

		fmt.Printf("[*] Running DNS query on %s...\n\n", target)

		// 执行查询
		runner, err := recon.NewDNSXRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize dnsx: %v\n", err)
			return
		}

		results, err := runner.Query(context.Background(), []string{target}, nil)
		if err != nil {
			red.Printf("[!] DNS query failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			yellow.Println("[*] No DNS records found")
			return
		}

		for _, r := range results {
			// 处理文本格式解析的结果
			if r.Domain != "" && r.RecordType != "" {
				green.Printf("[+] %s [%s] %s\n", r.Domain, r.RecordType, r.Value)
				continue
			}

			// 处理 JSON 格式的结果
			if r.Host != "" {
				green.Printf("[+] %s\n", r.Host)
				if len(r.A) > 0 {
					fmt.Printf("    A: %v\n", r.A)
				}
				if len(r.AAAA) > 0 {
					fmt.Printf("    AAAA: %v\n", r.AAAA)
				}
				if len(r.CNAME) > 0 {
					fmt.Printf("    CNAME: %v\n", r.CNAME)
				}
				if len(r.MX) > 0 {
					fmt.Printf("    MX: %v\n", r.MX)
				}
				if len(r.NS) > 0 {
					fmt.Printf("    NS: %v\n", r.NS)
				}
				fmt.Println()
			}
		}
	},
}

func init() {
	// HTTP 子命令
	reconHTTPCmd.Flags().StringP("target", "t", "", "Target domain or IP")
	reconHTTPCmd.Flags().StringP("output", "o", "", "Output file")
	reconCmd.AddCommand(reconHTTPCmd)

	// DNS 子命令
	reconDNSCmd.Flags().StringP("target", "t", "", "Target domain")
	reconDNSCmd.Flags().StringP("output", "o", "", "Output file")
	reconCmd.AddCommand(reconDNSCmd)
}
