package portscan

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
)

// NmapResult nmap 扫描结果
type NmapResult struct {
	Host     string        `json:"host"`
	Ports    []PortInfo    `json:"ports"`
	ScanTime time.Duration `json:"scan_time"`
	Command  string        `json:"command"`
}

// PortInfo 端口信息
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Version  string `json:"version,omitempty"`
}

// NmapRunner nmap 运行器
type NmapRunner struct {
	tool   *toolmgr.Tool
	runner *toolmgr.Runner
}

// NewNmapRunner 创建 nmap 运行器
func NewNmapRunner(tm *toolmgr.ToolManager) (*NmapRunner, error) {
	tool, err := tm.Get("nmap")
	if err != nil {
		return nil, fmt.Errorf("nmap not found in tool manager: %w", err)
	}
	if !tool.IsReady() {
		return nil, fmt.Errorf("nmap is not installed. Run 'fcapital deps install nmap'")
	}
	return &NmapRunner{
		tool:   tool,
		runner: toolmgr.NewRunner(tool),
	}, nil
}

// ScanOptions 扫描选项
type ScanOptions struct {
	Ports       string        // 端口范围 (e.g., "1-1000", "80,443,8080")
	ScanType    string        // 扫描类型 (syn, connect, udp)
	ServiceScan bool          // 服务探测
	VersionScan bool          // 版本探测
	OSDetection bool          // 操作系统探测
	Timing      int           // 时间模板 (0-5)
	Threads     int           // 最小速率
	Timeout     time.Duration // 超时时间
	Script      string        // NSE 脚本
}

// QuickScan 快速扫描 (Top 100)
func (r *NmapRunner) QuickScan(ctx context.Context, target string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       "-F",
		Timing:      4,
		ServiceScan: true,
	})
}

// FullScan 全端口扫描
func (r *NmapRunner) FullScan(ctx context.Context, target string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       "-",
		Timing:      3,
		ServiceScan: true,
	})
}

// CustomScan 自定义端口扫描
func (r *NmapRunner) CustomScan(ctx context.Context, target string, ports string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       ports,
		Timing:      4,
		ServiceScan: true,
	})
}

// Scan 执行扫描
func (r *NmapRunner) Scan(ctx context.Context, target string, opts *ScanOptions) (*NmapResult, error) {
	args := r.buildArgs(target, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}

	return &NmapResult{
		Host:     target,
		Command:  strings.Join(args, " "),
		ScanTime: result.Duration,
		Ports:    r.parseOutput(result.Output),
	}, nil
}

// ScanWithProgress 带进度的扫描
func (r *NmapRunner) ScanWithProgress(ctx context.Context, target string, opts *ScanOptions, progressFn func(string)) (*NmapResult, error) {
	args := r.buildArgs(target, opts)

	result, err := r.runner.RunWithProgress(ctx, args, func(line string) {
		if progressFn != nil {
			progressFn(line)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}

	return &NmapResult{
		Host:     target,
		Command:  strings.Join(args, " "),
		ScanTime: result.Duration,
		Ports:    r.parseOutput(result.Output),
	}, nil
}

func (r *NmapRunner) buildArgs(target string, opts *ScanOptions) []string {
	args := []string{}

	if opts == nil {
		opts = &ScanOptions{}
	}

	// 扫描类型
	switch opts.ScanType {
	case "syn", "sS":
		args = append(args, "-sS")
	case "connect", "sT":
		args = append(args, "-sT")
	case "udp", "sU":
		args = append(args, "-sU")
	default:
		// Windows 上默认使用 connect 扫描（syn 需要 root）
		args = append(args, "-sT")
	}

	// 端口
	if opts.Ports != "" {
		if opts.Ports == "-" {
			args = append(args, "-p-")
		} else if opts.Ports == "-F" {
			args = append(args, "-F")
		} else {
			args = append(args, "-p", opts.Ports)
		}
	} else {
		args = append(args, "-F") // 默认快速扫描
	}

	// 服务探测
	if opts.ServiceScan {
		args = append(args, "-sV")
	}

	// 操作系统探测
	if opts.OSDetection {
		args = append(args, "-O")
	}

	// 时间模板
	if opts.Timing >= 0 && opts.Timing <= 5 {
		args = append(args, fmt.Sprintf("-T%d", opts.Timing))
	} else {
		args = append(args, "-T4")
	}

	// 最小速率
	if opts.Threads > 0 {
		args = append(args, "--min-rate", fmt.Sprintf("%d", opts.Threads))
	}

	// 超时
	if opts.Timeout > 0 {
		args = append(args, "--max-time", fmt.Sprintf("%d", int(opts.Timeout.Seconds())))
	}

	// NSE 脚本
	if opts.Script != "" {
		args = append(args, "--script", opts.Script)
	}

	// 输出格式
	args = append(args, "-oN", "-")

	// 目标
	args = append(args, target)

	return args
}

// 端口行正则: 80/tcp   open  http    nginx 1.18.0
var portLineRegex = regexp.MustCompile(`^(\d+)/(tcp|udp)\s+(open|closed|filtered)\s+(.+)$`)

func (r *NmapRunner) parseOutput(output string) []PortInfo {
	var ports []PortInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 匹配端口行
		if matches := portLineRegex.FindStringSubmatch(line); matches != nil {
			port := PortInfo{
				Protocol: matches[2],
				State:    matches[3],
			}
			fmt.Sscanf(matches[1], "%d", &port.Port)

			// 解析服务信息
			serviceInfo := strings.TrimSpace(matches[4])
			parts := strings.Fields(serviceInfo)
			if len(parts) > 0 {
				port.Service = parts[0]
			}
			if len(parts) > 1 {
				port.Version = strings.Join(parts[1:], " ")
			}

			// 只记录开放端口
			if port.State == "open" {
				ports = append(ports, port)
			}
		}
	}

	return ports
}

// ========== 便捷函数 ==========

// QuickScan 快速扫描便捷函数
func QuickScan(ctx context.Context, tm *toolmgr.ToolManager, target string) (*NmapResult, error) {
	runner, err := NewNmapRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.QuickScan(ctx, target)
}

// FullScan 全端口扫描便捷函数
func FullScan(ctx context.Context, tm *toolmgr.ToolManager, target string) (*NmapResult, error) {
	runner, err := NewNmapRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.FullScan(ctx, target)
}

// CustomScan 自定义扫描便捷函数
func CustomScan(ctx context.Context, tm *toolmgr.ToolManager, target, ports string) (*NmapResult, error) {
	runner, err := NewNmapRunner(tm)
	if err != nil {
		return nil, err
	}
	return runner.CustomScan(ctx, target, ports)
}
