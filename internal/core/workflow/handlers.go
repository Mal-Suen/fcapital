package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/Mal-Suen/fcapital/internal/modules/portscan"
	"github.com/Mal-Suen/fcapital/internal/modules/recon"
	"github.com/Mal-Suen/fcapital/internal/modules/subdomain"
	"github.com/Mal-Suen/fcapital/internal/modules/vulnscan"
	"github.com/Mal-Suen/fcapital/internal/modules/webscan"
)

// ReconHandler 信息收集处理器
type ReconHandler struct {
	tm *toolmgr.ToolManager
}

func NewReconHandler(tm *toolmgr.ToolManager) *ReconHandler {
	return &ReconHandler{tm: tm}
}

func (h *ReconHandler) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	switch step.Action {
	case "http":
		return h.executeHTTPProbe(ctx, step, execCtx)
	case "dns":
		return h.executeDNSQuery(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unknown recon action: %s", step.Action)
	}
}

func (h *ReconHandler) executeHTTPProbe(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := recon.NewHTTPXRunner(h.tm)
	if err != nil {
		return nil, err
	}

	// 获取目标列表
	targets := h.getTargets(step, execCtx)
	if len(targets) == 0 {
		targets = []string{execCtx.Target}
	}

	opts := &recon.HTTPOptions{
		Title:         true,
		StatusCode:    true,
		WebServer:     true,
		TechDetect:    true,
	}

	results, err := runner.Probe(ctx, targets, opts)
	if err != nil {
		return nil, err
	}

	// 提取URL列表供后续步骤使用
	var urls []string
	for _, r := range results {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
	}

	return map[string]interface{}{
		"results": results,
		"urls":    urls,
		"count":   len(results),
	}, nil
}

