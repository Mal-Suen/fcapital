package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Mal-Suen/fcapital/internal/core/script"
	"github.com/spf13/cobra"
)

// scriptCmd AI生成脚本命令
var scriptCmd = &cobra.Command{
	Use:   "script <task>",
	Short: "AI 生成脚本",
	Long: `使用 AI 生成渗透测试脚本。

示例:
  fcapital script "waf bypass for example.com"
  fcapital script "custom poc for CVE-2024-xxxx" --language python
  fcapital script "encode bypass script" --execute`,
	Args: cobra.MinimumNArgs(1),
	Run:  runScript,
}

var (
	scriptLanguage    string
	scriptExecute     bool
	scriptOutput      string
	scriptProvider    string
	scriptModel       string
	scriptTarget      string
)

func init() {
	scriptCmd.Flags().StringVar(&scriptLanguage, "language", "python", "脚本语言 (python/bash/powershell)")
	scriptCmd.Flags().BoolVar(&scriptExecute, "execute", false, "生成后立即执行")
	scriptCmd.Flags().StringVarP(&scriptOutput, "output", "o", "", "保存脚本到文件")
	scriptCmd.Flags().StringVar(&scriptProvider, "provider", "openai", "AI提供者 (openai/deepseek/ollama)")
	scriptCmd.Flags().StringVar(&scriptModel, "model", "", "AI模型名称")
	scriptCmd.Flags().StringVarP(&scriptTarget, "target", "t", "", "目标")
}

func runScript(cmd *cobra.Command, args []string) {
	task := strings.Join(args, " ")
	printBanner()

	// Initialize AI provider
	apiKey := getAPIKey(scriptProvider)
	if apiKey == "" && scriptProvider != "ollama" {
		fmt.Printf("❌ 脚本生成需要 AI 支持，请配置 %s API 密钥\n", scriptProvider)
		fmt.Println()
		fmt.Println("配置方法:")
		fmt.Println("  1. 创建 .env 文件: OPENAI_API_KEY=your-key")
		fmt.Println("  2. 或使用本地 Ollama: fcapital script <task> --provider ollama")
		return
	}

	provider := createProvider(scriptProvider, apiKey, scriptModel)
	if provider == nil {
		fmt.Println("❌ AI 提供者初始化失败")
		return
	}

	fmt.Printf("🎯 任务: %s\n", task)
	fmt.Printf("📝 语言: %s\n\n", scriptLanguage)

	// Generate script
	gen := script.NewGenerator(provider)
	ctx := context.Background()

	req := &script.GenerateRequest{
		TaskDescription: task,
		Language:        scriptLanguage,
		Target:          scriptTarget,
		Context: map[string]interface{}{
			"target": scriptTarget,
		},
	}

	if scriptExecute {
		// Generate and execute
		result, err := gen.GenerateAndExecute(ctx, req, false)
		if err != nil {
			fmt.Printf("❌ 执行失败: %v\n", err)
			return
		}

		fmt.Println("\n📊 执行结果:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(result.Output)
		if result.Error != "" {
			fmt.Printf("\n❌ 错误: %s\n", result.Error)
		}
		fmt.Printf("\n⏱️  耗时: %v\n", result.Duration)
		return
	}

	// Generate only
	result, err := gen.Generate(ctx, req)
	if err != nil {
		fmt.Printf("❌ 生成失败: %v\n", err)
		return
	}

	// Display result
	fmt.Println("✅ 脚本生成成功")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("语言: %s\n", result.Language)
	fmt.Printf("安全评分: %d/100\n", result.SafetyScore)

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  警告:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	fmt.Println("\n📝 代码:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(result.Code)
	fmt.Println(strings.Repeat("-", 50))

	if result.Explanation != "" {
		fmt.Printf("\n💡 说明: %s\n", result.Explanation)
	}

	// Save to file
	if scriptOutput != "" {
		err := os.WriteFile(scriptOutput, []byte(result.Code), 0644)
		if err != nil {
			fmt.Printf("❌ 保存失败: %v\n", err)
		} else {
			fmt.Printf("\n📁 已保存到: %s\n", scriptOutput)
		}
	}
}
