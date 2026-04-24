# fcapital AI-Driven Penetration Testing Framework
# 系统设计文档 (System Design Document)

> **版本**: 2.3.0
> **状态**: 草案
> **创建日期**: 2026-04-22
> **最后更新**: 2026-04-25

---

## 变更记录

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 2.3.0 | 2026-04-25 | 重构工具调度设计：AI自由推荐最佳工具，添加工具检测和安装模块 |
| 2.2.0 | 2026-04-24 | 修改工作流程：执行→AI分析→用户选择，添加交互式决策设计 |
| 2.1.0 | 2026-04-24 | 添加混合模式架构 |
| 2.0.0 | 2026-04-22 | 初始版本 |

---

## 1. 系统架构

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLI Layer (CLI层)                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │  root   │ │   ai    │ │workflow │ │  deps   │ │ context │ │  report │   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Core Layer (核心层)                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    PhaseOrchestrator (阶段编排器)                     │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ Phase-01 │ │ Phase-02 │ │ Phase-03 │ │ Phase-04 │ │ Phase-05 │   │   │
│  │  │ 信息收集  │ │ 漏洞发现  │ │ 漏洞验证  │ │ 后渗透    │ │ 报告生成  │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                       │
│  ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐       │
│  │   AIEngine        │  │  ContextManager   │  │  ToolScheduler    │       │
│  │   (AI引擎)        │  │  (上下文管理器)    │  │  (工具调度器)      │       │
│  └───────────────────┘  └───────────────────┘  └───────────────────┘       │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Module Layer (模块层)                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │  recon  │ │subdomain│ │portscan │ │ webscan │ │vulnscan │ │ exploit │   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Infrastructure Layer (基础设施层)                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │  ToolRunner │ │  AIProvider │ │   Storage   │ │   Logger    │           │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 组件职责

| 组件 | 职责 | 依赖 |
|------|------|------|
| **CLI Layer** | 用户交互入口，命令解析 | Core Layer |
| **PhaseOrchestrator** | 阶段编排，流程控制 | AIEngine, ContextManager, ToolScheduler |
| **AIEngine** | AI 对话管理，决策分析 | AIProvider |
| **ContextManager** | 上下文收集、存储、压缩 | Storage |
| **ToolScheduler** | 工具选择、安装、执行 | ToolRunner |
| **ScriptGenerator** | AI脚本生成、安全检查、执行 | AIEngine, SandboxRunner |
| **Module Layer** | 具体功能实现 | ToolRunner |
| **Infrastructure Layer** | 底层抽象 | - |

### 1.3 混合模式架构 (新增)

> **版本**: 2.1.0
> **添加日期**: 2026-04-24

#### 1.3.1 架构决策背景

对比"AI分析+工具调度"与"AI直接生成Payload"两种模式后，采用混合模式：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Hybrid Mode Architecture                              │
│                              (混合模式架构)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────┐                                                       │
│   │   AI 分析结果    │                                                       │
│   └─────────────────┘                                                       │
│           │                                                                 │
│           ▼                                                                 │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                      决策分发器 (Decision Dispatcher)                │  │
│   │  ┌───────────────────────────────────────────────────────────────┐  │  │
│   │  │  场景判断逻辑:                                                  │  │  │
│   │  │  - 有成熟工具支持? → 标准任务                                   │  │  │
│   │  │  - 无现成工具? → 非标准场景                                     │  │  │
│   │  │  - 需要预处理/后处理? → 混合场景                                │  │  │
│   │  └───────────────────────────────────────────────────────────────┘  │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│           │                         │                       │              │
│           │ 标准任务                 │ 非标准场景             │ 混合场景     │
│           ▼                         ▼                       ▼              │
│   ┌───────────────┐         ┌───────────────┐       ┌───────────────┐     │
│   │ ToolScheduler │         │ScriptGenerator│       │   混合执行器   │     │
│   │               │         │               │       │               │     │
│   │ ┌───────────┐ │         │ ┌───────────┐ │       │ Tool + Script │     │
│   │ │   nmap    │ │         │ │ AI生成代码│ │       │               │     │
│   │ │  nuclei   │ │         │ │ 安全检查  │ │       │ ┌─────┬─────┐ │     │
│   │ │  sqlmap   │ │         │ │ 沙箱执行  │ │       │ │Tool│Script│ │     │
│   │ │ gobuster  │ │         │ │ 用户确认  │ │       │ └─────┴─────┘ │     │
│   │ │   ...     │ │         │ │ 执行脚本  │ │       │               │     │
│   │ └───────────┘ │         │ └───────────┘ │       │ └───────────┘ │     │
│   └───────────────┘         └───────────────┘       └───────────────┘     │
│           │                         │                       │              │
│           └─────────────────────────┴───────────────────────┘              │
│                                     │                                      │
│                                     ▼                                      │
│                         ┌───────────────────────┐                          │
│                         │     结果融合器        │                          │
│                         │   (Result Fusion)     │                          │
│                         └───────────────────────┘                          │
│                                     │                                      │
│                                     ▼                                      │
│                         ┌───────────────────────┐                          │
│                         │   AI 综合分析         │                          │
│                         │   生成建议列表        │                          │
│                         └───────────────────────┘                          │
│                                     │                                      │
│                                     ▼                                      │
│                         ┌───────────────────────┐                          │
│                         │   用户选择界面        │                          │
│                         │   (Interactive UI)    │                          │
│                         │                      │                          │
│                         │  [1] 建议操作1        │                          │
│                         │  [2] 建议操作2        │                          │
│                         │  [3] 建议操作3        │                          │
│                         │  [4] 自定义操作       │                          │
│                         │  [0] 结束测试         │                          │
│                         │                      │                          │
│                         │  用户选择 → 执行      │                          │
│                         └───────────────────────┘                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 1.3.2 交互式决策流程 (新增 v2.2.0)

> **核心变更**: AI 不自动执行下一步，而是生成建议列表供用户选择

```go
// internal/core/dispatcher/decision.go

// DecisionPoint 决策点
type DecisionPoint struct {
    CurrentPhase    string        `json:"current_phase"`     // 当前阶段
    Results         PhaseResult   `json:"results"`           // 当前结果
    AIAnalysis      string        `json:"ai_analysis"`       // AI 分析内容
    Recommendations []Recommendation `json:"recommendations"` // 建议列表
}

// Recommendation AI 建议
type Recommendation struct {
    ID          int      `json:"id"`           // 建议ID (用于用户选择)
    Title       string   `json:"title"`        // 建议标题
    Description string   `json:"description"`  // 详细描述
    Tool        string   `json:"tool"`         // 推荐工具 (可选)
    Script      string   `json:"script"`       // 推荐脚本 (可选)
    Priority    int      `json:"priority"`     // 优先级 1-5
    RiskLevel   string   `json:"risk_level"`   // 风险等级: low/medium/high
}

// UserChoice 用户选择结果
type UserChoice struct {
    SelectedID    int    `json:"selected_id"`    // 用户选择的建议ID
    CustomAction  string `json:"custom_action"`  // 自定义操作 (如果选择 [4])
    Confirm       bool   `json:"confirm"`        // 是否确认执行
}
```

