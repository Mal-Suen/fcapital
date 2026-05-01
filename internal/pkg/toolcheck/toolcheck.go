package toolcheck

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ToolInfo 工具信息
type ToolInfo struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Supported bool   `json:"supported"` // 是否支持当前系统
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
	Category  string `json:"category"` // scanner, enumerator, exploiter, utility
}

// ToolRegistry 工具注册表
var ToolRegistry = []ToolInfo{
	// 扫描器
	{Name: "nmap", Category: "scanner"},
	{Name: "nuclei", Category: "scanner"},
	{Name: "masscan", Category: "scanner"},
	{Name: "zmap", Category: "scanner"},
	
	// Web扫描器
	{Name: "nikto", Category: "scanner"},
	{Name: "wpscan", Category: "scanner"},
	{Name: "joomscan", Category: "scanner"},
	{Name: "whatweb", Category: "scanner"},
	{Name: "httpx", Category: "scanner"},
	{Name: "ffuf", Category: "scanner"},
	{Name: "gobuster", Category: "scanner"},
	{Name: "feroxbuster", Category: "scanner"},
	{Name: "dirsearch", Category: "scanner"},
	
	// 子域名枚举
	{Name: "subfinder", Category: "enumerator"},
	{Name: "amass", Category: "enumerator"},
	{Name: "dnsx", Category: "enumerator"},
	
	// 漏洞利用
	{Name: "sqlmap", Category: "exploiter"},
	{Name: "hydra", Category: "exploiter"},
	{Name: "medusa", Category: "exploiter"},
	{Name: "ncrack", Category: "exploiter"},
	
	// SSL/证书
	{Name: "sslscan", Category: "utility"},
	{Name: "testssl.sh", Category: "utility"},
	{Name: "openssl", Category: "utility"},
	
	// 其他工具
	{Name: "curl", Category: "utility"},
	{Name: "wget", Category: "utility"},
	{Name: "python", Category: "utility"},
	{Name: "python3", Category: "utility"},
	{Name: "ruby", Category: "utility"},
	{Name: "perl", Category: "utility"},
	{Name: "go", Category: "utility"},
	{Name: "git", Category: "utility"},
	{Name: "docker", Category: "utility"},
}

// CheckResult 检测结果
type CheckResult struct {
	Available   []ToolInfo `json:"available"`
	Missing     []ToolInfo `json:"missing"`
	TotalCount  int        `json:"total_count"`
	InstalledCount int     `json:"installed_count"`
}

// Checker 工具检测器
type Checker struct {
	tools []ToolInfo
}

// NewChecker 创建新的检测器
func NewChecker() *Checker {
	return &Checker{
		tools: ToolRegistry,
	}
}

// CheckAll 检测所有工具
func (c *Checker) CheckAll() *CheckResult {
	result := &CheckResult{
		Available:   []ToolInfo{},
		Missing:     []ToolInfo{},
		TotalCount:  len(c.tools),
		InstalledCount: 0,
	}

	for _, tool := range c.tools {
		info := c.CheckTool(tool.Name)
		info.Category = tool.Category
		
		if info.Installed {
			result.Available = append(result.Available, info)
			result.InstalledCount++
		} else {
			result.Missing = append(result.Missing, info)
		}
	}

	return result
}

// CheckTool 检测单个工具
func (c *Checker) CheckTool(name string) ToolInfo {
	info := ToolInfo{
		Name:      name,
		Installed: false,
		Supported: isToolSupported(name), // 检查是否支持当前系统
	}

	// 尝试查找工具路径
	path, err := exec.LookPath(name)
	if err == nil {
		info.Installed = true
		info.Path = path

		// 尝试获取版本
		version := c.getToolVersion(name, path)
		if version != "" {
			info.Version = version
		}
	}

	return info
}

// isToolSupported 检查工具是否支持当前系统
func isToolSupported(name string) bool {
	os := runtime.GOOS

	// Windows 不支持的工具列表
	windowsUnsupported := map[string]bool{
		"wpscan":    true, // 需要 Ruby + 特定依赖，Windows 上很难安装
		"hydra":     true, // 需要 Cygwin 或 WSL
		"medusa":    true,
		"ncrack":    true,
		"testssl.sh": true, // 需要 bash 环境
		"joomscan":  true,  // Perl 脚本，Windows 兼容性差
		"whatweb":   true,  // Ruby 工具，Windows 兼容性差
	}

	if os == "windows" {
		return !windowsUnsupported[name]
	}

	return true // Linux 和 macOS 支持大多数工具
}

