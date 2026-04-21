package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Manage tool dependencies",
	Long:  `Check, install, and update external tool dependencies.`,
}

var depsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check tool dependencies status",
	Long:  `Check if all required tools are installed and available.`,
	Run:   runDepsCheck,
}

var depsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all supported tools",
	Long:  `List all tools that fcapital can integrate with.`,
	Run:   runDepsList,
}

var depsInstallCmd = &cobra.Command{
	Use:   "install [tools...]",
	Short: "Install missing tools",
	Long:  `Install specified tools or all missing tools if no arguments provided.`,
	Run:   runDepsInstall,
}

func init() {
	depsCmd.AddCommand(depsCheckCmd)
	depsCmd.AddCommand(depsListCmd)
	depsCmd.AddCommand(depsInstallCmd)
}

func runDepsCheck(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Checking tool dependencies...")
	fmt.Printf("[*] OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	tm := InitToolManager()
	tools := tm.List()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOOL\tCATEGORY\tSTATUS\tSOURCE\tVERSION")

	ready := 0
	missing := 0

	for _, tool := range tools {
		status := "❌ Missing"
		if tool.Status == 1 { // StatusReady
			status = "✅ Ready"
			ready++
		} else {
			missing++
		}

		source := "-"
		if tool.Source == 1 { // SourceSystem
			source = "system"
		} else if tool.Source == 2 { // SourceLocal
			source = "local"
		}

		version := tool.Version
		if version == "" {
			version = "-"
		}
		if len(version) > 30 {
			version = version[:30] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", tool.Binary, tool.Category, status, source, version)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("[*] Ready: %d, Missing: %d\n", ready, missing)

	if missing > 0 {
		yellow.Println("\n  [!] Some tools are missing. Run 'fcapital deps install' for installation instructions.")
	} else {
		green.Println("\n  [+] All tools are ready!")
	}
}

func runDepsList(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Supported tools:")
	fmt.Println()

	tm := GetToolManager()
	tools := tm.List()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOOL\tCATEGORY\tDESCRIPTION")

	for _, tool := range tools {
		fmt.Fprintf(w, "%s\t%s\t%s\n", tool.Binary, tool.Category, tool.Description)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("[*] Run 'fcapital deps check' to see which tools are installed.")
}

func runDepsInstall(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Tool Installation")
	fmt.Println()

	tm := InitToolManager()

	if len(args) == 0 {
		missing := tm.GetMissingTools()
		if len(missing) == 0 {
			green.Println("  [+] All tools are already installed!")
			return
		}
		fmt.Printf("[*] Found %d missing tools:\n\n", len(missing))

		for _, tool := range missing {
			fmt.Printf("  %s (%s):\n", tool.Binary, tool.Description)
			if err := installTool(tool); err != nil {
				red.Printf("    [!] Failed: %v\n", err)
			}
			fmt.Println()
		}
	} else {
		for _, name := range args {
			tool, err := tm.Get(name)
			if err != nil {
				red.Printf("  [!] Unknown tool: %s\n", name)
				continue
			}
			fmt.Printf("  %s:\n", tool.Binary)
			if tool.IsReady() {
				green.Println("    [+] Already installed")
			} else {
				if err := installTool(tool); err != nil {
					red.Printf("    [!] Failed: %v\n", err)
				}
			}
			fmt.Println()
		}
	}

	// 重新检测工具状态
	fmt.Println("\n[*] Re-checking tools...")
	tm2 := InitToolManager()
	ready := 0
	missing := 0
	for _, t := range tm2.List() {
		if t.IsReady() {
			ready++
		} else {
			missing++
		}
	}
	fmt.Printf("[*] Ready: %d, Missing: %d\n", ready, missing)
}

// installTool 尝试安装工具
func installTool(tool *toolmgr.Tool) error {
	install := tool.Install

	// 优先尝试 Go 安装
	if install.Go != "" {
		fmt.Printf("    [*] Trying Go install...\n")
		if err := runGoInstall(install.Go); err != nil {
			fmt.Printf("    [!] Go install failed: %v\n", err)
		} else {
			green.Println("    [+] Go install successful!")
			return nil
		}
	}

	// 尝试 Pip 安装
	if install.Pip != "" {
		fmt.Printf("    [*] Trying Pip install...\n")
		if err := runPipInstall(install.Pip); err != nil {
			fmt.Printf("    [!] Pip install failed: %v\n", err)
		} else {
			green.Println("    [+] Pip install successful!")
			return nil
		}
	}

	// 尝试 Gem 安装
	if install.Gem != "" {
		fmt.Printf("    [*] Trying Gem install...\n")
		if err := runGemInstall(install.Gem); err != nil {
			fmt.Printf("    [!] Gem install failed: %v\n", err)
		} else {
			green.Println("    [+] Gem install successful!")
			return nil
		}
	}

	// 尝试系统包管理器
	if runtime.GOOS == "windows" {
		if install.Windows != "" {
			// 检查是否有 choco 或 scoop
			if hasCommand("choco") {
				fmt.Printf("    [*] Trying Chocolatey...\n")
				// 提取 choco 命令
				if strings.Contains(install.Windows, "choco") {
					if err := runCommand("choco", "install", tool.Binary, "-y"); err != nil {
						fmt.Printf("    [!] Chocolatey install failed: %v\n", err)
					} else {
						green.Println("    [+] Chocolatey install successful!")
						return nil
					}
				}
			}
			// 打印手动安装说明
			yellow.Printf("    [!] Automatic install not available. Manual install:\n")
			fmt.Printf("        %s\n", install.Windows)
		}
	} else if runtime.GOOS == "darwin" {
		if install.MacOS != "" && strings.Contains(install.MacOS, "brew") {
			fmt.Printf("    [*] Trying Homebrew...\n")
			if err := runCommand("brew", "install", tool.Binary); err != nil {
				fmt.Printf("    [!] Homebrew install failed: %v\n", err)
			} else {
				green.Println("    [+] Homebrew install successful!")
				return nil
			}
		}
	} else {
		if install.Linux != "" && strings.Contains(install.Linux, "apt") {
			fmt.Printf("    [*] Trying apt...\n")
			if err := runCommand("sudo", "apt", "install", tool.Binary, "-y"); err != nil {
				fmt.Printf("    [!] apt install failed: %v\n", err)
			} else {
				green.Println("    [+] apt install successful!")
				return nil
			}
		}
	}

	return fmt.Errorf("no automatic installation method available")
}

// runGoInstall 执行 Go 安装命令
func runGoInstall(pkg string) error {
	// 解析包路径，支持多种格式:
	// - "go install github.com/OJ/gobuster/v3@latest"
	// - "github.com/OJ/gobuster/v3@latest"
	pkgPath := strings.TrimSpace(pkg)
	pkgPath = strings.TrimPrefix(pkgPath, "go install ")
	pkgPath = strings.TrimPrefix(pkgPath, "go install")

	cmd := exec.Command("go", "install", pkgPath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// runPipInstall 执行 Pip 安装命令
func runPipInstall(pkg string) error {
	// 解析包名，支持多种格式:
	// - "pip install dirsearch"
	// - "dirsearch"
	pkgName := strings.TrimSpace(pkg)
	pkgName = strings.TrimPrefix(pkgName, "pip install ")
	pkgName = strings.TrimPrefix(pkgName, "pip install")

	// 使用 --user 确保用户安装
	cmd := exec.Command("pip", "install", "--user", pkgName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// runGemInstall 执行 Gem 安装命令
func runGemInstall(pkg string) error {
	cmd := exec.Command("gem", "install", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// runCommand 执行通用命令
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// hasCommand 检查命令是否存在
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// confirmInstall 确认安装
func confirmInstall(tool string) bool {
	fmt.Printf("    Install %s? [y/N]: ", tool)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
