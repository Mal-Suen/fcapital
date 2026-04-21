package webscan

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yourname/fcapital/internal/core/toolmgr"
)

// DirResult 目录扫描结果
type DirResult struct {
	URL        string `json:"url"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
	Redirect   string `json:"redirect,omitempty"`
}

// DirscanOptions 目录扫描选项
type DirscanOptions struct {
	URL         string        // 目标 URL
	Wordlist    string        // 字典文件
	Threads     int           // 线程数
	Timeout     time.Duration // 超时时间
	Extensions  []string      // 扩展名
	ExcludeCode []int         // 排除状态码
	Recursive   bool          // 递归扫描
}

// DirsearchRunner dirsearch 运行器
type DirsearchRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewDirsearchRunner 创建 dirsearch 运行器
func NewDirsearchRunner(tm *toolmgr.ToolManager) (*DirsearchRunner, error) {
	tool, err := tm.Get("dirsearch")
	if err != nil {
		return nil, fmt.Errorf("dirsearch not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("dirsearch is not installed. Run 'fcapital deps install dirsearch'")
	}
	return &DirsearchRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// Scan 执行目录扫描
func (r *DirsearchRunner) Scan(ctx context.Context, opts *DirscanOptions) ([]DirResult, error) {
	args := r.buildArgs(opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("dirsearch execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

// ScanWithProgress 带进度的扫描
func (r *DirsearchRunner) ScanWithProgress(ctx context.Context, opts *DirscanOptions, progressFn func(DirResult)) ([]DirResult, error) {
	args := r.buildArgs(opts)

	var results []DirResult

	result, err := r.runner.RunWithProgress(ctx, args, func(line string) {
		parsed := r.parseLine(line)
		if parsed != nil {
			results = append(results, *parsed)
			if progressFn != nil {
				progressFn(*parsed)
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("dirsearch execution failed: %w", err)
	}

	if len(results) == 0 {
		results = r.parseOutput(result.Output)
	}

	return results, nil
}

func (r *DirsearchRunner) buildArgs(opts *DirscanOptions) []string {
	args := []string{}

	// 目标 URL
	args = append(args, "-u", opts.URL)

	// 字典
	if opts.Wordlist != "" {
		args = append(args, "-w", opts.Wordlist)
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Threads))
	}

	// 超时
	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	// 扩展名
	if len(opts.Extensions) > 0 {
		args = append(args, "-e", strings.Join(opts.Extensions, ","))
	}

	// 排除状态码
	if len(opts.ExcludeCode) > 0 {
		codes := make([]string, len(opts.ExcludeCode))
		for i, c := range opts.ExcludeCode {
			codes[i] = fmt.Sprintf("%d", c)
		}
		args = append(args, "-x", strings.Join(codes, ","))
	}

	// 递归
	if opts.Recursive {
		args = append(args, "-r")
	}

	// 简单输出
	args = append(args, "--simple-report")

	return args
}

// 解析行: 200  1234  /path
var dirLineRegex = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+(.+)$`)

func (r *DirsearchRunner) parseLine(line string) *DirResult {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	if matches := dirLineRegex.FindStringSubmatch(line); matches != nil {
		var status int
		var size int64
		fmt.Sscanf(matches[1], "%d", &status)
		fmt.Sscanf(matches[2], "%d", &size)

		return &DirResult{
			Path:       strings.TrimSpace(matches[3]),
			StatusCode: status,
			Size:       size,
		}
	}

	return nil
}

func (r *DirsearchRunner) parseOutput(output string) []DirResult {
	var results []DirResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if result := r.parseLine(line); result != nil {
			results = append(results, *result)
		}
	}

	return results
}

