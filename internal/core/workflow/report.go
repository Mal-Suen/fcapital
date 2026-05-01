package workflow

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportGenerator 报告生成器
type ReportGenerator struct {
	templateDir string
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// GenerateHTML 生成HTML报告
func (g *ReportGenerator) GenerateHTML(result *WorkflowResult, outputPath string) error {
	tmpl := g.getHTMLTemplate()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	data := g.prepareTemplateData(result)

	return tmpl.Execute(f, data)
}

// GenerateJSON 生成JSON报告
func (g *ReportGenerator) GenerateJSON(result *WorkflowResult, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// GenerateMarkdown 生成Markdown报告
func (g *ReportGenerator) GenerateMarkdown(result *WorkflowResult, outputPath string) error {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# fcapital 扫描报告\n\n"))
	md.WriteString(fmt.Sprintf("**目标**: %s\n\n", result.Target))
	md.WriteString(fmt.Sprintf("**工作流**: %s\n\n", result.WorkflowName))
	md.WriteString(fmt.Sprintf("**开始时间**: %s\n\n", result.StartTime.Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**结束时间**: %s\n\n", result.EndTime.Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**持续时间**: %s\n\n", result.Duration))
	md.WriteString(fmt.Sprintf("**状态**: %s\n\n", g.getStatusEmoji(result.Status)))

	// 摘要
	if result.Summary != nil {
		md.WriteString("## 扫描摘要\n\n")
		md.WriteString(fmt.Sprintf("| 指标 | 数量 |\n"))
		md.WriteString(fmt.Sprintf("|------|------|\n"))
		md.WriteString(fmt.Sprintf("| 子域名 | %d |\n", result.Summary.Subdomains))
		md.WriteString(fmt.Sprintf("| 存活主机 | %d |\n", result.Summary.AliveHosts))
		md.WriteString(fmt.Sprintf("| 开放端口 | %d |\n", result.Summary.OpenPorts))
		md.WriteString(fmt.Sprintf("| 发现目录 | %d |\n", result.Summary.Directories))
		md.WriteString(fmt.Sprintf("| 发现漏洞 | %d |\n", result.Summary.Vulnerabilities))
		md.WriteString("\n")
	}

	// 步骤详情
	md.WriteString("## 扫描步骤\n\n")
	for stepID, stepResult := range result.Steps {
		md.WriteString(fmt.Sprintf("### %s\n\n", stepID))
		md.WriteString(fmt.Sprintf("- **状态**: %s\n", g.getStatusEmoji(stepResult.Status)))
		md.WriteString(fmt.Sprintf("- **持续时间**: %s\n", stepResult.Duration))
		if stepResult.Error != "" {
			md.WriteString(fmt.Sprintf("- **错误**: %s\n", stepResult.Error))
		}
		md.WriteString("\n")
	}

	// 严重漏洞
	if result.Summary != nil && len(result.Summary.CriticalVulns) > 0 {
		md.WriteString("## ⚠️ 严重漏洞\n\n")
		for _, vuln := range result.Summary.CriticalVulns {
			md.WriteString(fmt.Sprintf("### %s\n\n", vuln.Name))
			md.WriteString(fmt.Sprintf("- **严重程度**: %s\n", vuln.Severity))
			md.WriteString(fmt.Sprintf("- **主机**: %s\n", vuln.Host))
			if vuln.Description != "" {
				md.WriteString(fmt.Sprintf("- **描述**: %s\n", vuln.Description))
			}
			md.WriteString("\n")
		}
	}

	return os.WriteFile(outputPath, []byte(md.String()), 0644)
}

// TemplateData 模板数据
type TemplateData struct {
	Target        string
	WorkflowName  string
	StartTime     string
	EndTime       string
	Duration      string
	Status        string
	StatusClass   string
	Summary       *ScanSummary
	Steps         map[string]*StepResult
	CriticalVulns []VulnInfo
	GeneratedAt   string
}

func (g *ReportGenerator) prepareTemplateData(result *WorkflowResult) *TemplateData {
	statusClass := "success"
	if result.Status == "partial" {
		statusClass = "warning"
	} else if result.Status == "failed" {
		statusClass = "danger"
	}

	var criticalVulns []VulnInfo
	if result.Summary != nil {
		criticalVulns = result.Summary.CriticalVulns
	}

	return &TemplateData{
		Target:        result.Target,
		WorkflowName:  result.WorkflowName,
		StartTime:     result.StartTime.Format("2006-01-02 15:04:05"),
		EndTime:       result.EndTime.Format("2006-01-02 15:04:05"),
		Duration:      result.Duration,
		Status:        strings.ToUpper(result.Status),
		StatusClass:   statusClass,
		Summary:       result.Summary,
		Steps:         result.Steps,
		CriticalVulns: criticalVulns,
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (g *ReportGenerator) getStatusEmoji(status string) string {
	switch status {
	case "success":
		return "✅ 成功"
	case "running":
		return "🔄 运行中"
	case "partial":
		return "⚠️ 部分成功"
	case "failed":
		return "❌ 失败"
	case "skipped":
		return "⏭️ 跳过"
	default:
		return status
	}
}

func (g *ReportGenerator) getHTMLTemplate() *template.Template {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>fcapital 扫描报告 - {{.Target}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #eee;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            padding: 40px 0;
            border-bottom: 1px solid #333;
            margin-bottom: 30px;
        }
        .header h1 {
            font-size: 2.5em;
            background: linear-gradient(90deg, #00d4ff, #7c3aed);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }
        .header .subtitle {
            color: #888;
            font-size: 1.1em;
        }
        .status-badge {
            display: inline-block;
            padding: 8px 20px;
            border-radius: 20px;
            font-weight: bold;
            margin-top: 15px;
        }
        .status-success { background: #10b981; }
        .status-warning { background: #f59e0b; }
        .status-danger { background: #ef4444; }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .info-card {
            background: rgba(255,255,255,0.05);
            border-radius: 10px;
            padding: 20px;
            text-align: center;
        }
        .info-card .value {
            font-size: 2.5em;
            font-weight: bold;
            color: #00d4ff;
        }
        .info-card .label {
            color: #888;
            margin-top: 5px;
        }
        .section {
            background: rgba(255,255,255,0.05);
            border-radius: 10px;
            padding: 25px;
            margin-bottom: 20px;
        }
        .section h2 {
            color: #00d4ff;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid #333;
        }
        .step-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            background: rgba(255,255,255,0.03);
            border-radius: 8px;
            margin-bottom: 10px;
        }
        .step-status {
            padding: 5px 15px;
            border-radius: 15px;
            font-size: 0.9em;
        }
        .step-success { background: #10b981; }
        .step-failed { background: #ef4444; }
        .step-skipped { background: #6b7280; }
        .vuln-critical {
            background: rgba(239, 68, 68, 0.2);
            border-left: 4px solid #ef4444;
            padding: 15px;
            margin-bottom: 15px;
            border-radius: 0 8px 8px 0;
        }
        .vuln-high {
            background: rgba(249, 115, 22, 0.2);
            border-left: 4px solid #f97316;
            padding: 15px;
            margin-bottom: 15px;
            border-radius: 0 8px 8px 0;
        }
        .vuln-title {
            font-weight: bold;
            margin-bottom: 5px;
        }
        .vuln-meta {
            color: #888;
            font-size: 0.9em;
        }
        .footer {
            text-align: center;
            padding: 30px;
            color: #666;
            border-top: 1px solid #333;
            margin-top: 30px;
        }
        .target-info {
            background: rgba(0, 212, 255, 0.1);
            padding: 15px 25px;
            border-radius: 10px;
            margin-bottom: 20px;
        }
        .target-info span {
            margin-right: 30px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 fcapital 扫描报告</h1>
            <div class="subtitle">自动化渗透测试结果</div>
            <div class="status-badge status-{{.StatusClass}}">{{.Status}}</div>
        </div>

        <div class="target-info">
            <span><strong>目标:</strong> {{.Target}}</span>
            <span><strong>工作流:</strong> {{.WorkflowName}}</span>
            <span><strong>持续时间:</strong> {{.Duration}}</span>
        </div>

        {{if .Summary}}
        <div class="info-grid">
            <div class="info-card">
                <div class="value">{{.Summary.Subdomains}}</div>
                <div class="label">子域名</div>
            </div>
            <div class="info-card">
                <div class="value">{{.Summary.AliveHosts}}</div>
                <div class="label">存活主机</div>
            </div>
            <div class="info-card">
                <div class="value">{{.Summary.OpenPorts}}</div>
                <div class="label">开放端口</div>
            </div>
            <div class="info-card">
                <div class="value">{{.Summary.Directories}}</div>
                <div class="label">发现目录</div>
            </div>
            <div class="info-card">
                <div class="value">{{.Summary.Vulnerabilities}}</div>
                <div class="label">发现漏洞</div>
            </div>
        </div>
        {{end}}

        {{if .CriticalVulns}}
        <div class="section">
            <h2>⚠️ 严重漏洞</h2>
            {{range .CriticalVulns}}
            <div class="vuln-{{if eq .Severity "critical"}}critical{{else}}high{{end}}">
                <div class="vuln-title">{{.Name}}</div>
                <div class="vuln-meta">
                    <span>严重程度: {{.Severity}}</span> | 
                    <span>主机: {{.Host}}</span>
                </div>
                {{if .Description}}<p style="margin-top:10px;color:#aaa;">{{.Description}}</p>{{end}}
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="section">
            <h2>📋 扫描步骤</h2>
            {{range $id, $step := .Steps}}
            <div class="step-item">
                <div>
                    <strong>{{$id}}</strong>
                    <span style="color:#666;margin-left:10px;">{{$step.Duration}}</span>
                </div>
                <div class="step-status step-{{$step.Status}}">{{$step.Status}}</div>
            </div>
            {{if $step.Error}}
            <div style="color:#ef4444;font-size:0.9em;margin-top:-10px;margin-bottom:10px;padding-left:15px;">
                错误: {{$step.Error}}
            </div>
            {{end}}
            {{end}}
        </div>

        <div class="footer">
            <p>报告生成时间: {{.GeneratedAt}}</p>
            <p>fcapital - 综合渗透测试框架</p>
        </div>
    </div>
</body>
</html>`

	return template.Must(template.New("report").Parse(html))
}

// GenerateAll 生成所有格式报告
func (g *ReportGenerator) GenerateAll(result *WorkflowResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 生成 HTML
	if err := g.GenerateHTML(result, filepath.Join(outputDir, "report.html")); err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}

	// 生成 JSON
	if err := g.GenerateJSON(result, filepath.Join(outputDir, "report.json")); err != nil {
		return fmt.Errorf("failed to generate JSON: %w", err)
	}

	// 生成 Markdown
	if err := g.GenerateMarkdown(result, filepath.Join(outputDir, "report.md")); err != nil {
		return fmt.Errorf("failed to generate Markdown: %w", err)
	}

	return nil
}
