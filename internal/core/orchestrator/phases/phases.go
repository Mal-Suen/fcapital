// Package phases provides concrete phase implementations for penetration testing.
package phases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/orchestrator"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
)

// ReconPhase implements the reconnaissance phase.
type ReconPhase struct {
	scheduler *scheduler.Scheduler
}

// NewReconPhase creates a new reconnaissance phase.
func NewReconPhase(s *scheduler.Scheduler) *ReconPhase {
	return &ReconPhase{scheduler: s}
}

// ID returns the phase ID.
func (p *ReconPhase) ID() string {
	return "recon"
}

// Name returns the phase name.
func (p *ReconPhase) Name() string {
	return "信息收集"
}

// Description returns the phase description.
func (p *ReconPhase) Description() string {
	return "收集目标信息，包括子域名、开放端口、服务指纹等"
}

// CanSkip returns whether this phase can be skipped.
func (p *ReconPhase) CanSkip() bool {
	return false
}

// Dependencies returns the phase dependencies.
func (p *ReconPhase) Dependencies() []string {
	return []string{}
}

// Execute executes the reconnaissance phase.
func (p *ReconPhase) Execute(ctx context.Context, input *orchestrator.PhaseInput) (*orchestrator.PhaseOutput, error) {
	output := &orchestrator.PhaseOutput{
		PhaseID:   p.ID(),
		PhaseName: p.Name(),
		StartTime: time.Now(),
		Findings:  make(map[string]interface{}),
	}

	target := input.Target

	// 1. Subdomain enumeration
	subdomains, err := p.enumSubdomains(ctx, target)
	if err != nil {
		output.Findings["subdomain_error"] = err.Error()
	} else {
		output.Findings["subdomains"] = subdomains
	}

	// 2. HTTP probe
	var aliveHosts []map[string]interface{}
	if len(subdomains) > 0 {
		aliveHosts, err = p.probeHTTP(ctx, subdomains)
		if err != nil {
			output.Findings["http_probe_error"] = err.Error()
		} else {
			output.Findings["alive_hosts"] = aliveHosts
		}
	}

	// 3. Port scan on alive hosts
	var openPorts []map[string]interface{}
	if len(aliveHosts) > 0 {
		hosts := make([]string, len(aliveHosts))
		for i, h := range aliveHosts {
			hosts[i] = h["host"].(string)
		}
		openPorts, err = p.scanPorts(ctx, hosts)
		if err != nil {
			output.Findings["port_scan_error"] = err.Error()
		} else {
			output.Findings["open_ports"] = openPorts
		}
	}

	output.Status = "completed"
	output.EndTime = time.Now()
	return output, nil
}

// enumSubdomains performs subdomain enumeration.
func (p *ReconPhase) enumSubdomains(ctx context.Context, domain string) ([]map[string]interface{}, error) {
	result, err := p.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
		Capability: "subdomain_enum_passive",
		Target:     domain,
		Args:       []string{"-d", domain, "-silent"},
	})
	if err != nil {
		return nil, err
	}

	subdomains := make([]map[string]interface{}, 0)
	for _, line := range strings.Split(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		subdomains = append(subdomains, map[string]interface{}{
			"domain": line,
		})
	}

	return subdomains, nil
}

// probeHTTP performs HTTP probing.
func (p *ReconPhase) probeHTTP(ctx context.Context, subdomains []map[string]interface{}) ([]map[string]interface{}, error) {
	// Build target list
	targets := make([]string, len(subdomains))
	for i, sub := range subdomains {
		targets[i] = sub["domain"].(string)
	}

	result, err := p.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
		Capability: "http_probe",
		Target:     strings.Join(targets, ","),
		Args:       []string{"-silent", "-title", "-status-code", "-tech-detect"},
	})
	if err != nil {
		return nil, err
	}

	aliveHosts := make([]map[string]interface{}, 0)
	for _, line := range strings.Split(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse httpx output
		aliveHosts = append(aliveHosts, map[string]interface{}{
			"url":    line,
			"status": 200, // Simplified
		})
	}

	return aliveHosts, nil
}

