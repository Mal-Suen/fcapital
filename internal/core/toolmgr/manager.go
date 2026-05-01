package toolmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ToolManager 工具管理器
type ToolManager struct {
	tools      map[string]*Tool
	localPath  string
	extraPaths []string // 额外搜索路径
	mu         sync.RWMutex
}

// Tool 工具信息
type Tool struct {
	Name        string     `json:"name" yaml:"name"`
	Binary      string     `json:"binary" yaml:"binary"`
	Category    string     `json:"category" yaml:"category"`
	Description string     `json:"description" yaml:"description"`
	Install     InstallCmd `json:"install" yaml:"install"`

	// 运行时信息（检测后填充）
	SystemPath string     `json:"system_path,omitempty"`
	LocalPath  string     `json:"local_path,omitempty"`
	Version    string     `json:"version,omitempty"`
	Status     ToolStatus `json:"status"`
	Source     ToolSource `json:"source"`
}

// InstallCmd 安装命令 - 支持多种安装方式
type InstallCmd struct {
	// Linux 包管理器
	Apt    string `json:"apt,omitempty" yaml:"apt,omitempty"`       // Debian/Ubuntu/Kali
	Yum    string `json:"yum,omitempty" yaml:"yum,omitempty"`       // RHEL/CentOS (旧)
	Dnf    string `json:"dnf,omitempty" yaml:"dnf,omitempty"`       // Fedora/RHEL 8+
	Pacman string `json:"pacman,omitempty" yaml:"pacman,omitempty"` // Arch Linux
	Zypper string `json:"zypper,omitempty" yaml:"zypper,omitempty"` // openSUSE
	Apk    string `json:"apk,omitempty" yaml:"apk,omitempty"`       // Alpine Linux

	// macOS
	Brew string `json:"brew,omitempty" yaml:"brew,omitempty"` // Homebrew
	Port string `json:"port,omitempty" yaml:"port,omitempty"` // MacPorts

	// Windows
	Choco  string `json:"choco,omitempty" yaml:"choco,omitempty"`   // Chocolatey
	Scoop  string `json:"scoop,omitempty" yaml:"scoop,omitempty"`   // Scoop
	Winget string `json:"winget,omitempty" yaml:"winget,omitempty"` // Windows Package Manager

	// 跨平台
	Go    string `json:"go,omitempty" yaml:"go,omitempty"`       // Go install
	Pip   string `json:"pip,omitempty" yaml:"pip,omitempty"`     // pip install
	Pip3  string `json:"pip3,omitempty" yaml:"pip3,omitempty"`   // pip3 install
	Gem   string `json:"gem,omitempty" yaml:"gem,omitempty"`     // gem install
	Npm   string `json:"npm,omitempty" yaml:"npm,omitempty"`     // npm install
	Cargo string `json:"cargo,omitempty" yaml:"cargo,omitempty"` // cargo install

	// 源码安装
	Git    string `json:"git,omitempty" yaml:"git,omitempty"`       // git clone
	Docker string `json:"docker,omitempty" yaml:"docker,omitempty"` // docker pull

	// 手动安装说明
	Manual        string `json:"manual,omitempty" yaml:"manual,omitempty"`
	ManualWindows string `json:"manual_windows,omitempty" yaml:"manual_windows,omitempty"`
	ManualLinux   string `json:"manual_linux,omitempty" yaml:"manual_linux,omitempty"`
	ManualMacOS   string `json:"manual_macos,omitempty" yaml:"manual_macos,omitempty"`
}

// ToolStatus 工具状态
type ToolStatus int

const (
	StatusUnknown ToolStatus = iota
	StatusReady
	StatusMissing
)

func (s ToolStatus) String() string {
	switch s {
	case StatusReady:
		return "Ready"
	case StatusMissing:
		return "Missing"
	default:
		return "Unknown"
	}
}

// ToolSource 工具来源
type ToolSource int