#### 1.3.3 决策交互界面设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        fcapital 交互式决策界面                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  📊 当前阶段: 信息收集 (Phase-01)                                            │
│  🎯 目标: blog.mal-suen.cn                                                  │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  📋 执行结果摘要:                                                            │
│  ├─ 端口扫描: 发现 6 个开放端口 (22, 25, 80, 110, 143, 443)                  │
│  ├─ HTTP探测: WordPress 6.9.4, Nginx 1.22.1, PHP 8.3.27                     │
│  └─ 子域名: api.mal-suen.cn, blog.mal-suen.cn                               │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  🤖 AI 分析:                                                                 │
│  目标 blog.mal-suen.cn 是一个 WordPress 博客站点，运行在 Nginx + PHP 环境。  │
│  发现了多个开放端口，其中 22(SSH) 和 443(HTTP) 是重点关注对象。              │
│  WordPress 版本较新，建议检查插件和主题漏洞。                                 │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  📝 AI 建议的下一步操作:                                                     │
│                                                                             │
│  [1] 🔍 服务指纹识别 - 对开放端口进行详细服务探测 (nmap -sV)                  │
│      风险: 低 │ 优先级: 高                                                   │
│                                                                             │
│  [2] 📂 目录扫描 - 对 Web 服务进行目录爆破 (gobuster/ffuf)                   │
│      风险: 低 │ 优先级: 中                                                   │
│                                                                             │
│  [3] 🔎 漏洞扫描 - 使用 nuclei 扫描已知 CVE 漏洞                             │
│      风险: 低 │ 优先级: 高                                                   │
│                                                                             │
│  [4] 🔧 WordPress 专项 - 使用 wpscan 检查 WordPress 漏洞                     │
│      风险: 低 │ 优先级: 高                                                   │
│                                                                             │
│  [5] ⚡ 自定义操作 - 输入自定义命令或脚本                                     │
│      风险: 用户评估 │ 优先级: -                                              │
│                                                                             │
│  [0] 📄 结束测试 - 生成渗透测试报告                                          │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  请选择下一步操作 [0-5]: _                                                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 1.3.4 场景判断规则

```go
// internal/core/dispatcher/rules.go

type ScenarioType int

const (
    ScenarioStandard    ScenarioType = iota  // 标准任务：有成熟工具
    ScenarioNonStandard                       // 非标准：需AI生成脚本
    ScenarioMixed                             // 混合：工具+脚本
)

// 场景判断规则
var scenarioRules = map[string]ScenarioType{
    // 标准任务（有成熟工具）
    "port_scan":           ScenarioStandard,    // nmap, masscan, rustscan
    "subdomain_enum":      ScenarioStandard,    // subfinder, amass
    "directory_bruteforce": ScenarioStandard,   // gobuster, ffuf, dirsearch
    "vulnerability_scan":  ScenarioStandard,    // nuclei, nikto
    "sql_injection":       ScenarioStandard,    // sqlmap
    "wordpress_scan":      ScenarioStandard,    // wpscan

    // 非标准场景（需AI生成）
    "custom_poc":          ScenarioNonStandard, // 特定CMS/框架漏洞
    "waf_bypass":          ScenarioNonStandard, // WAF绕过脚本
    "encoding_bypass":     ScenarioNonStandard, // 编码转换
    "custom_protocol":     ScenarioNonStandard, // 非标准协议
    "data_processing":     ScenarioNonStandard, // 数据清洗/转换

    // 混合场景
    "targeted_exploit":    ScenarioMixed,       // 工具发现 + 脚本利用
    "chain_attack":        ScenarioMixed,       // 多步骤攻击链
}
```

#### 1.3.3 ScriptGenerator 组件设计

```go
// internal/core/script/generator.go

package script

import "context"

// ScriptGenerator AI脚本生成器
type ScriptGenerator struct {
    ai       ai.Provider
    sandbox  *SandboxRunner
    auditor  *CodeAuditor
    languages []string  // 支持的语言: python, bash, powershell, go
}

// GenerateRequest 脚本生成请求
type GenerateRequest struct {
    TaskDescription string            `json:"task_description"`  // 任务描述
    Context         map[string]interface{} `json:"context"`       // 上下文信息
    Language        string            `json:"language"`          // 目标语言
    Constraints     []string          `json:"constraints"`       // 约束条件
}

// GeneratedScript 生成的脚本
type GeneratedScript struct {
    Code        string   `json:"code"`
    Language    string   `json:"language"`
    SafetyScore int      `json:"safety_score"`  // 0-100, 安全评分
    Warnings    []string `json:"warnings"`      // 安全警告
    Estimate    string   `json:"estimate"`      // 预估执行时间
}

// Generate 生成脚本
func (g *ScriptGenerator) Generate(ctx context.Context, req *GenerateRequest) (*GeneratedScript, error) {
    // 1. 构建Prompt
    prompt := g.buildPrompt(req)

    // 2. AI生成代码
    resp, err := g.ai.Chat(ctx, prompt)
    if err != nil {
        return nil, err
    }

    // 3. 解析代码
    script := g.parseCode(resp.Content)

    // 4. 安全审计
    auditResult := g.auditor.Audit(script.Code)
    script.SafetyScore = auditResult.Score
    script.Warnings = auditResult.Warnings

    // 5. 返回结果
    return script, nil
}

// ExecuteWithConfirm 执行脚本（需用户确认）
func (g *ScriptGenerator) ExecuteWithConfirm(ctx context.Context, script *GeneratedScript, autoConfirm bool) (*ExecutionResult, error) {
    // 1. 展示代码给用户
    if !autoConfirm {
        confirmed := g.showCodeAndWaitConfirm(script)
        if !confirmed {
            return nil, errors.New("user cancelled")
        }
    }

    // 2. 沙箱预执行
    sandboxResult := g.sandbox.Run(script.Code, script.Language)
    if sandboxResult.Error != nil {
        return nil, sandboxResult.Error
    }

    // 3. 正式执行
    result := g.executor.Run(script.Code, script.Language)

    return result, nil
}
```

#### 1.3.4 代码安全审计器

