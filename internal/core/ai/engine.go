// Package ai provides AI integration for the fcapital framework.
// It provides a unified interface for AI-powered decision making.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
)

// Engine is the main AI engine.
type Engine struct {
	provider     providers.Provider
	promptMgr    *PromptManager
	parser       *ResponseParser
	tokenCounter *TokenCounter
	mu           sync.RWMutex
	totalTokens  int64
	maxTokens    int64
}

// EngineOption is a functional option for Engine.
type EngineOption func(*Engine)

// WithMaxTokens sets the maximum total tokens.
func WithMaxTokens(max int64) EngineOption {
	return func(e *Engine) {
		e.maxTokens = max
	}
}

// NewEngine creates a new AI engine.
func NewEngine(provider providers.Provider, opts ...EngineOption) *Engine {
	e := &Engine{
		provider:     provider,
		promptMgr:    NewPromptManager(),
		parser:       NewResponseParser(),
		tokenCounter: NewTokenCounter(),
		maxTokens:    100000,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Chat sends a chat request and returns the response.
func (e *Engine) Chat(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	resp, err := e.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI chat failed: %w", err)
	}

	e.mu.Lock()
	e.totalTokens += int64(resp.TokensUsed)
	e.mu.Unlock()

	return resp, nil
}

// StreamChat sends a chat request and returns a streaming response.
func (e *Engine) StreamChat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return e.provider.StreamChat(ctx, req)
}

// Decision represents an AI decision response.
type Decision struct {
	Analysis    string       `json:"analysis"`
	NextAction  string       `json:"next_action"`
	Priority    string       `json:"priority"`
	ToolsNeeded []string     `json:"tools_needed"`
	Reasoning   string       `json:"reasoning"`
	NextPhase   string       `json:"next_phase,omitempty"`
	SkipPhases  []string     `json:"skip_phases,omitempty"`
	InstallPlan *InstallPlan `json:"install_plan,omitempty"`
}

// InstallPlan represents a tool installation plan.
type InstallPlan struct {
	Tool        string `json:"tool"`
	Method      string `json:"method"`
	Command     string `json:"command"`
	PostInstall string `json:"post_install"`
	VerifyCmd   string `json:"verify_cmd"`
}

// AnalyzePhaseResult analyzes a phase result and returns a decision.
func (e *Engine) AnalyzePhaseResult(ctx context.Context, phaseID, phaseName, target string, findings map[string]interface{}) (*Decision, error) {
	prompt := e.promptMgr.BuildPhaseAnalysisPrompt(phaseID, phaseName, target, findings)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "system", Content: e.promptMgr.GetSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
	}

	resp, err := e.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	decision, err := e.parser.ParseDecision(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return decision, nil
}

// GenerateInstallPlan generates an installation plan for a missing tool.
func (e *Engine) GenerateInstallPlan(ctx context.Context, toolName string, systemInfo map[string]interface{}) (*InstallPlan, error) {
	prompt := e.promptMgr.BuildInstallPrompt(toolName, systemInfo)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "system", Content: e.promptMgr.GetSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	resp, err := e.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	plan, err := e.parser.ParseInstallPlan(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse install plan: %w", err)
	}

	return plan, nil
}

// InitializeSession initializes an AI session with system context.
func (e *Engine) InitializeSession(ctx context.Context, systemInfo, toolsInfo string) (string, error) {
	prompt := e.promptMgr.BuildInitPrompt(systemInfo, toolsInfo)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "system", Content: e.promptMgr.GetSystemPrompt()},
			{Role: "user", Content: prompt},
		},
	}

	resp, err := e.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// GetTokenUsage returns the total tokens used.
func (e *Engine) GetTokenUsage() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.totalTokens
}

// ShouldWarnTokens returns true if token usage exceeds the threshold.
func (e *Engine) ShouldWarnTokens() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.totalTokens > e.maxTokens
}

// CountTokens counts tokens in the given text.
func (e *Engine) CountTokens(text string) int {
	return e.provider.CountTokens(text)
}

// Provider returns the current provider name.
func (e *Engine) Provider() string {
	return e.provider.Name()
}

// ResponseParser parses AI responses.
type ResponseParser struct{}

// NewResponseParser creates a new response parser.
func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

// ParseDecision parses a decision from the AI response.
func (p *ResponseParser) ParseDecision(content string) (*Decision, error) {
	jsonStr := p.extractJSON(content)
	if jsonStr == "" {
		return &Decision{
			Analysis: content,
			Priority: "medium",
		}, nil
	}

	var decision Decision
	if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse decision JSON: %w", err)
	}

	return &decision, nil
}

// ParseInstallPlan parses an install plan from the AI response.
func (p *ResponseParser) ParseInstallPlan(content string) (*InstallPlan, error) {
	jsonStr := p.extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var plan InstallPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse install plan JSON: %w", err)
	}

	return &plan, nil
}