// scanPorts performs port scanning.
func (p *ReconPhase) scanPorts(ctx context.Context, hosts []string) ([]map[string]interface{}, error) {
	result, err := p.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
		Capability: "port_scan",
		Target:     strings.Join(hosts, ","),
		Args:       []string{"-sV", "-T4", "-F", "--open", strings.Join(hosts, ",")},
	})
	if err != nil {
		return nil, err
	}

	// Parse nmap output (simplified)
	openPorts := make([]map[string]interface{}, 0)
	openPorts = append(openPorts, map[string]interface{}{
		"raw_output": result.Output,
	})

	return openPorts, nil
}

// DiscoveryPhase implements the vulnerability discovery phase.
type DiscoveryPhase struct {
	scheduler *scheduler.Scheduler
}

// NewDiscoveryPhase creates a new discovery phase.
func NewDiscoveryPhase(s *scheduler.Scheduler) *DiscoveryPhase {
	return &DiscoveryPhase{scheduler: s}
}

// ID returns the phase ID.
func (p *DiscoveryPhase) ID() string {
	return "discovery"
}

// Name returns the phase name.
func (p *DiscoveryPhase) Name() string {
	return "漏洞发现"
}

// Description returns the phase description.
func (p *DiscoveryPhase) Description() string {
	return "使用各种工具发现潜在漏洞"
}

// CanSkip returns whether this phase can be skipped.
func (p *DiscoveryPhase) CanSkip() bool {
	return true
}

// Dependencies returns the phase dependencies.
func (p *DiscoveryPhase) Dependencies() []string {
	return []string{"recon"}
}

// Execute executes the discovery phase.
func (p *DiscoveryPhase) Execute(ctx context.Context, input *orchestrator.PhaseInput) (*orchestrator.PhaseOutput, error) {
	output := &orchestrator.PhaseOutput{
		PhaseID:   p.ID(),
		PhaseName: p.Name(),
		StartTime: time.Now(),
		Findings:  make(map[string]interface{}),
	}

	// Get alive hosts from recon phase
	reconResult, exists := input.PrevResults["recon"]
	if !exists {
		output.Status = "skipped"
		output.Error = "recon phase not completed"
		output.EndTime = time.Now()
		return output, nil
	}

	aliveHosts, ok := reconResult.Findings["alive_hosts"].([]map[string]interface{})
	if !ok || len(aliveHosts) == 0 {
		output.Status = "skipped"
		output.Error = "no alive hosts found"
		output.EndTime = time.Now()
		return output, nil
	}

	// Run nuclei scan
	vulns := make([]map[string]interface{}, 0)
	for _, host := range aliveHosts {
		url, ok := host["url"].(string)
		if !ok {
			continue
		}

		result, err := p.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
			Capability: "vulnerability_scan",
			Target:     url,
			Args:       []string{"-u", url, "-silent", "-severity", "critical,high,medium"},
		})
		if err != nil {
			output.Findings["nuclei_error"] = err.Error()
			continue
		}

		if result.Output != "" {
			vulns = append(vulns, map[string]interface{}{
				"url":      url,
				"findings": result.Output,
			})
		}
	}

	output.Findings["vulnerabilities"] = vulns
	output.Status = "completed"
	output.EndTime = time.Now()
	return output, nil
}

// VerificationPhase implements the vulnerability verification phase.
type VerificationPhase struct {
	scheduler *scheduler.Scheduler
}

// NewVerificationPhase creates a new verification phase.
func NewVerificationPhase(s *scheduler.Scheduler) *VerificationPhase {
	return &VerificationPhase{scheduler: s}
}

// ID returns the phase ID.
func (p *VerificationPhase) ID() string {
	return "verification"
}