```go
// internal/core/script/auditor.go

type CodeAuditor struct {
    dangerousPatterns []Pattern
    allowedOperations []string
}

// Pattern 危险模式
type Pattern struct {
    Name        string
    Pattern     string  // 正则表达式
    Severity    string  // critical, high, medium, low
    Description string
}

// 内置危险模式
var defaultDangerousPatterns = []Pattern{
    {
        Name:     "file_deletion",
        Pattern:  `(rm -rf|os\.remove|shutil\.rmtree|del /s)`,
        Severity: "critical",
    },
    {
        Name:     "system_modification",
        Pattern:  `(reg add|net user|chmod 777|setuid)`,
        Severity: "high",
    },
    {
        Name:     "network_listen",
        Pattern:  `(socket\.listen|nc -l|netcat -l)`,
        Severity: "medium",
    },
    {
        Name:     "infinite_loop",
        Pattern:  `(while True:|for \(;;\)|while \(1\))`,
        Severity: "low",
    },
}

// Audit 审计代码
func (a *CodeAuditor) Audit(code string) *AuditResult {
    result := &AuditResult{
        Score:    100,
        Warnings: []string{},
    }

    for _, pattern := range a.dangerousPatterns {
        if matched, _ := regexp.MatchString(pattern.Pattern, code); matched {
            result.Warnings = append(result.Warnings,
                fmt.Sprintf("[%s] %s: %s", pattern.Severity, pattern.Name, pattern.Description))

            // 降低安全评分
            switch pattern.Severity {
            case "critical":
                result.Score -= 40
            case "high":
                result.Score -= 20
            case "medium":
                result.Score -= 10
            case "low":
                result.Score -= 5
            }
        }
    }

    return result
}
```

#### 1.3.5 沙箱执行器

```go
// internal/core/script/sandbox.go

type SandboxRunner struct {
    timeout    time.Duration
    workDir    string
    restricted bool
}

// Run 在沙箱中执行脚本
func (s *SandboxRunner) Run(code, language string) *SandboxResult {
    // 1. 创建隔离工作目录
    workDir := s.createIsolatedDir()

    // 2. 写入脚本文件
    scriptFile := s.writeScript(workDir, code, language)

    // 3. 设置资源限制
    // - CPU时间限制
    // - 内存限制
    // - 网络限制（可选）
    // - 文件系统限制

    // 4. 执行并监控
    ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
    defer cancel()

    cmd := s.buildCommand(ctx, scriptFile, language)
    output, err := s.runWithMonitoring(cmd)

    // 5. 清理工作目录
    s.cleanup(workDir)

    return &SandboxResult{
        Output: output,
        Error:  err,
        Duration: time.Since(startTime),
    }
}
```

---

## 2. 核心组件设计

### 2.1 AIEngine (AI引擎)

#### 2.1.1 接口定义

```go
// internal/core/ai/engine.go

package ai

import "context"

// Provider AI提供者接口
type Provider interface {
    // Chat 发送消息并获取响应
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    
    // StreamChat 流式对话
    StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
    
    // CountTokens 计算Token数量
    CountTokens(text string) int
    
    // Name 返回提供者名称
    Name() string
}

// ChatRequest 对话请求
type ChatRequest struct {
    Messages  []Message `json:"messages"`
    MaxTokens int       `json:"max_tokens,omitempty"`
    Temperature float   `json:"temperature,omitempty"`
}

// Message 消息
type Message struct {
    Role    string `json:"role"`    // system, user, assistant
    Content string `json:"content"`
}

// ChatResponse 对话响应
type ChatResponse struct {
    Content      string `json:"content"`
    TokensUsed   int    `json:"tokens_used"`
    FinishReason string `json:"finish_reason"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
    Content string `json:"content"`
    Done    bool   `json:"done"`
    Error   error  `json:"error,omitempty"`
}
```

#### 2.1.2 Provider 实现

```go
// internal/core/ai/providers/openai.go

type OpenAIProvider struct {
    apiKey     string
    model      string
    baseURL    string
    httpClient *http.Client
}

// internal/core/ai/providers/deepseek.go

type DeepSeekProvider struct {
    apiKey     string
    model      string
    httpClient *http.Client
}

// internal/core/ai/providers/ollama.go

type OllamaProvider struct {
    baseURL    string
    model      string
    httpClient *http.Client
}
```

#### 2.1.3 Prompt 模板管理

```go
// internal/core/ai/prompts/prompts.go

type PromptManager struct {
    templates map[string]*template.Template
}

// 内置模板
const (
    // 系统初始化Prompt
    PromptSystemInit = `你是一个专业的渗透测试助手。当前环境信息：
{{.SystemInfo}}
已安装工具：{{.InstalledTools}}
缺失工具：{{.MissingTools}}

请根据环境信息，准备协助用户进行渗透测试。`

    // 阶段分析Prompt
    PromptPhaseAnalysis = `当前阶段：{{.PhaseName}}
目标：{{.Target}}
发现结果：
{{.Findings}}

请分析以上结果，并给出下一步建议。输出格式：
{
  "analysis": "分析内容",
  "next_action": "建议的下一步操作",
  "priority": "high/medium/low",
  "tools_needed": ["工具列表"],
  "reasoning": "推理过程"
}`

    // 工具安装Prompt
    PromptToolInstall = `需要安装工具：{{.ToolName}}
当前系统：{{.OS}} {{.Arch}}
可用包管理器：{{.PackageManagers}}

请生成安装命令。输出格式：
{
  "method": "包管理器名称",
  "command": "安装命令",
  "post_install": "安装后需要执行的命令",
  "verify_command": "验证安装的命令"
}`
)
```

---

### 2.2 ContextManager (上下文管理器)

#### 2.2.1 数据结构

```go
// internal/core/context/manager.go

package context

import "time"

// Session 会话信息
type Session struct {
    ID        string    `json:"id"`
    StartTime time.Time `json:"start_time"`
    Target    string    `json:"target"`
    Status    string    `json:"status"` // running, paused, completed, failed
}

// SystemInfo 系统信息
type SystemInfo struct {
    OS           string `json:"os"`            // windows, linux, darwin
    OSVersion    string `json:"os_version"`
    Arch         string `json:"arch"`          // amd64, arm64
    Hostname     string `json:"hostname"`
    Username     string `json:"username"`
    NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
}

// NetworkInterface 网络接口
type NetworkInterface struct {
    Name        string   `json:"name"`
    IPAddresses []string `json:"ip_addresses"`
    MAC         string   `json:"mac"`
}

// ToolInfo 工具信息
type ToolInfo struct {
    Name      string `json:"name"`
    Version   string `json:"version"`
    Path      string `json:"path"`
    Status    string `json:"status"` // ready, missing, error
    Capabilities []string `json:"capabilities"`
}