func (h *ReconHandler) executeDNSQuery(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := recon.NewDNSXRunner(h.tm)
	if err != nil {
		return nil, err
	}

	target := execCtx.Target
	if step.Params != nil {
		if t, ok := step.Params["target"].(string); ok {
			target = t
		}
	}

	result, err := runner.Query(ctx, []string{target}, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *ReconHandler) getTargets(step *Step, execCtx *ExecutionContext) []string {
	if step.InputFrom == "" {
		return nil
	}

	prevResult, ok := execCtx.StepResults[step.InputFrom]
	if !ok || prevResult.Output == nil {
		return nil
	}

	output, ok := prevResult.Output.(map[string]interface{})
	if !ok {
		return nil
	}

	fieldData, ok := output[step.InputField]
	if !ok {
		return nil
	}

	switch v := fieldData.(type) {
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}

// SubdomainHandler 子域名枚举处理器
type SubdomainHandler struct {
	tm *toolmgr.ToolManager
}

func NewSubdomainHandler(tm *toolmgr.ToolManager) *SubdomainHandler {
	return &SubdomainHandler{tm: tm}
}

func (h *SubdomainHandler) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	switch step.Action {
	case "passive":
		return h.executePassive(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unknown subdomain action: %s", step.Action)
	}
}

func (h *SubdomainHandler) executePassive(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := subdomain.NewSubfinderRunner(h.tm)
	if err != nil {
		return nil, err
	}

	opts := &subdomain.SubfinderOptions{
		Timeout: 30 * time.Second,
		Silent:  true,
	}

	results, err := runner.Enumerate(ctx, execCtx.Target, opts)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"subdomains": results,
		"count":      len(results),
	}, nil
}

// PortscanHandler 端口扫描处理器
type PortscanHandler struct {
	tm *toolmgr.ToolManager
}

func NewPortscanHandler(tm *toolmgr.ToolManager) *PortscanHandler {
	return &PortscanHandler{tm: tm}
}

func (h *PortscanHandler) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	switch step.Action {
	case "quick":
		return h.executeQuickScan(ctx, step, execCtx)
	case "full":
		return h.executeFullScan(ctx, step, execCtx)
	case "custom":
		return h.executeCustomScan(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unknown portscan action: %s", step.Action)
	}
}

func (h *PortscanHandler) executeQuickScan(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := portscan.NewNmapRunner(h.tm)
	if err != nil {
		return nil, err
	}

	result, err := runner.QuickScan(ctx, execCtx.Target)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *PortscanHandler) executeFullScan(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := portscan.NewNmapRunner(h.tm)
	if err != nil {
		return nil, err
	}

	result, err := runner.FullScan(ctx, execCtx.Target)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *PortscanHandler) executeCustomScan(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := portscan.NewNmapRunner(h.tm)
	if err != nil {
		return nil, err
	}

	ports := "1-1000"
	if step.Params != nil {
		if p, ok := step.Params["ports"].(string); ok {
			ports = p
		}
	}

	result, err := runner.CustomScan(ctx, execCtx.Target, ports)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// WebscanHandler Web扫描处理器
type WebscanHandler struct {
	tm *toolmgr.ToolManager
}

func NewWebscanHandler(tm *toolmgr.ToolManager) *WebscanHandler {
	return &WebscanHandler{tm: tm}
}

func (h *WebscanHandler) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	switch step.Action {
	case "dir":
		return h.executeDirScan(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unknown webscan action: %s", step.Action)
	}
}

func (h *WebscanHandler) executeDirScan(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	// 获取目标URL列表
	targets := h.getTargets(step, execCtx)
	if len(targets) == 0 {
		targets = []string{execCtx.Target}
	}

	toolName := "gobuster"
	if step.Tool != "" {
		toolName = step.Tool
	}

	var allResults []webscan.DirResult

	for _, target := range targets {
		opts := &webscan.DirscanOptions{
			URL:     target,
			Threads: 20,
		}

		var results []webscan.DirResult
		var err error

		switch toolName {
		case "dirsearch":
			runner, e := webscan.NewDirsearchRunner(h.tm)
			if e != nil {
				continue
			}
			results, err = runner.Scan(ctx, opts)
		case "gobuster":
			runner, e := webscan.NewGobusterRunner(h.tm)
			if e != nil {
				continue
			}
			results, err = runner.Scan(ctx, opts)
		case "ffuf":
			runner, e := webscan.NewFfufRunner(h.tm)
			if e != nil {
				continue
			}
			results, err = runner.Scan(ctx, opts)
		default:
			runner, e := webscan.NewGobusterRunner(h.tm)
			if e != nil {
				continue
			}
			results, err = runner.Scan(ctx, opts)
		}

		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return map[string]interface{}{
		"results": allResults,
		"count":   len(allResults),
	}, nil
}

func (h *WebscanHandler) getTargets(step *Step, execCtx *ExecutionContext) []string {
	if step.InputFrom == "" {
		return nil
	}

	prevResult, ok := execCtx.StepResults[step.InputFrom]
	if !ok || prevResult.Output == nil {
		return nil
	}

	output, ok := prevResult.Output.(map[string]interface{})
	if !ok {
		return nil
	}

	fieldData, ok := output[step.InputField]
	if !ok {
		return nil
	}

	switch v := fieldData.(type) {
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}

// VulnscanHandler 漏洞扫描处理器
type VulnscanHandler struct {
	tm *toolmgr.ToolManager
}

func NewVulnscanHandler(tm *toolmgr.ToolManager) *VulnscanHandler {
	return &VulnscanHandler{tm: tm}
}

func (h *VulnscanHandler) Execute(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	switch step.Action {
	case "nuclei":
		return h.executeNuclei(ctx, step, execCtx)
	case "sqlmap":
		return h.executeSQLMap(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unknown vulnscan action: %s", step.Action)
	}
}

func (h *VulnscanHandler) executeNuclei(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := vulnscan.NewNucleiRunner(h.tm)
	if err != nil {
		return nil, err
	}

	// 获取目标URL列表
	targets := h.getTargets(step, execCtx)
	if len(targets) == 0 {
		targets = []string{execCtx.Target}
	}

	opts := &vulnscan.NucleiOptions{
		Silent:    true,
		Severity:  []string{"critical", "high", "medium"},
		Templates: []string{},
	}

	var allResults []vulnscan.NucleiResult
	for _, target := range targets {
		results, err := runner.Scan(ctx, target, opts)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	// 统计严重漏洞
	var criticalVulns []vulnscan.NucleiResult
	for _, r := range allResults {
		if strings.ToLower(r.Severity) == "critical" || strings.ToLower(r.Severity) == "high" {
			criticalVulns = append(criticalVulns, r)
		}
	}

	return map[string]interface{}{
		"results":        allResults,
		"count":          len(allResults),
		"critical":       criticalVulns,
		"critical_count": len(criticalVulns),
	}, nil
}

func (h *VulnscanHandler) executeSQLMap(ctx context.Context, step *Step, execCtx *ExecutionContext) (interface{}, error) {
	runner, err := vulnscan.NewSQLMapRunner(h.tm)
	if err != nil {
		return nil, err
	}

	target := execCtx.Target
	if step.Params != nil {
		if t, ok := step.Params["url"].(string); ok {
			target = t
		}
	}

	result, err := runner.Scan(ctx, target, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *VulnscanHandler) getTargets(step *Step, execCtx *ExecutionContext) []string {
	if step.InputFrom == "" {
		return nil
	}

	prevResult, ok := execCtx.StepResults[step.InputFrom]
	if !ok || prevResult.Output == nil {
		return nil
	}

	output, ok := prevResult.Output.(map[string]interface{})
	if !ok {
		return nil
	}

	fieldData, ok := output[step.InputField]
	if !ok {
		return nil
	}

	switch v := fieldData.(type) {
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}
