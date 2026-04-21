# fcapital 技术设计文档 v2.0

> 文档版本: v2.0
> 编写日期: 2026年4月21日
> 项目位置: E:\PrometheusProjects\RedCoastII\fcapital

---

## 目录

1. [项目概述](#1-项目概述)
2. [技术架构](#2-技术架构)
3. [工具管理系统](#3-工具管理系统)
4. [模块集成方案](#4-模块集成方案)
5. [核心代码设计](#5-核心代码设计)
6. [配置系统设计](#6-配置系统设计)
7. [输出系统设计](#7-输出系统设计)
8. [第三方依赖](#8-第三方依赖)
9. [开发规范](#9-开发规范)

---

## 1. 项目概述

### 1.1 项目简介

fcapital 是一个综合渗透测试框架，采用"最大化集成"策略，复用现有成熟安全工具，提供统一的交互界面和命令行接口。

### 1.2 核心设计理念

| 理念 | 说明 |
|------|------|
| **最大化集成** | 复用现有工具，避免重复造轮子 |
| **智能检测** | 优先使用系统已安装工具（如 Kali Linux） |
| **统一入口** | 一个命令调用所有工具 |
| **结果聚合** | 统一格式输出，便于后续处理 |

### 1.3 技术选型

| 组件 | 选型 | 说明 |
|------|------|------|
| 编程语言 | Go 1.21+ | 跨平台、高性能 |
| CLI 框架 | cobra | 业界标准 |
| 配置管理 | viper | 多格式支持 |
| 交互 UI | survey | 终端交互 |
| 日志框架 | zap | 高性能日志 |

---

## 2. 技术架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                          fcapital                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      用户界面层                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │   │
│  │  │  交互菜单   │  │  命令行参数  │  │    配置管理     │   │   │
│  │  │  (survey)   │  │   (cobra)   │  │     (viper)     │   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    工具管理层 (核心)                        │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │   │
│  │  │ 工具检测  │ │ 工具安装  │ │ 工具调用  │ │ 结果解析  │     │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│              ┌───────────────┴───────────────┐                  │
│              ▼                               ▼                   │
│  ┌─────────────────────┐       ┌─────────────────────────┐      │
│  │   Go 库直接集成      │       │     外部工具调用         │      │
│  │  ┌───────────────┐  │       │  ┌─────────────────┐    │      │
│  │  │ httpx         │  │       │  │ nmap            │    │      │
│  │  │ subfinder     │  │       │  │ dirsearch       │    │      │
│  │  │ dnsx          │  │       │  │ dirb            │    │      │
│  │  │ nuclei        │  │       │  │ gobuster        │    │      │
│  │  └───────────────┘  │       │  │ ffuf            │    │      │
│  └─────────────────────┘       │  │ sqlmap          │    │      │
│                                │  │ wpscan          │    │      │
│                                │  │ hydra           │    │      │
│                                │  └─────────────────┘    │      │
│                                └─────────────────────────┘      │
│                                          │                       │
│                                          ▼                       │
│                         ┌─────────────────────────┐             │
│                         │   优先使用系统安装工具    │             │
│                         │   (Kali Linux 预装)      │             │
│                         └─────────────────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 目录结构

```
fcapital/
├── cmd/
│   └── fcapital/
│       └── main.go                 # 程序入口
│
├── internal/
│   ├── cli/                        # 命令行处理
│   │   ├── root.go                 # 根命令
│   │   ├── interactive.go          # 交互模式
│   │   ├── banner.go               # Banner 显示
│   │   └── deps.go                 # 依赖管理命令
│   │
│   ├── core/                       # 核心功能
│   │   ├── toolmgr/                # 工具管理器 (核心)
│   │   │   ├── manager.go          # 管理器主逻辑
│   │   │   ├── detect.go           # 工具检测
│   │   │   ├── install.go          # 工具安装
│   │   │   ├── runner.go           # 工具调用
│   │   │   └── parser.go           # 结果解析
│   │   ├── config.go               # 配置管理
│   │   ├── output.go               # 输出处理
│   │   └── logger.go               # 日志管理
│   │
│   └── modules/                    # 功能模块
│       ├── recon/                  # 信息收集
│       │   └── recon.go
│       ├── subdomain/              # 子域名枚举
│       │   └── subdomain.go
│       ├── portscan/               # 端口扫描
│       │   └── portscan.go
│       ├── webscan/                # Web 扫描
│       │   └── webscan.go
│       ├── vulnscan/               # 漏洞扫描
│       │   └── vulnscan.go
│       ├── password/               # 密码攻击
│       │   └── password.go
│       └── utils/                  # 辅助工具
│           └── utils.go
│
├── configs/
│   ├── config.yaml                 # 默认配置
│   ├── tools.yaml                  # 工具定义
│   └── wordlists/                  # 字典文件
│       ├── dirs.txt
│       ├── subdomains.txt
│       └── passwords.txt
│
├── scripts/
│   ├── install.sh                  # Linux/macOS 安装脚本
│   ├── install.bat                 # Windows 安装脚本
│   └── install-tools.sh            # 工具安装脚本
│
├── docs/
│   ├── README.md
│   ├── INSTALL.md
│   └── TOOLS.md                    # 工具文档
│
├── .gitignore
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## 3. 工具管理系统

### 3.1 工具检测优先级

```
优先级 1: 系统安装 (Kali Linux 预装)
    ↓ 未找到
优先级 2: fcapital 本地目录 (~/.fcapital/tools/)
    ↓ 未找到
优先级 3: 提示用户安装 / 自动安装
```

### 3.2 工具定义配置

```yaml
# configs/tools.yaml

tools:
  # 端口扫描
  nmap:
    name: "Nmap"
    binary: "nmap"
    category: "portscan"
    description: "Network Security Scanner"
    kali_preinstalled: true
    install:
      linux: "sudo apt install nmap"
      macos: "brew install nmap"
      windows: "choco install nmap"
  
  # 目录扫描
  dirsearch:
    name: "Dirsearch"
    binary: "dirsearch"
    category: "webscan"
    description: "Web Path Scanner"
    kali_preinstalled: true
    install:
      linux: "sudo apt install dirsearch"
      pip: "pip install dirsearch"
  
  dirb:
    name: "Dirb"
    binary: "dirb"
    category: "webscan"
    description: "Web Content Scanner"
    kali_preinstalled: true
    install:
      linux: "sudo apt install dirb"
      macos: "brew install dirb"
  
  gobuster:
    name: "Gobuster"
    binary: "gobuster"
    category: "webscan"
    description: "Directory/File/DNS Bustng Tool"
    kali_preinstalled: true
    install:
      linux: "sudo apt install gobuster"
      macos: "brew install gobuster"
      go: "go install github.com/OJ/gobuster/v3@latest"
  
  ffuf:
    name: "Ffuf"
    binary: "ffuf"
    category: "webscan"
    description: "Fast Web Fuzzer"
    kali_preinstalled: true
    install:
      linux: "sudo apt install ffuf"
      go: "go install github.com/ffuf/ffuf/v2@latest"
  
  # SQL 注入
  sqlmap:
    name: "SQLMap"
    binary: "sqlmap"
    category: "vulnscan"
    description: "Automatic SQL Injection Tool"
    kali_preinstalled: true
    install:
      linux: "sudo apt install sqlmap"
      pip: "pip install sqlmap"
  
  # WordPress 扫描
  wpscan:
    name: "WPScan"
    binary: "wpscan"
    category: "webscan"
    description: "WordPress Security Scanner"
    kali_preinstalled: true
    install:
      linux: "sudo apt install wpscan"
      gem: "gem install wpscan"
  
  # 密码破解
  hydra:
    name: "Hydra"
    binary: "hydra"
    category: "password"
    description: "Network Logon Cracker"
    kali_preinstalled: true
    install:
      linux: "sudo apt install hydra"
      macos: "brew install hydra"
  
  # Go 工具 (可能需要安装)
  nuclei:
    name: "Nuclei"
    binary: "nuclei"
    category: "vulnscan"
    description: "Vulnerability Scanner"
    kali_preinstalled: false
    install:
      linux: "sudo apt install nuclei"
      go: "go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest"
  
  subfinder:
    name: "Subfinder"
    binary: "subfinder"
    category: "subdomain"
    description: "Subdomain Discovery Tool"
    kali_preinstalled: false
    install:
      linux: "sudo apt install subfinder"
      go: "go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest"
  
  httpx:
    name: "Httpx"
    binary: "httpx"
    category: "recon"
    description: "HTTP Toolkit"
    kali_preinstalled: false
    install:
      linux: "sudo apt install httpx"
      go: "go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest"
  
  dnsx:
    name: "Dnsx"
    binary: "dnsx"
    category: "recon"
    description: "DNS Toolkit"
    kali_preinstalled: false
    install:
      linux: "sudo apt install dnsx"
      go: "go install -v github.com/projectdiscovery/dnsx/cmd/dnsx@latest"
```

### 3.3 工具管理器代码

```go
// internal/core/toolmgr/manager.go

package toolmgr

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "sync"
)

// ToolManager 工具管理器
type ToolManager struct {
    tools     map[string]*Tool
    localPath string
    mu        sync.RWMutex
}

// Tool 工具信息
type Tool struct {
    Name            string     `json:"name"`
    Binary          string     `json:"binary"`
    Category        string     `json:"category"`
    Description     string     `json:"description"`
    KaliPreinstalled bool      `json:"kali_preinstalled"`
    Install         InstallCmd `json:"install"`
    
    // 运行时信息
    SystemPath string     `json:"system_path,omitempty"`
    LocalPath  string     `json:"local_path,omitempty"`
    Version    string     `json:"version,omitempty"`
    Status     ToolStatus `json:"status"`
    Source     ToolSource `json:"source"`
}

// InstallCmd 安装命令
type InstallCmd struct {
    Linux   string `json:"linux,omitempty"`
    MacOS   string `json:"macos,omitempty"`
    Windows string `json:"windows,omitempty"`
    Go      string `json:"go,omitempty"`
    Pip     string `json:"pip,omitempty"`
    Gem     string `json:"gem,omitempty"`
}

// ToolStatus 工具状态
type ToolStatus int

const (
    StatusUnknown ToolStatus = iota
    StatusReady
    StatusMissing
    StatusError
)

// ToolSource 工具来源
type ToolSource int

const (
    SourceNone ToolSource = iota
    SourceSystem
    SourceLocal
)

// NewToolManager 创建工具管理器
func NewToolManager() *ToolManager {
    homeDir, _ := os.UserHomeDir()
    localPath := filepath.Join(homeDir, ".fcapital", "tools")
    
    return &ToolManager{
        tools:     make(map[string]*Tool),
        localPath: localPath,
    }
}

// LoadTools 从配置加载工具定义
func (tm *ToolManager) LoadTools(configPath string) error {
    data, err := os.ReadFile(configPath)
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

// DetectAll 检测所有工具
func (tm *ToolManager) DetectAll() {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    var wg sync.WaitGroup
    for name := range tm.tools {
        wg.Add(1)
        go func(n string) {
            defer wg.Done()
            tm.detect(n)
        }(name)
    }
    wg.Wait()
}

// Detect 检测单个工具
func (tm *ToolManager) Detect(name string) (*Tool, error) {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    return tm.detect(name)
}

func (tm *ToolManager) detect(name string) (*Tool, error) {
    tool, ok := tm.tools[name]
    if !ok {
        return nil, fmt.Errorf("unknown tool: %s", name)
    }
    
    // 1. 检测系统安装
    if path, err := exec.LookPath(tool.Binary); err == nil {
        tool.SystemPath = path
        tool.Source = SourceSystem
        tool.Status = StatusReady
        tool.Version = tm.getVersion(tool)
        return tool, nil
    }
    
    // 2. 检测本地安装
    localBin := filepath.Join(tm.localPath, tool.Binary)
    if runtime.GOOS == "windows" {
        localBin += ".exe"
    }
    if _, err := os.Stat(localBin); err == nil {
        tool.LocalPath = localBin
        tool.Source = SourceLocal
        tool.Status = StatusReady
        tool.Version = tm.getVersion(tool)
        return tool, nil
    }
    
    // 3. 未安装
    tool.Status = StatusMissing
    tool.Source = SourceNone
    return tool, nil
}

// GetPath 获取工具路径
func (t *Tool) GetPath() string {
    if t.Source == SourceSystem {
        return t.SystemPath
    }
    return t.LocalPath
}

// IsReady 检查工具是否可用
func (t *Tool) IsReady() bool {
    return t.Status == StatusReady
}

// getVersion 获取工具版本
func (tm *ToolManager) getVersion(tool *Tool) string {
    path := tool.GetPath()
    if path == "" {
        return ""
    }
    
    // 常见版本参数
    versionFlags := []string{"--version", "-version", "-V", "version"}
    
    for _, flag := range versionFlags {
        cmd := exec.Command(path, flag)
        output, err := cmd.CombinedOutput()
        if err == nil && len(output) > 0 {
            // 简单提取版本号
            return parseVersion(string(output))
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
```

### 3.4 工具调用器

```go
// internal/core/toolmgr/runner.go

package toolmgr

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

// Runner 工具运行器
type Runner struct {
    tool    *Tool
    timeout time.Duration
    env     []string
}

// RunResult 运行结果
type RunResult struct {
    Tool      string        `json:"tool"`
    Command   string        `json:"command"`
    Args      []string      `json:"args"`
    Output    string        `json:"output"`
    Error     string        `json:"error,omitempty"`
    Duration  time.Duration `json:"duration"`
    ExitCode  int           `json:"exit_code"`
    StartTime time.Time     `json:"start_time"`
}

// NewRunner 创建运行器
func NewRunner(tool *Tool) *Runner {
    return &Runner{
        tool:    tool,
        timeout: 10 * time.Minute,
    }
}

// SetTimeout 设置超时
func (r *Runner) SetTimeout(d time.Duration) {
    r.timeout = d
}

// SetEnv 设置环境变量
func (r *Runner) SetEnv(env []string) {
    r.env = env
}

// Run 执行工具
func (r *Runner) Run(ctx context.Context, args []string) (*RunResult, error) {
    if !r.tool.IsReady() {
        return nil, fmt.Errorf("tool %s is not ready", r.tool.Name)
    }
    
    // 创建带超时的上下文
    ctx, cancel := context.WithTimeout(ctx, r.timeout)
    defer cancel()
    
    // 构建命令
    cmd := exec.CommandContext(ctx, r.tool.GetPath(), args...)
    cmd.Env = append(cmd.Env, r.env...)
    
    result := &RunResult{
        Tool:      r.tool.Name,
        Command:   r.tool.GetPath(),
        Args:      args,
        StartTime: time.Now(),
    }
    
    // 执行命令
    output, err := cmd.CombinedOutput()
    result.Duration = time.Since(result.StartTime)
    result.Output = string(output)
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        }
        result.Error = err.Error()
        return result, err
    }
    
    result.ExitCode = 0
    return result, nil
}

// RunJSON 执行工具并解析 JSON 输出
func (r *Runner) RunJSON(ctx context.Context, args []string, v interface{}) error {
    result, err := r.Run(ctx, args)
    if err != nil {
        return err
    }
    
    return json.Unmarshal([]byte(result.Output), v)
}

// RunWithProgress 执行工具并显示进度
func (r *Runner) RunWithProgress(ctx context.Context, args []string, progressFn func(string)) (*RunResult, error) {
    if !r.tool.IsReady() {
        return nil, fmt.Errorf("tool %s is not ready", r.tool.Name)
    }
    
    ctx, cancel := context.WithTimeout(ctx, r.timeout)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, r.tool.GetPath(), args...)
    
    // 获取 stdout 和 stderr
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    cmd.Stderr = cmd.Stdout
    
    result := &RunResult{
        Tool:      r.tool.Name,
        Command:   r.tool.GetPath(),
        Args:      args,
        StartTime: time.Now(),
    }
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    // 实时读取输出
    var outputBuilder strings.Builder
    buf := make([]byte, 1024)
    for {
        n, err := stdout.Read(buf)
        if n > 0 {
            chunk := string(buf[:n])
            outputBuilder.WriteString(chunk)
            if progressFn != nil {
                progressFn(chunk)
            }
        }
        if err != nil {
            break
        }
    }
    
    err = cmd.Wait()
    result.Duration = time.Since(result.StartTime)
    result.Output = outputBuilder.String()
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        }
        result.Error = err.Error()
    }
    
    return result, nil
}
```

---

## 4. 模块集成方案

### 4.1 信息收集模块

```go
// internal/modules/recon/recon.go

package recon

import (
    "context"
    "fmt"
    
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

type ReconModule struct {
    toolMgr *toolmgr.ToolManager
}

func NewReconModule(tm *toolmgr.ToolManager) *ReconModule {
    return &ReconModule{toolMgr: tm}
}

// HTTPProbe HTTP 探测
func (m *ReconModule) HTTPProbe(ctx context.Context, targets []string) ([]HTTPResult, error) {
    // 使用 httpx
    tool, err := m.toolMgr.Detect("httpx")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("httpx is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    
    // 构建参数
    args := []string{
        "-silent",
        "-json",
        "-title",
        "-status-code",
        "-content-length",
        "-web-server",
        "-tech-detect",
    }
    
    // 从 stdin 读取目标
    // 或者使用 -l 参数指定文件
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    // 解析结果
    return m.parseHTTPXOutput(result.Output)
}

// DNSQuery DNS 查询
func (m *ReconModule) DNSQuery(ctx context.Context, domain string) (*DNSResult, error) {
    // 使用 dnsx
    tool, err := m.toolMgr.Detect("dnsx")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("dnsx is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    args := []string{
        "-d", domain,
        "-silent",
        "-json",
        "-a", "-aaaa", "-mx", "-txt", "-ns", "-cname",
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseDNSXOutput(result.Output)
}
```

### 4.2 子域名枚举模块

```go
// internal/modules/subdomain/subdomain.go

package subdomain

import (
    "context"
    "fmt"
    
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

type SubdomainModule struct {
    toolMgr *toolmgr.ToolManager
}

func NewSubdomainModule(tm *toolmgr.ToolManager) *SubdomainModule {
    return &SubdomainModule{toolMgr: tm}
}

// Enumerate 子域名枚举
func (m *SubdomainModule) Enumerate(ctx context.Context, domain string) ([]string, error) {
    tool, err := m.toolMgr.Detect("subfinder")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("subfinder is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    args := []string{
        "-d", domain,
        "-silent",
        "-json",
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseSubfinderOutput(result.Output)
}
```

### 4.3 端口扫描模块

```go
// internal/modules/portscan/portscan.go

package portscan

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

type PortscanModule struct {
    toolMgr *toolmgr.ToolManager
}

func NewPortscanModule(tm *toolmgr.ToolManager) *PortscanModule {
    return &PortscanModule{toolMgr: tm}
}

// Scan 端口扫描
func (m *PortscanModule) Scan(ctx context.Context, target string, ports string) (*ScanResult, error) {
    tool, err := m.toolMgr.Detect("nmap")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("nmap is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    runner.SetTimeout(30 * time.Minute)
    
    args := []string{
        "-p", ports,
        "-sV",  // 服务版本检测
        "-T4",  // 加速
        "--open",
        "-oX", "-",  // XML 输出到 stdout
        target,
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseNmapXML(result.Output)
}

// QuickScan 快速扫描 (Top 100 端口)
func (m *PortscanModule) QuickScan(ctx context.Context, target string) (*ScanResult, error) {
    return m.Scan(ctx, target, "-F")  // -F = Fast mode (Top 100)
}

// FullScan 全端口扫描
func (m *PortscanModule) FullScan(ctx context.Context, target string) (*ScanResult, error) {
    return m.Scan(ctx, target, "1-65535")
}
```

### 4.4 Web 扫描模块

```go
// internal/modules/webscan/webscan.go

package webscan

import (
    "context"
    "fmt"
    
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

type WebscanModule struct {
    toolMgr *toolmgr.ToolManager
}

func NewWebscanModule(tm *toolmgr.ToolManager) *WebscanModule {
    return &WebscanModule{toolMgr: tm}
}

// DirScan 目录扫描
func (m *WebscanModule) DirScan(ctx context.Context, target string, toolName string, wordlist string) (*DirScanResult, error) {
    // 支持多种工具
    var tool *toolmgr.Tool
    var err error
    
    switch toolName {
    case "dirsearch":
        tool, err = m.toolMgr.Detect("dirsearch")
    case "dirb":
        tool, err = m.toolMgr.Detect("dirb")
    case "gobuster":
        tool, err = m.toolMgr.Detect("gobuster")
    case "ffuf":
        tool, err = m.toolMgr.Detect("ffuf")
    default:
        tool, err = m.toolMgr.Detect("dirsearch")  // 默认
    }
    
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("%s is not installed", toolName)
    }
    
    runner := toolmgr.NewRunner(tool)
    
    var args []string
    switch toolName {
    case "dirsearch":
        args = []string{
            "-u", target,
            "-w", wordlist,
            "--quiet",
            "--format", "json",
        }
    case "dirb":
        args = []string{
            target,
            wordlist,
            "-N", "404",  // 忽略 404
        }
    case "gobuster":
        args = []string{
            "dir",
            "-u", target,
            "-w", wordlist,
            "-q",
        }
    case "ffuf":
        args = []string{
            "-u", target + "/FUZZ",
            "-w", wordlist,
            "-v",
        }
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseOutput(toolName, result.Output)
}
```

### 4.5 漏洞扫描模块

```go
// internal/modules/vulnscan/vulnscan.go

package vulnscan

import (
    "context"
    "fmt"
    "time"
    
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

type VulnscanModule struct {
    toolMgr *toolmgr.ToolManager
}

func NewVulnscanModule(tm *toolmgr.ToolManager) *VulnscanModule {
    return &VulnscanModule{toolMgr: tm}
}

// NucleiScan Nuclei 漏洞扫描
func (m *VulnscanModule) NucleiScan(ctx context.Context, target string, templates string) (*VulnResult, error) {
    tool, err := m.toolMgr.Detect("nuclei")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("nuclei is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    runner.SetTimeout(60 * time.Minute)
    
    args := []string{
        "-u", target,
        "-silent",
        "-json",
    }
    
    if templates != "" {
        args = append(args, "-t", templates)
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseNucleiOutput(result.Output)
}

// SQLMapScan SQL 注入扫描
func (m *VulnscanModule) SQLMapScan(ctx context.Context, target string) (*SQLMapResult, error) {
    tool, err := m.toolMgr.Detect("sqlmap")
    if err != nil {
        return nil, err
    }
    
    if !tool.IsReady() {
        return nil, fmt.Errorf("sqlmap is not installed")
    }
    
    runner := toolmgr.NewRunner(tool)
    runner.SetTimeout(30 * time.Minute)
    
    args := []string{
        "-u", target,
        "--batch",  // 非交互模式
        "--random-agent",
    }
    
    result, err := runner.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return m.parseSQLMapOutput(result.Output)
}
```

---

## 5. 核心代码设计

### 5.1 主程序入口

```go
// cmd/fcapital/main.go

package main

import (
    "fmt"
    "os"
    
    "github.com/yourname/fcapital/internal/cli"
)

var (
    version = "1.0.0"
    commit  = "none"
    date    = "unknown"
)

func main() {
    if err := cli.Execute(version, commit, date); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### 5.2 CLI 根命令

```go
// internal/cli/root.go

package cli

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
)

var rootCmd = &cobra.Command{
    Use:   "fcapital",
    Short: "A comprehensive penetration testing framework",
    Long: `fcapital is a penetration testing framework that integrates
multiple security tools with a unified interface.

It provides both interactive menu and command-line interface
for various security testing tasks.`,
}

func Execute(version, commit, date string) error {
    rootCmd.Version = version
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fcapital/config.yaml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
    
    // 添加子命令
    rootCmd.AddCommand(depsCmd)
    rootCmd.AddCommand(reconCmd)
    rootCmd.AddCommand(subdomainCmd)
    rootCmd.AddCommand(portscanCmd)
    rootCmd.AddCommand(webscanCmd)
    rootCmd.AddCommand(vulnscanCmd)
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        cobra.CheckErr(err)
        
        viper.AddConfigPath(home + "/.fcapital")
        viper.AddConfigPath(".")
        viper.AddConfigPath("./configs")
        viper.SetConfigType("yaml")
        viper.SetConfigName("config")
    }
    
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err == nil {
        if verbose {
            fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
        }
    }
}
```

### 5.3 依赖管理命令

```go
// internal/cli/deps.go

package cli

import (
    "fmt"
    "os"
    "text/tabwriter"
    
    "github.com/spf13/cobra"
    "github.com/yourname/fcapital/internal/core/toolmgr"
)

var depsCmd = &cobra.Command{
    Use:   "deps",
    Short: "Manage tool dependencies",
    Long:  `Check, install, and update external tool dependencies.`,
}

var depsCheckCmd = &cobra.Command{
    Use:   "check",
    Short: "Check tool dependencies status",
    Run:   runDepsCheck,
}

var depsInstallCmd = &cobra.Command{
    Use:   "install [tools...]",
    Short: "Install missing tools",
    Run:   runDepsInstall,
}

var depsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all tools",
    Run:   runDepsList,
}

func init() {
    depsCmd.AddCommand(depsCheckCmd)
    depsCmd.AddCommand(depsInstallCmd)
    depsCmd.AddCommand(depsListCmd)
}

func runDepsCheck(cmd *cobra.Command, args []string) {
    tm := toolmgr.NewToolManager()
    
    // 加载工具配置
    if err := tm.LoadTools("configs/tools.yaml"); err != nil {
        fmt.Fprintln(os.Stderr, "Error loading tools config:", err)
        os.Exit(1)
    }
    
    // 检测所有工具
    fmt.Println("Checking tool dependencies...")
    tm.DetectAll()
    
    // 显示结果
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
    fmt.Fprintln(w, "TOOL\tSTATUS\tPATH\tSOURCE")
    
    for _, tool := range tm.List() {
        status := "❌ Missing"
        if tool.IsReady() {
            status = "✅ Ready"
        }
        
        path := "-"
        if tool.IsReady() {
            path = tool.GetPath()
        }
        
        source := "-"
        switch tool.Source {
        case toolmgr.SourceSystem:
            source = "system"
        case toolmgr.SourceLocal:
            source = "local"
        }
        
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Name, status, path, source)
    }
    w.Flush()
    
    // 统计
    ready := len(tm.GetReadyTools())
    missing := len(tm.GetMissingTools())
    fmt.Printf("\n%d tools ready, %d missing\n", ready, missing)
}

func runDepsInstall(cmd *cobra.Command, args []string) {
    fmt.Println("Installing tools...")
    // TODO: 实现安装逻辑
}

func runDepsList(cmd *cobra.Command, args []string) {
    tm := toolmgr.NewToolManager()
    
    if err := tm.LoadTools("configs/tools.yaml"); err != nil {
        fmt.Fprintln(os.Stderr, "Error loading tools config:", err)
        os.Exit(1)
    }
    
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
    fmt.Fprintln(w, "TOOL\tCATEGORY\tDESCRIPTION\tKALI")
    
    for _, tool := range tm.List() {
        kali := "❌"
        if tool.KaliPreinstalled {
            kali = "✅"
        }
        fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tool.Name, tool.Category, tool.Description, kali)
    }
    w.Flush()
}
```

---

## 6. 配置系统设计

### 6.1 配置文件

```yaml
# configs/config.yaml

# 应用配置
app:
  name: fcapital
  version: 1.0.0
  debug: false

# 输出配置
output:
  format: text  # text, json, csv, html
  file: ""
  color: true
  verbose: false

# 工具配置
tools:
  config: "configs/tools.yaml"
  local_path: "~/.fcapital/tools"
  timeout: 10m

# 模块默认配置
modules:
  recon:
    timeout: 5m
  subdomain:
    timeout: 10m
  portscan:
    timeout: 30m
    default_ports: "-F"  # Fast scan
  webscan:
    timeout: 20m
    default_tool: "dirsearch"
    wordlist: "configs/wordlists/dirs.txt"
  vulnscan:
    timeout: 60m
    templates: ""

# 代理配置
proxy:
  http: ""
  https: ""
  socks5: ""
```

---

## 7. 输出系统设计

### 7.1 统一输出格式

```go
// internal/core/output/output.go

package output

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "time"
)

// Result 统一结果格式
type Result struct {
    ScanID    string      `json:"scan_id"`
    Module    string      `json:"module"`
    Tool      string      `json:"tool"`
    Target    string      `json:"target"`
    Status    string      `json:"status"`
    StartTime time.Time   `json:"start_time"`
    EndTime   time.Time   `json:"end_time"`
    Duration  string      `json:"duration"`
    Data      interface{} `json:"data,omitempty"`
    Error     string      `json:"error,omitempty"`
    RawOutput string      `json:"raw_output,omitempty"`
}

// Writer 输出写入器
type Writer struct {
    format Format
    output io.Writer
}

type Format int

const (
    FormatText Format = iota
    FormatJSON
    FormatCSV
)

func NewWriter(format Format, output io.Writer) *Writer {
    return &Writer{
        format: format,
        output: output,
    }
}

func (w *Writer) Write(result *Result) error {
    switch w.format {
    case FormatJSON:
        return w.writeJSON(result)
    case FormatCSV:
        return w.writeCSV(result)
    default:
        return w.writeText(result)
    }
}

func (w *Writer) writeJSON(result *Result) error {
    encoder := json.NewEncoder(w.output)
    encoder.SetIndent("", "  ")
    return encoder.Encode(result)
}

func (w *Writer) writeText(result *Result) error {
    fmt.Fprintf(w.output, "\n[%s] %s - %s\n", result.Module, result.Target, result.Status)
    fmt.Fprintf(w.output, "  Tool: %s\n", result.Tool)
    fmt.Fprintf(w.output, "  Duration: %s\n", result.Duration)
    if result.Error != "" {
        fmt.Fprintf(w.output, "  Error: %s\n", result.Error)
    }
    return nil
}

func (w *Writer) writeCSV(result *Result) error {
    // CSV 格式输出
    fmt.Fprintf(w.output, "%s,%s,%s,%s,%s\n",
        result.Module, result.Tool, result.Target, result.Status, result.Duration)
    return nil
}
```

---

## 8. 第三方依赖

### 8.1 go.mod

```go
module github.com/yourname/fcapital

go 1.21

require (
    // CLI
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    
    // 日志
    go.uber.org/zap v1.26.0
    
    // 交互 UI
    github.com/AlecAivazis/survey/v2 v2.3.7
    
    // 颜色输出
    github.com/fatih/color v1.16.0
    
    // 表格输出
    github.com/olekukonko/tablewriter v0.0.5
    
    // JSON 处理
    github.com/tidwall/gjson v1.17.0
    
    // YAML
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## 9. 开发规范

### 9.1 工具集成规范

```go
// 1. 每个工具调用前检查是否可用
tool, err := tm.Detect("nmap")
if err != nil || !tool.IsReady() {
    return fmt.Errorf("nmap is not available. Run 'fcapital deps install nmap'")
}

// 2. 使用 context 控制超时
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()

// 3. 统一结果格式
result := &Result{
    Module:    "portscan",
    Tool:      "nmap",
    Target:    target,
    StartTime: time.Now(),
}

// 4. 错误处理
if err != nil {
    result.Status = "failed"
    result.Error = err.Error()
} else {
    result.Status = "success"
}
```

### 9.2 命令行参数规范

```bash
# 统一参数命名
--target, -t     # 目标
--output, -o     # 输出文件
--format, -f     # 输出格式
--timeout        # 超时时间
--threads        # 并发数
--proxy          # 代理
--wordlist, -w   # 字典文件
--tool           # 指定工具
--verbose, -v    # 详细输出
--quiet, -q      # 静默模式
```

---

## 附录

### A. 支持的工具列表

| 工具 | 分类 | Kali 预装 | 说明 |
|------|------|----------|------|
| nmap | 端口扫描 | ✅ | 网络安全扫描器 |
| dirsearch | Web扫描 | ✅ | 目录扫描 |
| dirb | Web扫描 | ✅ | 目录爆破 |
| gobuster | Web扫描 | ✅ | 目录/DNS爆破 |
| ffuf | Web扫描 | ✅ | 模糊测试 |
| sqlmap | 漏洞扫描 | ✅ | SQL注入 |
| wpscan | Web扫描 | ✅ | WordPress扫描 |
| hydra | 密码攻击 | ✅ | 密码破解 |
| nuclei | 漏洞扫描 | ⚠️ | 模板化扫描 |
| subfinder | 子域名 | ⚠️ | 子域名枚举 |
| httpx | 信息收集 | ⚠️ | HTTP探测 |
| dnsx | 信息收集 | ⚠️ | DNS工具 |

### B. 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/yourname/fcapital.git
cd fcapital

# 2. 编译
go build -o fcapital ./cmd/fcapital

# 3. 检查依赖
./fcapital deps check

# 4. 安装缺失工具
./fcapital deps install

# 5. 运行
./fcapital
```

### C. 使用示例

```bash
# 信息收集
fcapital recon -t example.com

# 子域名枚举
fcapital subdomain -d example.com

# 端口扫描
fcapital portscan -t 192.168.1.1 -p 1-1000

# 目录扫描
fcapital webscan dir -u https://example.com -w wordlist.txt

# 漏洞扫描
fcapital vulnscan nuclei -u https://example.com

# 检查依赖
fcapital deps check
```