// PhaseResult 阶段结果
type PhaseResult struct {
    PhaseID      string                 `json:"phase_id"`
    PhaseName    string                 `json:"phase_name"`
    StartTime    time.Time              `json:"start_time"`
    EndTime      time.Time              `json:"end_time"`
    Status       string                 `json:"status"`
    Findings     map[string]interface{} `json:"findings"`
    ToolsUsed    []string               `json:"tools_used"`
    AISummary    string                 `json:"ai_summary"`
    RawOutput    string                 `json:"raw_output,omitempty"`
}

// Context 完整上下文
type Context struct {
    Session      Session        `json:"session"`
    SystemInfo   SystemInfo     `json:"system_info"`
    Tools        []ToolInfo     `json:"tools"`
    PhaseHistory []PhaseResult  `json:"phase_history"`
    CurrentPhase string         `json:"current_phase"`
    Conversation []Message      `json:"conversation"` // AI对话历史
    Metadata     map[string]interface{} `json:"metadata"`
}
```

#### 2.2.2 接口定义

```go
// ContextManager 上下文管理器接口
type ContextManager interface {
    // Initialize 初始化上下文（收集系统信息）
    Initialize() error
    
    // GetContext 获取完整上下文
    GetContext() *Context
    
    // AddPhaseResult 添加阶段结果
    AddPhaseResult(result *PhaseResult) error
    
    // AddMessage 添加AI对话消息
    AddMessage(role, content string) error
    
    // GetRelevantContext 获取与当前任务相关的上下文（用于压缩）
    GetRelevantContext(task string) string
    
    // Save 持久化上下文
    Save(path string) error
    
    // Load 加载上下文
    Load(path string) error
    
    // Compress 压缩上下文（减少Token消耗）
    Compress() error
}
```

#### 2.2.3 上下文压缩策略

```go
// internal/core/context/compressor.go

type Compressor struct {
    maxTokens int
    provider  ai.Provider
}

// Compress 压缩上下文
// 策略：
// 1. 保留最近N条对话
// 2. 保留关键发现摘要
// 3. 压缩历史阶段详细数据为摘要
// 4. 使用AI生成压缩摘要
func (c *Compressor) Compress(ctx *Context) error {
    // 1. 保留系统信息（不压缩）
    // 2. 保留工具列表（不压缩）
    // 3. 压缩对话历史
    if len(ctx.Conversation) > 20 {
        // 保留最近10条，其余生成摘要
        oldMessages := ctx.Conversation[:len(ctx.Conversation)-10]
        summary := c.generateSummary(oldMessages)
        ctx.Conversation = append(
            []Message{{Role: "system", Content: "历史对话摘要: " + summary}},
            ctx.Conversation[len(ctx.Conversation)-10:]...,
        )
    }
    // 4. 压缩阶段历史
    for i := range ctx.PhaseHistory {
        if ctx.PhaseHistory[i].RawOutput != "" {
            ctx.PhaseHistory[i].RawOutput = "" // 清除原始输出
        }
    }
    return nil
}
```

---

### 2.3 ToolChecker (工具检测器) - 新增 v2.3.0

> **版本**: 2.3.0 新增
> **模块路径**: `internal/pkg/toolcheck`

#### 2.3.1 设计目标

独立于调度器的工具检测模块，负责：
1. 检测本地已安装的工具
2. 获取工具版本信息
3. 提供安装说明
4. 尝试自动安装

#### 2.3.2 接口定义

```go
// internal/pkg/toolcheck/toolcheck.go

package toolcheck

// ToolInfo 工具信息
type ToolInfo struct {
    Name      string `json:"name"`
    Installed bool   `json:"installed"`
    Version   string `json:"version,omitempty"`
    Path      string `json:"path,omitempty"`
    Category  string `json:"category"` // scanner, enumerator, exploiter, utility
}

// CheckResult 检测结果
type CheckResult struct {
    Available      []ToolInfo `json:"available"`
    Missing        []ToolInfo `json:"missing"`
    TotalCount     int        `json:"total_count"`
    InstalledCount int        `json:"installed_count"`
}

// Checker 工具检测器
type Checker struct {
    tools []ToolInfo
}

// 核心方法
func (c *Checker) CheckAll() *CheckResult
func (c *Checker) CheckTool(name string) ToolInfo
func (c *Checker) IsToolAvailable(name string) bool
func (c *Checker) FormatAvailableTools(result *CheckResult) string

// 安装相关函数
func GetInstallInstructions(name string) string
func TryAutoInstall(name string) (bool, string)
```

#### 2.3.3 工具注册表

```go
// ToolRegistry 工具注册表
var ToolRegistry = []ToolInfo{
    // 扫描器
    {Name: "nmap", Category: "scanner"},
    {Name: "nuclei", Category: "scanner"},
    {Name: "masscan", Category: "scanner"},
    
    // Web扫描器
    {Name: "nikto", Category: "scanner"},
    {Name: "wpscan", Category: "scanner"},
    {Name: "whatweb", Category: "scanner"},
    {Name: "httpx", Category: "scanner"},
    {Name: "ffuf", Category: "scanner"},
    {Name: "gobuster", Category: "scanner"},
    
    // 子域名枚举
    {Name: "subfinder", Category: "enumerator"},
    {Name: "amass", Category: "enumerator"},
    
    // 漏洞利用
    {Name: "sqlmap", Category: "exploiter"},
    {Name: "hydra", Category: "exploiter"},
    
    // 其他工具
    {Name: "curl", Category: "utility"},
    {Name: "python", Category: "utility"},
    {Name: "git", Category: "utility"},
    {Name: "docker", Category: "utility"},
}
```

#### 2.3.4 工具检测流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        工具检测流程                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    CheckAll() 检测所有工具                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  for each tool in ToolRegistry:                                      │   │
│  │      1. exec.LookPath(tool.Name)  // 查找工具路径                    │   │
│  │      2. 如果找到:                                                     │   │
│  │         - 获取版本 (tool --version)                                  │   │
│  │         - 添加到 Available 列表                                      │   │
│  │      3. 如果未找到:                                                   │   │
│  │         - 添加到 Missing 列表                                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  返回 CheckResult:                                                   │   │
│  │  - Available: [{nmap, 7.80, /usr/bin/nmap}, ...]                    │   │
│  │  - Missing: [{wpscan, false}, ...]                                  │   │
│  │  - TotalCount: 20                                                    │   │
│  │  - InstalledCount: 15                                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 2.3.5 自动安装策略

```go
// TryAutoInstall 尝试自动安装工具
func TryAutoInstall(name string) (bool, string) {
    os := runtime.GOOS
    
    switch name {
    // Go 工具 - 跨平台自动安装
    case "nuclei", "subfinder", "httpx", "gobuster", "ffuf", "amass":
        cmd := exec.Command("go", "install", getGoPackage(name), "@latest")
        // ...
        
    // Python 工具 - pip 安装
    case "sqlmap", "dirsearch":
        cmd := exec.Command("pip", "install", name)
        // ...
        
    // Linux/macOS 包管理器
    case "hydra", "nikto", "sslscan":
        if os == "linux" {
            cmd := exec.Command("sudo", "apt", "install", "-y", name)
        } else if os == "darwin" {
            cmd := exec.Command("brew", "install", name)
        }
        
    // Ruby 工具 - gem 安装
    case "wpscan":
        if os == "windows" {
            return false, "Windows 上建议使用 Docker"
        }
        cmd := exec.Command("gem", "install", name)
        
    default:
        return false, "无法自动安装，请参考安装说明"
    }
}
```

#### 2.3.6 与 AI 推荐的集成

```go
// 在 hybrid.go 中的使用

