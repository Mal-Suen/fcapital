package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mal-Suen/fcapital/internal/modules/utils"
)

func runInteractive() {
	printBanner()
	printWarning()

	reader := bufio.NewReader(os.Stdin)

	// 初始化工具管理器
	InitToolManager()

	for {
		printMenu()
		fmt.Print("  >> Enter your choice: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil {
			red.Println("  [!] Invalid input. Please enter a number.")
			continue
		}

		switch choice {
		case 1:
			runReconInteractive(reader)
		case 2:
			runSubdomainInteractive(reader)
		case 3:
			runPortscanInteractive(reader)
		case 4:
			runWebscanInteractive(reader)
		case 5:
			runVulnscanInteractive(reader)
		case 6:
			runPasswordInteractive(reader)
		case 7:
			runUtilsInteractive(reader)
		case 8:
			runDepsCheckInteractive()
		case 0:
			cyan.Println("\n  [!] Goodbye!")
			return
		default:
			red.Println("  [!] Invalid choice. Please try again.")
		}
	}
}

func runReconInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Information Gathering ===")
	fmt.Println()
	fmt.Println("  [1] HTTP Probe (httpx)")
	fmt.Println("  [2] DNS Query (dnsx)")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter target (domain/IP): ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running HTTP probe on %s...\n", target)
			// TODO: 调用 httpx
			green.Println("  [+] HTTP probe completed (placeholder)")
		}
	case "2":
		fmt.Print("  >> Enter domain: ")
		domain, _ := reader.ReadString('\n')
		domain = strings.TrimSpace(domain)
		if domain != "" {
			fmt.Printf("\n  [*] Running DNS query on %s...\n", domain)
			// TODO: 调用 dnsx
			green.Println("  [+] DNS query completed (placeholder)")
		}
	case "0":
		return
	}
}

func runSubdomainInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Subdomain Enumeration ===")
	fmt.Println()
	fmt.Println("  [1] Passive Enumeration (subfinder)")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter domain: ")
		domain, _ := reader.ReadString('\n')
		domain = strings.TrimSpace(domain)
		if domain != "" {
			fmt.Printf("\n  [*] Running subdomain enumeration on %s...\n", domain)
			// TODO: 调用 subfinder
			green.Println("  [+] Subdomain enumeration completed (placeholder)")
		}
	case "0":
		return
	}
}

func runPortscanInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Port Scanning ===")
	fmt.Println()
	fmt.Println("  [1] Quick Scan (Top 100 ports)")
	fmt.Println("  [2] Full Scan (All ports)")
	fmt.Println("  [3] Custom Scan")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter target: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running quick port scan on %s...\n", target)
			// TODO: 调用 nmap
			green.Println("  [+] Port scan completed (placeholder)")
		}
	case "2":
		fmt.Print("  >> Enter target: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running full port scan on %s...\n", target)
			// TODO: 调用 nmap
			green.Println("  [+] Port scan completed (placeholder)")
		}
	case "3":
		fmt.Print("  >> Enter target: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		fmt.Print("  >> Enter port range (e.g., 1-1000): ")
		ports, _ := reader.ReadString('\n')
		ports = strings.TrimSpace(ports)
		if target != "" && ports != "" {
			fmt.Printf("\n  [*] Running port scan on %s (ports: %s)...\n", target, ports)
			// TODO: 调用 nmap
			green.Println("  [+] Port scan completed (placeholder)")
		}
	case "0":
		return
	}
}

func runWebscanInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Web Scanning ===")
	fmt.Println()
	fmt.Println("  [1] Directory Scan (dirsearch)")
	fmt.Println("  [2] Directory Scan (gobuster)")
	fmt.Println("  [3] Directory Scan (ffuf)")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1", "2", "3":
		fmt.Print("  >> Enter target URL: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running directory scan on %s...\n", target)
			// TODO: 调用相应工具
			green.Println("  [+] Directory scan completed (placeholder)")
		}
	case "0":
		return
	}
}

func runVulnscanInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Vulnerability Scanning ===")
	fmt.Println()
	fmt.Println("  [1] Nuclei Scan")
	fmt.Println("  [2] SQLMap Scan")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter target URL: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running Nuclei scan on %s...\n", target)
			// TODO: 调用 nuclei
			green.Println("  [+] Nuclei scan completed (placeholder)")
		}
	case "2":
		fmt.Print("  >> Enter target URL: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running SQLMap on %s...\n", target)
			// TODO: 调用 sqlmap
			green.Println("  [+] SQLMap scan completed (placeholder)")
		}
	case "0":
		return
	}
}

func runPasswordInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Password Attacks ===")
	fmt.Println()
	fmt.Println("  [1] Hydra Brute Force")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter target: ")
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target != "" {
			fmt.Printf("\n  [*] Running Hydra on %s...\n", target)
			// TODO: 调用 hydra
			green.Println("  [+] Hydra completed (placeholder)")
		}
	case "0":
		return
	}
}

func runUtilsInteractive(reader *bufio.Reader) {
	fmt.Println()
	bold.Println("  === Utilities ===")
	fmt.Println()
	fmt.Println("  [1] Base64 Encode/Decode")
	fmt.Println("  [2] URL Encode/Decode")
	fmt.Println("  [3] Hash Calculator")
	fmt.Println("  [0] Back")
	fmt.Println()
	fmt.Print("  >> Enter your choice: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		fmt.Print("  >> Enter string: ")
		str, _ := reader.ReadString('\n')
		str = strings.TrimSpace(str)
		fmt.Printf("\n  [+] Base64 encoded: %s\n", base64Encode(str))
	case "2":
		fmt.Print("  >> Enter string: ")
		str, _ := reader.ReadString('\n')
		str = strings.TrimSpace(str)
		fmt.Printf("\n  [+] URL encoded: %s\n", urlEncode(str))
	case "3":
		fmt.Print("  >> Enter string: ")
		str, _ := reader.ReadString('\n')
		str = strings.TrimSpace(str)
		fmt.Printf("\n  [+] MD5: %s\n", md5Hash(str))
		fmt.Printf("  [+] SHA256: %s\n", sha256Hash(str))
	case "0":
		return
	}
}

func runDepsCheckInteractive() {
	fmt.Println()
	runDepsCheck(nil, nil)
}

// 辅助函数
func base64Encode(s string) string {
	return utils.Base64Encode(s)
}

func urlEncode(s string) string {
	return utils.URLEncode(s)
}

func md5Hash(s string) string {
	return utils.MD5Hash(s)
}

func sha256Hash(s string) string {
	return utils.SHA256Hash(s)
}
