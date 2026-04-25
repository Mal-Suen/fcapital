package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mal-Suen/fcapital/internal/core/ai/providers"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
)

// getAPIKey 获取指定提供者的 API 密钥
func getAPIKey(provider string) string {
	providerUpper := strings.ToUpper(provider)

	// 尝试多种环境变量格式
	keys := []string{
		fmt.Sprintf("%s_API_KEY", providerUpper),
		fmt.Sprintf("%s_APIKEY", providerUpper),
		"OPENAI_API_KEY",
		"AI_API_KEY",
	}

	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}

	return ""
}

// createProvider 创建 AI 提供者
func createProvider(providerName, apiKey, model string) providers.Provider {
	providerUpper := strings.ToUpper(providerName)

	// 获取 base URL
	baseURL := os.Getenv(fmt.Sprintf("%s_BASE_URL", providerUpper))
	if baseURL == "" {
		baseURL = os.Getenv(fmt.Sprintf("%s_BASEURL", providerUpper))
	}

	// 获取模型
	if model == "" {
		model = os.Getenv("AI_MODEL")
	}

	switch strings.ToLower(providerName) {
	case "openai":
		if apiKey == "" {
			return nil
		}
		return providers.NewOpenAIProvider(apiKey, model, baseURL)
	case "deepseek":
		if apiKey == "" {
			return nil
		}
		if baseURL == "" {
			baseURL = "https://api.deepseek.com/v1"
		}
		return providers.NewDeepSeekProvider(apiKey, model, baseURL)
	case "ollama":
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return providers.NewOllamaProvider(baseURL, model)
	default:
		// 默认使用 OpenAI 兼容接口
		if apiKey == "" {
			return nil
		}
		return providers.NewOpenAIProvider(apiKey, model, baseURL)
	}
}

// registerTools 注册工具到调度器
func registerTools(sched *scheduler.Scheduler) {
	// 端口扫描工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "nmap",
		Description:  "Network port scanner",
		Category:     "portscan",
		Capabilities: []string{"port_scan", "service_detection", "os_detection"},
		VerifyCmd:    "nmap --version",
	})

	// HTTP 探测工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "httpx",
		Description:  "HTTP probe tool",
		Category:     "recon",
		Capabilities: []string{"http_probe", "tech_detection", "screenshot"},
		VerifyCmd:    "httpx -version",
	})

	// DNS 工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "dnsx",
		Description:  "DNS query tool",
		Category:     "recon",
		Capabilities: []string{"dns_query", "dns_bruteforce"},
		VerifyCmd:    "dnsx -version",
	})

	// 子域名枚举工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "subfinder",
		Description:  "Subdomain discovery tool",
		Category:     "recon",
		Capabilities: []string{"subdomain_enum"},
		VerifyCmd:    "subfinder -version",
	})

	// 漏洞扫描工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "nuclei",
		Description:  "Vulnerability scanner",
		Category:     "vulnscan",
		Capabilities: []string{"vuln_scan", "cve_detection"},
		VerifyCmd:    "nuclei -version",
	})

	// 目录扫描工具
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "gobuster",
		Description:  "Directory brute force tool",
		Category:     "webscan",
		Capabilities: []string{"dir_bruteforce", "dns_bruteforce"},
		VerifyCmd:    "gobuster version",
	})

	// Web 应用扫描
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "wpscan",
		Description:  "WordPress scanner",
		Category:     "webscan",
		Capabilities: []string{"wordpress_scan"},
		VerifyCmd:    "wpscan --version",
	})

	// SQL 注入检测
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "sqlmap",
		Description:  "SQL injection tool",
		Category:     "vulnscan",
		Capabilities: []string{"sql_injection"},
		VerifyCmd:    "sqlmap --version",
	})

	// 指纹识别
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "wafw00f",
		Description:  "WAF detection tool",
		Category:     "recon",
		Capabilities: []string{"waf_detection"},
		VerifyCmd:    "wafw00f --version",
	})

	// 端口扫描 (masscan)
	sched.RegisterTool(&scheduler.ToolDefinition{
		Name:         "masscan",
		Description:  "Fast port scanner",
		Category:     "portscan",
		Capabilities: []string{"fast_port_scan"},
		VerifyCmd:    "masscan --version",
	})
}