const (
	SourceNone ToolSource = iota
	SourceSystem
	SourceLocal
)

func (s ToolSource) String() string {
	switch s {
	case SourceSystem:
		return "system"
	case SourceLocal:
		return "local"
	default:
		return "none"
	}
}

// NewToolManager 创建工具管理器
func NewToolManager() *ToolManager {
	homeDir, _ := os.UserHomeDir()
	localPath := filepath.Join(homeDir, ".fcapital", "tools")

	// 初始化额外搜索路径
	extraPaths := []string{}

	if runtime.GOOS == "windows" {
		// Windows 常见路径 - Go 工具优先
		extraPaths = []string{
			filepath.Join(homeDir, "go", "bin"), // Go 工具 (优先)
		}

		// 尝试添加 Python Scripts 路径
		if pythonPath := findPythonScriptsPath(); pythonPath != "" {
			extraPaths = append(extraPaths, pythonPath)
		}

		// 其他路径
		extraPaths = append(extraPaths,
			filepath.Join(homeDir, ".local", "bin"), // 本地安装
			// Nmap 默认安装路径
			"C:\\Program Files (x86)\\Nmap",
			"C:\\Program Files\\Nmap",
		)
	} else {
		// Linux/macOS
		extraPaths = []string{
			filepath.Join(homeDir, "go", "bin"),
			filepath.Join(homeDir, ".local", "bin"),
			"/usr/local/bin",
			"/usr/bin",
		}
	}

	tm := &ToolManager{
		tools:      make(map[string]*Tool),
		localPath:  localPath,
		extraPaths: extraPaths,
	}
	// Load default tools
	tm.loadDefaultTools()
	// Auto-detect all tools
	tm.DetectAll()
	return tm
}

