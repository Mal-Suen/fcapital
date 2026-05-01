// Package merger provides result merging and analysis for hybrid mode execution.
package merger

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ResultType represents the type of result.
type ResultType string

const (
	ResultTypeTool   ResultType = "tool"
	ResultTypeScript ResultType = "script"
	ResultTypeAI     ResultType = "ai"
)

// SourceResult represents a result from a single source.
type SourceResult struct {
	Type      ResultType        `json:"type"`
	Source    string            `json:"source"`
	Success   bool              `json:"success"`
	Output    string            `json:"output"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
	Findings  []Finding         `json:"findings,omitempty"`
}

// Finding represents a security finding.
type Finding struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Evidence    string   `json:"evidence,omitempty"`
	References  []string `json:"references,omitempty"`
	Source      string   `json:"source"`
}

// MergedResult represents a merged result from multiple sources.
type MergedResult struct {
	Success        bool               `json:"success"`
	Summary        string             `json:"summary"`
	AllFindings    []Finding          `json:"all_findings"`
	UniqueFindings []Finding          `json:"unique_findings"`
	Duplicates     []DuplicateFinding `json:"duplicates,omitempty"`
	Statistics     Statistics         `json:"statistics"`
	Sources        []string           `json:"sources"`
	Timestamp      time.Time          `json:"timestamp"`
	TotalDuration  time.Duration      `json:"total_duration"`
	RawOutputs     map[string]string  `json:"raw_outputs,omitempty"`
}

// DuplicateFinding represents a duplicate finding.
type DuplicateFinding struct {
	Finding Finding  `json:"finding"`
	Sources []string `json:"sources"`
}

// Statistics represents execution statistics.
type Statistics struct {
	TotalSources    int            `json:"total_sources"`
	SuccessfulCount int            `json:"successful_count"`
	FailedCount     int            `json:"failed_count"`
	TotalFindings   int            `json:"total_findings"`
	UniqueFindings  int            `json:"unique_findings"`
	BySeverity      map[string]int `json:"by_severity"`
	BySource        map[string]int `json:"by_source"`
}

// Merger merges results from multiple sources.
type Merger struct {
	dedupStrategy DeduplicationStrategy
	normalizers   map[ResultType]Normalizer
}

// DeduplicationStrategy defines how to handle duplicate findings.
type DeduplicationStrategy int

const (
	// DedupKeepFirst keeps the first occurrence.
	DedupKeepFirst DeduplicationStrategy = iota
	// DedupKeepHighestSeverity keeps the finding with highest severity.
	DedupKeepHighestSeverity
	// DedupMerge merges duplicate findings.
	DedupMerge
)

// Normalizer normalizes results from different sources.
type Normalizer interface {
	Normalize(output string) ([]Finding, error)
}

// MergerOption is a functional option for Merger.
type MergerOption func(*Merger)

// WithDedupStrategy sets the deduplication strategy.
func WithDedupStrategy(strategy DeduplicationStrategy) MergerOption {
	return func(m *Merger) {
		m.dedupStrategy = strategy
	}
}

// NewMerger creates a new result merger.
func NewMerger(opts ...MergerOption) *Merger {
	m := &Merger{
		dedupStrategy: DedupKeepHighestSeverity,
		normalizers: map[ResultType]Normalizer{
			ResultTypeTool:   &ToolNormalizer{},
			ResultTypeScript: &ScriptNormalizer{},
			ResultTypeAI:     &AINormalizer{},
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Merge merges results from multiple sources.
func (m *Merger) Merge(results []SourceResult) *MergedResult {
	merged := &MergedResult{
		Timestamp:  time.Now(),
		RawOutputs: make(map[string]string),
		Statistics: Statistics{
			BySeverity: make(map[string]int),
			BySource:   make(map[string]int),
		},
	}

	// 1. Collect all findings
	allFindings := []Finding{}
	for _, result := range results {
		merged.Sources = append(merged.Sources, result.Source)
		merged.TotalDuration += result.Duration

		if result.Success {
			merged.Statistics.SuccessfulCount++
		} else {
			merged.Statistics.FailedCount++
		}

		merged.RawOutputs[result.Source] = result.Output

		// Normalize and extract findings
		if normalizer, ok := m.normalizers[result.Type]; ok {
			findings, err := normalizer.Normalize(result.Output)
			if err == nil {
				for i := range findings {
					findings[i].Source = result.Source
				}
				allFindings = append(allFindings, findings...)
			}
		}

		// Add explicit findings
		allFindings = append(allFindings, result.Findings...)
	}

	merged.AllFindings = allFindings
	merged.Statistics.TotalSources = len(results)
	merged.Statistics.TotalFindings = len(allFindings)

	// 2. Deduplicate findings
	merged.UniqueFindings, merged.Duplicates = m.deduplicate(allFindings)
	merged.Statistics.UniqueFindings = len(merged.UniqueFindings)

	// 3. Calculate statistics
	for _, f := range merged.UniqueFindings {
		merged.Statistics.BySeverity[f.Severity]++
		merged.Statistics.BySource[f.Source]++
	}

	// 4. Generate summary
	merged.Summary = m.generateSummary(merged)
	merged.Success = merged.Statistics.FailedCount == 0

	return merged
}

// deduplicate removes duplicate findings based on strategy.
func (m *Merger) deduplicate(findings []Finding) ([]Finding, []DuplicateFinding) {
	unique := []Finding{}
	duplicates := []DuplicateFinding{}
	seen := make(map[string][]Finding)

	// Group by title/ID
	for _, f := range findings {
		key := m.getFindingKey(f)
		seen[key] = append(seen[key], f)
	}

	// Process each group
	for _, group := range seen {
		if len(group) == 1 {
			unique = append(unique, group[0])
		} else {
			// Apply deduplication strategy
			selected := m.selectFinding(group)
			unique = append(unique, selected)

			sources := []string{}
			for _, f := range group {
				sources = append(sources, f.Source)
			}
			duplicates = append(duplicates, DuplicateFinding{
				Finding: selected,
				Sources: sources,
			})
		}
	}

	return unique, duplicates
}

// getFindingKey generates a key for finding deduplication.
func (m *Merger) getFindingKey(f Finding) string {
	// Use ID if available
	if f.ID != "" {
		return f.ID
	}
	// Use title
	return strings.ToLower(f.Title)
}

// selectFinding selects a finding from duplicates based on strategy.
func (m *Merger) selectFinding(findings []Finding) Finding {
	if len(findings) == 0 {
		return Finding{}
	}

	switch m.dedupStrategy {
	case DedupKeepFirst:
		return findings[0]

	case DedupKeepHighestSeverity:
		severityOrder := map[string]int{
			"critical": 4,
			"high":     3,
			"medium":   2,
			"low":      1,
			"info":     0,
		}
		selected := findings[0]
		for _, f := range findings {
			if severityOrder[f.Severity] > severityOrder[selected.Severity] {
				selected = f
			}
		}
		return selected

	case DedupMerge:
		// Merge all findings
		merged := findings[0]
		for _, f := range findings[1:] {
			if f.Description != "" && merged.Description == "" {
				merged.Description = f.Description
			}
			if f.Evidence != "" && merged.Evidence == "" {
				merged.Evidence = f.Evidence
			}
			merged.References = append(merged.References, f.References...)
		}
		return merged

	default:
		return findings[0]
	}
}

// generateSummary generates a summary of the merged results.
func (m *Merger) generateSummary(merged *MergedResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("执行完成: %d 个数据源, %d 成功, %d 失败\n",
		merged.Statistics.TotalSources,
		merged.Statistics.SuccessfulCount,
		merged.Statistics.FailedCount))

	sb.WriteString(fmt.Sprintf("发现: %d 个结果 (%d 个唯一)\n",
		merged.Statistics.TotalFindings,
		merged.Statistics.UniqueFindings))

	if len(merged.Statistics.BySeverity) > 0 {
		sb.WriteString("严重程度分布:\n")
		for sev, count := range merged.Statistics.BySeverity {
			sb.WriteString(fmt.Sprintf("  - %s: %d\n", sev, count))
		}
	}

	return sb.String()
}

// AddNormalizer adds a custom normalizer for a result type.
func (m *Merger) AddNormalizer(resultType ResultType, normalizer Normalizer) {
	m.normalizers[resultType] = normalizer
}

// ToolNormalizer normalizes tool output.
type ToolNormalizer struct{}

// Normalize extracts findings from tool output.
func (n *ToolNormalizer) Normalize(output string) ([]Finding, error) {
	findings := []Finding{}

	// Try to parse as JSON first
	var jsonData []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		for _, item := range jsonData {
			finding := Finding{}
			if id, ok := item["id"].(string); ok {
				finding.ID = id
			}
			if name, ok := item["name"].(string); ok {
				finding.Title = name
			}
			if severity, ok := item["severity"].(string); ok {
				finding.Severity = severity
			}
			if desc, ok := item["description"].(string); ok {
				finding.Description = desc
			}
			findings = append(findings, finding)
		}
		return findings, nil
	}

	// Parse text output
	// Look for common vulnerability patterns
	patterns := []struct {
		regex    string
		severity string
	}{
		{`(?i)(SQL\s*injection|sqli)`, "high"},
		{`(?i)(XSS|cross-site scripting)`, "medium"},
		{`(?i)(RCE|remote code execution)`, "critical"},
		{`(?i)(LFI|local file inclusion)`, "high"},
		{`(?i)(RFI|remote file inclusion)`, "high"},
		{`(?i)(SSRF|server-side request forgery)`, "high"},
		{`(?i)(Open\s*Redirect)`, "low"},
		{`(?i)(Information\s*Disclosure)`, "low"},
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p.regex)
		matches := re.FindAllString(output, -1)
		for _, match := range matches {
			findings = append(findings, Finding{
				Title:    match,
				Severity: p.severity,
				Evidence: match,
			})
		}
	}

	return findings, nil
}

// ScriptNormalizer normalizes script output.
type ScriptNormalizer struct{}

// Normalize extracts findings from script output.
func (n *ScriptNormalizer) Normalize(output string) ([]Finding, error) {
	findings := []Finding{}

	// Parse JSON output from scripts
	var jsonData struct {
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		return jsonData.Findings, nil
	}

	// Look for FINDING: markers
	re := regexp.MustCompile(`(?i)FINDING:\s*(.+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) > 1 {
			findings = append(findings, Finding{
				Title:    strings.TrimSpace(match[1]),
				Severity: "medium",
			})
		}
	}

	return findings, nil
}

// AINormalizer normalizes AI output.
type AINormalizer struct{}

// Normalize extracts findings from AI output.
func (n *AINormalizer) Normalize(output string) ([]Finding, error) {
	findings := []Finding{}

	// Try to parse as JSON
	var jsonData struct {
		Findings        []Finding `json:"findings"`
		Vulnerabilities []Finding `json:"vulnerabilities"`
	}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		findings = append(findings, jsonData.Findings...)
		findings = append(findings, jsonData.Vulnerabilities...)
		return findings, nil
	}

	return findings, nil
}