// getToolVersion 获取工具版本
func (c *Checker) getToolVersion(name, path string) string {
	var cmd *exec.Cmd
	
	switch name {
	case "nmap":
		cmd = exec.Command(path, "--version")
	case "nuclei":
		cmd = exec.Command(path, "--version")
	case "gobuster":
		cmd = exec.Command(path, "--version")
	case "ffuf":
		cmd = exec.Command(path, "-V")
	case "subfinder":
		cmd = exec.Command(path, "--version")
	case "amass":
		cmd = exec.Command(path, "--version")
	case "httpx":
		cmd = exec.Command(path, "--version")
	case "sqlmap":
		cmd = exec.Command(path, "--version")
	case "hydra":
		cmd = exec.Command(path, "-h")
	case "curl":
		cmd = exec.Command(path, "--version")
	case "wget":
		cmd = exec.Command(path, "--version")
	case "python", "python3":
		cmd = exec.Command(path, "--version")
	case "go":
		cmd = exec.Command(path, "version")
	case "git":
		cmd = exec.Command(path, "--version")
	case "docker":
		cmd = exec.Command(path, "--version")
	case "openssl":
		cmd = exec.Command(path, "version")
	default:
		// 尝试通用 --version
		cmd = exec.Command(path, "--version")
	}

	if cmd == nil {
		return ""
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 尝试 -V
		cmd = exec.Command(path, "-V")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return ""
		}
	}

	// 提取版本号（取第一行）
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// 截取前50个字符
		if len(firstLine) > 50 {
			return firstLine[:50]
		}
		return firstLine
	}

	return ""
}

// IsToolAvailable 检查工具是否可用
func (c *Checker) IsToolAvailable(name string) bool {
	info := c.CheckTool(name)
	return info.Installed
}

// GetToolInfo 获取工具信息
func (c *Checker) GetToolInfo(name string) ToolInfo {
	return c.CheckTool(name)
}

// FormatToolList 格式化工具列表用于 AI prompt
func (c *Checker) FormatToolList(result *CheckResult) string {
	var sb strings.Builder
	
	sb.WriteString("## 本机已安装的工具:\n")
	for _, tool := range result.Available {
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", tool.Name, tool.Category, tool.Version))
	}
	
	sb.WriteString("\n## 本机未安装的工具:\n")
	for _, tool := range result.Missing {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", tool.Name, tool.Category))
	}
	
	return sb.String()
}

// FormatAvailableTools 格式化可用工具列表（简洁版）
func (c *Checker) FormatAvailableTools(result *CheckResult) string {
	var tools []string
	for _, tool := range result.Available {
		tools = append(tools, tool.Name)
	}
	return strings.Join(tools, ", ")
}