// extractJSON extracts JSON from a response that may contain markdown code blocks.
func (p *ResponseParser) extractJSON(content string) string {
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json")
		if start != -1 {
			start += 7
			end := strings.Index(content[start:], "```")
			if end != -1 {
				return strings.TrimSpace(content[start : start+end])
			}
		}
	}

	if strings.Contains(content, "```") {
		start := strings.Index(content, "```")
		if start != -1 {
			start += 3
			if idx := strings.Index(content[start:], "\n"); idx != -1 {
				start += idx + 1
			}
			end := strings.Index(content[start:], "```")
			if end != -1 {
				return strings.TrimSpace(content[start : start+end])
			}
		}
	}

	start := strings.Index(content, "{")
	if start != -1 {
		end := strings.LastIndex(content, "}")
		if end != -1 && end > start {
			return content[start : end+1]
		}
	}

	return ""
}

// TokenCounter counts tokens in text.
type TokenCounter struct {
	charsPerToken float64
}

// NewTokenCounter creates a new token counter.
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		charsPerToken: 4.0,
	}
}

// Count estimates the number of tokens in the text.
func (tc *TokenCounter) Count(text string) int {
	return int(float64(len(text)) / tc.charsPerToken)
}

// PromptManager manages AI prompts.
type PromptManager struct {
	systemPrompt string
}

// NewPromptManager creates a new prompt manager.
func NewPromptManager() *PromptManager {
	return &PromptManager{
		systemPrompt: defaultSystemPrompt,
	}
}

// GetSystemPrompt returns the system prompt.
func (pm *PromptManager) GetSystemPrompt() string {
	return pm.systemPrompt
}

// SetSystemPrompt sets a custom system prompt.
func (pm *PromptManager) SetSystemPrompt(prompt string) {
	pm.systemPrompt = prompt
}

// BuildInitPrompt builds the initialization prompt.
func (pm *PromptManager) BuildInitPrompt(systemInfo, toolsInfo string) string {
	return fmt.Sprintf(initPromptTemplate, systemInfo, toolsInfo)
}

// BuildPhaseAnalysisPrompt builds a phase analysis prompt.
func (pm *PromptManager) BuildPhaseAnalysisPrompt(phaseID, phaseName, target string, findings map[string]interface{}) string {
	findingsJSON, _ := json.MarshalIndent(findings, "", "  ")
	return fmt.Sprintf(phaseAnalysisTemplate, phaseID, phaseName, target, string(findingsJSON))
}

// BuildInstallPrompt builds an installation prompt.
func (pm *PromptManager) BuildInstallPrompt(toolName string, systemInfo map[string]interface{}) string {
	infoJSON, _ := json.MarshalIndent(systemInfo, "", "  ")
	return fmt.Sprintf(installPromptTemplate, toolName, string(infoJSON))
}

// Default prompts
const defaultSystemPrompt = `你是一个专业的渗透测试助手，协助安全研究员进行授权的渗透测试工作。

你的职责：
1. 分析渗透测试各阶段的结果，提供专业的安全洞察
2. 根据发现的信息，推荐下一步测试方向
3. 评估漏洞严重程度，优先处理高风险问题
4. 在工具缺失时，提供安装建议

重要原则：
- 始终假设测试已获得授权
- 关注真实的安全风险，避免误报
- 提供可操作的建议，而非泛泛而谈
- 输出JSON格式时，确保格式正确可解析`

const initPromptTemplate = `请根据以下环境信息，准备协助进行渗透测试。

系统信息：
%s

工具状态：
%s

请确认你已了解当前环境，并准备开始协助。简要回复即可。`

const phaseAnalysisTemplate = `当前阶段：%s (%s)
目标：%s

发现结果：
%s

请分析以上结果，并给出下一步建议。以JSON格式输出：
{
  "analysis": "对当前发现的分析",
  "next_action": "建议的下一步操作",
  "priority": "high/medium/low",
  "tools_needed": ["需要的工具列表"],
  "reasoning": "推理过程",
  "next_phase": "建议的下一阶段（可选）",
  "skip_phases": ["建议跳过的阶段（可选）"]
}`

const installPromptTemplate = `需要安装工具：%s

当前系统环境：
%s

请生成安装方案。以JSON格式输出：
{
  "tool": "工具名称",
  "method": "安装方法（winget/apt/brew/go/pip等）",
  "command": "安装命令",
  "post_install": "安装后需要执行的命令（可选）",
  "verify_cmd": "验证安装的命令"
}`

// ProviderConfig represents provider configuration.
type ProviderConfig struct {
	Type    string        `json:"type" yaml:"type"`
	APIKey  string        `json:"api_key" yaml:"api_key"`
	Model   string        `json:"model" yaml:"model"`
	BaseURL string        `json:"base_url" yaml:"base_url"`
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultProviderConfigs returns default provider configurations.
func DefaultProviderConfigs() map[string]*ProviderConfig {
	return map[string]*ProviderConfig{
		"openai": {
			Type:    "openai",
			Model:   "gpt-4o",
			BaseURL: "https://api.openai.com/v1",
			Timeout: 60 * time.Second,
		},
		"deepseek": {
			Type:    "deepseek",
			Model:   "deepseek-chat",
			BaseURL: "https://api.deepseek.com/v1",
			Timeout: 60 * time.Second,
		},
		"ollama": {
			Type:    "ollama",
			Model:   "llama3",
			BaseURL: "http://localhost:11434",
			Timeout: 120 * time.Second,
		},
	}
}
