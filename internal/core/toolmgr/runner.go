package toolmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Runner 工具运行器
type Runner struct {
	tool    *Tool
	timeout time.Duration
	env     []string
}

// RunResult 运行结果
type RunResult struct {
	Tool      string        `json:"tool"`
	Command   string        `json:"command"`
	Args      []string      `json:"args"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	ExitCode  int           `json:"exit_code"`
	StartTime time.Time     `json:"start_time"`
}

// NewRunner 创建运行器
func NewRunner(tool *Tool) *Runner {
	return &Runner{
		tool:    tool,
		timeout: 10 * time.Minute,
	}
}

// SetTimeout 设置超时
func (r *Runner) SetTimeout(d time.Duration) {
	r.timeout = d
}

// SetEnv 设置环境变量
func (r *Runner) SetEnv(env []string) {
	r.env = env
}

// Tool 返回工具
func (r *Runner) Tool() *Tool {
	return r.tool
}

// Run 执行工具
func (r *Runner) Run(ctx context.Context, args []string) (*RunResult, error) {
	if !r.tool.IsReady() {
		return nil, fmt.Errorf("tool %s is not ready", r.tool.Name)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// 构建命令
	cmd := exec.CommandContext(ctx, r.tool.GetPath(), args...)
	if len(r.env) > 0 {
		cmd.Env = append(cmd.Env, r.env...)
	}

	result := &RunResult{
		Tool:      r.tool.Name,
		Command:   r.tool.GetPath(),
		Args:      args,
		StartTime: time.Now(),
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(result.StartTime)
	result.Output = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		// 如果有输出，即使退出码非零也返回结果（某些工具输出到 stderr）
		if len(output) > 0 {
			return result, nil
		}
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}

// RunJSON 执行工具并解析 JSON 输出
func (r *Runner) RunJSON(ctx context.Context, args []string, v interface{}) error {
	result, err := r.Run(ctx, args)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(result.Output), v)
}

// RunWithProgress 执行工具并实时显示进度
func (r *Runner) RunWithProgress(ctx context.Context, args []string, progressFn func(string)) (*RunResult, error) {
	if !r.tool.IsReady() {
		return nil, fmt.Errorf("tool %s is not ready", r.tool.Name)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.tool.GetPath(), args...)

	// 获取 stdout 和 stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	result := &RunResult{
		Tool:      r.tool.Name,
		Command:   r.tool.GetPath(),
		Args:      args,
		StartTime: time.Now(),
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 实时读取输出
	var outputBuilder strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			outputBuilder.WriteString(chunk)
			if progressFn != nil {
				progressFn(chunk)
			}
		}
		if err != nil {
			break
		}
	}

	err = cmd.Wait()
	result.Duration = time.Since(result.StartTime)
	result.Output = outputBuilder.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
	}

	return result, nil
}

// RunAsync 异步执行工具
func (r *Runner) RunAsync(ctx context.Context, args []string) (<-chan *RunResult, <-chan error) {
	resultCh := make(chan *RunResult, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(resultCh)
		defer close(errCh)

		result, err := r.Run(ctx, args)
		if err != nil {
			errCh <- err
		}
		resultCh <- result
	}()

	return resultCh, errCh
}

// RunWithStdin 执行工具并通过 stdin 传递输入
func (r *Runner) RunWithStdin(ctx context.Context, args []string, stdinInput string) (*RunResult, error) {
	if !r.tool.IsReady() {
		return nil, fmt.Errorf("tool %s is not ready", r.tool.Name)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.tool.GetPath(), args...)
	if len(r.env) > 0 {
		cmd.Env = append(cmd.Env, r.env...)
	}

	// 设置 stdin
	cmd.Stdin = strings.NewReader(stdinInput)

	result := &RunResult{
		Tool:      r.tool.Name,
		Command:   r.tool.GetPath(),
		Args:      args,
		StartTime: time.Now(),
	}

	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(result.StartTime)
	result.Output = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		if len(output) > 0 {
			return result, nil
		}
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}
