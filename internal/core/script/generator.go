// Package script provides AI-powered script generation for non-standard penetration testing scenarios.
package script

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
)

// Generator generates scripts using AI for non-standard scenarios.
type Generator struct {
	ai       providers.Provider
	auditor  *Auditor
	executor *Executor
	prompts  *PromptManager
}

// GeneratorOption is a functional option for Generator.
type GeneratorOption func(*Generator)

// WithAuditor sets a custom auditor.
func WithAuditor(auditor *Auditor) GeneratorOption {
	return func(g *Generator) {
		g.auditor = auditor
	}
}

// WithExecutor sets a custom executor.
func WithExecutor(executor *Executor) GeneratorOption {
	return func(g *Generator) {
		g.executor = executor
	}
}

// NewGenerator creates a new script generator.
func NewGenerator(ai providers.Provider, opts ...GeneratorOption) *Generator {
	g := &Generator{
		ai:       ai,
		auditor:  NewAuditor(),
		executor: NewExecutor(),
		prompts:  NewPromptManager(),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// GenerateRequest represents a script generation request.
type GenerateRequest struct {
	TaskDescription string                 `json:"task_description"`
	Context         map[string]interface{} `json:"context"`
	Language        string                 `json:"language"`
	Constraints     []string               `json:"constraints"`
	Target          string                 `json:"target"`
	ScenarioType    string                 `json:"scenario_type"`
}

// GeneratedScript represents a generated script.
type GeneratedScript struct {
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	SafetyScore int      `json:"safety_score"`
	Warnings    []string `json:"warnings"`
	Estimate    string   `json:"estimate"`
	Explanation string   `json:"explanation"`
}

// Generate generates a script based on the request.
func (g *Generator) Generate(ctx context.Context, req *GenerateRequest) (*GeneratedScript, error) {
	// 1. Build prompt
	prompt := g.prompts.BuildGeneratePrompt(req)

	// 2. Call AI
	aiReq := &providers.ChatRequest{
		Messages: []providers.Message{
			{Role: "system", Content: g.prompts.GetSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	resp, err := g.ai.Chat(ctx, aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 3. Parse response
	script, err := g.parseResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 4. Audit code
	auditResult := g.auditor.Audit(script.Code)
	script.SafetyScore = auditResult.Score
	script.Warnings = auditResult.Warnings

	return script, nil
}

// GenerateAndExecute generates a script and executes it with user confirmation.
func (g *Generator) GenerateAndExecute(ctx context.Context, req *GenerateRequest, autoConfirm bool) (*ExecutionResult, error) {
	// 1. Generate script
	script, err := g.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. Check safety score
	if script.SafetyScore < 50 {
		return nil, fmt.Errorf("script safety score too low (%d), execution blocked", script.SafetyScore)
	}

	// 3. Show code and get confirmation
	if !autoConfirm {
		confirmed := g.showCodeAndWaitConfirm(script)
		if !confirmed {
			return nil, fmt.Errorf("user cancelled execution")
		}
	}

	// 4. Execute in sandbox first
	sandboxResult := g.executor.RunInSandbox(script.Code, script.Language, 30*time.Second)
	if sandboxResult.Error != nil {
		return nil, fmt.Errorf("sandbox execution failed: %w", sandboxResult.Error)
	}

	// 5. Execute for real
	result := g.executor.Execute(script.Code, script.Language, req.Target)

	return result, nil
}

// parseResponse parses the AI response to extract the script.
func (g *Generator) parseResponse(content string) (*GeneratedScript, error) {
	script := &GeneratedScript{
		Language: "python",
	}

	// Try to parse as JSON first
	jsonStr := g.extractJSON(content)
	if jsonStr != "" {
		var data struct {
			Code        string `json:"code"`
			Language    string `json:"language"`
			Explanation string `json:"explanation"`
			Estimate    string `json:"estimate"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
			script.Code = data.Code
			script.Language = data.Language
			script.Explanation = data.Explanation
			script.Estimate = data.Estimate
			return script, nil
		}
	}

	// Extract code from markdown code blocks
	script.Code = g.extractCode(content)
	if script.Code == "" {
		return nil, fmt.Errorf("no code found in response")
	}

	script.Explanation = g.extractExplanation(content)

	return script, nil
}

// extractJSON extracts JSON from markdown code blocks.
func (g *Generator) extractJSON(content string) string {
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
	return ""
}

// extractCode extracts code from markdown code blocks.
func (g *Generator) extractCode(content string) string {
	// Try different language markers
	languages := []string{"python", "bash", "powershell", "go", "sh", ""}
	for _, lang := range languages {
		marker := "```" + lang
		if strings.Contains(content, marker) {
			start := strings.Index(content, marker)
			if start != -1 {
				start += len(marker)
				// Skip newline after marker
				if start < len(content) && content[start] == '\n' {
					start++
				}
				end := strings.Index(content[start:], "```")
				if end != -1 {
					return strings.TrimSpace(content[start : start+end])
				}
			}
		}
	}

	// Try generic code block
	if strings.Contains(content, "```") {
		start := strings.Index(content, "```")
		if start != -1 {
			start += 3
			// Skip language identifier and newline
			if idx := strings.Index(content[start:], "\n"); idx != -1 {
				start += idx + 1
			}
			end := strings.Index(content[start:], "```")
			if end != -1 {
				return strings.TrimSpace(content[start : start+end])
			}
		}
	}

	return ""
}

// extractExplanation extracts explanation from the response.
func (g *Generator) extractExplanation(content string) string {
	// Look for explanation section
	markers := []string{"Explanation:", "说明:", "描述:", "Description:"}
	for _, marker := range markers {
		if idx := strings.Index(content, marker); idx != -1 {
			start := idx + len(marker)
			end := len(content)
			// Find next section or code block
			if codeIdx := strings.Index(content[start:], "```"); codeIdx != -1 {
				end = start + codeIdx
			}
			return strings.TrimSpace(content[start:end])
		}
	}
	return ""
}

// showCodeAndWaitConfirm displays the code and waits for user confirmation.
func (g *Generator) showCodeAndWaitConfirm(script *GeneratedScript) bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 Generated Script")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Language: %s\n", script.Language)
	fmt.Printf("Safety Score: %d/100\n", script.SafetyScore)
	if len(script.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, w := range script.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	fmt.Println("\n📝 Code:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(script.Code)
	fmt.Println(strings.Repeat("-", 60))
	if script.Explanation != "" {
		fmt.Printf("\n💡 Explanation: %s\n", script.Explanation)
	}
	fmt.Println()

	// In a real implementation, this would prompt the user
	// For now, we return true to allow execution
	fmt.Print("Execute this script? [y/N]: ")
	// In automated mode, default to no
	return false
}

// PromptManager manages prompts for script generation.
type PromptManager struct {
	systemPrompt string
}

// NewPromptManager creates a new prompt manager.
func NewPromptManager() *PromptManager {
	return &PromptManager{
		systemPrompt: scriptSystemPrompt,
	}
}

// GetSystemPrompt returns the system prompt.
func (pm *PromptManager) GetSystemPrompt() string {
	return pm.systemPrompt
}

// BuildGeneratePrompt builds a script generation prompt.
func (pm *PromptManager) BuildGeneratePrompt(req *GenerateRequest) string {
	contextJSON, _ := json.MarshalIndent(req.Context, "", "  ")
	constraints := strings.Join(req.Constraints, "\n- ")

	return fmt.Sprintf(generatePromptTemplate,
		req.ScenarioType,
		req.TaskDescription,
		req.Target,
		string(contextJSON),
		req.Language,
		constraints,
	)
}

const scriptSystemPrompt = `你是一个专业的渗透测试脚本生成助手。你的任务是为授权的渗透测试生成辅助脚本。

重要原则：
1. 只生成用于授权测试的脚本
2. 代码必须安全、高效、可读
3. 添加必要的错误处理
4. 包含清晰的注释说明
5. 避免危险的系统操作（如删除文件、修改系统配置）

输出格式：
{
  "language": "python/bash/powershell",
  "code": "脚本代码",
  "explanation": "脚本说明",
  "estimate": "预估执行时间"
}`

const generatePromptTemplate = `场景类型: %s

任务描述: %s

目标: %s

上下文信息:
%s

目标语言: %s

约束条件:
- %s

请生成一个脚本来完成上述任务。确保代码安全、高效。`
