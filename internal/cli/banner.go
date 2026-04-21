package cli

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan).Add(color.Bold)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgYellow)
	red    = color.New(color.FgRed)
	bold   = color.New(color.Bold)
)

func printBanner() {
	banner := `
  ███████╗ ██████╗ █████╗ ██╗     ███████╗██╗██████╗ ███████╗
  ██╔════╝██╔════╝██╔══██╗██║     ██╔════╝██║██╔══██╗██╔════╝
  █████╗  ██║     ███████║██║     █████╗  ██║██║  ██║█████╗
  ██╔══╝  ██║     ██╔══██║██║     ██╔══╝  ██║██║  ██║██╔══╝
  ██║     ╚██████╗██║  ██║███████╗███████╗██║██████╔╝███████╗
  ╚═╝      ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚═╝╚═════╝ ╚══════╝
`
	cyan.Println(banner)
	fmt.Println()
	green.Println("  [!] A Comprehensive Penetration Testing Framework")
	green.Println("  [!] Version: 1.0.0")
	fmt.Println()
}

func printWarning() {
	yellow.Println("  ⚠️  WARNING - LEGAL DISCLAIMER")
	fmt.Println()
	fmt.Println("  fcapital is designed for authorized security testing and educational")
	fmt.Println("  purposes only. Unauthorized use of this tool against systems you do")
	fmt.Println("  not own or have explicit permission to test is ILLEGAL.")
	fmt.Println()
	fmt.Println("  By using fcapital, you agree to:")
	fmt.Println("  1. Only test systems you own or have written authorization to test")
	fmt.Println("  2. Comply with all applicable laws and regulations")
	fmt.Println("  3. Accept full responsibility for your actions")
	fmt.Println()
}

func printMenu() {
	fmt.Println()
	bold.Println("  ┌─────────────────────────────────────────┐")
	bold.Println("  │           MAIN MENU                      │")
	bold.Println("  ├─────────────────────────────────────────┤")
	fmt.Println("  │  [1] Information Gathering               │")
	fmt.Println("  │  [2] Subdomain Enumeration               │")
	fmt.Println("  │  [3] Port Scanning                       │")
	fmt.Println("  │  [4] Web Scanning                        │")
	fmt.Println("  │  [5] Vulnerability Scanning              │")
	fmt.Println("  │  [6] Password Attacks                    │")
	fmt.Println("  │  [7] Utilities                           │")
	fmt.Println("  │  [8] Check Dependencies                  │")
	fmt.Println("  │  [0] Exit                                │")
	bold.Println("  └─────────────────────────────────────────┘")
	fmt.Println()
}
