# fcapital 需求文档

> 文档版本: v1.0
> 编写日期: 2026年4月21日
> 项目位置: E:\PrometheusProjects\RedCoastII\fcapital

---

## 1. 项目概述

### 1.1 项目背景

参考 fsociety 工具的设计理念，开发一个现代化的综合渗透测试框架。fcapital 将集成多种安全测试工具，提供统一的交互界面，覆盖渗透测试全流程。

### 1.2 项目名称

**fcapital** (F-Society Capital)

命名灵感来源于《黑客军团》(Mr. Robot) 剧集中的 fsociety 组织，"capital" 寓意"首都/中心"，象征这是一个集中的安全测试平台。

### 1.3 项目定位

- **类型**: 综合渗透测试框架
- **目标用户**: 安全研究人员、渗透测试工程师、CTF 选手
- **核心价值**: 一站式安全测试工具集，降低工具使用门槛

---

## 2. 功能需求

### 2.1 功能模块划分

参考 fsociety 的分类方式，fcapital 划分为以下模块：

```
fcapital
├── 1. 信息收集 (Information Gathering)
├── 2. 子域名枚举 (Subdomain Enumeration)
├── 3. 端口扫描 (Port Scanning)
├── 4. Web 扫描 (Web Scanning)
├── 5. 漏洞扫描 (Vulnerability Scanning)
├── 6. 密码攻击 (Password Attacks)
├── 7. 漏洞利用 (Exploitation)
├── 8. 后渗透 (Post Exploitation)
└── 9. 辅助工具 (Utilities)
```

### 2.2 各模块详细功能

#### 2.2.1 信息收集模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| WHOIS 查询 | 域名 WHOIS 信息查询 | 调用 whois 命令/API |
| DNS 枚举 | DNS 记录查询 (A, AAAA, MX, TXT, NS, CNAME) | dnsx 工具 |
| IP 地理位置 | IP 地址地理位置查询 | 调用 IP 地理位置 API |
| 端口服务识别 | 常见端口服务识别 | httpx 探测 |
| CMS 识别 | 识别目标使用的 CMS 系统 | httpx 技术检测 |
| WAF 识别 | 识别目标使用的 WAF | 响应特征分析 |
| 技术栈识别 | 识别目标使用的技术栈 | httpx 技术检测 |

#### 2.2.2 子域名枚举模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| 被动枚举 | 从公开数据源获取子域名 | subfinder 工具 |
| 主动枚举 | 字典暴力枚举 | 可选工具 |
| DNS 区域传送 | 尝试 DNS 区域传送 | DNS AXFR 查询 |
| 子域名接管检测 | 检测可能的子域名接管 | CNAME 记录分析 |
| 存活探测 | 探测子域名是否存活 | httpx 探测 |

#### 2.2.3 端口扫描模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| TCP 连接扫描 | 全连接端口扫描 | nmap 工具 |
| SYN 扫描 | 半连接扫描 | nmap -sS |
| 服务识别 | 识别端口运行的服务 | nmap -sV |
| 常见端口扫描 | 扫描 Top 100/1000 端口 | nmap -F |
| 自定义端口扫描 | 扫描指定端口范围 | nmap -p |

#### 2.2.4 Web 扫描模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| 目录枚举 | 目录/文件暴力枚举 | dirsearch/dirb/gobuster/ffuf |
| 备份文件发现 | 发现备份文件 | 常见备份文件名枚举 |
| 敏感文件发现 | 发现敏感文件 (.git, .env, config 等) | 特定路径探测 |
| 爬虫 | 爬取网站链接 | 网页解析 + 递归爬取 |
| 参数发现 | 发现隐藏参数 | 参数模糊测试 |
| JavaScript 分析 | 分析 JS 文件中的敏感信息 | 正则匹配 |
| 响应头分析 | 分析 HTTP 安全头 | 安全头检查 |

#### 2.2.5 漏洞扫描模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| 模板化扫描 | 基于 YAML 模板的漏洞扫描 | nuclei 工具 |
| SQL 注入检测 | 检测 SQL 注入漏洞 | sqlmap 工具 |
| WordPress 漏洞 | WordPress 专用扫描 | wpscan 工具 |
| CVE 扫描 | 已知漏洞扫描 | nuclei 模板 |

#### 2.2.6 密码攻击模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| 多协议破解 | 支持多种协议的密码破解 | hydra 工具 |
| 字典攻击 | 使用字典进行密码破解 | hydra 工具 |

#### 2.2.7 辅助工具模块

| 功能 | 描述 | 实现方式 |
|------|------|---------|
| 编码/解码工具 | 各种编码转换 | Base64, Hex, URL 等 |
| Hash 计算 | 计算文件/字符串 Hash | MD5, SHA1, SHA256 等 |
| 字典处理 | 字典去重、合并、分割 | 文件操作 |

---

## 3. 非功能需求

### 3.1 性能需求

| 指标 | 要求 |
|------|------|
| 启动时间 | < 1 秒 |
| 内存占用 | < 100 MB (空闲状态) |
| 工具调用 | 异步执行，支持超时控制 |

### 3.2 可用性需求

| 指标 | 要求 |
|------|------|
| 界面类型 | 交互式菜单 + 命令行参数 |
| 输出格式 | 文本、JSON、CSV、HTML |
| 日志记录 | 支持日志文件输出 |
| 进度显示 | 实时进度显示 |
| 错误处理 | 友好的错误提示 |

