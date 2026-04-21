package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mal-Suen/fcapital/internal/modules/webscan"
	"github.com/spf13/cobra"
)

var webscanCmd = &cobra.Command{
	Use:   "webscan",
	Short: "Web scanning module",
	Long:  `Perform web directory/file scanning.`,
}

var webscanDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Directory scanning",
	Long:  `Scan for hidden directories and files.`,
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		tool, _ := cmd.Flags().GetString("tool")
		wordlist, _ := cmd.Flags().GetString("wordlist")
		extensions, _ := cmd.Flags().GetString("extensions")

		if url == "" {
			fmt.Println("[!] Error: URL is required. Use -u or --url")
			return
		}

		if tool == "" {
			tool = "dirsearch"
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if t, err := tm.Get(tool); err != nil || !t.IsReady() {
			red.Printf("[!] %s is not installed. Run 'fcapital deps install %s'\n", tool, tool)
			return
		}

		fmt.Printf("[*] Running directory scan on %s using %s...\n\n", url, tool)

		// 构建选项
		opts := &webscan.DirscanOptions{
			URL: url,
		}
		if wordlist != "" {
			opts.Wordlist = wordlist
		}
		if extensions != "" {
			opts.Extensions = splitExtensions(extensions)
		}

		// 执行扫描
		results, err := webscan.Scan(context.Background(), tm, tool, opts)
		if err != nil {
			red.Printf("[!] Directory scan failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			yellow.Println("[*] No directories/files found")
			return
		}

		fmt.Printf("STATUS    SIZE      PATH\n")
		for _, r := range results {
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				green.Printf("%d       %d       %s\n", r.StatusCode, r.Size, r.Path)
			} else if r.StatusCode >= 300 && r.StatusCode < 400 {
				yellow.Printf("%d       %d       %s\n", r.StatusCode, r.Size, r.Path)
			} else {
				red.Printf("%d       %d       %s\n", r.StatusCode, r.Size, r.Path)
			}
		}

		fmt.Println()
		green.Printf("[+] Found %d result(s)\n", len(results))
	},
}

var webscanFuzzCmd = &cobra.Command{
	Use:   "fuzz",
	Short: "Fuzzing using ffuf",
	Long:  `Perform fuzzing on target URL.`,
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		wordlist, _ := cmd.Flags().GetString("wordlist")

		if url == "" {
			fmt.Println("[!] Error: URL is required. Use -u or --url")
			return
		}

		tm := InitToolManager()

		// 检查工具是否可用
		if tool, err := tm.Get("ffuf"); err != nil || !tool.IsReady() {
			red.Println("[!] ffuf is not installed. Run 'fcapital deps install ffuf'")
			return
		}

		fmt.Printf("[*] Running fuzzing on %s...\n\n", url)

		// 构建选项
		opts := &webscan.DirscanOptions{
			URL: url,
		}
		if wordlist != "" {
			opts.Wordlist = wordlist
		}

		// 执行扫描
		results, err := webscan.Scan(context.Background(), tm, "ffuf", opts)
		if err != nil {
			red.Printf("[!] Fuzzing failed: %v\n", err)
			return
		}

		// 输出结果
		if len(results) == 0 {
			yellow.Println("[*] No results found")
			return
		}

		for _, r := range results {
			green.Printf("[+] %s\n", r.URL)
		}

		fmt.Println()
		green.Printf("[+] Found %d result(s)\n", len(results))
	},
}

func splitExtensions(ext string) []string {
	if ext == "" {
		return nil
	}
	// 移除空格并按逗号分割
	exts := make([]string, 0)
	for _, e := range strings.Split(ext, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			if e[0] != '.' {
				e = "." + e
			}
			exts = append(exts, e)
		}
	}
	return exts
}

func init() {
	// Dir scan
	webscanDirCmd.Flags().StringP("url", "u", "", "Target URL")
	webscanDirCmd.Flags().StringP("tool", "T", "dirsearch", "Tool to use (dirsearch, gobuster, ffuf)")
	webscanDirCmd.Flags().StringP("wordlist", "w", "", "Wordlist file")
	webscanDirCmd.Flags().StringP("extensions", "e", "", "File extensions (e.g., php,asp,jsp)")
	webscanCmd.AddCommand(webscanDirCmd)

	// Fuzz
	webscanFuzzCmd.Flags().StringP("url", "u", "", "Target URL with FUZZ keyword")
	webscanFuzzCmd.Flags().StringP("wordlist", "w", "", "Wordlist file")
	webscanCmd.AddCommand(webscanFuzzCmd)
}
