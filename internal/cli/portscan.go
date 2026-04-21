package cli

import (
	"context"
	"fmt"

	"github.com/Mal-Suen/fcapital/internal/modules/portscan"
	"github.com/spf13/cobra"
)

var portscanCmd = &cobra.Command{
	Use:   "portscan",
	Short: "Port scanning module",
	Long:  `Scan ports on target hosts using nmap.`,
}

var portscanQuickCmd = &cobra.Command{
	Use:   "quick",
	Short: "Quick scan (Top 100 ports)",
	Long:  `Perform a quick port scan on the most common 100 ports.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("nmap"); err != nil || !tool.IsReady() {
			red.Println("[!] nmap is not installed. Run 'fcapital deps install nmap'")
			return
		}

		fmt.Printf("[*] Running quick port scan on %s...\n\n", target)

		// 执行扫描
		runner, err := portscan.NewNmapRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize nmap: %v\n", err)
			return
		}

		result, err := runner.QuickScan(context.Background(), target)
		if err != nil {
			red.Printf("[!] Port scan failed: %v\n", err)
			return
		}

		// 输出结果
		if len(result.Ports) == 0 {
			yellow.Println("[*] No open ports found")
			return
		}

		fmt.Printf("PORT      STATE   SERVICE\n")
		for _, port := range result.Ports {
			green.Printf("%d/%s  open    %s\n", port.Port, port.Protocol, port.Service)
			if port.Version != "" {
				fmt.Printf("          Version: %s\n", port.Version)
			}
		}

		fmt.Println()
		green.Printf("[+] Found %d open port(s) in %v\n", len(result.Ports), result.ScanTime)
	},
}

var portscanFullCmd = &cobra.Command{
	Use:   "full",
	Short: "Full scan (All ports)",
	Long:  `Perform a full port scan on all 65535 ports.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("nmap"); err != nil || !tool.IsReady() {
			red.Println("[!] nmap is not installed. Run 'fcapital deps install nmap'")
			return
		}

		fmt.Printf("[*] Running full port scan on %s...\n", target)
		yellow.Println("[!] This may take a while...")

		// 执行扫描
		runner, err := portscan.NewNmapRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize nmap: %v\n", err)
			return
		}

		result, err := runner.FullScan(context.Background(), target)
		if err != nil {
			red.Printf("[!] Port scan failed: %v\n", err)
			return
		}

		// 输出结果
		if len(result.Ports) == 0 {
			yellow.Println("[*] No open ports found")
			return
		}

		fmt.Printf("\nPORT      STATE   SERVICE\n")
		for _, port := range result.Ports {
			green.Printf("%d/%s  open    %s\n", port.Port, port.Protocol, port.Service)
		}

		fmt.Println()
		green.Printf("[+] Found %d open port(s) in %v\n", len(result.Ports), result.ScanTime)
	},
}

var portscanCustomCmd = &cobra.Command{
	Use:   "custom",
	Short: "Custom port scan",
	Long:  `Perform a custom port scan on specified ports.`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		ports, _ := cmd.Flags().GetString("ports")
		if target == "" {
			fmt.Println("[!] Error: target is required. Use -t or --target")
			return
		}
		if ports == "" {
			fmt.Println("[!] Error: ports is required. Use -p or --ports")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("nmap"); err != nil || !tool.IsReady() {
			red.Println("[!] nmap is not installed. Run 'fcapital deps install nmap'")
			return
		}

		fmt.Printf("[*] Running port scan on %s (ports: %s)...\n\n", target, ports)

		// 执行扫描
		runner, err := portscan.NewNmapRunner(tm)
		if err != nil {
			red.Printf("[!] Failed to initialize nmap: %v\n", err)
			return
		}

		result, err := runner.CustomScan(context.Background(), target, ports)
		if err != nil {
			red.Printf("[!] Port scan failed: %v\n", err)
			return
		}

		// 输出结果
		if len(result.Ports) == 0 {
			yellow.Println("[*] No open ports found")
			return
		}

		fmt.Printf("PORT      STATE   SERVICE\n")
		for _, port := range result.Ports {
			green.Printf("%d/%s  open    %s\n", port.Port, port.Protocol, port.Service)
		}

		fmt.Println()
		green.Printf("[+] Found %d open port(s) in %v\n", len(result.Ports), result.ScanTime)
	},
}

func init() {
	// Quick scan
	portscanQuickCmd.Flags().StringP("target", "t", "", "Target host or IP")
	portscanCmd.AddCommand(portscanQuickCmd)

	// Full scan
	portscanFullCmd.Flags().StringP("target", "t", "", "Target host or IP")
	portscanCmd.AddCommand(portscanFullCmd)

	// Custom scan
	portscanCustomCmd.Flags().StringP("target", "t", "", "Target host or IP")
	portscanCustomCmd.Flags().StringP("ports", "p", "", "Port range (e.g., 1-1000, 80,443,8080)")
	portscanCmd.AddCommand(portscanCustomCmd)
}
