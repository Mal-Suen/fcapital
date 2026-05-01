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
	Host       string        `json:"host"`
	Ports      []PortInfo    `json:"ports"`
	Vulns      []VulnInfo    `json:"vulns,omitempty"`
	ScanTime   time.Duration `json:"scan_time"`
	Command    string        `json:"command"`
	RawOutput  string        `json:"raw_output,omitempty"`
}

// PortInfo 端口信息
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Version  string `json:"version,omitempty"`
	Scripts  []ScriptResult `json:"scripts,omitempty"`
}

// ScriptResult NSE 脚本结果
type ScriptResult struct {
	ID      string `json:"id"`
	Output  string `json:"output"`
}

// VulnInfo 漏洞信息
type VulnInfo struct {
	Port        int    `json:"port"`
	Service     string `json:"service"`
	VulnID      string `json:"vuln_id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
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
	VulnScan    bool          // 漏洞扫描
}

// QuickScan 快速扫描 (Top 100)
func (r *NmapRunner) QuickScan(ctx context.Context, target string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       "-F",
		Timing:      4,
		ServiceScan: true,
		VersionScan: true,
		VulnScan:    false,
	})
}

// FullScan 全端口扫描
func (r *NmapRunner) FullScan(ctx context.Context, target string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       "-",
		Timing:      4,
		ServiceScan: true,
		VersionScan: true,
		VulnScan:    true, // 全端口扫描默认启用漏洞扫描
	})
}

// CustomScan 自定义端口扫描
func (r *NmapRunner) CustomScan(ctx context.Context, target string, ports string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       ports,
		Timing:      4,
		ServiceScan: true,
		VersionScan: true,
		VulnScan:    false,
	})
}

// QuickScanWithVuln 快速扫描 + 漏洞扫描
func (r *NmapRunner) QuickScanWithVuln(ctx context.Context, target string) (*NmapResult, error) {
	return r.Scan(ctx, target, &ScanOptions{
		Ports:       "-F",
		Timing:      4,
		ServiceScan: true,
		VersionScan: true,
		VulnScan:    true,
	})
}

// Scan 执行扫描
func (r *NmapRunner) Scan(ctx context.Context, target string, opts *ScanOptions) (*NmapResult, error) {
	args := r.buildArgs(target, opts)

	result, err := r.runner.Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("nmap execution failed: %w", err)
	}

	ports, vulns := r.parseOutput(result.Output)

	return &NmapResult{
		Host:      target,
		Command:   strings.Join(args, " "),
		ScanTime:  result.Duration,
		Ports:     ports,
		Vulns:     vulns,
		RawOutput: result.Output,
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

	ports, vulns := r.parseOutput(result.Output)

	return &NmapResult{
		Host:      target,
		Command:   strings.Join(args, " "),
		ScanTime:  result.Duration,
		Ports:     ports,
		Vulns:     vulns,
		RawOutput: result.Output,
	}, nil
}

func (r *NmapRunner) buildArgs(target string, opts *ScanOptions) []string {
	args := []string{}

	if opts == nil {
		opts = &ScanOptions{}
	}

	// SYN 扫描 (需要 root/管理员权限，更快)
	// 如果没有权限会自动回退到 connect 扫描
	args = append(args, "-sS")

	// 跳过主机发现（直接扫描端口）
	args = append(args, "-Pn")

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

	// 版本探测
	if opts.VersionScan {
		args = append(args, "--version-intensity", "5")
	}

	// 操作系统探测
	if opts.OSDetection {
		args = append(args, "-O")
	}

	// 时间模板 - 默认 T4
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

	// NSE 脚本 - 漏洞扫描
	if opts.VulnScan {
		// 使用 default 和 vulners 脚本
		// default: 基础信息收集脚本
		// vulners: 基于 CVE 的漏洞检测
		args = append(args, "--script", "default,vulners")
	} else if opts.Script != "" {
		args = append(args, "--script", opts.Script)
	}

	// 输出格式
	args = append(args, "-oN", "-")

	// 目标
	args = append(args, target)

	return args
}

// 端口行正则: 22/tcp  open  ssh  OpenSSH 6.6.1p1
// 允许前导空格，匹配端口/协议 状态 服务 版本
var portLineRegex = regexp.MustCompile(`^\s*(\d+)/(tcp|udp)\s+(open|closed|filtered)\s+(.+)$`)

// 脚本结果正则: | ssl-heartbleed: NOT VULNERABLE
var scriptOutputRegex = regexp.MustCompile(`^\|\s+([a-zA-Z0-9_-]+):\s*(.*)$`)

func (r *NmapRunner) parseOutput(output string) ([]PortInfo, []VulnInfo) {
	var ports []PortInfo
	var vulns []VulnInfo
	var currentPort *PortInfo
	var inScriptOutput bool
	var currentScriptID string
	var scriptOutput strings.Builder

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// 不要 TrimSpace，保留前导空格用于正则匹配

		// 匹配端口行
		if matches := portLineRegex.FindStringSubmatch(line); matches != nil {
			// 保存之前的端口
			if currentPort != nil && currentPort.State == "open" {
				ports = append(ports, *currentPort)
			}

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

			currentPort = &port
			inScriptOutput = false
			currentScriptID = ""
			scriptOutput.Reset()
			continue
		}

		// 解析脚本输出
		if currentPort != nil {
			trimmedLine := strings.TrimSpace(line)
			// 脚本输出开始
			if strings.HasPrefix(trimmedLine, "|") {
				inScriptOutput = true
				if matches := scriptOutputRegex.FindStringSubmatch(trimmedLine); matches != nil {
					if currentScriptID != "" && scriptOutput.Len() > 0 {
						// 保存之前的脚本结果
						currentPort.Scripts = append(currentPort.Scripts, ScriptResult{
							ID:     currentScriptID,
							Output: strings.TrimSpace(scriptOutput.String()),
						})
					}
					currentScriptID = matches[1]
					scriptOutput.Reset()
					scriptOutput.WriteString(matches[2])
				} else {
					// 续行
					scriptOutput.WriteString("\n" + strings.TrimPrefix(trimmedLine, "|"))
				}

				// 检查是否发现漏洞
				if strings.Contains(trimmedLine, "VULNERABLE") || strings.Contains(trimmedLine, "VULN") {
					vuln := VulnInfo{
						Port:    currentPort.Port,
						Service: currentPort.Service,
						VulnID:  currentScriptID,
						Name:    currentScriptID,
					}
					if strings.Contains(trimmedLine, "HIGH") || strings.Contains(trimmedLine, "CRITICAL") {
						vuln.Severity = "high"
					} else if strings.Contains(trimmedLine, "MEDIUM") {
						vuln.Severity = "medium"
					} else {
						vuln.Severity = "low"
					}
					vulns = append(vulns, vuln)
				}
			} else if strings.HasPrefix(trimmedLine, "|_") {
				// 脚本输出结束
				if currentScriptID != "" {
					currentPort.Scripts = append(currentPort.Scripts, ScriptResult{
						ID:     currentScriptID,
						Output: strings.TrimSpace(scriptOutput.String()),
					})
				}
				inScriptOutput = false
				currentScriptID = ""
				scriptOutput.Reset()
			} else if inScriptOutput && trimmedLine != "" {
				// 续行内容
				scriptOutput.WriteString("\n" + strings.TrimPrefix(strings.TrimPrefix(trimmedLine, "|"), " "))
			}
		}
	}

	// 保存最后一个端口
	if currentPort != nil && currentPort.State == "open" {
		ports = append(ports, *currentPort)
	}

	return ports, vulns
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
