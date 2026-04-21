package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/fcapital/internal/core/toolmgr"
)

// HTTPXResult httpx 扫描结果
type HTTPXResult struct {
	URL          string   `json:"url"`
	Host         string   `json:"host"`
	Port         string   `json:"port"`
	Scheme       string   `json:"scheme"`
	Title        string   `json:"title"`
	StatusCode   int      `json:"status_code"`
	ContentLength int     `json:"content_length"`
	WebServer    string   `json:"webserver"`
	ContentType  string   `json:"content_type"`
	CDN          bool     `json:"cdn"`
	Technologies []string `json:"tech"`
	ResponseTime string   `json:"time"`
	Input        string   `json:"input"`
}

// DNSXResult dnsx 查询结果
type DNSXResult struct {
	Host       string   `json:"host"`
	Domain     string   `json:"domain,omitempty"`
	RecordType string   `json:"record_type,omitempty"`
	Value      string   `json:"value,omitempty"`
	A          []string `json:"a"`
	AAAA       []string `json:"aaaa"`
	CNAME      []string `json:"cname"`
	MX         []string `json:"mx"`
	NS         []string `json:"ns"`
	TXT        []string `json:"txt"`
}

// HTTPXRunner httpx 运行器
type HTTPXRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewHTTPXRunner 创建 httpx 运行器
func NewHTTPXRunner(tm *toolmgr.ToolManager) (*HTTPXRunner, error) {
	tool, err := tm.Get("httpx")
	if err != nil {
		return nil, fmt.Errorf("httpx not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("httpx is not installed. Run 'fcapital deps install httpx'")
	}
	return &HTTPXRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// HTTPOptions HTTP 探测选项
type HTTPOptions struct {
	Ports           string        // 端口范围
	Threads         int           // 线程数
	Timeout         time.Duration // 超时时间
	FollowRedirects bool          // 跟随重定向
	Title           bool          // 获取标题
	StatusCode      bool          // 获取状态码
	ContentLength   bool          // 获取内容长度
	WebServer       bool          // 获取服务器信息
	TechDetect      bool          // 技术检测
}

// Probe 探测 HTTP 服务
func (r *HTTPXRunner) Probe(ctx context.Context, targets []string, opts *HTTPOptions) ([]HTTPXResult, error) {
	args := r.buildArgs(targets, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("httpx execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

// ProbeWithProgress 带进度的探测
func (r *HTTPXRunner) ProbeWithProgress(ctx context.Context, targets []string, opts *HTTPOptions, progressFn func(HTTPXResult)) ([]HTTPXResult, error) {
	args := r.buildArgs(targets, opts)

	var results []HTTPXResult

	result, err := r.runner.RunWithProgress(ctx, args, func(line string) {
		// 尝试解析每行 JSON
		var httpxResult HTTPXResult
		if err := json.Unmarshal([]byte(line), &httpxResult); err == nil {
			results = append(results, httpxResult)
			if progressFn != nil {
				progressFn(httpxResult)
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("httpx execution failed: %w", err)
	}

	// 如果没有通过进度回调解析，尝试解析完整输出
	if len(results) == 0 {
		results = r.parseOutput(result.Output)
	}

	return results, nil
}

func (r *HTTPXRunner) buildArgs(targets []string, opts *HTTPOptions) []string {
	args := []string{}

	// 目标 - httpx 使用 -u 参数
	if len(targets) == 1 {
		args = append(args, "-u", targets[0])
	} else {
		// 多目标通过 stdin
		args = append(args, "-l", "-")
	}

	if opts == nil {
		opts = &HTTPOptions{}
	}

	// 端口
	if opts.Ports != "" {
		args = append(args, "-p", opts.Ports)
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-threads", fmt.Sprintf("%d", opts.Threads))
	}

	// 超时
	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	// 重定向
	if opts.FollowRedirects {
		args = append(args, "-fr")
	}

	// 信息收集选项
	args = append(args, "-title", "-sc", "-cl", "-web-server", "-tech-detect")

	// JSON 输出 (使用 -j 而不是 -json)
	args = append(args, "-j", "-silent")

	return args
}

func (r *HTTPXRunner) parseOutput(output string) []HTTPXResult {
	var results []HTTPXResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result HTTPXResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		results = append(results, result)
	}

	return results
}

// DNSXRunner dnsx 运行器
type DNSXRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewDNSXRunner 创建 dnsx 运行器
func NewDNSXRunner(tm *toolmgr.ToolManager) (*DNSXRunner, error) {
	tool, err := tm.Get("dnsx")
	if err != nil {
		return nil, fmt.Errorf("dnsx not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("dnsx is not installed. Run 'fcapital deps install dnsx'")
	}
	return &DNSXRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// DNSOptions DNS 查询选项
type DNSOptions struct {
	RecordTypes []string      // 记录类型 (A, AAAA, CNAME, MX, NS, TXT)
	Resolver    string        // DNS 服务器
	Threads     int           // 线程数
	Timeout     time.Duration // 超时时间
}

// Query 查询 DNS 记录
func (r *DNSXRunner) Query(ctx context.Context, domains []string, opts *DNSOptions) ([]DNSXResult, error) {
	args := r.buildArgs(domains, opts)

	// 通过 stdin 传递域名
	domainInput := strings.Join(domains, "\n")

	result, err := r.runner.RunWithStdin(ctx, args, domainInput)
	if err != nil {
		return nil, fmt.Errorf("dnsx execution failed: %w", err)
	}

	return r.parseOutput(result.Output), nil
}

func (r *DNSXRunner) buildArgs(domains []string, opts *DNSOptions) []string {
	args := []string{}

	// 目标 - 使用 stdin 方式
	args = append(args, "-l", "-")

	if opts == nil {
		opts = &DNSOptions{}
	}

	// 记录类型
	if len(opts.RecordTypes) > 0 {
		for _, rt := range opts.RecordTypes {
			args = append(args, "-"+strings.ToLower(rt))
		}
	} else {
		// 默认查询常用记录
		args = append(args, "-a", "-aaaa", "-cname")
	}

	// 显示响应
	args = append(args, "-resp")

	// DNS 服务器
	if opts.Resolver != "" {
		args = append(args, "-resolver", opts.Resolver)
	}

	// 线程
	if opts.Threads > 0 {
		args = append(args, "-threads", fmt.Sprintf("%d", opts.Threads))
	}

	// 静默模式
	args = append(args, "-silent")

	return args
}

func (r *DNSXRunner) parseOutput(output string) []DNSXResult {
	var results []DNSXResult
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试解析 JSON 格式
		var result DNSXResult
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			results = append(results, result)
			continue
		}

		// 解析文本格式: "example.com [A] [1.2.3.4]"
		// 格式: domain [record_type] [value]
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				domain := parts[0]
				recordType := strings.Trim(parts[1], "[]")
				value := strings.Trim(parts[2], "[]")

				results = append(results, DNSXResult{
					Domain:     domain,
					RecordType: recordType,
					Value:      value,
				})
			}
		}
	}

	return results
}

// ========== 便捷函数 ==========

// HTTPProbe HTTP 探测便捷函数
func HTTPProbe(ctx context.Context, tm *toolmgr.ToolManager, targets []string) ([]HTTPXResult, error) {
	runner, err := NewHTTPXRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Probe(ctx, targets, nil)
}

// DNSQuery DNS 查询便捷函数
func DNSQuery(ctx context.Context, tm *toolmgr.ToolManager, domains []string) ([]DNSXResult, error) {
	runner, err := NewDNSXRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.Query(ctx, domains, nil)
}
