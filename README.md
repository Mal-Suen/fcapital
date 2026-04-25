# fcapital

> **AI-Driven Penetration Testing Framework with Intelligent Workflow Orchestration.**
> **AI驱动的渗透测试框架：智能工作流编排与自动化决策。**

[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![ProjectDiscovery](https://img.shields.io/badge/ProjectDiscovery-Integrated-orange.svg)](https://github.com/projectdiscovery)
[![AI-Powered](https://img.shields.io/badge/AI-Powered-green.svg)]()
[![Cross-Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue.svg)]()

---

## 🇬🇧 English Documentation

### 📖 Introduction

**fcapital** is an **AI-driven penetration testing framework** that revolutionizes how security assessments are conducted. Unlike traditional tool wrappers, fcapital integrates **AI decision-making** at every phase, enabling:

- **🤖 AI-Guided Analysis**: Each phase result is analyzed by AI to determine the optimal next step
- **🔄 Phase-Based Workflow**: Structured execution with dependency resolution and AI decision points
- **🛠️ Intelligent Tool Scheduling**: Automatic tool detection, installation, and fallback mechanisms
- **📊 Context-Aware Testing**: System information and tool status are synchronized with AI for informed decisions
- **🌐 Cross-Platform Support**: Native support for Windows, Linux, and macOS with platform-specific optimizations

### 🚀 Key Features

| Feature | Detail |
| :--- | :--- |
| **🤖 AI Integration** | OpenAI, DeepSeek, Ollama support. AI analyzes phase results and suggests next actions. |
| **🔄 Smart Mode** | Standard tasks use mature tools, non-standard tasks use AI-generated scripts. |
| **🔗 Smart Data Flow** | Results flow between phases. AI determines what to focus on next. |
| **🛠️ Auto-Install** | 11+ package managers. Missing tools are auto-detected and installed. |
| **📊 Context Management** | System info, tool status, and history are maintained for AI context. |
| **⚡ Go Performance** | Native binaries, concurrent execution, streaming I/O. |
| **🌐 Cross-Platform** | Windows, Linux, macOS with native package manager support. |

### 🆕 CLI Commands

```bash
# === Information Gathering (Default) ===
fcapital recon -t example.com                    # Full recon (subdomains, HTTP, DNS, ports)
fcapital recon -t example.com --no-portscan      # Skip port scanning
fcapital recon -t example.com --subdomains-only  # Only enumerate subdomains

# === AI-Driven Penetration Testing ===
fcapital ai -t example.com                       # AI-driven scan
fcapital ai -t example.com --provider deepseek   # Use DeepSeek
fcapital ai -t example.com --provider ollama     # Use local Ollama
fcapital ai -t example.com --auto-confirm        # Auto-confirm script execution
fcapital ai --session <session-id>               # Resume interrupted session

# === AI Script Generation ===
fcapital script "waf bypass for example.com"     # Generate script
fcapital script "custom poc for CVE-2024-xxxx" --language python
fcapital script "encode bypass script" --execute # Generate and execute

# === Dependency Management ===
fcapital deps check              # Check all tools
fcapital deps list               # List supported tools
fcapital deps install <tool>     # Install specific tool
fcapital deps install --all      # Install all missing tools

# === Traditional Commands ===
fcapital portscan quick -t 192.168.1.1           # Quick port scan
fcapital webscan dir -u https://example.com      # Directory scan
fcapital vulnscan nuclei -t example.com          # Vulnerability scan
fcapital workflow run full -t example.com        # Full workflow
```

### 📊 AI Workflow Phases

| Phase | Description | AI Decision Point |
| :--- | :--- | :--- |
| **Recon** | Subdomain enum, HTTP probe, port scan | Which hosts to focus on |
| **Discovery** | Vulnerability scanning with nuclei | Priority ranking of findings |
| **Verification** | SQL injection, CMS exploits | Which vulns to exploit |
| **Report** | Generate comprehensive report | Report content optimization |

### 🛠️ Supported Tools (20+)

| Category | Tools |
| :--- | :--- |
| **Recon** | subfinder, httpx, dnsx, amass |
| **Port Scan** | nmap, masscan, zmap |
| **Web Scan** | dirsearch, gobuster, ffuf, feroxbuster, nikto, whatweb |
| **Vuln Scan** | nuclei, sqlmap, wpscan, joomscan |
| **Password** | hydra, medusa, ncrack |
| **SSL/TLS** | sslscan, testssl.sh, openssl |
| **Utility** | curl, wget, python, python3, ruby, go, git, docker |

### 🌐 Cross-Platform Support

| Platform | Package Managers | Tool Paths |
| :--- | :--- | :--- |
| **Windows** | winget, choco, scoop | `C:\Program Files`, `%USERPROFILE%\go\bin` |
| **Linux** | apt, yum, dnf, pacman, snap | `/usr/bin`, `/usr/local/bin`, `~/.local/bin` |
| **macOS** | brew, port | `/usr/local/bin`, `/opt/homebrew/bin` |

### 🚀 Getting Started

#### Installation

```bash
# Clone and build
git clone https://github.com/Mal-Suen/fcapital.git
cd fcapital
go build -o build/fcapital ./cmd/fcapital

# Or using Make
make build

# Windows (PowerShell)
go build -o build/fcapital.exe ./cmd/fcapital
```

#### Quick Start

```bash
# Set AI API key (choose one)
export DEEPSEEK_API_KEY="your-key"    # Recommended (cost-effective)
export OPENAI_API_KEY="your-key"      # GPT-4o
# Or use local Ollama (no API key needed)

# Run information gathering
./build/fcapital recon -t example.com

# Run AI-driven scan
./build/fcapital ai -t example.com

# Generate custom script
./build/fcapital script "waf bypass" -t example.com

# Check tool dependencies
./build/fcapital deps check

# Install missing tools
./build/fcapital deps install --all
```

### 📂 Project Structure

```text
fcapital/
├── cmd/fcapital/              # Entry point
├── internal/
│   ├── cli/                   # CLI commands
│   │   ├── root.go            # Root command
│   │   ├── ai_cmd.go          # AI-driven commands
│   │   ├── script_cmd.go      # Script generation
│   │   ├── recon_all.go       # Full recon command
│   │   ├── helpers.go         # Helper functions
│   │   ├── deps.go            # Dependency management
│   │   └── ...
│   ├── core/
│   │   ├── ai/                # AI engine
│   │   │   └── providers/     # AI providers (OpenAI, DeepSeek, Ollama)
│   │   ├── dispatcher/        # Task dispatcher
│   │   ├── scheduler/         # Tool scheduler
│   │   ├── script/            # Script generation
│   │   └── toolmgr/           # Tool manager
│   ├── modules/               # Feature modules
│   │   ├── recon/             # HTTPX, DNSX runners
│   │   ├── subdomain/         # Subfinder runner
│   │   ├── portscan/          # Nmap runner
│   │   └── ...
│   └── pkg/
│       ├── logger/            # Session logging
│       └── toolcheck/         # Tool detection
├── configs/
│   └── tools.yaml             # Tool definitions
├── build/                     # Compiled binaries
└── README.md
```

---

## 🇨🇳 中文文档

### 📖 项目简介

**fcapital** 不仅仅是一个工具包装器——它是为专业渗透测试人员设计的**AI驱动工作流自动化引擎**。当类似框架专注于工具安装时，fcapital 强调**智能工具链联动**、**AI决策分析**和**自动化执行**。采用 Go 语言构建，将侦察、扫描和漏洞评估编排成符合真实渗透测试方法论的工作流。

### 🚀 核心特性

| 特性 | 细节 |
| :--- | :--- |
| **🤖 AI 集成** | 支持 OpenAI、DeepSeek、Ollama。AI 分析阶段结果并建议下一步操作。 |
| **🔗 智能数据流** | 子域名枚举 → HTTP探测 → 目录扫描 → 漏洞扫描。零手动干预。 |
| **📊 报告生成** | HTML（深色主题）、JSON、Markdown 格式。执行摘要 + 技术细节。 |
| **🛠️ 自动安装** | 支持 11+ 种包管理器：apt、yum、dnf、pacman、brew、choco、scoop、winget、go、pip、cargo。 |
| **🎯 统一接口** | 单一 CLI 控制 20+ 工具。统一的参数、输出格式和错误处理。 |
| **⚡ Go 性能** | 原生二进制，并发执行，流式 I/O 处理大输出。 |
| **🌐 跨平台** | 原生支持 Windows、Linux、macOS，平台特定优化。 |

### 🆕 命令行示例

```bash
# === 信息收集（默认） ===
fcapital recon -t example.com                    # 完整侦察（子域名、HTTP、DNS、端口）
fcapital recon -t example.com --no-portscan      # 跳过端口扫描
fcapital recon -t example.com --subdomains-only  # 仅枚举子域名

# === AI 驱动渗透测试 ===
fcapital ai -t example.com                       # AI 驱动扫描
fcapital ai -t example.com --provider deepseek   # 使用 DeepSeek
fcapital ai -t example.com --provider ollama     # 使用本地 Ollama
fcapital ai -t example.com --auto-confirm        # 自动确认脚本执行
fcapital ai --session <session-id>               # 恢复中断的会话

# === AI 脚本生成 ===
fcapital script "waf bypass for example.com"     # 生成脚本
fcapital script "custom poc for CVE-2024-xxxx" --language python
fcapital script "encode bypass script" --execute # 生成并执行

# === 依赖管理 ===
fcapital deps check              # 检查所有工具
fcapital deps list               # 列出支持的工具有
fcapital deps install <tool>     # 安装指定工具
fcapital deps install --all      # 安装所有缺失工具

# === 传统命令 ===
fcapital portscan quick -t 192.168.1.1           # 快速端口扫描
fcapital webscan dir -u https://example.com      # 目录扫描
fcapital vulnscan nuclei -t example.com          # 漏洞扫描
fcapital workflow run full -t example.com        # 完整工作流
```

### 📊 内置工作流

| 工作流 | 步骤 | 用途 |
| :--- | :--- | :--- |
| **full** | 子域名 → HTTP → 端口 → 目录 → 漏洞 | 完整渗透测试 |
| **recon** | 子域名 → HTTP | 快速侦察 |
| **webapp** | HTTP → 目录 → 漏洞 | Web应用评估 |
| **vuln** | HTTP → Nuclei | 漏洞扫描 |

### 🛠️ 支持的工具 (20+)

| 类别 | 工具 |
| :--- | :--- |
| **侦察** | subfinder, httpx, dnsx, amass |
| **端口扫描** | nmap, masscan, zmap |
| **Web扫描** | dirsearch, gobuster, ffuf, feroxbuster, nikto, whatweb |
| **漏洞扫描** | nuclei, sqlmap, wpscan, joomscan |
| **密码攻击** | hydra, medusa, ncrack |
| **SSL/TLS** | sslscan, testssl.sh, openssl |
| **工具** | curl, wget, python, python3, ruby, go, git, docker |

### 🌐 跨平台支持

| 平台 | 包管理器 | 工具路径 |
| :--- | :--- | :--- |
| **Windows** | winget, choco, scoop | `C:\Program Files`, `%USERPROFILE%\go\bin` |
| **Linux** | apt, yum, dnf, pacman, snap | `/usr/bin`, `/usr/local/bin`, `~/.local/bin` |
| **macOS** | brew, port | `/usr/local/bin`, `/opt/homebrew/bin` |

### 🚀 快速开始

#### 安装

```bash
# 克隆并构建
git clone https://github.com/Mal-Suen/fcapital.git
cd fcapital
go build -o build/fcapital ./cmd/fcapital

# 或使用 Make
make build

# Windows (PowerShell)
go build -o build/fcapital.exe ./cmd/fcapital
```

#### 快速上手

```bash
# 设置 AI API 密钥（选择一个）
export DEEPSEEK_API_KEY="your-key"    # 推荐（性价比高）
export OPENAI_API_KEY="your-key"      # GPT-4o
# 或使用本地 Ollama（无需 API 密钥）

# 运行信息收集
./build/fcapital recon -t example.com

# 运行 AI 驱动扫描
./build/fcapital ai -t example.com

# 生成自定义脚本
./build/fcapital script "waf bypass" -t example.com

# 检查工具依赖
./build/fcapital deps check

# 安装缺失工具
./build/fcapital deps install --all
```

### 📂 目录结构

```text
fcapital/
├── cmd/fcapital/              # 入口点
├── internal/
│   ├── cli/                   # CLI命令
│   │   ├── root.go            # 根命令
│   │   ├── ai_cmd.go          # AI 驱动命令
│   │   ├── script_cmd.go      # 脚本生成
│   │   ├── recon_all.go       # 完整侦察命令
│   │   ├── helpers.go         # 辅助函数
│   │   └── ...
│   ├── core/
│   │   ├── ai/                # AI 引擎
│   │   │   └── providers/     # AI 提供者 (OpenAI, DeepSeek, Ollama)
│   │   ├── dispatcher/        # 任务调度器
│   │   ├── scheduler/         # 工具调度器
│   │   ├── script/            # 脚本生成
│   │   └── toolmgr/           # 工具管理器
│   ├── modules/               # 功能模块
│   │   ├── recon/             # HTTPX, DNSX 运行器
│   │   ├── subdomain/         # Subfinder 运行器
│   │   ├── portscan/          # Nmap 运行器
│   │   └── ...
│   └── pkg/
│       ├── logger/            # 会话日志
│       └── toolcheck/         # 工具检测
├── configs/
│   └── tools.yaml             # 工具定义
├── build/                     # 编译产物
└── README.md
```

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
