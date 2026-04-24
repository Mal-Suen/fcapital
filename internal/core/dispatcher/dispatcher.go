// Package dispatcher provides decision dispatching for hybrid mode execution.
package dispatcher

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
	"github.com/Mal-Suen/fcapital/internal/core/script"
	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
)

// ScenarioType represents the type of scenario.
type ScenarioType int

const (
	// ScenarioStandard indicates a standard task with mature tools.
	ScenarioStandard ScenarioType = iota
	// ScenarioNonStandard indicates a non-standard scenario requiring AI-generated scripts.
	ScenarioNonStandard
	// ScenarioMixed indicates a mixed scenario requiring both tools and scripts.
	ScenarioMixed
)

// String returns the string representation of ScenarioType.
func (s ScenarioType) String() string {
	switch s {
	case ScenarioStandard:
		return "standard"
	case ScenarioNonStandard:
		return "non-standard"
	case ScenarioMixed:
		return "mixed"
	default:
		return "unknown"
	}
}

// DispatchResult represents the result of a dispatch decision.
type DispatchResult struct {
	ScenarioType   ScenarioType `json:"scenario_type"`
	ToolName       string       `json:"tool_name,omitempty"`
	ScriptNeeded   bool         `json:"script_needed"`
	ScriptLanguage string       `json:"script_language,omitempty"`
	ScriptTask     string       `json:"script_task,omitempty"`
	Reasoning      string       `json:"reasoning"`
}

// Dispatcher dispatches tasks to appropriate handlers.
type Dispatcher struct {
	scheduler    *scheduler.Scheduler
	generator    *script.Generator
	toolmgr      *toolmgr.ToolManager
	rules        map[string]ScenarioType
	capabilities map[string]string
}

// DispatcherOption is a functional option for Dispatcher.
type DispatcherOption func(*Dispatcher)

// WithScheduler sets the scheduler.
func WithScheduler(s *scheduler.Scheduler) DispatcherOption {
	return func(d *Dispatcher) {
		d.scheduler = s
	}
}

// WithGenerator sets the script generator.
func WithGenerator(g *script.Generator) DispatcherOption {
	return func(d *Dispatcher) {
		d.generator = g
	}
}