func runSession(...) {
    // 1. 初始化工具检测器
    toolChecker := toolcheck.NewChecker()
    toolCheckResult := toolChecker.CheckAll()
    
    // 2. 显示工具状态
    fmt.Printf("工具检测: 已安装 %d/%d 个工具\n", 
        toolCheckResult.InstalledCount, toolCheckResult.TotalCount)
    
    // 3. AI 推荐时传入已安装工具信息（仅供参考）
    recommendations := getAIRecommendations(ctx, provider, session, toolCheckResult)
    
    // 4. 显示推荐时标注工具状态
    for _, rec := range recommendations {
        toolStatus := ""
        if rec.Tool != "" && !toolChecker.IsToolAvailable(rec.Tool) {
            toolStatus = " ⚠️[未安装]"
        }
        fmt.Printf("[%d] %s (工具: %s%s)\n", rec.ID, rec.Title, rec.Tool, toolStatus)
    }
    
    // 5. 用户选择未安装工具时，进入安装流程
    if selected.Tool != "" && !toolChecker.IsToolAvailable(selected.Tool) {
        installed := handleToolInstallation(selected.Tool, reader, provider, ctx)
        if !installed {
            // 安装失败，AI 推荐替代方案
            continue
        }
    }
}
```

---

### 2.4 ToolScheduler (工具调度器)

#### 2.3.1 接口定义

```go
// internal/core/scheduler/scheduler.go

package scheduler

import "context"

// ToolDefinition 工具定义
type ToolDefinition struct {
    Name         string            `json:"name"`
    Description  string            `json:"description"`
    Capabilities []string          `json:"capabilities"`
    InstallMethods []InstallMethod `json:"install_methods"`
    Fallbacks    []string          `json:"fallbacks"`
    Category     string            `json:"category"`
}

// InstallMethod 安装方法
type InstallMethod struct {
    Type       string `json:"type"`        // winget, apt, brew, go, pip, etc.
    Package    string `json:"package"`     // 包名或URL
    PostInstall string `json:"post_install"` // 安装后命令
    VerifyCmd  string `json:"verify_cmd"`  // 验证命令
}

// ScheduleResult 调度结果
type ScheduleResult struct {
    ToolName string   `json:"tool_name"`
    Action   string   `json:"action"` // execute, install, fallback
    Command  string   `json:"command"`
    Args     []string `json:"args"`
}

// Scheduler 工具调度器接口
type Scheduler interface {
    // FindToolByCapability 根据能力查找工具
    FindToolByCapability(capability string) (*ToolDefinition, error)
    
    // CheckAvailability 检查工具可用性
    CheckAvailability(toolName string) (bool, *ToolInfo)
    
    // Schedule 调度工具执行
    Schedule(ctx context.Context, req *ScheduleRequest) (*ScheduleResult, error)
    
    // InstallTool 安装工具
    InstallTool(ctx context.Context, toolName string) error
    
    // Execute 执行工具
    Execute(ctx context.Context, toolName string, args []string) (*ExecutionResult, error)
}
```

#### 2.3.2 工具能力映射

```yaml
# configs/tools.yaml

capabilities:
  port_scan:
    primary: nmap
    fallbacks: [masscan, rustscan]
    
  subdomain_enum_passive:
    primary: subfinder
    fallbacks: [amass]
    
  subdomain_enum_active:
    primary: gobuster
    fallbacks: [dnsrecon]
    
  http_probe:
    primary: httpx
    fallbacks: [httprobe]
    
  directory_bruteforce:
    primary: dirsearch
    fallbacks: [gobuster, ffuf, dirb]
    
  vulnerability_scan:
    primary: nuclei
    fallbacks: [nikto]
    
  sql_injection:
    primary: sqlmap
    fallbacks: []
    
  wordpress_scan:
    primary: wpscan
    fallbacks: []
    
  password_attack:
    primary: hydra
    fallbacks: [medusa]
```

#### 2.3.3 自动安装流程

```go
// internal/core/scheduler/installer.go

func (s *scheduler) InstallTool(ctx context.Context, toolName string) error {
    def, err := s.getToolDefinition(toolName)
    if err != nil {
        return err
    }
    
    // 获取可用包管理器
    managers := s.detectPackageManagers()
    
    // 按优先级尝试安装
    for _, method := range def.InstallMethods {
        if !s.isManagerAvailable(method.Type, managers) {
            continue
        }
        
        err := s.tryInstall(ctx, method)
        if err == nil {
            // 验证安装
            if s.verifyInstallation(toolName) {
                return nil
            }
        }
    }
    
    // 所有方法都失败，请求AI生成安装方案
    return s.requestAIInstallGuide(ctx, toolName)
}
```

---

### 2.4 PhaseOrchestrator (阶段编排器)

#### 2.4.1 接口定义

```go
// internal/core/orchestrator/orchestrator.go

package orchestrator

import "context"

// Phase 阶段接口
type Phase interface {
    // ID 阶段ID
    ID() string
    
    // Name 阶段名称
    Name() string
    
    // Execute 执行阶段
    Execute(ctx context.Context, input *PhaseInput) (*PhaseOutput, error)
    
    // CanSkip 是否可以跳过
    CanSkip() bool
    
    // Dependencies 依赖的阶段
    Dependencies() []string
}

// PhaseInput 阶段输入
type PhaseInput struct {
    Target      string                 `json:"target"`
    Context     *context.Context       `json:"context"`
    PrevResults map[string]*PhaseOutput `json:"prev_results"`
    Options     map[string]interface{} `json:"options"`
}

