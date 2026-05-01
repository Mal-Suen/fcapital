package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
)

// Workflow 工作流定义
type Workflow struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Steps       []Step    `json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
}

// Step 工作流步骤
type Step struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Module      string                 `json:"module"`      // recon, subdomain, portscan, webscan, vulnscan
	Action      string                 `json:"action"`      // http, dns, passive, quick, dir, nuclei
	Tool        string                 `json:"tool"`        // 可选：指定工具
	DependsOn   []string               `json:"depends_on"`  // 依赖的步骤
	InputFrom   string                 `json:"input_from"`  // 从哪个步骤获取输入
	InputField  string                 `json:"input_field"` // 输入字段名
	Condition   string                 `json:"condition"`   // 执行条件
	Params      map[string]interface{} `json:"params"`      // 额外参数
	SkipOnError bool                   `json:"skip_on_error"`
	Timeout     time.Duration          `json:"timeout"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Target      string
	OutputDir   string
	StartTime   time.Time
	StepResults map[string]*StepResult
	mu          sync.RWMutex
}

// StepResult 步骤结果
type StepResult struct {
	StepID    string      `json:"step_id"`
	Status    string      `json:"status"` // pending, running, success, failed, skipped
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Duration  string      `json:"duration"`
	Output    interface{} `json:"output"`
	Error     string      `json:"error,omitempty"`
	RawOutput string      `json:"raw_output,omitempty"`
}

// WorkflowResult 工作流结果
type WorkflowResult struct {
	WorkflowName string                 `json:"workflow_name"`
	Target       string                 `json:"target"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     string                 `json:"duration"`
	Status       string                 `json:"status"` // running, success, partial, failed
	Steps        map[string]*StepResult `json:"steps"`
	Summary      *ScanSummary           `json:"summary"`
	OutputFile   string                 `json:"output_file"`
}

// ScanSummary 扫描摘要
type ScanSummary struct {
	Subdomains      int           `json:"subdomains"`
	AliveHosts      int           `json:"alive_hosts"`
	OpenPorts       int           `json:"open_ports"`
	Directories     int           `json:"directories"`
	Vulnerabilities int           `json:"vulnerabilities"`
	Technologies    []string      `json:"technologies"`
	Services        []ServiceInfo `json:"services"`
	CriticalVulns   []VulnInfo    `json:"critical_vulns"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
	Version string `json:"version"`
	Host    string `json:"host"`
}

// VulnInfo 漏洞信息
type VulnInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Host        string `json:"host"`
	Description string `json:"description"`
}

// Engine 工作流引擎
type Engine struct {
	toolMgr   *toolmgr.ToolManager
	workflows map[string]*Workflow
	handlers  map[string]StepHandler
}

// StepHandler 步骤处理器接口
type StepHandler interface {
	Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error)
}

// NewEngine 创建工作流引擎
func NewEngine(tm *toolmgr.ToolManager) *Engine {
	e := &Engine{
		toolMgr:   tm,
		workflows: make(map[string]*Workflow),
		handlers:  make(map[string]StepHandler),
	}

	// 注册内置工作流
	e.registerBuiltinWorkflows()

	return e
}

// RegisterHandler 注册步骤处理器
func (e *Engine) RegisterHandler(module string, handler StepHandler) {
	e.handlers[module] = handler
}

// GetWorkflow 获取工作流
func (e *Engine) GetWorkflow(name string) (*Workflow, bool) {
	wf, ok := e.workflows[name]
	return wf, ok
}

// ListWorkflows 列出所有工作流
func (e *Engine) ListWorkflows() []*Workflow {
	var workflows []*Workflow
	for _, wf := range e.workflows {
		workflows = append(workflows, wf)
	}
	return workflows
}

