package vulnscan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/fcapital/internal/core/toolmgr"
)

// NucleiResult nuclei 扫描结果
type NucleiResult struct {
	TemplateID   string `json:"template-id"`
	TemplateName string `json:"info.name"`
	Severity     string `json:"info.severity"`
	Host         string `json:"host"`
	MatchedAt    string `json:"matched-at"`
	Type         string `json:"type"`
}

// NucleiOptions nuclei 扫描选项
type NucleiOptions struct {
	Templates   []string      // 模板目录
	Tags        []string      // 标签过滤
	Severity    []string      // 严重级别过滤
	Threads     int           // 线程数
	Timeout     time.Duration // 超时时间
	RateLimit   int           // 速率限制
	Silent      bool          // 静默模式
}

// NucleiRunner nuclei 运行器
type NucleiRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewNucleiRunner 创建 nuclei 运行器
func NewNucleiRunner(tm *toolmgr.ToolManager) (*NucleiRunner, error) {
	tool, err := tm.Get("nuclei")
	if err != nil {
		return nil, fmt.Errorf("nuclei not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("nuclei is not installed. Run 'fcapital deps install nuclei'")
	}
	return &NucleiRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// Scan 执行漏洞扫描
func (r *NucleiRunner) Scan(ctx context.Context, target string, opts *NucleiOptions) ([]NucleiResult, error) {
	args := r.buildArgs(target, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("nuclei execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

// ScanWithProgress 带进度的扫描
func (r *NucleiRunner) ScanWithProgress(ctx context.Context, target string, opts *NucleiOptions, progressFn func(NucleiResult)) ([]NucleiResult, error) {
	args := r.buildArgs(target, opts)

	var results []NucleiResult

	result, err := r.runner.RunWithProgress(ctx, args, func(line string) {
		var nucleiResult NucleiResult
		if err := json.Unmarshal([]byte(line), &nucleiResult); err == nil {
			results = append(results, nucleiResult)
			if progressFn != nil {
				progressFn(nucleiResult)
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("nuclei execution failed: %w", err)
	}

	if len(results) == 0 {
		results = r.parseOutput(result.Output)
	}

	return results, nil
}

func (r *NucleiRunner) buildArgs(target string, opts *NucleiOptions) []string {
	args := []string{}

	// 目标
	args = append(args, "-u", target)

	if opts == nil {
		opts = &NucleiOptions{}
	}

	// 模板目录
	if len(opts.Templates) > 0 {
		args = append(args, "-t", strings.Join(opts.Templates, ","))
	}

	// 标签过滤
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}

	// 严重级别
	if len(opts.Severity) > 0 {
		args = append(args, "-severity", strings.Join(opts.Severity, ","))
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", opts.Threads))
	}

	// 超时
	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	// 速率限制
	if opts.RateLimit > 0 {
		args = append(args, "-rate-limit", fmt.Sprintf("%d", opts.RateLimit))
	}

	// JSON 输出
	args = append(args, "-json")

	// 静默模式
	args = append(args, "-silent")

	return args
}

func (r *NucleiRunner) parseOutput(output string) []NucleiResult {
	var results []NucleiResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		results = append(results, result)
	}

	return results
}

// SQLMapResult sqlmap 扫描结果
type SQLMapResult struct {
	URL           string   `json:"url"`
	Parameter     string   `json:"parameter"`
	InjectionType string   `json:"injection_type"`
	DBMS          string   `json:"dbms"`
	Payload       string   `json:"payload"`
	Vulnerable    bool     `json:"vulnerable"`
	Databases     []string `json:"databases,omitempty"`
}

// SQLMapOptions sqlmap 扫描选项
type SQLMapOptions struct {
	Data        string        // POST 数据
	Cookie      string        // Cookie
	Level       int           // 测试级别 (1-5)
	Risk        int           // 风险级别 (1-3)
	Threads     int           // 线程数
	Techniques  string        // 注入技术
	DBMS        string        // 指定 DBMS
	RandomAgent bool          // 随机 User-Agent
	Batch       bool          // 非交互模式
}

// SQLMapRunner sqlmap 运行器
type SQLMapRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewSQLMapRunner 创建 sqlmap 运行器
func NewSQLMapRunner(tm *toolmgr.ToolManager) (*SQLMapRunner, error) {
	tool, err := tm.Get("sqlmap")
	if err != nil {
		return nil, fmt.Errorf("sqlmap not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("sqlmap is not installed. Run 'fcapital deps install sqlmap'")
	}
	return &SQLMapRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// Scan 执行 SQL 注入扫描
func (r *SQLMapRunner) Scan(ctx context.Context, url string, opts *SQLMapOptions) (*SQLMapResult, error) {
	args := r.buildArgs(url, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("sqlmap execution failed: %w", err)
	}

	return r.parseOutput(result.Output, url), nil
}

func (r *SQLMapRunner) buildArgs(url string, opts *SQLMapOptions) []string {
	args := []string{}

	// 目标 URL
	args = append(args, "-u", url)

	if opts == nil {
		opts = &SQLMapOptions{}
	}

	// POST 数据
	if opts.Data != "" {
		args = append(args, "--data", opts.Data)
	}

	// Cookie
	if opts.Cookie != "" {
		args = append(args, "--cookie", opts.Cookie)
	}

	// 测试级别
	if opts.Level > 0 && opts.Level <= 5 {
		args = append(args, fmt.Sprintf("--level=%d", opts.Level))
	}

	// 风险级别
	if opts.Risk > 0 && opts.Risk <= 3 {
		args = append(args, fmt.Sprintf("--risk=%d", opts.Risk))
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "--threads", fmt.Sprintf("%d", opts.Threads))
	}

	// 注入技术
	if opts.Techniques != "" {
		args = append(args, "--technique", opts.Techniques)
	}

	// 指定 DBMS
	if opts.DBMS != "" {
		args = append(args, "--dbms", opts.DBMS)
	}

	// 随机 User-Agent
	if opts.RandomAgent {
		args = append(args, "--random-agent")
	}

	// 非交互模式
	args = append(args, "--batch")

	return args
}

func (r *SQLMapRunner) parseOutput(output string, url string) *SQLMapResult {
	result := &SQLMapResult{
		URL: url,
	}

	// 检测是否存在注入点
	if strings.Contains(output, "Parameter:") {
		result.Vulnerable = true

		// 提取参数名
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Parameter:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					result.Parameter = parts[1]
				}
			}
			if strings.Contains(line, "Type:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					result.InjectionType = parts[1]
				}
			}
			if strings.Contains(line, "back-end DBMS:") {
				idx := strings.Index(line, "back-end DBMS:")
				if idx != -1 {
					result.DBMS = strings.TrimSpace(line[idx+14:])
				}
			}
		}
	}

	return result
}

// ========== 便捷函数 ==========

// NucleiScan nuclei 扫描便捷函数
func NucleiScan(ctx context.Context, tm *toolmgr.ToolManager, target string) ([]NucleiResult, error) {
	runner, err := NewNucleiRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Scan(ctx, target, nil)
}

// SQLMapScan sqlmap 扫描便捷函数
func SQLMapScan(ctx context.Context, tm *toolmgr.ToolManager, url string) (*SQLMapResult, error) {
	runner, err := NewSQLMapRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Scan(ctx, url, nil)
}