// PhaseOutput 阶段输出
type PhaseOutput struct {
    PhaseID   string                 `json:"phase_id"`
    Status    string                 `json:"status"`
    Findings  map[string]interface{} `json:"findings"`
    NextPhase string                 `json:"next_phase,omitempty"`
    SkipPhases []string              `json:"skip_phases,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

// Orchestrator 编排器接口
type Orchestrator interface {
    // RegisterPhase 注册阶段
    RegisterPhase(phase Phase) error
    
    // Run 运行工作流
    Run(ctx context.Context, target string, options *RunOptions) (*RunResult, error)
    
    // Pause 暂停
    Pause() error
    
    // Resume 恢复
    Resume() error
    
    // SkipPhase 跳过阶段
    SkipPhase(phaseID string) error
    
    // GetCurrentPhase 获取当前阶段
    GetCurrentPhase() Phase
}
```

#### 2.4.2 阶段定义

```go
// internal/core/orchestrator/phases/recon.go

type ReconPhase struct {
    scheduler scheduler.Scheduler
    ai        ai.Provider
}

func (p *ReconPhase) Execute(ctx context.Context, input *PhaseInput) (*PhaseOutput, error) {
    output := &PhaseOutput{
        PhaseID: p.ID(),
        Findings: make(map[string]interface{}),
    }
    
    // 1. 子域名枚举
    subdomains, err := p.enumSubdomains(ctx, input.Target)
    if err != nil {
        return nil, err
    }
    output.Findings["subdomains"] = subdomains
    
    // 2. HTTP探测
    aliveHosts, err := p.probeHTTP(ctx, subdomains)
    if err != nil {
        return nil, err
    }
    output.Findings["alive_hosts"] = aliveHosts
    
    // 3. 端口扫描
    openPorts, err := p.scanPorts(ctx, aliveHosts)
    if err != nil {
        return nil, err
    }
    output.Findings["open_ports"] = openPorts
    
    output.Status = "completed"
    return output, nil
}
```

#### 2.4.3 AI决策集成

```go
// internal/core/orchestrator/orchestrator.go

func (o *orchestrator) runPhaseWithAI(ctx context.Context, phase Phase, input *PhaseInput) (*PhaseOutput, error) {
    // 1. 执行阶段
    output, err := phase.Execute(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // 2. 格式化结果
    formattedResult := o.formatResult(output)
    
    // 3. 发送给AI分析
    aiResponse, err := o.ai.Chat(ctx, &ai.ChatRequest{
        Messages: []ai.Message{
            {Role: "system", Content: o.buildAnalysisPrompt(phase, formattedResult)},
        },
    })
    if err != nil {
        // AI失败不影响流程，记录警告
        o.logger.Warn("AI analysis failed", "error", err)
    } else {
        output.AISummary = aiResponse.Content
    }
    
    // 4. 解析AI建议
    decision := o.parseAIDecision(aiResponse.Content)
    
    // 5. 应用决策
    if decision.SkipPhases != nil {
        for _, phaseID := range decision.SkipPhases {
            o.SkipPhase(phaseID)
        }
    }
    if decision.NextPhase != "" {
        output.NextPhase = decision.NextPhase
    }
    
    return output, nil
}
```

---

## 3. 数据流设计

### 3.1 完整数据流

```
用户输入目标
      │
      ▼
┌─────────────────┐
│ ContextManager  │ ←── 收集系统信息
│ Initialize()    │
└─────────────────┘
      │
      ▼
┌─────────────────┐
│   AIEngine      │ ←── 发送初始上下文
│   Chat()        │     AI确认准备就绪
└─────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────┐
│              PhaseOrchestrator.Run()                │
│  ┌───────────────────────────────────────────────┐  │
│  │              Phase-01: 信息收集                │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐         │  │
│  │  │subfinder│→│ httpx   │→│  nmap   │         │  │
│  │  └─────────┘ └─────────┘ └─────────┘         │  │
│  │                   │                           │  │
│  │                   ▼                           │  │
│  │           ┌─────────────┐                     │  │
│  │           │ AI 分析结果 │                     │  │
│  │           │ 决定下一步  │                     │  │
│  │           └─────────────┘                     │  │
│  └───────────────────────────────────────────────┘  │
│                        │                            │
│                        ▼                            │
│  ┌───────────────────────────────────────────────┐  │
│  │              Phase-02: 漏洞发现                │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐         │  │
│  │  │ nuclei  │→│ sqlmap  │→│ wpscan  │         │  │
│  │  └─────────┘ └─────────┘ └─────────┘         │  │
│  │                   │                           │  │
│  │                   ▼                           │  │
│  │           ┌─────────────┐                     │  │
│  │           │ AI 分析结果 │                     │  │
│  │           │ 优先级排序  │                     │  │
│  │           └─────────────┘                     │  │
│  └───────────────────────────────────────────────┘  │
│                        │                            │
│                        ▼                            │
│                    ... 更多阶段 ...                  │
│                                                     │
└─────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────┐
│  ReportGenerator │
│  Generate()      │
└─────────────────┘
      │
      ▼
   输出报告
```

### 3.2 AI交互数据流

```
┌──────────────────────────────────────────────────────────────┐
│                      AI Interaction Flow                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │   Context   │                                            │
│  │  Manager    │                                            │
│  └──────┬──────┘                                            │
│         │                                                    │
│         │ getContext()                                       │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Prompt Builder                    │    │
│  │  ┌─────────────────────────────────────────────┐    │    │
│  │  │ System: 你是渗透测试助手...                    │    │    │
│  │  │ Context: {系统信息, 工具状态, 历史发现}        │    │    │
│  │  │ Current Phase: 信息收集                       │    │    │
│  │  │ Findings: {子域名: [...], 端口: [...]}       │    │    │
│  │  │ Task: 分析结果并建议下一步                     │    │    │
│  │  └─────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────┘    │
│         │                                                    │
│         ▼                                                    │
│  ┌─────────────┐                                            │
│  │ AI Provider │ ←── OpenAI / DeepSeek / Ollama            │
│  │  Chat()     │                                            │
│  └──────┬──────┘                                            │
│         │                                                    │
│         │ Response                                          │
│         ▼                                                    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  Response Parser                     │    │
│  │  {                                                   │    │
│  │    "analysis": "发现多个潜在入口...",                │    │
│  │    "next_action": "对api.example.com进行深度扫描",   │    │
│  │    "tools_needed": ["nuclei", "ffuf"],              │    │
│  │    "priority": "high"                               │    │
│  │  }                                                   │    │
│  └─────────────────────────────────────────────────────┘    │
│         │                                                    │
│         ▼                                                    │
│  ┌─────────────┐                                            │
│  │  Decision   │                                            │
│  │  Executor   │ → 执行AI建议的操作                         │
│  └─────────────┘                                            │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. 目录结构

```
fcapital/
├── cmd/
│   └── fcapital/
│       └── main.go                    # 入口
├── internal/
│   ├── cli/                           # CLI命令层
│   │   ├── root.go                    # 根命令
│   │   ├── ai.go                      # ai命令 (新增)
│   │   ├── workflow.go                # workflow命令
│   │   ├── deps.go                    # deps命令
│   │   ├── context.go                 # context命令 (新增)
│   │   └── config.go                  # config命令
│   │
│   ├── core/                          # 核心层
│   │   ├── ai/                        # AI引擎 (新增)
│   │   │   ├── engine.go              # AI引擎接口
│   │   │   ├── prompts.go             # Prompt模板
│   │   │   ├── parser.go              # 响应解析器
│   │   │   └── providers/
│   │   │       ├── openai.go
│   │   │       ├── deepseek.go
│   │   │       ├── anthropic.go
│   │   │       └── ollama.go
│   │   │
│   │   ├── context/                   # 上下文管理 (新增)
│   │   │   ├── manager.go             # 上下文管理器
│   │   │   ├── collector.go           # 信息收集器
│   │   │   ├── compressor.go          # 上下文压缩
│   │   │   └── storage.go             # 持久化存储
│   │   │
│   │   ├── scheduler/                 # 工具调度 (新增)
│   │   │   ├── scheduler.go           # 调度器接口
│   │   │   ├── installer.go           # 自动安装
│   │   │   ├── executor.go            # 工具执行
│   │   │   └── capability.go          # 能力映射
│   │   │
│   │   ├── orchestrator/              # 阶段编排 (新增)
│   │   │   ├── orchestrator.go        # 编排器
│   │   │   ├── phase.go               # 阶段接口
│   │   │   └── phases/
│   │   │       ├── recon.go           # 信息收集阶段
│   │   │       ├── discovery.go       # 漏洞发现阶段
│   │   │       ├── verification.go    # 漏洞验证阶段
│   │   │       ├── postexploit.go     # 后渗透阶段
│   │   │       └── report.go          # 报告生成阶段
│   │   │
│   │   └── toolmgr/                   # 工具管理 (保留)
│   │       ├── manager.go
│   │       └── runner.go
│   │
│   ├── modules/                       # 功能模块
│   │   ├── recon/
│   │   ├── subdomain/
│   │   ├── portscan/
│   │   ├── webscan/
│   │   ├── vulnscan/
│   │   └── utils/
│   │
│   └── pkg/                           # 公共包
│       ├── errors/
│       ├── formatter/                 # 结果格式化 (新增)
│       └── reporter/                  # 报告生成 (新增)
│
├── configs/
│   ├── config.yaml                    # 主配置
│   ├── tools.yaml                     # 工具定义
│   └── prompts/                       # Prompt模板 (新增)
│       ├── system_init.md
│       ├── phase_analysis.md
│       └── tool_install.md
│
├── docs/
│   ├── TODO.md                        # 任务清单
│   ├── REQUIREMENTS.md                # 需求文档
│   └── DESIGN.md                      # 设计文档
│
├── tests/
│   ├── unit/
│   ├── integration/
│   └── mocks/
│       └── ai_mock.go                 # AI Mock
│
├── go.mod
├── Makefile
└── README.md
```

---

## 5. 配置设计

### 5.1 主配置文件

```yaml
# ~/.fcapital/config.yaml

# AI配置
ai:
  provider: "deepseek"           # openai, deepseek, anthropic, ollama
  model: "deepseek-chat"
  api_key: ""                    # 环境变量: FCAPITAL_AI_API_KEY
  base_url: ""                   # 自定义API地址
  max_tokens: 4096
  temperature: 0.7
  
  # Token消耗预警
  token_warning:
    enabled: true
    threshold: 100000            # 累计Token超过此值时警告
    
  # 离线模式
  offline_mode: false
  fallback_provider: "ollama"    # 主Provider不可用时回退

# 工作流配置
workflow:
  auto_continue: false           # AI决策后是否自动继续
  confirm_critical: true         # 关键操作是否需要确认
  timeout: "30m"                 # 单阶段超时
  
# 工具配置
tools:
  install_automatically: true    # 缺失工具是否自动安装
  preferred:                     # 首选工具
    directory_scan: "dirsearch"
    port_scan: "nmap"
    vuln_scan: "nuclei"
    
# 输出配置
output:
  format: "text"                 # text, json
  color: true
  verbose: false
  log_file: "~/.fcapital/logs/fcapital.log"
  
# 报告配置
report:
  default_format: "html"
  output_dir: "~/.fcapital/reports"
  include_raw_output: false
```

### 5.2 工具定义文件

```yaml
# configs/tools.yaml

version: "1.0"

tools:
  nmap:
    name: "Nmap"
    description: "Network Security Scanner"
    category: "port_scan"
    capabilities:
      - port_scan
      - service_detection
      - os_fingerprinting
      - vuln_scan
    install:
      winget: "Insecure.Nmap"
      choco: "nmap"
      apt: "nmap"
      brew: "nmap"
    verify: "nmap --version"
    fallbacks: ["masscan", "rustscan"]
    
  subfinder:
    name: "Subfinder"
    description: "Subdomain Discovery Tool"
    category: "recon"
    capabilities:
      - subdomain_enum_passive
    install:
      go: "github.com/projectdiscovery/subfinder/v2/cmd/subfinder"
    verify: "subfinder -version"
    fallbacks: ["amass"]
    
  httpx:
    name: "HTTPX"
    description: "HTTP Toolkit"
    category: "recon"
    capabilities:
      - http_probe
      - technology_detection
    install:
      go: "github.com/projectdiscovery/httpx/cmd/httpx"
    verify: "httpx -version"
    fallbacks: ["httprobe"]
    
  nuclei:
    name: "Nuclei"
    description: "Vulnerability Scanner"
    category: "vuln_scan"
    capabilities:
      - vulnerability_scan
      - cve_scan
    install:
      go: "github.com/projectdiscovery/nuclei/v3/cmd/nuclei"
    verify: "nuclei -version"
    fallbacks: ["nikto"]
    
  dirsearch:
    name: "Dirsearch"
    description: "Web Path Scanner"
    category: "web_scan"
    capabilities:
      - directory_bruteforce
    install:
      pip: "dirsearch"
      pip3: "dirsearch"
    verify: "dirsearch --version"
    fallbacks: ["gobuster", "ffuf", "dirb"]
    
  gobuster:
    name: "Gobuster"
    description: "Directory/File/DNS Busting Tool"
    category: "web_scan"
    capabilities:
      - directory_bruteforce
      - dns_bruteforce
    install:
      go: "github.com/OJ/gobuster/v3"
    verify: "gobuster version"
    fallbacks: ["dirsearch", "ffuf"]
    
  ffuf:
    name: "FFUF"
    description: "Fast Web Fuzzer"
    category: "web_scan"
    capabilities:
      - directory_bruteforce
      - fuzzing
    install:
      go: "github.com/ffuf/ffuf/v2"
    verify: "ffuf -V"
    fallbacks: ["gobuster", "dirsearch"]
    
  sqlmap:
    name: "SQLMap"
    description: "Automatic SQL Injection Tool"
    category: "vuln_scan"
    capabilities:
      - sql_injection
    install:
      pip: "sqlmap"
      pip3: "sqlmap"
    verify: "sqlmap --version"
    fallbacks: []
    
  wpscan:
    name: "WPScan"
    description: "WordPress Security Scanner"
    category: "web_scan"
    capabilities:
      - wordpress_scan
      - cms_detection
    install:
      gem: "wpscan"
    verify: "wpscan --version"
    fallbacks: []
    
  hydra:
    name: "Hydra"
    description: "Network Logon Cracker"
    category: "password"
    capabilities:
      - password_attack
      - brute_force
    install:
      apt: "hydra"
      brew: "hydra"
      choco: "hydra"
    verify: "hydra -h"
    fallbacks: ["medusa"]
    
  dnsx:
    name: "DNSX"
    description: "DNS Toolkit"
    category: "recon"
    capabilities:
      - dns_query
      - dns_bruteforce
    install:
      go: "github.com/projectdiscovery/dnsx/cmd/dnsx"
    verify: "dnsx -version"
    fallbacks: []
```

---

## 6. 错误处理策略

### 6.1 错误分类

| 错误类型 | 处理策略 | 示例 |
|----------|----------|------|
| **工具缺失** | 自动安装 → 备选工具 → AI建议 | nmap未安装 |
| **工具执行失败** | 重试 → 降级 → 记录并继续 | nmap超时 |
| **AI调用失败** | 回退到预设流程 → 离线模式 | API限流 |
| **网络错误** | 重试 → 离线模式 → 终止 | 无法访问目标 |
| **权限不足** | 提示用户 → 降级执行 | 需要root权限 |

### 6.2 错误恢复

```go
// internal/pkg/errors/recovery.go

type RecoveryStrategy struct {
    MaxRetries    int
    RetryDelay    time.Duration
    FallbackFunc  func() error
}

func (s *RecoveryStrategy) Execute(fn func() error) error {
    var lastErr error
    for i := 0; i < s.MaxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        lastErr = err
        
        // 判断是否可重试
        if !s.isRetryable(err) {
            break
        }
        
        time.Sleep(s.RetryDelay)
    }
    
    // 尝试回退
    if s.FallbackFunc != nil {
        return s.FallbackFunc()
    }
    
    return lastErr
}
```

---

## 7. 安全考虑

### 7.1 API密钥管理

```go
// internal/core/config/secrets.go