// Execute 执行工作流
func (e *Engine) Execute(ctx context.Context, workflowName string, target string, outputDir string) (*WorkflowResult, error) {
	wf, ok := e.workflows[workflowName]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflowName)
	}

	// 创建输出目录
	if outputDir == "" {
		homeDir, _ := os.UserHomeDir()
		outputDir = filepath.Join(homeDir, ".fcapital", "results", fmt.Sprintf("%s_%d", target, time.Now().Unix()))
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 初始化执行上下文
	execCtx := &ExecutionContext{
		Target:      target,
		OutputDir:   outputDir,
		StartTime:   time.Now(),
		StepResults: make(map[string]*StepResult),
	}

	// 初始化结果
	result := &WorkflowResult{
		WorkflowName: wf.Name,
		Target:       target,
		StartTime:    time.Now(),
		Status:       "running",
		Steps:        make(map[string]*StepResult),
		Summary:      &ScanSummary{},
	}

	// 初始化所有步骤状态
	for _, step := range wf.Steps {
		execCtx.StepResults[step.ID] = &StepResult{
			StepID: step.ID,
			Status: "pending",
		}
		result.Steps[step.ID] = execCtx.StepResults[step.ID]
	}

	// 执行步骤（拓扑排序处理依赖）
	executed := make(map[string]bool)
	for {
		progress := false
		for i := range wf.Steps {
			step := &wf.Steps[i]
			if executed[step.ID] {
				continue
			}

			// 检查依赖是否满足
			depsMet := true
			for _, dep := range step.DependsOn {
				if !executed[dep] {
					depsMet = false
					break
				}
				// 检查依赖步骤是否成功
				if depResult := execCtx.StepResults[dep]; depResult.Status != "success" && !step.SkipOnError {
					// 依赖失败，跳过此步骤
					execCtx.StepResults[step.ID].Status = "skipped"
					execCtx.StepResults[step.ID].Error = fmt.Sprintf("dependency %s failed", dep)
					executed[step.ID] = true
					progress = true
					break
				}
			}

			if !depsMet || executed[step.ID] {
				continue
			}

			// 执行步骤
			progress = true
			executed[step.ID] = true

			stepResult := execCtx.StepResults[step.ID]
			stepResult.Status = "running"
			stepResult.StartTime = time.Now()

			// 获取处理器
			handler, ok := e.handlers[step.Module]
			if !ok {
				stepResult.Status = "failed"
				stepResult.Error = fmt.Sprintf("no handler for module: %s", step.Module)
				stepResult.EndTime = time.Now()
				stepResult.Duration = stepResult.EndTime.Sub(stepResult.StartTime).String()
				continue
			}

			// 设置超时并执行步骤
			output, err := e.executeStep(ctx, step, handler, execCtx, stepResult)

			stepResult.EndTime = time.Now()
			stepResult.Duration = stepResult.EndTime.Sub(stepResult.StartTime).String()

			if err != nil {
				stepResult.Status = "failed"
				stepResult.Error = err.Error()
				fmt.Printf("[!] Step %s failed: %s\n", step.Name, err.Error())
				if !step.SkipOnError {
					result.Status = "partial"
				}
			} else {
				stepResult.Status = "success"
				stepResult.Output = output
				fmt.Printf("[+] Step %s completed in %s\n", step.Name, stepResult.Duration)
			}
		}

		// 检查是否所有步骤都已执行
		allExecuted := true
		for _, step := range wf.Steps {
			if !executed[step.ID] {
				allExecuted = false
				break
			}
		}

		if allExecuted || !progress {
			break
		}
	}

	// 计算摘要
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).String()

	if result.Status == "running" {
		result.Status = "success"
	}

	// 保存结果
	resultFile := filepath.Join(outputDir, "result.json")
	result.OutputFile = resultFile
	e.saveResult(result, resultFile)

	return result, nil
}

// executeStep 执行单个步骤（独立函数以正确处理 context cancel）
func (e *Engine) executeStep(ctx context.Context, step *Step, handler StepHandler, execCtx *ExecutionContext, stepResult *StepResult) (interface{}, error) {
	// 打印步骤开始信息
	fmt.Printf("\n[*] Running step: %s (%s/%s)\n", step.Name, step.Module, step.Action)

	// 设置超时
	if step.Timeout > 0 {
		stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
		defer cancel()
		return handler.Execute(stepCtx, step, execCtx)
	}

	return handler.Execute(ctx, step, execCtx)
}

