# fcapital

> **AI-Driven Penetration Testing Framework with Intelligent Workflow Orchestration.**
> **AI驱动的渗透测试框架：智能工作流编排与自动化决策。**

[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![ProjectDiscovery](https://img.shields.io/badge/ProjectDiscovery-Integrated-orange.svg)](https://github.com/projectdiscovery)
[![AI-Powered](https://img.shields.io/badge/AI-Powered-green.svg)]()

---

## 🇬🇧 English Documentation

### 📖 Introduction

**fcapital** is an **AI-driven penetration testing framework** that revolutionizes how security assessments are conducted. Unlike traditional tool wrappers, fcapital integrates **AI decision-making** at every phase, enabling:

- **🤖 AI-Guided Analysis**: Each phase result is analyzed by AI to determine the optimal next step
- **🔄 Phase-Based Workflow**: Structured execution with dependency resolution and AI decision points
- **🛠️ Intelligent Tool Scheduling**: Automatic tool detection, installation, and fallback mechanisms
- **📊 Context-Aware Testing**: System information and tool status are synchronized with AI for informed decisions

### 🚀 Key Features

| Feature | Detail |
| :--- | :--- |
| **🤖 AI Integration** | OpenAI, DeepSeek, Ollama support. AI analyzes phase results and suggests next actions. |
| **🔄 Phase-Based Workflow** | Recon → Discovery → Verification → Report. AI decides phase transitions. |
| **🔗 Smart Data Flow** | Results flow between phases. AI determines what to focus on next. |
| **🛠️ Auto-Install** | 11 package managers. Missing tools are auto-detected and installed. |
| **📊 Context Management** | System info, tool status, and history are maintained for AI context. |
| **⚡ Go Performance** | Native binaries, concurrent execution, streaming I/O. |

### 🆕 AI-Powered Commands

```bash
# AI-driven scan (recommended)
fcapital ai-scan -t example.com

# With specific AI provider
fcapital ai-scan -t example.com --provider deepseek
fcapital ai-scan -t example.com --provider openai
fcapital ai-scan -t example.com --provider ollama

# Auto-continue without prompts
fcapital ai-scan -t example.com --auto-continue

# Interactive AI chat
fcapital ai-chat

# View context information
fcapital context show
```

### 📊 AI Workflow Phases

| Phase | Description | AI Decision Point |
| :--- | :--- | :--- |
| **Recon** | Subdomain enum, HTTP probe, port scan | Which hosts to focus on |
| **Discovery** | Vulnerability scanning with nuclei | Priority ranking of findings |
| **Verification** | SQL injection, CMS exploits | Which vulns to exploit |
| **Report** | Generate comprehensive report | Report content optimization |

### 🛠️ Supported Tools (12+)

| Category | Tools |
| :--- | :--- |
| **Recon** | subfinder, httpx, dnsx |
| **Port Scan** | nmap |
| **Web Scan** | dirsearch, gobuster, ffuf |
| **Vuln Scan** | nuclei, sqlmap, wpscan |
| **Password** | hydra |

### 🚀 Getting Started

#### Installation

```bash
# Clone and build
git clone https://github.com/Mal-Suen/fcapital.git
cd fcapital
go build -o build/fcapital ./cmd/fcapital

# Or using Make
make build
```

#### Quick Start

```bash
# Set AI API key (choose one)
export DEEPSEEK_API_KEY="your-key"    # Recommended (cost-effective)
export OPENAI_API_KEY="your-key"      # GPT-4o
# Or use local Ollama (no API key needed)

# Run AI-driven scan
./build/fcapital ai-scan -t example.com

# Check tool dependencies
./build/fcapital deps check

# Install missing tools
./build/fcapital deps install --all
```

#### CLI Commands

```bash
# === AI-Powered Commands (NEW) ===
fcapital ai-scan -t example.com              # AI-driven penetration test
fcapital ai-scan -t example.com --auto-continue
fcapital ai-chat                             # Interactive AI assistant
fcapital context show                        # View current context

# === Dependency Management ===
fcapital deps check              # Check all tools
fcapital deps list               # List supported tools
fcapital deps install <tool>     # Install specific tool
fcapital deps install --all      # Install all missing tools

# === Traditional Workflows ===
fcapital workflow run full -t example.com
fcapital workflow run recon -t example.com
fcapital workflow run webapp -t example.com

# === Web Scanning ===
fcapital webscan dir -u https://example.com              # Default (gobuster)
fcapital webscan dir -u https://example.com -T dirsearch # Specific tool
fcapital webscan dir -u https://example.com -w wordlist.txt -e php,asp

# === Vulnerability Scanning ===
fcapital vulnscan nuclei -t https://example.com
fcapital vulnscan nuclei -t https://example.com --tags cve,rce
fcapital vulnscan sqlmap -t "https://example.com?id=1"

# === Workflow Automation ===
fcapital workflow run full -t example.com      # Full pentest
fcapital workflow run recon -t example.com     # Quick recon
fcapital workflow run webapp -t example.com    # Web app scan
fcapital workflow run vuln -t example.com      # Vuln scan
fcapital workflow list                         # List workflows
```

### 📂 Project Structure

```text
fcapital/
├── cmd/fcapital/              # Entry point
├── internal/
│   ├── cli/                   # CLI commands
│   │   ├── root.go            # Root command
│   │   ├── deps.go            # Dependency management
│   │   ├── workflow.go        # Workflow commands
│   │   ├── recon.go           # Recon commands
│   │   ├── subdomain.go       # Subdomain commands
│   │   ├── portscan.go        # Port scan commands
│   │   ├── webscan.go         # Web scan commands
│   │   └── vulnscan.go        # Vuln scan commands
│   ├── core/
│   │   ├── toolmgr/           # Tool manager
│   │   │   ├── manager.go     # Tool detection & installation
│   │   │   └── runner.go      # Tool execution
│   │   └── workflow/          # Workflow engine
│   │       ├── engine.go      # Core engine with topological sort
│   │       ├── handlers.go    # Step handlers for each module
│   │       └── report.go      # Report generation (HTML/JSON/MD)
│   ├── modules/               # Feature modules
│   │   ├── recon/             # HTTPX, DNSX runners
│   │   ├── subdomain/         # Subfinder runner
│   │   ├── portscan/          # Nmap runner
│   │   ├── webscan/           # Dirsearch, Gobuster, FFUF runners
│   │   ├── vulnscan/          # Nuclei, SQLMap runners
│   │   └── utils/             # Encoding, hashing utilities
│   └── pkg/errors/            # Unified error handling
├── configs/
│   └── tools.yaml             # Tool definitions
├── build/                     # Compiled binaries
└── README.md
```

### 🔬 Architecture Highlights

1. **Workflow Engine**: Uses topological sorting to resolve step dependencies. Each step declares `DependsOn`, `InputFrom`, and `InputField` for automatic data flow.

2. **Tool Runner**: Abstracts tool execution with timeout control, stdin/stdout handling, and progress callbacks. Supports both synchronous and streaming modes.

3. **Auto-Install**: Detects OS and available package managers, then attempts installation in priority order. Falls back to manual instructions when automatic install fails.

---

## 🇨🇳 中文文档

### 📖 项目简介

**fcapital** 不仅仅是一个工具包装器——它是为专业渗透测试人员设计的**工作流自动化引擎**。当类似框架专注于工具安装时，fcapital 强调**智能工具链联动**、**自动化数据流转**和**全面报告生成**。采用 Go 语言构建，将侦察、扫描和漏洞评估编排成符合真实渗透测试方法论的工作流。

### 🚀 核心特性

| 特性 | 细节 |
| :--- | :--- |
| **🔄 工作流引擎** | 拓扑排序执行，依赖自动解析。步骤通过 `InputFrom`/`InputField` 机制自动串联。 |
| **🔗 智能数据流** | 子域名枚举 → HTTP探测 → 目录扫描 → 漏洞扫描。零手动干预。 |
| **📊 报告生成** | HTML（深色主题）、JSON、Markdown 格式。执行摘要 + 技术细节。 |
| **🛠️ 自动安装** | 支持 11 种包管理器：`apt`、`yum`、`dnf`、`pacman`、`brew`、`choco`、`scoop`、`winget`、`go`、`pip`、`cargo`。 |
| **🎯 统一接口** | 单一 CLI 控制 12+ 工具。统一的参数、输出格式和错误处理。 |
| **⚡ Go 性能** | 原生二进制，并发执行，流式 I/O 处理大输出。 |

### 📊 内置工作流

| 工作流 | 步骤 | 用途 |
| :--- | :--- | :--- |
| **full** | 子域名 → HTTP → 端口 → 目录 → 漏洞 | 完整渗透测试 |
| **recon** | 子域名 → HTTP | 快速侦察 |
| **webapp** | HTTP → 目录 → 漏洞 | Web应用评估 |
| **vuln** | HTTP → Nuclei | 漏洞扫描 |

### 🛠️ 支持的工具 (12)

| 类别 | 工具 |
| :--- | :--- |
| **侦察** | httpx, dnsx |
| **子域名** | subfinder |
| **端口扫描** | nmap |
| **Web扫描** | dirsearch, gobuster, ffuf, dirb |
| **漏洞扫描** | nuclei, sqlmap, wpscan |
| **密码攻击** | hydra |

### 🚀 快速开始

#### 安装

```bash
# 克隆并构建
git clone https://github.com/Mal-Suen/fcapital.git
cd fcapital
go build -o build/fcapital ./cmd/fcapital

# 或使用 Make
make build
```

#### 快速上手

```bash
# 检查工具依赖
./build/fcapital deps check

# 安装缺失工具（自动检测包管理器）
./build/fcapital deps install nmap
./build/fcapital deps install --all

# 运行工作流
./build/fcapital workflow run full -t example.com

# 列出可用工作流
./build/fcapital workflow list
```

#### 命令行示例

```bash
# === 依赖管理 ===
fcapital deps check              # 检查所有工具
fcapital deps list               # 列出支持的工具有
fcapital deps install <tool>     # 安装指定工具
fcapital deps install --all      # 安装所有缺失工具

# === 信息收集 ===
fcapital recon http -t example.com           # HTTP探测
fcapital recon http -t sub1.com,sub2.com     # 多目标
fcapital recon dns -d example.com            # DNS查询

# === 子域名枚举 ===
fcapital subdomain passive -d example.com    # 被动枚举

# === 端口扫描 ===
fcapital portscan quick -t 192.168.1.1       # Top 100端口
fcapital portscan full -t 192.168.1.1        # 全端口
fcapital portscan custom -t 192.168.1.1 -p 80,443,8080-9000

# === Web扫描 ===
fcapital webscan dir -u https://example.com              # 默认(gobuster)
fcapital webscan dir -u https://example.com -T dirsearch # 指定工具
fcapital webscan dir -u https://example.com -w wordlist.txt -e php,asp

# === 漏洞扫描 ===
fcapital vulnscan nuclei -t https://example.com
fcapital vulnscan nuclei -t https://example.com --tags cve,rce
fcapital vulnscan sqlmap -t "https://example.com?id=1"

# === 工作流自动化 ===
fcapital workflow run full -t example.com      # 完整渗透
fcapital workflow run recon -t example.com     # 快速侦察
fcapital workflow run webapp -t example.com    # Web应用扫描
fcapital workflow run vuln -t example.com      # 漏洞扫描
fcapital workflow list                         # 列出工作流
```

### 📂 目录结构

```text
fcapital/
├── cmd/fcapital/              # 入口点
├── internal/
│   ├── cli/                   # CLI命令
│   │   ├── root.go            # 根命令
│   │   ├── deps.go            # 依赖管理
│   │   ├── workflow.go        # 工作流命令
│   │   ├── recon.go           # 侦察命令
│   │   ├── subdomain.go       # 子域名命令
│   │   ├── portscan.go        # 端口扫描命令
│   │   ├── webscan.go         # Web扫描命令
│   │   └── vulnscan.go        # 漏洞扫描命令
│   ├── core/
│   │   ├── toolmgr/           # 工具管理器
│   │   │   ├── manager.go     # 工具检测与安装
│   │   │   └── runner.go      # 工具执行
│   │   └── workflow/          # 工作流引擎
│   │       ├── engine.go      # 核心引擎（拓扑排序）
│   │       ├── handlers.go    # 各模块步骤处理器
│   │       └── report.go      # 报告生成（HTML/JSON/MD）
│   ├── modules/               # 功能模块
│   │   ├── recon/             # HTTPX, DNSX 运行器
│   │   ├── subdomain/         # Subfinder 运行器
│   │   ├── portscan/          # Nmap 运行器
│   │   ├── webscan/           # Dirsearch, Gobuster, FFUF 运行器
│   │   ├── vulnscan/          # Nuclei, SQLMap 运行器
│   │   └── utils/             # 编码、哈希工具
│   └── pkg/errors/            # 统一错误处理
├── configs/
│   └── tools.yaml             # 工具定义
├── build/                     # 编译产物
└── README.md
```

### 🔬 架构亮点

1. **工作流引擎**：使用拓扑排序解析步骤依赖。每个步骤声明 `DependsOn`、`InputFrom` 和 `InputField` 实现自动数据流转。

2. **工具运行器**：抽象工具执行，支持超时控制、stdin/stdout 处理和进度回调。支持同步和流式两种模式。

3. **自动安装**：检测操作系统和可用包管理器，按优先级尝试安装。自动安装失败时提供手动安装指南。

---

## ⚠️ Disclaimer / 免责声明

**fcapital is designed for authorized security testing and educational purposes only.**

Unauthorized use of this tool against systems you do not own or have explicit permission to test is **ILLEGAL**. By using fcapital, you agree to:

1. Only test systems you own or have written authorization to test
2. Comply with all applicable laws and regulations
3. Accept full responsibility for your actions

**fcapital 仅用于授权安全测试和教育目的。**

未经授权对您不拥有或未获得明确测试许可的系统使用本工具是**违法的**。使用 fcapital 即表示您同意：

1. 仅测试您拥有或获得书面授权测试的系统
2. 遵守所有适用法律法规
3. 对您的行为承担全部责任

---

## 🤝 Contribution & Contact / 贡献与联系

*   **Author:** Mal-Suen
*   **Blog:** [Mal-Suen's Blog](https://blog.mal-suen.cn)
*   **GitHub:** [https://github.com/Mal-Suen/fcapital](https://github.com/Mal-Suen/fcapital)

*Copyright © 2024-2026 Mal-Suen. Released under MIT License.*