// findPythonScriptsPath 查找 Python Scripts 路径
func findPythonScriptsPath() string {
	// 方法1: 通过 where/which 命令查找 python
	var pythonPath string
	if runtime.GOOS == "windows" {
		if output, err := exec.Command("where", "python").Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) > 0 {
				pythonPath = strings.TrimSpace(lines[0])
			}
		}
	} else {
		if output, err := exec.Command("which", "python3").Output(); err == nil {
			pythonPath = strings.TrimSpace(string(output))
		} else if output, err := exec.Command("which", "python").Output(); err == nil {
			pythonPath = strings.TrimSpace(string(output))
		}
	}

	// 从 python 可执行文件路径推断 Scripts 目录
	if pythonPath != "" {
		dir := filepath.Dir(pythonPath)
		scriptsDir := filepath.Join(dir, "Scripts")
		if _, err := os.Stat(scriptsDir); err == nil {
			return scriptsDir
		}
		// 某些安装方式 Scripts 和 python 在同一目录
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// 方法2: 检查常见路径
	homeDir, _ := os.UserHomeDir()
	var commonPaths []string

	if runtime.GOOS == "windows" {
		commonPaths = []string{
			// 用户 AppData 路径
			filepath.Join(homeDir, "AppData", "Local", "Programs", "Python", "Python313", "Scripts"),
			filepath.Join(homeDir, "AppData", "Local", "Programs", "Python", "Python312", "Scripts"),
			filepath.Join(homeDir, "AppData", "Local", "Programs", "Python", "Python311", "Scripts"),
			filepath.Join(homeDir, "AppData", "Roaming", "Python", "Python313", "Scripts"),
			filepath.Join(homeDir, "AppData", "Roaming", "Python", "Python312", "Scripts"),
			// Anaconda 路径
			filepath.Join(homeDir, "anaconda3", "Scripts"),
			filepath.Join(homeDir, "miniconda3", "Scripts"),
			"C:\\ProgramData\\anaconda3\\Scripts",
			"C:\\ProgramData\\miniconda3\\Scripts",
			// 系统 Python 路径
			"C:\\Program Files\\Python313\\Scripts",
			"C:\\Program Files\\Python312\\Scripts",
			"C:\\Python313\\Scripts",
			"C:\\Python312\\Scripts",
		}
	} else {
		commonPaths = []string{
			filepath.Join(homeDir, ".local", "bin"),
			"/usr/local/bin",
			"/usr/bin",
			filepath.Join(homeDir, "anaconda3", "bin"),
			filepath.Join(homeDir, "miniconda3", "bin"),
			"/opt/homebrew/bin",
			"/usr/local/anaconda3/bin",
		}
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// LoadToolsFromYAML 从 YAML 文件加载工具定义
func (tm *ToolManager) LoadToolsFromYAML(path string) error {
	// 先加载默认工具
	tm.loadDefaultTools()

	// 如果有配置文件，尝试加载（目前简化处理，直接使用默认工具）
	return nil
}

// LoadToolsFromJSON 从 JSON 文件加载工具定义
func (tm *ToolManager) LoadToolsFromJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tools config: %w", err)
	}

	var cfg struct {
		Tools map[string]Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse tools config: %w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	for name, tool := range cfg.Tools {
		tool.Status = StatusUnknown
		tool.Source = SourceNone
		tm.tools[name] = &tool
	}

	return nil
}

// loadDefaultTools 加载默认工具列表（包含安装命令）
func (tm *ToolManager) loadDefaultTools() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	defaultTools := []Tool{
		{
			Name: "Nmap", Binary: "nmap", Category: "portscan",
			Description: "Network Security Scanner",
			Install: InstallCmd{
				Apt:           "sudo apt install nmap -y",
				Yum:           "sudo yum install nmap -y",
				Dnf:           "sudo dnf install nmap -y",
				Pacman:        "sudo pacman -S nmap --noconfirm",
				Zypper:        "sudo zypper install -y nmap",
				Apk:           "sudo apk add nmap",
				Brew:          "brew install nmap",
				Choco:         "choco install nmap -y",
				Scoop:         "scoop install nmap",
				Winget:        "Insecure.Nmap",
				ManualWindows: "Download from https://nmap.org/download.html",
			},
		},
		{
			Name: "Dirsearch", Binary: "dirsearch", Category: "webscan",
			Description: "Web Path Scanner",
			Install: InstallCmd{
				Apt: "sudo apt install dirsearch -y",
				Yum: "sudo yum install dirsearch -y",
				Pip: "pip install dirsearch",
				Git: "git clone https://github.com/maurosoria/dirsearch.git",
			},
		},
		{
			Name: "Dirb", Binary: "dirb", Category: "webscan",
			Description: "Web Content Scanner",
			Install: InstallCmd{
				Apt:    "sudo apt install dirb -y",
				Yum:    "sudo yum install dirb -y",
				Brew:   "brew install dirb",
				Manual: "Download from https://dirb.sourceforge.net/",
			},
		},
		{
			Name: "Gobuster", Binary: "gobuster", Category: "webscan",
			Description: "Directory/File/DNS Busting Tool",
			Install: InstallCmd{
				Apt:  "sudo apt install gobuster -y",
				Yum:  "sudo yum install gobuster -y",
				Brew: "brew install gobuster",
				Go:   "go install github.com/OJ/gobuster/v3@latest",
			},
		},
		{
			Name: "Ffuf", Binary: "ffuf", Category: "webscan",
			Description: "Fast Web Fuzzer",
			Install: InstallCmd{
				Apt: "sudo apt install ffuf -y",
				Yum: "sudo yum install ffuf -y",
				Go:  "go install github.com/ffuf/ffuf/v2@latest",
			},
		},
		{
			Name: "SQLMap", Binary: "sqlmap", Category: "vulnscan",
			Description: "Automatic SQL Injection Tool",
			Install: InstallCmd{
				Apt: "sudo apt install sqlmap -y",
				Yum: "sudo yum install sqlmap -y",
				Pip: "pip install sqlmap",
				Git: "git clone --depth 1 https://github.com/sqlmapproject/sqlmap.git",
			},
		},
		{
			Name: "WPScan", Binary: "wpscan", Category: "webscan",
			Description: "WordPress Security Scanner",
			Install: InstallCmd{
				Apt:    "sudo apt install wpscan -y",
				Yum:    "sudo yum install wpscan -y",
				Gem:    "gem install wpscan",
				Docker: "docker pull wpscanteam/wpscan",
			},
		},
		{
			Name: "Hydra", Binary: "hydra", Category: "password",
			Description: "Network Logon Cracker",
			Install: InstallCmd{
				Apt:    "sudo apt install hydra -y",
				Yum:    "sudo yum install hydra -y",
				Brew:   "brew install hydra",
				Manual: "Download from https://github.com/vanhauser-thc/thc-hydra",
			},
		},
		{
			Name: "Nuclei", Binary: "nuclei", Category: "vulnscan",
			Description: "Vulnerability Scanner",
			Install: InstallCmd{
				Apt: "sudo apt install nuclei -y",
				Go:  "go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest",
			},
		},
		{
			Name: "Subfinder", Binary: "subfinder", Category: "subdomain",
			Description: "Subdomain Discovery Tool",
			Install: InstallCmd{
				Apt: "sudo apt install subfinder -y",
				Go:  "go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest",
			},
		},
		{
			Name: "Httpx", Binary: "httpx", Category: "recon",
			Description: "HTTP Toolkit",
			Install: InstallCmd{
				Apt: "sudo apt install httpx -y",
				Go:  "go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest",
			},
		},
		{
			Name: "Dnsx", Binary: "dnsx", Category: "recon",
			Description: "DNS Toolkit",
			Install: InstallCmd{
				Apt: "sudo apt install dnsx -y",
				Go:  "go install -v github.com/projectdiscovery/dnsx/cmd/dnsx@latest",
			},
		},
	}

	for _, tool := range defaultTools {
		t := tool
		t.Status = StatusUnknown
		t.Source = SourceNone
		tm.tools[strings.ToLower(t.Binary)] = &t
	}
}