// saveResult 保存结果
func (e *Engine) saveResult(result *WorkflowResult, path string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// registerBuiltinWorkflows 注册内置工作流
func (e *Engine) registerBuiltinWorkflows() {
	// 完整渗透测试工作流
	e.workflows["full"] = &Workflow{
		Name:        "full",
		Description: "完整渗透测试流程：子域名枚举 → HTTP探测 → 端口扫描 → 目录扫描 → 漏洞扫描",
		CreatedAt:   time.Now(),
		Steps: []Step{
			{
				ID:          "subdomain",
				Name:        "子域名枚举",
				Module:      "subdomain",
				Action:      "passive",
				Timeout:     5 * time.Minute,
				SkipOnError: true,
			},
			{
				ID:         "http_probe",
				Name:       "HTTP探测",
				Module:     "recon",
				Action:     "http",
				InputFrom:  "subdomain",
				InputField: "subdomains",
				Timeout:    3 * time.Minute,
				DependsOn:  []string{"subdomain"},
			},
			{
				ID:          "port_scan",
				Name:        "端口扫描",
				Module:      "portscan",
				Action:      "quick",
				Timeout:     10 * time.Minute,
				SkipOnError: true,
			},
			{
				ID:          "dir_scan",
				Name:        "目录扫描",
				Module:      "webscan",
				Action:      "dir",
				InputFrom:   "http_probe",
				InputField:  "urls",
				Timeout:     10 * time.Minute,
				DependsOn:   []string{"http_probe"},
				SkipOnError: true,
			},
			{
				ID:          "vuln_scan",
				Name:        "漏洞扫描",
				Module:      "vulnscan",
				Action:      "nuclei",
				InputFrom:   "http_probe",
				InputField:  "urls",
				Timeout:     15 * time.Minute,
				DependsOn:   []string{"http_probe"},
				SkipOnError: true,
			},
		},
	}

	// 快速侦察工作流
	e.workflows["recon"] = &Workflow{
		Name:        "recon",
		Description: "快速侦察：子域名枚举 → HTTP探测",
		CreatedAt:   time.Now(),
		Steps: []Step{
			{
				ID:          "subdomain",
				Name:        "子域名枚举",
				Module:      "subdomain",
				Action:      "passive",
				Timeout:     5 * time.Minute,
				SkipOnError: true,
			},
			{
				ID:         "http_probe",
				Name:       "HTTP探测",
				Module:     "recon",
				Action:     "http",
				InputFrom:  "subdomain",
				InputField: "subdomains",
				Timeout:    3 * time.Minute,
				DependsOn:  []string{"subdomain"},
			},
		},
	}

	// Web应用扫描工作流
	e.workflows["webapp"] = &Workflow{
		Name:        "webapp",
		Description: "Web应用扫描：HTTP探测 → 目录扫描 → 漏洞扫描",
		CreatedAt:   time.Now(),
		Steps: []Step{
			{
				ID:      "http_probe",
				Name:    "HTTP探测",
				Module:  "recon",
				Action:  "http",
				Timeout: 2 * time.Minute,
			},
			{
				ID:          "dir_scan",
				Name:        "目录扫描",
				Module:      "webscan",
				Action:      "dir",
				InputFrom:   "http_probe",
				InputField:  "urls",
				Timeout:     10 * time.Minute,
				DependsOn:   []string{"http_probe"},
				SkipOnError: true,
			},
			{
				ID:          "vuln_scan",
				Name:        "漏洞扫描",
				Module:      "vulnscan",
				Action:      "nuclei",
				InputFrom:   "http_probe",
				InputField:  "urls",
				Timeout:     15 * time.Minute,
				DependsOn:   []string{"http_probe"},
				SkipOnError: true,
			},
		},
	}

	// 漏洞扫描工作流
	e.workflows["vuln"] = &Workflow{
		Name:        "vuln",
		Description: "漏洞扫描：HTTP探测 → Nuclei漏洞扫描",
		CreatedAt:   time.Now(),
		Steps: []Step{
			{
				ID:      "http_probe",
				Name:    "HTTP探测",
				Module:  "recon",
				Action:  "http",
				Timeout: 2 * time.Minute,
			},
			{
				ID:          "vuln_scan",
				Name:        "漏洞扫描",
				Module:      "vulnscan",
				Action:      "nuclei",
				InputFrom:   "http_probe",
				InputField:  "urls",
				Timeout:     20 * time.Minute,
				DependsOn:   []string{"http_probe"},
				SkipOnError: true,
			},
		},
	}
}