// GetInstallInstructions 获取安装指令
func GetInstallInstructions(name string) string {
	os := runtime.GOOS
	
	switch name {
	case "wpscan":
		switch os {
		case "windows":
			return `# WPScan 安装 (Windows)
方法1: 使用 Docker (推荐)
  docker pull wpscanteam/wpscan
  docker run --rm wpscanteam/wpscan --url https://target.com

方法2: 使用 Ruby (需要 Ruby 环境)
  gem install wpscan

方法3: 手动下载
  访问 https://github.com/wpscanteam/wpscan/releases`
		case "linux", "darwin":
			return `# WPScan 安装
方法1: Ruby gem (推荐)
  gem install wpscan

方法2: Docker
  docker pull wpscanteam/wpscan

方法3: 包管理器
  # Ubuntu/Debian
  sudo apt install wpscan
  
  # macOS
  brew install wpscan`
		}
		
	case "nuclei":
		switch os {
		case "windows":
			return `# Nuclei 安装 (Windows)
方法1: Go install (推荐)
  go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

方法2: 下载预编译版本
  访问 https://github.com/projectdiscovery/nuclei/releases

方法3: Docker
  docker pull projectdiscovery/nuclei`
		case "linux", "darwin":
			return `# Nuclei 安装
方法1: Go install (推荐)
  go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

方法2: 包管理器
  # Ubuntu/Debian (需要先安装 go)
  sudo snap install nuclei
  
  # macOS
  brew install nuclei

方法3: Docker
  docker pull projectdiscovery/nuclei`
		}
		
	case "gobuster":
		switch os {
		case "windows":
			return `# Gobuster 安装 (Windows)
方法1: Go install
  go install github.com/OJ/gobuster/v3@latest

方法2: 下载预编译版本
  访问 https://github.com/OJ/gobuster/releases`
		case "linux", "darwin":
			return `# Gobuster 安装
方法1: Go install
  go install github.com/OJ/gobuster/v3@latest

方法2: 包管理器
  # macOS
  brew install gobuster`
		}
		
	case "ffuf":
		switch os {
		case "windows":
			return `# FFUF 安装 (Windows)
方法1: Go install
  go install github.com/ffuf/ffuf/v2@latest

方法2: 下载预编译版本
  访问 https://github.com/ffuf/ffuf/releases`
		case "linux", "darwin":
			return `# FFUF 安装
方法1: Go install
  go install github.com/ffuf/ffuf/v2@latest

方法2: 包管理器
  # macOS
  brew install ffuf`
		}
		
	case "subfinder":
		switch os {
		case "windows":
			return `# Subfinder 安装 (Windows)
方法1: Go install
  go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest

方法2: 下载预编译版本
  访问 https://github.com/projectdiscovery/subfinder/releases`
		case "linux", "darwin":
			return `# Subfinder 安装
方法1: Go install
  go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest

方法2: 包管理器
  # macOS
  brew install subfinder`
		}
		
	case "httpx":
		switch os {
		case "windows":
			return `# HTTPX 安装 (Windows)
方法1: Go install
  go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest

方法2: 下载预编译版本
  访问 https://github.com/projectdiscovery/httpx/releases`
		case "linux", "darwin":
			return `# HTTPX 安装
方法1: Go install
  go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest

方法2: 包管理器
  # macOS
  brew install httpx`
		}
		
	case "sqlmap":
		return `# SQLMap 安装
方法1: Git clone (推荐)
  git clone --depth 1 https://github.com/sqlmapproject/sqlmap.git sqlmap-dev
  python sqlmap-dev/sqlmap.py -u "target"

方法2: pip
  pip install sqlmap

方法3: Docker
  docker pull sqlmap/sqlmap`
		
	case "hydra":
		switch os {
		case "windows":
			return `# Hydra 安装 (Windows)
方法1: 下载预编译版本
  访问 https://github.com/maaaaz/thc-hydra-windows-release

方法2: Cygwin
  通过 Cygwin 安装 hydra 包`
		case "linux", "darwin":
			return `# Hydra 安装
方法1: 包管理器 (推荐)
  # Ubuntu/Debian
  sudo apt install hydra
  
  # macOS
  brew install hydra

方法2: 源码编译
  git clone https://github.com/vanhauser-thc/thc-hydra
  cd thc-hydra
  ./configure && make && make install`
		}
		
	case "nikto":
		return `# Nikto 安装
方法1: Git clone (推荐)
  git clone https://github.com/sullo/nikto.git
  cd nikto/program
  perl nikto.pl -h target

方法2: 包管理器
  # Ubuntu/Debian
  sudo apt install nikto
  
  # macOS
  brew install nikto

方法3: Docker
  docker pull sullo/nikto`
		
	case "amass":
		switch os {
		case "windows":
			return `# Amass 安装 (Windows)
方法1: Go install
  go install -v github.com/owasp-amass/amass/v4/cmd/amass@latest

方法2: 下载预编译版本
  访问 https://github.com/owasp-amass/amass/releases`
		case "linux", "darwin":
			return `# Amass 安装
方法1: Go install
  go install -v github.com/owasp-amass/amass/v4/cmd/amass@latest

方法2: 包管理器
  # macOS
  brew install amass`
		}
		
	case "dirsearch":
		return `# Dirsearch 安装
方法1: Git clone (推荐)
  git clone https://github.com/maurosoria/dirsearch
  cd dirsearch
  python3 dirsearch.py -u target

方法2: pip
  pip install dirsearch

方法3: Docker
  docker pull maurosoria/dirsearch`
		
	case "feroxbuster":
		switch os {
		case "windows":
			return `# Feroxbuster 安装 (Windows)
方法1: 下载预编译版本
  访问 https://github.com/epi052/feroxbuster/releases

方法2: PowerShell
  Invoke-Expression (Invoke-WebRequest -Uri https://raw.githubusercontent.com/epi052/feroxbuster/main/install.ps1)`
		case "linux", "darwin":
			return `# Feroxbuster 安装
方法1: 下载脚本
  curl -sL https://raw.githubusercontent.com/epi052/feroxbuster/main/install.sh | bash

方法2: 包管理器
  # macOS
  brew install feroxbuster`
		}
		
	case "sslscan":
		switch os {
		case "windows":
			return `# SSLScan 安装 (Windows)
方法1: 下载预编译版本
  访问 https://github.com/rbsec/sslscan/releases`
		case "linux", "darwin":
			return `# SSLScan 安装
方法1: 包管理器 (推荐)
  # Ubuntu/Debian
  sudo apt install sslscan
  
  # macOS
  brew install sslscan`
		}
		
	case "testssl.sh":
		return `# testssl.sh 安装
方法1: Git clone (推荐)
  git clone --depth 1 https://github.com/drwetter/testssl.sh.git
  cd testssl.sh
  ./testssl.sh target

方法2: Docker
  docker pull drwetter/testssl.sh`
		
	case "whatweb":
		switch os {
		case "windows":
			return `# WhatWeb 安装 (Windows)
方法1: Docker
  docker pull whatweb/whatweb

方法2: Ruby gem (需要 Ruby 环境)
  gem install whatweb`
		case "linux", "darwin":
			return `# WhatWeb 安装
方法1: 包管理器 (推荐)
  # Ubuntu/Debian
  sudo apt install whatweb
  
  # macOS
  brew install whatweb`
		}
		
	case "joomscan":
		return `# JoomScan 安装
方法1: Git clone (推荐)
  git clone https://github.com/OWASP/joomscan.git
  cd joomscan
  perl joomscan.pl -u target

方法2: Docker
  docker pull owasp/joomscan`
		
	case "masscan":
		switch os {
		case "windows":
			return `# Masscan 安装 (Windows)
方法1: 下载预编译版本
  访问 https://github.com/robertdavidgraham/masscan/releases`
		case "linux", "darwin":
			return `# Masscan 安装
方法1: 源码编译
  git clone https://github.com/robertdavidgraham/masscan
  cd masscan
  make && make install`
		}
		
	default:
		return fmt.Sprintf(`# %s 安装
请查阅官方文档或使用搜索引擎查找安装方法。
常见安装方式:
- Go 工具: go install github.com/xxx/%s@latest
- Python 工具: pip install %s
- Ruby 工具: gem install %s
- 包管理器: apt/brew/snap install %s
- Docker: docker pull xxx/%s`, name, name, name, name, name, name)
	}
	
	return ""
}