// Name returns the phase name.
func (p *VerificationPhase) Name() string {
	return "漏洞验证"
}

// Description returns the phase description.
func (p *VerificationPhase) Description() string {
	return "验证发现的漏洞是否真实存在"
}

// CanSkip returns whether this phase can be skipped.
func (p *VerificationPhase) CanSkip() bool {
	return true
}

// Dependencies returns the phase dependencies.
func (p *VerificationPhase) Dependencies() []string {
	return []string{"discovery"}
}

// Execute executes the verification phase.
func (p *VerificationPhase) Execute(ctx context.Context, input *orchestrator.PhaseInput) (*orchestrator.PhaseOutput, error) {
	output := &orchestrator.PhaseOutput{
		PhaseID:   p.ID(),
		PhaseName: p.Name(),
		StartTime: time.Now(),
		Findings:  make(map[string]interface{}),
	}

	// Get vulnerabilities from discovery phase
	discoveryResult, exists := input.PrevResults["discovery"]
	if !exists {
		output.Status = "skipped"
		output.Error = "discovery phase not completed"
		output.EndTime = time.Now()
		return output, nil
	}

	vulns, ok := discoveryResult.Findings["vulnerabilities"].([]map[string]interface{})
	if !ok || len(vulns) == 0 {
		output.Status = "skipped"
		output.Error = "no vulnerabilities found"
		output.EndTime = time.Now()
		return output, nil
	}

	// Verify SQL injection vulnerabilities
	verifiedVulns := make([]map[string]interface{}, 0)
	for _, vuln := range vulns {
		// Check if it's a potential SQL injection
		if strings.Contains(fmt.Sprintf("%v", vuln["findings"]), "sqli") {
			url := vuln["url"].(string)
			result, err := p.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
				Capability: "sql_injection",
				Target:     url,
				Args:       []string{"-u", url, "--batch", "--level=1", "--risk=1"},
			})
			if err == nil && strings.Contains(result.Output, "Parameter") {
				verifiedVulns = append(verifiedVulns, map[string]interface{}{
					"url":      url,
					"type":     "sql_injection",
					"verified": true,
					"details":  result.Output,
				})
			}
		}
	}

	output.Findings["verified_vulnerabilities"] = verifiedVulns
	output.Status = "completed"
	output.EndTime = time.Now()
	return output, nil
}

// ReportPhase implements the report generation phase.
type ReportPhase struct {
	scheduler *scheduler.Scheduler
}

// NewReportPhase creates a new report phase.
func NewReportPhase(s *scheduler.Scheduler) *ReportPhase {
	return &ReportPhase{scheduler: s}
}

// ID returns the phase ID.
func (p *ReportPhase) ID() string {
	return "report"
}

// Name returns the phase name.
func (p *ReportPhase) Name() string {
	return "报告生成"
}

// Description returns the phase description.
func (p *ReportPhase) Description() string {
	return "生成渗透测试报告"
}

// CanSkip returns whether this phase can be skipped.
func (p *ReportPhase) CanSkip() bool {
	return false
}

// Dependencies returns the phase dependencies.
func (p *ReportPhase) Dependencies() []string {
	return []string{"verification"}
}

// Execute executes the report phase.
func (p *ReportPhase) Execute(ctx context.Context, input *orchestrator.PhaseInput) (*orchestrator.PhaseOutput, error) {
	output := &orchestrator.PhaseOutput{
		PhaseID:   p.ID(),
		PhaseName: p.Name(),
		StartTime: time.Now(),
		Findings:  make(map[string]interface{}),
	}

	// Collect all findings
	allFindings := make(map[string]interface{})

	for phaseID, result := range input.PrevResults {
		allFindings[phaseID] = result.Findings
	}

	output.Findings["all_findings"] = allFindings
	output.Findings["report_generated"] = true
	output.Status = "completed"
	output.EndTime = time.Now()
	return output, nil
}
