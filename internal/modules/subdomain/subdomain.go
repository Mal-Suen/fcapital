package subdomain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/fcapital/internal/core/toolmgr"
)

// SubfinderRunner subfinder 运行器
type SubfinderRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewSubfinderRunner 创建 subfinder 运行器
func NewSubfinderRunner(tm *toolmgr.ToolManager) (*SubfinderRunner, error) {
	tool, err := tm.Get("subfinder")
	if err != nil {
		return nil, fmt.Errorf("subfinder not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("subfinder is not installed. Run 'fcapital deps install subfinder'")
	}
	return &SubfinderRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// SubfinderOptions subfinder 选项
type SubfinderOptions struct {
	Threads      int           // 线程数
	Timeout      time.Duration // 超时时间
	Sources      []string      // 数据源
	Exclude      []string      // 排除的数据源
	Recursive    bool          // 递归枚举
	All          bool          // 使用所有数据源
	Silent       bool          // 静默模式
}

// Enumerate 枚举子域名
func (r *SubfinderRunner) Enumerate(ctx context.Context, domain string, opts *SubfinderOptions) ([]string, error) {
	args := r.buildArgs(domain, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("subfinder execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

// EnumerateWithProgress 带进度的枚举
func (r *SubfinderRunner) EnumerateWithProgress(ctx context.Context, domain string, opts *SubfinderOptions, progressFn func(string)) ([]string, error) {
	args := r.buildArgs(domain, opts)

	var results []string

	result, err := r.runner.RunWithProgress(ctx, args, func(line string) {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
			if progressFn != nil {
				progressFn(line)
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("subfinder execution failed: %w", err)
	}

	// 如果没有通过进度回调解析，尝试解析完整输出
	if len(results) == 0 {
		results = r.parseOutput(result.Output)
	}

	return results, nil
}

func (r *SubfinderRunner) buildArgs(domain string, opts *SubfinderOptions) []string {
	args := []string{}

	// 目标域名
	args = append(args, "-d", domain)

	if opts == nil {
		opts = &SubfinderOptions{}
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Threads))
	}

	// 超时
	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	// 数据源
	if len(opts.Sources) > 0 {
		args = append(args, "-s", strings.Join(opts.Sources, ","))
	}

	// 排除数据源
	if len(opts.Exclude) > 0 {
		args = append(args, "-es", strings.Join(opts.Exclude, ","))
	}

	// 递归
	if opts.Recursive {
		args = append(args, "-recursive")
	}

	// 所有数据源
	if opts.All {
		args = append(args, "-all")
	}

	// 静默模式
	args = append(args, "-silent")

	return args
}

func (r *SubfinderRunner) parseOutput(output string) []string {
	var results []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}

	return results
}

// ========== 便捷函数 ==========

// PassiveEnum 被动子域名枚举
func PassiveEnum(ctx context.Context, tm *toolmgr.ToolManager, domain string) ([]string, error) {
	runner, err := NewSubfinderRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Enumerate(ctx, domain, nil)
}

// PassiveEnumWithOpts 带选项的被动子域名枚举
func PassiveEnumWithOpts(ctx context.Context, tm *toolmgr.ToolManager, domain string, opts *SubfinderOptions) ([]string, error) {
	runner, err := NewSubfinderRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Enumerate(ctx, domain, opts)
}