// 使用系统密钥环存储API密钥
// Windows: DPAPI
// macOS: Keychain
// Linux: Secret Service API

type SecretManager interface {
    Store(key, value string) error
    Retrieve(key string) (string, error)
    Delete(key string) error
}

// 配置文件中不存储明文密钥
// 使用环境变量或密钥环
func LoadAPIKey(provider string) string {
    // 1. 检查环境变量
    if key := os.Getenv(fmt.Sprintf("FCAPITAL_%s_API_KEY", strings.ToUpper(provider))); key != "" {
        return key
    }
    
    // 2. 检查密钥环
    sm := NewSecretManager()
    if key, err := sm.Retrieve(fmt.Sprintf("api_key_%s", provider)); err == nil {
        return key
    }
    
    return ""
}
```

### 7.2 敏感数据脱敏

```go
// internal/pkg/formatter/sanitizer.go

type Sanitizer struct {
    patterns []*regexp.Regexp
}

func NewSanitizer() *Sanitizer {
    return &Sanitizer{
        patterns: []*regexp.Regexp{
            regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|key|api_key)\s*[=:]\s*\S+`),
            regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+\S+`),
            regexp.MustCompile(`(?i)Cookie:\s*\S+`),
        },
    }
}

func (s *Sanitizer) Sanitize(text string) string {
    for _, p := range s.patterns {
        text = p.ReplaceAllString(text, "$1=***REDACTED***")
    }
    return text
}
```

---

## 8. 测试策略

### 8.1 单元测试

```go
// tests/unit/ai/engine_test.go