// DetectAll 检测所有工具
func (tm *ToolManager) DetectAll() {
	tm.mu.RLock()
	names := make([]string, 0, len(tm.tools))
	for name := range tm.tools {
		names = append(names, name)
	}
	tm.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			tool, err := tm.detect(n)
			if err == nil && tool != nil {
				mu.Lock()
				tm.tools[n] = tool
				mu.Unlock()
			}
		}(name)
	}
	wg.Wait()
}

// Detect 检测单个工具
func (tm *ToolManager) Detect(name string) (*Tool, error) {
	return tm.detect(name)
}

func (tm *ToolManager) detect(name string) (*Tool, error) {
	tm.mu.RLock()
	tool, ok := tm.tools[name]
	tm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	// 复制工具信息进行检测
	result := *tool

	// 获取要检测的二进制文件名
	binary := tool.Binary
	if runtime.GOOS == "windows" {
		// Windows 上尝试多种可能的名称
		candidates := []string{
			binary + ".exe",
			binary + ".bat",
			binary + ".cmd",
			binary,
		}

		// 1. 先检查额外路径 (优先级高于系统 PATH)
		for _, searchPath := range tm.extraPaths {
			for _, candidate := range candidates {
				fullPath := filepath.Join(searchPath, candidate)
				if _, err := os.Stat(fullPath); err == nil {
					result.SystemPath = fullPath
					result.Source = SourceSystem
					result.Status = StatusReady
					result.Version = tm.getVersion(fullPath)
					return &result, nil
				}
			}
		}

		// 2. 再检查系统 PATH
		for _, candidate := range candidates {
			if path, err := exec.LookPath(candidate); err == nil {
				result.SystemPath = path
				result.Source = SourceSystem
				result.Status = StatusReady
				result.Version = tm.getVersion(path)
				return &result, nil
			}
		}
	} else {
		// Linux/macOS
		// 1. 先检查系统 PATH
		if path, err := exec.LookPath(binary); err == nil {
			result.SystemPath = path
			result.Source = SourceSystem
			result.Status = StatusReady
			result.Version = tm.getVersion(path)
			return &result, nil
		}

		// 2. 检查额外路径
		for _, searchPath := range tm.extraPaths {
			fullPath := filepath.Join(searchPath, binary)
			if _, err := os.Stat(fullPath); err == nil {
				result.SystemPath = fullPath
				result.Source = SourceSystem
				result.Status = StatusReady
				result.Version = tm.getVersion(fullPath)
				return &result, nil
			}
		}
	}

	// 3. 检测本地安装目录
	localBin := filepath.Join(tm.localPath, binary)
	if runtime.GOOS == "windows" {
		localBin += ".exe"
	}
	if _, err := os.Stat(localBin); err == nil {
		result.LocalPath = localBin
		result.Source = SourceLocal
		result.Status = StatusReady
		result.Version = tm.getVersion(localBin)
		return &result, nil
	}

	// 4. 未找到
	result.Status = StatusMissing
	result.Source = SourceNone
	return &result, nil
}

