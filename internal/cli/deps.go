package cli

import (
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
		yellow.Println("\n  [!] Some tools are missing. Run 'fcapital deps install' to install them.")
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
	fmt.Printf("[*] OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
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

	// 1. 优先尝试 Go 安装 (跨平台)
	if install.Go != "" && hasCommand("go") {
		fmt.Printf("    [*] Trying Go install...\n")
		if err := runGoInstall(install.Go); err != nil {
			fmt.Printf("    [!] Go install failed: %v\n", err)
		} else {
			green.Println("    [+] Go install successful!")
			return nil
		}
	}

	// 2. 尝试 Pip 安装 (跨平台)
	if install.Pip != "" && hasCommand("pip") {
		fmt.Printf("    [*] Trying Pip install...\n")
		if err := runPipInstall(install.Pip); err != nil {
			fmt.Printf("    [!] Pip install failed: %v\n", err)
		} else {
			green.Println("    [+] Pip install successful!")
			return nil
		}
	}

	// 3. 尝试 Pip3 安装 (跨平台)
	if install.Pip3 != "" && hasCommand("pip3") {
		fmt.Printf("    [*] Trying Pip3 install...\n")
		if err := runPip3Install(install.Pip3); err != nil {
			fmt.Printf("    [!] Pip3 install failed: %v\n", err)
		} else {
			green.Println("    [+] Pip3 install successful!")
			return nil
		}
	}

	// 4. 尝试 Gem 安装 (跨平台)
	if install.Gem != "" && hasCommand("gem") {
		fmt.Printf("    [*] Trying Gem install...\n")
		if err := runGemInstall(install.Gem); err != nil {
			fmt.Printf("    [!] Gem install failed: %v\n", err)
		} else {
			green.Println("    [+] Gem install successful!")
			return nil
		}
	}

	// 5. 尝试 Cargo 安装 (跨平台)
	if install.Cargo != "" && hasCommand("cargo") {
		fmt.Printf("    [*] Trying Cargo install...\n")
		if err := runCargoInstall(install.Cargo); err != nil {
			fmt.Printf("    [!] Cargo install failed: %v\n", err)
		} else {
			green.Println("    [+] Cargo install successful!")
			return nil
		}
	}

	// 6. 尝试 Npm 安装 (跨平台)
	if install.Npm != "" && hasCommand("npm") {
		fmt.Printf("    [*] Trying Npm install...\n")
		if err := runNpmInstall(install.Npm); err != nil {
			fmt.Printf("    [!] Npm install failed: %v\n", err)
		} else {
			green.Println("    [+] Npm install successful!")
			return nil
		}
	}

	// 7. 根据操作系统尝试系统包管理器
	switch runtime.GOOS {
	case "windows":
		return installWindows(tool)
	case "darwin":
		return installMacOS(tool)
	default:
		return installLinux(tool)
	}
}

// installWindows Windows 系统安装
func installWindows(tool *toolmgr.Tool) error {
	install := tool.Install

	// Chocolatey
	if install.Choco != "" && hasCommand("choco") {
		fmt.Printf("    [*] Trying Chocolatey...\n")
		if err := runCommand("choco", "install", tool.Binary, "-y"); err != nil {
			fmt.Printf("    [!] Chocolatey failed: %v\n", err)
		} else {
			green.Println("    [+] Chocolatey install successful!")
			return nil
		}
	}

	// Scoop
	if install.Scoop != "" && hasCommand("scoop") {
		fmt.Printf("    [*] Trying Scoop...\n")
		if err := runCommand("scoop", "install", tool.Binary); err != nil {
			fmt.Printf("    [!] Scoop failed: %v\n", err)
		} else {
			green.Println("    [+] Scoop install successful!")
			return nil
		}
	}

	// Winget
	if install.Winget != "" && hasCommand("winget") {
		fmt.Printf("    [*] Trying Winget...\n")
		// Winget 需要使用包 ID
		if err := runCommand("winget", "install", "--id", install.Winget, "-e", "--accept-source-agreements", "--accept-package-agreements"); err != nil {
			fmt.Printf("    [!] Winget failed: %v\n", err)
		} else {
			green.Println("    [+] Winget install successful!")
			return nil
		}
	}

	// 手动安装说明
	printManualInstall(tool, "windows")
	return fmt.Errorf("no automatic installation method available on Windows")
}

// installMacOS macOS 系统安装
func installMacOS(tool *toolmgr.Tool) error {
	install := tool.Install

	// Homebrew
	if install.Brew != "" && hasCommand("brew") {
		fmt.Printf("    [*] Trying Homebrew...\n")
		if err := runCommand("brew", "install", tool.Binary); err != nil {
			fmt.Printf("    [!] Homebrew failed: %v\n", err)
		} else {
			green.Println("    [+] Homebrew install successful!")
			return nil
		}
	}

	// MacPorts
	if install.Port != "" && hasCommand("port") {
		fmt.Printf("    [*] Trying MacPorts...\n")
		if err := runCommand("sudo", "port", "install", tool.Binary); err != nil {
			fmt.Printf("    [!] MacPorts failed: %v\n", err)
		} else {
			green.Println("    [+] MacPorts install successful!")
			return nil
		}
	}

	// 手动安装说明
	printManualInstall(tool, "macos")
	return fmt.Errorf("no automatic installation method available on macOS")
}

// installLinux Linux 系统安装
func installLinux(tool *toolmgr.Tool) error {
	install := tool.Install

	// 检测 Linux 发行版并选择合适的包管理器
	// 优先级: apt > dnf > yum > pacman > zypper > apk

	// Debian/Ubuntu/Kali
	if install.Apt != "" && hasCommand("apt") {
		fmt.Printf("    [*] Trying apt (Debian/Ubuntu/Kali)...\n")
		if err := runCommand("sudo", "apt", "install", tool.Binary, "-y"); err != nil {
			fmt.Printf("    [!] apt failed: %v\n", err)
		} else {
			green.Println("    [+] apt install successful!")
			return nil
		}
	}

	// Fedora/RHEL 8+
	if install.Dnf != "" && hasCommand("dnf") {
		fmt.Printf("    [*] Trying dnf (Fedora/RHEL 8+)...\n")
		if err := runCommand("sudo", "dnf", "install", tool.Binary, "-y"); err != nil {
			fmt.Printf("    [!] dnf failed: %v\n", err)
		} else {
			green.Println("    [+] dnf install successful!")
			return nil
		}
	}

	// RHEL/CentOS (旧版本)
	if install.Yum != "" && hasCommand("yum") && !hasCommand("dnf") {
		fmt.Printf("    [*] Trying yum (RHEL/CentOS)...\n")
		if err := runCommand("sudo", "yum", "install", tool.Binary, "-y"); err != nil {
			fmt.Printf("    [!] yum failed: %v\n", err)
		} else {
			green.Println("    [+] yum install successful!")
			return nil
		}
	}

	// Arch Linux
	if install.Pacman != "" && hasCommand("pacman") {
		fmt.Printf("    [*] Trying pacman (Arch Linux)...\n")
		if err := runCommand("sudo", "pacman", "-S", tool.Binary, "--noconfirm"); err != nil {
			fmt.Printf("    [!] pacman failed: %v\n", err)
		} else {
			green.Println("    [+] pacman install successful!")
			return nil
		}
	}

	// openSUSE
	if install.Zypper != "" && hasCommand("zypper") {
		fmt.Printf("    [*] Trying zypper (openSUSE)...\n")
		if err := runCommand("sudo", "zypper", "install", "-y", tool.Binary); err != nil {
			fmt.Printf("    [!] zypper failed: %v\n", err)
		} else {
			green.Println("    [+] zypper install successful!")
			return nil
		}
	}

	// Alpine Linux
	if install.Apk != "" && hasCommand("apk") {
		fmt.Printf("    [*] Trying apk (Alpine Linux)...\n")
		if err := runCommand("sudo", "apk", "add", tool.Binary); err != nil {
			fmt.Printf("    [!] apk failed: %v\n", err)
		} else {
			green.Println("    [+] apk install successful!")
			return nil
		}
	}

	// 手动安装说明
	printManualInstall(tool, "linux")
	return fmt.Errorf("no automatic installation method available on Linux")
}

// printManualInstall 打印手动安装说明
func printManualInstall(tool *toolmgr.Tool, os string) {
	yellow.Println("    [!] Automatic install not available. Manual install options:")

	install := tool.Install

	// 打印所有可用的安装方式
	if install.Manual != "" {
		fmt.Printf("        - %s\n", install.Manual)
	}

	switch os {
	case "windows":
		if install.ManualWindows != "" {
			fmt.Printf("        - %s\n", install.ManualWindows)
		}
		if install.Choco != "" && !hasCommand("choco") {
			fmt.Printf("        - Install Chocolatey: https://chocolatey.org/install\n")
			fmt.Printf("          Then: choco install %s -y\n", tool.Binary)
		}
		if install.Scoop != "" && !hasCommand("scoop") {
			fmt.Printf("        - Install Scoop: https://scoop.sh\n")
			fmt.Printf("          Then: scoop install %s\n", tool.Binary)
		}
	case "macos":
		if install.ManualMacOS != "" {
			fmt.Printf("        - %s\n", install.ManualMacOS)
		}
		if install.Brew != "" && !hasCommand("brew") {
			fmt.Printf("        - Install Homebrew: https://brew.sh\n")
			fmt.Printf("          Then: brew install %s\n", tool.Binary)
		}
	case "linux":
		if install.ManualLinux != "" {
			fmt.Printf("        - %s\n", install.ManualLinux)
		}
		if install.Git != "" {
			fmt.Printf("        - Clone: %s\n", install.Git)
		}
	}

	if install.Docker != "" {
		fmt.Printf("        - Docker: %s\n", install.Docker)
	}
}

// runGoInstall 执行 Go 安装命令
func runGoInstall(pkg string) error {
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
	pkgName := strings.TrimSpace(pkg)
	pkgName = strings.TrimPrefix(pkgName, "pip install ")
	pkgName = strings.TrimPrefix(pkgName, "pip install")

	cmd := exec.Command("pip", "install", "--user", pkgName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// runPip3Install 执行 Pip3 安装命令
func runPip3Install(pkg string) error {
	pkgName := strings.TrimSpace(pkg)
	cmd := exec.Command("pip3", "install", "--user", pkgName)
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

// runCargoInstall 执行 Cargo 安装命令
func runCargoInstall(pkg string) error {
	cmd := exec.Command("cargo", "install", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// runNpmInstall 执行 Npm 安装命令
func runNpmInstall(pkg string) error {
	cmd := exec.Command("npm", "install", "-g", pkg)
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