// WithToolManager sets the tool manager.
func WithToolManager(tm *toolmgr.ToolManager) DispatcherOption {
	return func(d *Dispatcher) {
		d.toolmgr = tm
	}
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(opts ...DispatcherOption) *Dispatcher {
	d := &Dispatcher{
		rules:       defaultScenarioRules,
		capabilities: defaultCapabilityMapping,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Dispatch analyzes a task and determines the execution strategy.
func (d *Dispatcher) Dispatch(ctx context.Context, task string, context map[string]interface{}) (*DispatchResult, error) {
	result := &DispatchResult{}

	// 1. Determine scenario type
	scenarioType := d.determineScenario(task)
	result.ScenarioType = scenarioType

	// 2. Based on scenario type, determine execution strategy
	switch scenarioType {
	case ScenarioStandard:
		// Find appropriate tool
		toolName := d.findToolForTask(task)
		result.ToolName = toolName
		result.ScriptNeeded = false
		result.Reasoning = fmt.Sprintf("Task '%s' is a standard task, using tool: %s", task, toolName)

	case ScenarioNonStandard:
		// Need to generate script
		result.ScriptNeeded = true
		result.ScriptLanguage = d.determineBestLanguage(task, context)
		result.ScriptTask = task
		result.Reasoning = fmt.Sprintf("Task '%s' requires custom script generation", task)

	case ScenarioMixed:
		// Both tool and script needed
		toolName := d.findToolForTask(task)
		result.ToolName = toolName
		result.ScriptNeeded = true
		result.ScriptLanguage = d.determineBestLanguage(task, context)
		result.ScriptTask = fmt.Sprintf("Post-processing for %s", toolName)
		result.Reasoning = fmt.Sprintf("Task '%s' requires both tool (%s) and script", task, toolName)
	}

	return result, nil
}

// determineScenario determines the scenario type for a task.
func (d *Dispatcher) determineScenario(task string) ScenarioType {
	taskLower := strings.ToLower(task)

	// Check against rules
	for pattern, scenarioType := range d.rules {
		if strings.Contains(taskLower, pattern) {
			return scenarioType
		}
	}

	// Default to standard if we have a tool for it
	if d.hasToolForTask(task) {
		return ScenarioStandard
	}

	// Default to non-standard
	return ScenarioNonStandard
}

// findToolForTask finds the best tool for a task.
func (d *Dispatcher) findToolForTask(task string) string {
	taskLower := strings.ToLower(task)
	// Normalize task: replace spaces with underscores for matching
	taskNormalized := strings.ReplaceAll(taskLower, " ", "_")

	// Check capability mapping
	for capability, tool := range d.capabilities {
		if strings.Contains(taskNormalized, capability) || strings.Contains(taskLower, capability) {
			return tool
		}
	}

	// Check with scheduler if available
	if d.scheduler != nil {
		tools := d.scheduler.GetToolStatus()
		for name := range tools {
			// Simple matching
			if strings.Contains(taskLower, strings.ToLower(name)) {
				return name
			}
		}
	}

	return ""
}

// hasToolForTask checks if there's a tool available for the task.
func (d *Dispatcher) hasToolForTask(task string) bool {
	return d.findToolForTask(task) != ""
}

// determineBestLanguage determines the best scripting language for a task.
func (d *Dispatcher) determineBestLanguage(task string, context map[string]interface{}) string {
	// Check context for preferred language
	if lang, ok := context["preferred_language"].(string); ok {
		return lang
	}

	// Default based on task type
	taskLower := strings.ToLower(task)

	if strings.Contains(taskLower, "windows") || strings.Contains(taskLower, "powershell") {
		return "powershell"
	}

	if strings.Contains(taskLower, "bash") || strings.Contains(taskLower, "linux") {
		return "bash"
	}

	// Default to Python for complex tasks
	return "python"
}

// ExecuteStandard executes a standard task using tools.
func (d *Dispatcher) ExecuteStandard(ctx context.Context, toolName string, args []string) (*ExecutionResult, error) {
	// Build tool-specific arguments
	toolArgs := d.buildToolArgs(toolName, args)

	// Get tool path using toolmgr if available
	var toolPath string
	if d.toolmgr != nil {
		tool, err := d.toolmgr.Get(toolName)
		if err == nil && tool.IsReady() {
			toolPath = tool.GetPath()
		}
	}

	// If toolmgr found a path, use it directly
	if toolPath != "" {
		return d.executeTool(ctx, toolPath, toolArgs)
	}

	// Fallback to scheduler
	if d.scheduler == nil {
		return nil, fmt.Errorf("scheduler not configured")
	}

	// Check if tool is available via scheduler
	status, err := d.scheduler.CheckAvailability(toolName)
	if err != nil || status.Status != "ready" {
		// Tool not available - return error with tool name
		return nil, fmt.Errorf("tool %s is not available: %s", toolName, status.Status)
	}

	output, err := d.scheduler.Execute(ctx, toolName, toolArgs)
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{
		Success: output.Success,
		Output:  output.Output,
		Error:   output.Error,
	}, nil
}

// GetToolPath returns the tool path for debugging.
func (d *Dispatcher) GetToolPath(toolName string) string {
	if d.toolmgr != nil {
		tool, err := d.toolmgr.Get(toolName)
		if err == nil && tool.IsReady() {
			return tool.GetPath()
		}
	}
	return ""
}

// executeTool executes a tool with the given path and arguments.
func (d *Dispatcher) executeTool(ctx context.Context, toolPath string, args []string) (*ExecutionResult, error) {
	start := time.Now()
	result := &ExecutionResult{}

	// Create timeout context
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(runCtx, toolPath, args...)

	// Execute
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result, fmt.Errorf("command failed: %w", err)
	}

	result.Success = true
	return result, nil
}

// buildToolArgs builds tool-specific arguments based on tool name.
func (d *Dispatcher) buildToolArgs(toolName string, targets []string) []string {
	if len(targets) == 0 {
		return []string{}
	}

	switch toolName {
	case "nmap":
		// 简化的 nmap 参数 - 快速扫描优先
		// -Pn: 跳过主机发现（解决防火墙阻塞ping问题）
		// -sV: 服务版本探测
		// -sC: 默认脚本扫描（相当于 --script=default）
		// -T4: 较快速度
		// --top-ports 100: 扫描最常用的100个端口（更快）
		// --open: 只显示开放端口
		return []string{
			"-Pn", "-sV", "-sC",
			"-T4",
			"--top-ports", "100",
			"--open",
			targets[0],
		}

	case "httpx":
		// httpx needs: -u target -json
		return []string{"-u", targets[0], "-json"}

	case "subfinder":
		// subfinder needs: -d target
		return []string{"-d", targets[0]}

	case "nuclei":
		// nuclei needs: -u target
		return []string{"-u", targets[0]}

	case "sqlmap":
		// sqlmap needs: -u target
		return []string{"-u", targets[0], "--batch"}

	case "gobuster":
		// gobuster needs: dir -u target
		return []string{"dir", "-u", targets[0]}

	case "ffuf":
		// ffuf needs: -u target/FUZZ
		return []string{"-u", targets[0] + "/FUZZ"}

	case "wpscan":
		// wpscan needs: --url target --enumerate u,vp,vt,cb,dbe
		// --enumerate u: 用户枚举
		// --enumerate vp: 易受攻击的插件
		// --enumerate vt: 易受攻击的主题
		// --enumerate cb: 配置备份
		// --enumerate dbe: 数据库导出错误
		return []string{
			"--url", targets[0],
			"--enumerate", "u,vp,vt,cb,dbe",
			"--random-user-agent",
			"--disable-tls-checks",
		}

	default:
		// Default: just pass targets
		return targets
	}
}

// ExecuteNonStandard executes a non-standard task using AI-generated scripts.
func (d *Dispatcher) ExecuteNonStandard(ctx context.Context, task string, context map[string]interface{}, autoConfirm bool) (*ExecutionResult, error) {
	if d.generator == nil {
		return nil, fmt.Errorf("script generator not configured")
	}

	// Generate and execute script
	req := &script.GenerateRequest{
		TaskDescription: task,
		Context:         context,
		Language:        d.determineBestLanguage(task, context),
	}

	result, err := d.generator.GenerateAndExecute(ctx, req, autoConfirm)
	if err != nil {
		return nil, err
	}

	return &ExecutionResult{
		Success:  result.Success,
		Output:   result.Output,
		Error:    result.Error,
		Duration: result.Duration,
	}, nil
}

// ExecuteMixed executes a mixed task using both tools and scripts.
func (d *Dispatcher) ExecuteMixed(ctx context.Context, toolName, task string, args []string, context map[string]interface{}, autoConfirm bool) (*ExecutionResult, error) {
	// 1. Execute tool first
	toolResult, err := d.ExecuteStandard(ctx, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// 2. Execute script for post-processing
	scriptContext := context
	scriptContext["tool_output"] = toolResult.Output

	scriptResult, err := d.ExecuteNonStandard(ctx, task, scriptContext, autoConfirm)
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// 3. Combine results
	return &ExecutionResult{
		Success:  scriptResult.Success,
		Output:   fmt.Sprintf("Tool Output:\n%s\n\nScript Output:\n%s", toolResult.Output, scriptResult.Output),
		Duration: toolResult.Duration + scriptResult.Duration,
	}, nil
}

// ExecutionResult represents the result of execution.
type ExecutionResult struct {
	Success  bool          `json:"success"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// Default scenario rules
var defaultScenarioRules = map[string]ScenarioType{
	// Standard tasks (have mature tools)
	"port scan":        ScenarioStandard,
	"port_scan":        ScenarioStandard,
	"subdomain":        ScenarioStandard,
	"subfinder":        ScenarioStandard,
	"directory":        ScenarioStandard,
	"vulnerability":    ScenarioStandard,
	"sql injection":    ScenarioStandard,
	"sqlmap":           ScenarioStandard,
	"wordpress":        ScenarioStandard,
	"wpscan":           ScenarioStandard,
	"nuclei":           ScenarioStandard,
	"nmap":             ScenarioStandard,
	"gobuster":         ScenarioStandard,
	"ffuf":             ScenarioStandard,
	"httpx":            ScenarioStandard,
	"dirsearch":        ScenarioStandard,
	"nikto":            ScenarioStandard,
	"sslscan":          ScenarioStandard,
	"whatweb":          ScenarioStandard,

	// Non-standard tasks (need custom scripts)
	"custom poc":       ScenarioNonStandard,
	"custom_poc":       ScenarioNonStandard,
	"waf bypass":       ScenarioNonStandard,
	"waf_bypass":       ScenarioNonStandard,
	"encoding bypass":  ScenarioNonStandard,
	"custom protocol":  ScenarioNonStandard,
	"data processing":  ScenarioNonStandard,
	"special encoding": ScenarioNonStandard,
	"cms exploit":      ScenarioNonStandard,
	"custom exploit":   ScenarioNonStandard,

	// Mixed tasks
	"targeted exploit": ScenarioMixed,
	"chain attack":     ScenarioMixed,
	"post exploitation": ScenarioMixed,
	"persistence":      ScenarioMixed,
}

// Default capability to tool mapping
var defaultCapabilityMapping = map[string]string{
	"port_scan":           "nmap",
	"subdomain_enum":      "subfinder",
	"directory_bruteforce": "gobuster",
	"vulnerability_scan":  "nuclei",
	"sql_injection":       "sqlmap",
	"wordpress_scan":      "wpscan",
	"http_probe":          "httpx",
	"fuzzing":             "ffuf",
	"wpscan":              "wpscan",
	"dirsearch":           "dirsearch",
	"nikto":               "nikto",
	"sslscan":             "sslscan",
	"whatweb":             "whatweb",
	"nmap":                "nmap",
	"nuclei":              "nuclei",
	"gobuster":            "gobuster",
	"ffuf":                "ffuf",
	"httpx":               "httpx",
	"subfinder":           "subfinder",
	"sqlmap":              "sqlmap",
}

// AddRule adds a custom scenario rule.
func (d *Dispatcher) AddRule(pattern string, scenarioType ScenarioType) {
	d.rules[strings.ToLower(pattern)] = scenarioType
}

// AddCapability adds a custom capability mapping.
func (d *Dispatcher) AddCapability(capability, tool string) {
	d.capabilities[strings.ToLower(capability)] = tool
}

// GetScenarioRules returns the current scenario rules.
func (d *Dispatcher) GetScenarioRules() map[string]ScenarioType {
	return d.rules
}