### 3.3 兼容性需求

| 平台 | 支持级别 |
|------|---------|
| Linux (x64) | 完全支持 |
| Linux (ARM) | 完全支持 |
| Windows (x64) | 完全支持 |
| macOS | 完全支持 |
| Kali Linux | 优先支持（自动检测预装工具） |

### 3.4 安全需求

| 要求 | 描述 |
|------|------|
| 授权检查 | 启动时显示法律警告 |
| 敏感信息保护 | 不记录敏感信息到日志 |
| 代理支持 | 支持 HTTP/SOCKS5 代理 |

---

## 4. 技术方案

### 4.1 技术栈选择

| 组件 | 技术选型 | 版本 | 理由 |
|------|---------|------|------|
| 编程语言 | Go | 1.21+ | 跨平台编译、高性能、单二进制部署 |
| CLI 框架 | cobra | v1.8+ | Go 最流行的 CLI 框架 |
| 配置管理 | viper | v1.18+ | 支持多种配置格式 |
| 日志框架 | zap | v1.26+ | 高性能结构化日志 |
| HTTP 客户端 | net/http | 标准库 | 稳定可靠 |
| UI 组件 | survey | v2.3+ | 交互式 UI |

### 4.2 工具集成策略

#### 核心原则

1. **最大化集成现有工具**，避免重复造轮子
2. **智能检测系统工具**，优先使用 Kali Linux 预装版本
3. **统一入口和输出格式**

#### 工具检测优先级

```
优先级 1: 系统安装 (Kali Linux 预装)
    ↓ 未找到
优先级 2: fcapital 本地目录 (~/.fcapital/tools/)
    ↓ 未找到
优先级 3: 提示用户安装 / 自动安装
```

#### Go 库直接集成

| 工具 | 功能 | 说明 |
|------|------|------|
| httpx | HTTP 探测 | 存活探测、技术栈识别 |
| subfinder | 子域名枚举 | 被动子域名发现 |
| dnsx | DNS 操作 | DNS 查询 |
| nuclei | 漏洞扫描 | 模板化漏洞扫描 |

#### 外部工具调用

| 工具 | 功能 | Kali 预装 |
|------|------|----------|
| nmap | 端口扫描 | ✅ |
| dirsearch | 目录扫描 | ✅ |
| dirb | 目录扫描 | ✅ |
| gobuster | 目录扫描 | ✅ |
| ffuf | 模糊测试 | ✅ |
| sqlmap | SQL 注入 | ✅ |
| wpscan | WordPress 扫描 | ✅ |
| hydra | 密码破解 | ✅ |

### 4.3 项目结构

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
│       ├── subdomain/              # 子域名枚举
│       ├── portscan/               # 端口扫描
│       ├── webscan/                # Web 扫描
│       ├── vulnscan/               # 漏洞扫描
│       ├── password/               # 密码攻击
│       └── utils/                  # 辅助工具
│
├── configs/
│   ├── config.yaml                 # 默认配置
│   ├── tools.yaml                  # 工具定义
│   └── wordlists/                  # 字典文件
│
├── scripts/
│   ├── install.sh                  # Linux/macOS 安装脚本
│   └── install.bat                 # Windows 安装脚本
│
├── docs/
│   ├── README.md
│   └── TOOLS.md
│
├── Makefile
├── go.mod
└── README.md
```

---

## 5. 开发计划

### 5.1 开发阶段

| 阶段 | 内容 |
|------|------|
| Phase 1 | 项目框架搭建、CLI 框架、工具管理系统 |
| Phase 2 | 信息收集模块、子域名枚举模块 |
| Phase 3 | 端口扫描模块、Web 扫描模块 |
| Phase 4 | 漏洞扫描模块、密码攻击模块 |
| Phase 5 | 交互菜单、辅助工具模块 |
| Phase 6 | 测试、文档完善 |

### 5.2 优先级排序

**P0 (必须实现)**:
- CLI 框架和交互式菜单
- 工具管理系统（检测、调用）
- 信息收集模块
- 子域名枚举模块
- 端口扫描模块

**P1 (重要功能)**:
- Web 扫描模块
- 漏洞扫描模块
- 辅助工具模块

**P2 (扩展功能)**:
- 密码攻击模块
- 高级功能

---

## 6. 法律声明

```
⚠️  WARNING - LEGAL DISCLAIMER

fcapital is designed for authorized security testing and educational
purposes only. Unauthorized use of this tool against systems you do
not own or have explicit permission to test is ILLEGAL.

By using fcapital, you agree to:
1. Only test systems you own or have written authorization to test
2. Comply with all applicable laws and regulations
3. Accept full responsibility for your actions

The developers assume no liability for misuse of this software.
```

---

## 7. 参考资料

- [fsociety](https://github.com/Manisso/fsociety) - 项目灵感来源
- [Nuclei](https://github.com/projectdiscovery/nuclei) - 模板化扫描
- [httpx](https://github.com/projectdiscovery/httpx) - HTTP 探测
- [subfinder](https://github.com/projectdiscovery/subfinder) - 子域名枚举
- [Nmap](https://nmap.org/) - 端口扫描