// TryAutoInstall 尝试自动安装工具
func TryAutoInstall(name string) (bool, string) {
	os := runtime.GOOS
	
	switch name {
	// Go 工具 - 可以自动安装
	case "nuclei", "subfinder", "httpx", "gobuster", "ffuf", "amass", "feroxbuster":
		cmd := exec.Command("go", "install", "-v", getGoPackage(name), "@latest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("Go install 失败: %v\n%s", err, string(output))
		}
		return true, fmt.Sprintf("成功通过 go install 安装 %s", name)

	// Python 工具 - 可以尝试 pip
	case "sqlmap", "dirsearch":
		cmd := exec.Command("pip", "install", name)
		_, err := cmd.CombinedOutput()
		if err != nil {
			// 尝试 pip3
			cmd = exec.Command("pip3", "install", name)
			_, err = cmd.CombinedOutput()
			if err != nil {
				return false, "pip install 失败，建议使用 git clone"
			}
		}
		return true, fmt.Sprintf("成功通过 pip 安装 %s", name)

	// Linux/macOS 包管理器
	case "hydra", "nikto", "sslscan", "whatweb":
		if os == "linux" {
			cmd := exec.Command("sudo", "apt", "install", "-y", name)
			_, err := cmd.CombinedOutput()
			if err != nil {
				return false, fmt.Sprintf("apt install 失败: %v", err)
			}
			return true, fmt.Sprintf("成功通过 apt 安装 %s", name)
		} else if os == "darwin" {
			cmd := exec.Command("brew", "install", name)
			_, err := cmd.CombinedOutput()
			if err != nil {
				return false, fmt.Sprintf("brew install 失败: %v", err)
			}
			return true, fmt.Sprintf("成功通过 brew 安装 %s", name)
		}
		return false, "Windows 上需要手动安装"

	// Ruby 工具
	case "wpscan":
		if os == "windows" {
			return false, "Windows 上建议使用 Docker: docker pull wpscanteam/wpscan"
		}
		cmd := exec.Command("gem", "install", name)
		_, err := cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("gem install 失败: %v\n建议使用 Docker", err)
		}
		return true, fmt.Sprintf("成功通过 gem 安装 %s", name)

	default:
		return false, "无法自动安装，请参考安装说明手动安装"
	}
}

// getGoPackage 获取 Go 包路径
func getGoPackage(name string) string {
	switch name {
	case "nuclei":
		return "github.com/projectdiscovery/nuclei/v3/cmd/nuclei"
	case "subfinder":
		return "github.com/projectdiscovery/subfinder/v2/cmd/subfinder"
	case "httpx":
		return "github.com/projectdiscovery/httpx/cmd/httpx"
	case "gobuster":
		return "github.com/OJ/gobuster/v3"
	case "ffuf":
		return "github.com/ffuf/ffuf/v2"
	case "amass":
		return "github.com/owasp-amass/amass/v4/cmd/amass"
	case "feroxbuster":
		return "github.com/epi052/feroxbuster"
	default:
		return name
	}
}