// GobusterRunner gobuster 运行器
type GobusterRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewGobusterRunner 创建 gobuster 运行器
func NewGobusterRunner(tm *toolmgr.ToolManager) (*GobusterRunner, error) {
	tool, err := tm.Get("gobuster")
	if err != nil {
		return nil, fmt.Errorf("gobuster not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("gobuster is not installed. Run 'fcapital deps install gobuster'")
	}
	return &GobusterRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// Scan 执行目录扫描
func (r *GobusterRunner) Scan(ctx context.Context, opts *DirscanOptions) ([]DirResult, error) {
	args := r.buildArgs(opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("gobuster execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

func (r *GobusterRunner) buildArgs(opts *DirscanOptions) []string {
	args := []string{"dir"}

	// 目标 URL
	args = append(args, "-u", opts.URL)

	// 字典
	if opts.Wordlist != "" {
		args = append(args, "-w", opts.Wordlist)
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Threads))
	}

	// 扩展名
	if len(opts.Extensions) > 0 {
		args = append(args, "-x", strings.Join(opts.Extensions, ","))
	}

	// 静默模式
	args = append(args, "-q")

	return args
}

// 解析: /path (Status: 200) [Size: 1234]
var gobusterLineRegex = regexp.MustCompile(`^(.+?)\s+\(Status:\s*(\d+)\)\s+\[Size:\s*(\d+)\]`)

func (r *GobusterRunner) parseOutput(output string) []DirResult {
	var results []DirResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if matches := gobusterLineRegex.FindStringSubmatch(line); matches != nil {
			var status int
			var size int64
			fmt.Sscanf(matches[2], "%d", &status)
			fmt.Sscanf(matches[3], "%d", &size)

			results = append(results, DirResult{
				Path:       strings.TrimSpace(matches[1]),
				StatusCode: status,
				Size:       size,
			})
		}
	}

	return results
}

// FfufRunner ffuf 运行器
type FfufRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewFfufRunner 创建 ffuf 运行器
func NewFfufRunner(tm *toolmgr.ToolManager) (*FfufRunner, error) {
	tool, err := tm.Get("ffuf")
	if err != nil {
		return nil, fmt.Errorf("ffuf not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("ffuf is not installed. Run 'fcapital deps install ffuf'")
	}
	return &FfufRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// Scan 执行目录扫描
func (r *FfufRunner) Scan(ctx context.Context, opts *DirscanOptions) ([]DirResult, error) {
	args := r.buildArgs(opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("ffuf execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

func (r *FfufRunner) buildArgs(opts *DirscanOptions) []string {
	args := []string{}

	// 目标 URL (FUZZ 关键字)
	url := opts.URL
	if !strings.Contains(url, "FUZZ") {
		if strings.HasSuffix(url, "/") {
			url += "FUZZ"
		} else {
			url += "/FUZZ"
		}
	}
	args = append(args, "-u", url)

	// 字典
	if opts.Wordlist != "" {
		args = append(args, "-w", opts.Wordlist)
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Threads))
	}

	// 静默模式
	args = append(args, "-s")

	return args
}

// 解析 ffuf 输出 (静默模式只输出 URL)
func (r *FfufRunner) parseOutput(output string) []DirResult {
	var results []DirResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 静默模式输出完整 URL
		results = append(results, DirResult{
			URL:  line,
			Path: extractPath(line),
		})
	}

	return results
}

func extractPath(url string) string {
	// 从 URL 提取路径
	idx := strings.Index(url, "://")
	if idx != -1 {
		url = url[idx+3:]
	}
	idx = strings.Index(url, "/")
	if idx != -1 {
		return url[idx:]
	}
	return "/"
}

// ========== 便捷函数 ==========

// Scan 目录扫描便捷函数
func Scan(ctx context.Context, tm *toolmgr.ToolManager, tool string, opts *DirscanOptions) ([]DirResult, error) {
	switch tool {
	case "dirsearch":
		runner, err := NewDirsearchRunner(tm)
		if err != nil {
			return nil, err
		}
		return runner.Scan(ctx, opts)
	case "gobuster":
		runner, err := NewGobusterRunner(tm)
		if err != nil {
			return nil, err
		}
		return runner.Scan(ctx, opts)
	case "ffuf":
		runner, err := NewFfufRunner(tm)
		if err != nil {
			return nil, err
		}
		return runner.Scan(ctx, opts)
	default:
		return nil, fmt.Errorf("unsupported tool: %s", tool)
	}
}
