// Package script provides code auditing for AI-generated scripts.
package script

import (
	"fmt"
	"regexp"
	"strings"
)

// Auditor audits generated scripts for security concerns.
type Auditor struct {
	patterns []DangerousPattern
	rules    []AuditRule
}

// DangerousPattern represents a dangerous code pattern.
type DangerousPattern struct {
	Name        string
	Pattern     string
	Severity    string // critical, high, medium, low
	Description string
	Category    string
}

// AuditRule represents an audit rule.
type AuditRule struct {
	ID          string
	Name        string
	Description string
	Check       func(code string) (bool, string)
}

// AuditResult represents the result of a code audit.
type AuditResult struct {
	Score      int      `json:"score"`
	Warnings   []string `json:"warnings"`
	Findings   []Finding `json:"findings"`
	Passed     bool     `json:"passed"`
	Recommendations []string `json:"recommendations"`
}

// Finding represents a specific audit finding.
type Finding struct {
	Line       int    `json:"line"`
	Pattern    string `json:"pattern"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// NewAuditor creates a new code auditor with default patterns.
func NewAuditor() *Auditor {
	a := &Auditor{
		patterns: defaultDangerousPatterns,
		rules:    defaultAuditRules,
	}
	return a
}

// Audit performs a security audit on the given code.
func (a *Auditor) Audit(code string) *AuditResult {
	result := &AuditResult{
		Score:      100,
		Warnings:   []string{},
		Findings:   []Finding{},
		Passed:     true,
		Recommendations: []string{},
	}

	// Check dangerous patterns
	for _, pattern := range a.patterns {
		matches := a.findPattern(code, pattern.Pattern)
		for _, match := range matches {
			severity := pattern.Severity
			message := pattern.Description
			if message == "" {
				message = pattern.Name
			}

			result.Findings = append(result.Findings, Finding{
				Line:     match.Line,
				Pattern:  pattern.Name,
				Severity: severity,
				Message:  message,
			})

			result.Warnings = append(result.Warnings,
				formatWarning(severity, pattern.Name, message))

			result.Score -= getScorePenalty(severity)
		}
	}

	// Run audit rules
	for _, rule := range a.rules {
		if passed, msg := rule.Check(code); !passed {
			result.Findings = append(result.Findings, Finding{
				Severity:   "medium",
				Pattern:    rule.Name,
				Message:    msg,
			})
			result.Warnings = append(result.Warnings,
				formatWarning("medium", rule.Name, msg))
			result.Score -= 10
		}
	}

	// Ensure score doesn't go below 0
	if result.Score < 0 {
		result.Score = 0
	}

	// Determine if passed
	result.Passed = result.Score >= 50 && !a.hasCriticalFindings(result.Findings)

	// Add recommendations
	result.Recommendations = a.generateRecommendations(result.Findings)

	return result
}

// PatternMatch represents a pattern match result.
type PatternMatch struct {
	Line    int
	Content string
}

// findPattern finds all matches of a pattern in the code.
func (a *Auditor) findPattern(code, pattern string) []PatternMatch {
	matches := []PatternMatch{}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return matches
	}

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, PatternMatch{
				Line:    i + 1,
				Content: strings.TrimSpace(line),
			})
		}
	}

	return matches
}

// hasCriticalFindings checks if there are any critical findings.
func (a *Auditor) hasCriticalFindings(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == "critical" {
			return true
		}
	}
	return false
}

// generateRecommendations generates recommendations based on findings.
func (a *Auditor) generateRecommendations(findings []Finding) []string {
	recommendations := []string{}
	seen := make(map[string]bool)

	for _, f := range findings {
		rec := getRecommendation(f.Pattern)
		if rec != "" && !seen[f.Pattern] {
			recommendations = append(recommendations, rec)
			seen[f.Pattern] = true
		}
	}

	return recommendations
}

// AddPattern adds a custom dangerous pattern.
func (a *Auditor) AddPattern(pattern DangerousPattern) {
	a.patterns = append(a.patterns, pattern)
}

// AddRule adds a custom audit rule.
func (a *Auditor) AddRule(rule AuditRule) {
	a.rules = append(a.rules, rule)
}

// Helper functions

func formatWarning(severity, name, message string) string {
	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(severity), name, message)
}

func getScorePenalty(severity string) int {
	switch severity {
	case "critical":
		return 40
	case "high":
		return 25
	case "medium":
		return 15
	case "low":
		return 5
	default:
		return 10
	}
}

func getRecommendation(pattern string) string {
	recommendations := map[string]string{
		"file_deletion":       "避免使用文件删除操作，或使用临时目录",
		"system_modification": "避免修改系统配置，使用用户级配置替代",
		"network_listen":      "避免监听网络端口，使用出站连接替代",
		"privilege_escalation": "避免提权操作，在脚本文档中说明所需权限",
		"credential_exposure": "不要在代码中硬编码凭据，使用环境变量或配置文件",
		"remote_code_exec":    "避免远程代码执行，使用白名单验证输入",
		"infinite_loop":       "添加循环计数器或超时机制",
		"resource_exhaustion": "添加资源使用限制和清理逻辑",
	}
	return recommendations[pattern]
}

// Default dangerous patterns
var defaultDangerousPatterns = []DangerousPattern{
	// Critical patterns
	{
		Name:        "file_deletion",
		Pattern:     `(rm\s+-rf|os\.remove|shutil\.rmtree|del\s+/s|Remove-Item.*-Recurse)`,
		Severity:    "critical",
		Description: "检测到文件删除操作",
		Category:    "filesystem",
	},
	{
		Name:        "system_modification",
		Pattern:     `(reg\s+add|net\s+user|chmod\s+777|setuid|Set-ExecutionPolicy)`,
		Severity:    "critical",
		Description: "检测到系统修改操作",
		Category:    "system",
	},
	{
		Name:        "privilege_escalation",
		Pattern:     `(sudo\s+|runas\s+|Invoke-Expression.*-Verb\s+RunAs)`,
		Severity:    "critical",
		Description: "检测到提权操作",
		Category:    "privilege",
	},

	// High severity patterns
	{
		Name:        "network_listen",
		Pattern:     `(socket\.listen|nc\s+-l|netcat\s+-l|\.Listen\()`,
		Severity:    "high",
		Description: "检测到网络监听操作",
		Category:    "network",
	},
	{
		Name:        "credential_exposure",
		Pattern:     `(password\s*=\s*['"]|api_key\s*=\s*['"]|secret\s*=\s*['"])`,
		Severity:    "high",
		Description: "检测到可能的凭据暴露",
		Category:    "security",
	},
	{
		Name:        "remote_code_exec",
		Pattern:     `(eval\s*\(|exec\s*\(|subprocess\.call.*shell=True|Invoke-Expression)`,
		Severity:    "high",
		Description: "检测到远程代码执行风险",
		Category:    "security",
	},

	// Medium severity patterns
	{
		Name:        "infinite_loop",
		Pattern:     `(while\s+True:|for\s*\(\s*;;\s*\)|while\s*\(\s*1\s*\))`,
		Severity:    "medium",
		Description: "检测到可能的无限循环",
		Category:    "logic",
	},
	{
		Name:        "resource_exhaustion",
		Pattern:     `(fork\(\)|multiprocessing\.Process|Start-Thread)`,
		Severity:    "medium",
		Description: "检测到可能的资源耗尽风险",
		Category:    "resource",
	},
	{
		Name:        "unsafe_deserialization",
		Pattern:     `(pickle\.loads|yaml\.load\(|Marshal\.Load)`,
		Severity:    "medium",
		Description: "检测到不安全的反序列化",
		Category:    "security",
	},

	// Low severity patterns
	{
		Name:        "debug_code",
		Pattern:     `(print\s*\(|console\.log|Write-Host|echo\s+)`,
		Severity:    "low",
		Description: "检测到调试输出代码",
		Category:    "style",
	},
	{
		Name:        "hardcoded_path",
		Pattern:     `(/etc/|/usr/|C:\\Windows|C:\\Program Files)`,
		Severity:    "low",
		Description: "检测到硬编码路径",
		Category:    "portability",
	},
}

// Default audit rules
var defaultAuditRules = []AuditRule{
	{
		ID:          "R001",
		Name:        "error_handling",
		Description: "Check for proper error handling",
		Check: func(code string) (bool, string) {
			// Check if code has try-catch or error handling
			if strings.Contains(code, "try") && strings.Contains(code, "except") {
				return true, ""
			}
			if strings.Contains(code, "try") && strings.Contains(code, "catch") {
				return true, ""
			}
			if strings.Contains(code, "try") && strings.Contains(code, "finally") {
				return true, ""
			}
			return false, "缺少错误处理机制"
		},
	},
	{
		ID:          "R002",
		Name:        "input_validation",
		Description: "Check for input validation",
		Check: func(code string) (bool, string) {
			// Simple check for input validation patterns
			validationPatterns := []string{
				"if", "validate", "check", "sanitize", "escape",
				"filter", "whitelist", "blacklist",
			}
			for _, p := range validationPatterns {
				if strings.Contains(strings.ToLower(code), p) {
					return true, ""
				}
			}
			return false, "缺少输入验证"
		},
	},
	{
		ID:          "R003",
		Name:        "timeout_mechanism",
		Description: "Check for timeout mechanism in network operations",
		Check: func(code string) (bool, string) {
			// Check if code has timeout for network operations
			if strings.Contains(code, "timeout") || strings.Contains(code, "Timeout") {
				return true, ""
			}
			// If no network operations, pass
			if !strings.Contains(code, "http") && !strings.Contains(code, "socket") && !strings.Contains(code, "request") {
				return true, ""
			}
			return false, "网络操作缺少超时机制"
		},
	},
}