// GetPath 获取工具路径
func (t *Tool) GetPath() string {
	if t.Source == SourceSystem {
		return t.SystemPath
	}
	if t.Source == SourceLocal {
		return t.LocalPath
	}
	return ""
}

// IsReady 检查工具是否可用
func (t *Tool) IsReady() bool {
	return t.Status == StatusReady
}

// getVersion 获取工具版本
func (tm *ToolManager) getVersion(path string) string {
	if path == "" {
		return ""
	}

	// 常见版本参数
	versionFlags := []string{"--version", "-V", "-version", "version"}

	for _, flag := range versionFlags {
		cmd := exec.Command(path, flag)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			// 提取第一行非空内容
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					// 截取前100个字符
					if len(line) > 100 {
						line = line[:100] + "..."
					}
					return line
				}
			}
		}
	}

	return "unknown"
}

// List 列出所有工具
func (tm *ToolManager) List() []*Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tools := make([]*Tool, 0, len(tm.tools))
	for _, tool := range tm.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListByCategory 按分类列出工具
func (tm *ToolManager) ListByCategory(category string) []*Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tools []*Tool
	for _, tool := range tm.tools {
		if tool.Category == category {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetReadyTools 获取可用工具列表
func (tm *ToolManager) GetReadyTools() []*Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tools []*Tool
	for _, tool := range tm.tools {
		if tool.Status == StatusReady {
			tools = append(tools, tool)
		}
	}
	return tools
}

// GetMissingTools 获取缺失工具列表
func (tm *ToolManager) GetMissingTools() []*Tool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tools []*Tool
	for _, tool := range tm.tools {
		if tool.Status == StatusMissing {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Get 获取指定工具
func (tm *ToolManager) Get(name string) (*Tool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 支持小写名称查找
	name = strings.ToLower(name)
	tool, ok := tm.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// GetInstallCommand 获取当前系统的安装命令
func (t *Tool) GetInstallCommand() string {
	switch runtime.GOOS {
	case "windows":
		if t.Install.Choco != "" {
			return t.Install.Choco
		}
		if t.Install.Scoop != "" {
			return t.Install.Scoop
		}
		return t.Install.ManualWindows
	case "darwin":
		if t.Install.Brew != "" {
			return t.Install.Brew
		}
		return t.Install.ManualMacOS
	default:
		if t.Install.Apt != "" {
			return t.Install.Apt
		}
		if t.Install.Dnf != "" {
			return t.Install.Dnf
		}
		if t.Install.Yum != "" {
			return t.Install.Yum
		}
		return t.Install.ManualLinux
	}
}