func TestAIEngine_Chat(t *testing.T) {
    mockProvider := &MockProvider{
        Response: &ai.ChatResponse{
            Content: `{"analysis": "test", "next_action": "test"}`,
        },
    }
    
    engine := ai.NewEngine(mockProvider)
    resp, err := engine.Chat(context.Background(), &ai.ChatRequest{
        Messages: []ai.Message{{Role: "user", Content: "test"}},
    })
    
    assert.NoError(t, err)
    assert.Contains(t, resp.Content, "analysis")
}
```

### 8.2 集成测试

```go
// tests/integration/workflow_test.go

func TestWorkflow_FullScan(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // 使用测试目标
    target := "testphp.vulnweb.com"
    
    orch := orchestrator.New(
        orchestrator.WithAIMock(NewMockAI()),
        orchestrator.WithToolMock(NewMockTools()),
    )
    
    result, err := orch.Run(context.Background(), target, nil)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.PhaseResults)
}
```

### 8.3 AI Mock

```go
// tests/mocks/ai_mock.go

type MockAIProvider struct {
    Responses map[string]string
}

func (m *MockAIProvider) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
    // 根据消息内容返回预设响应
    for key, resp := range m.Responses {
        if strings.Contains(req.Messages[0].Content, key) {
            return &ai.ChatResponse{Content: resp}, nil
        }
    }
    return &ai.ChatResponse{Content: `{"analysis": "mock response"}`}, nil
}
```

---

## 9. 部署方案

### 9.1 编译

```makefile
# Makefile

.PHONY: build
build:
	go build -ldflags="-s -w" -o build/fcapital ./cmd/fcapital

.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/fcapital-linux-amd64 ./cmd/fcapital
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o build/fcapital-windows-amd64.exe ./cmd/fcapital
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o build/fcapital-darwin-amd64 ./cmd/fcapital
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/fcapital-darwin-arm64 ./cmd/fcapital
```

### 9.2 安装脚本

```bash
#!/bin/bash
# scripts/install.sh

# 检测操作系统
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 下载对应版本
curl -L "https://github.com/Mal-Suen/fcapital/releases/latest/download/fcapital-${OS}-${ARCH}" -o /usr/local/bin/fcapital
chmod +x /usr/local/bin/fcapital

# 初始化配置
fcapital config init
```

---

## 10. 变更历史

| 版本 | 日期 | 作者 | 变更内容 |
|------|------|------|----------|
| 1.0.0 | 2026-04-20 | - | 初始架构设计 |
| 2.0.0 | 2026-04-22 | - | AI驱动架构升级 |
| 2.1.0 | 2026-04-24 | - | 混合模式架构：新增ScriptGenerator组件、决策分发器、代码审计器、沙箱执行器 |
