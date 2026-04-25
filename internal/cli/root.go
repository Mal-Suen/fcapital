package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "fcapital",
	Short: "A comprehensive penetration testing framework",
	Long: `fcapital is a penetration testing framework that integrates
multiple security tools with a unified interface.

It provides both interactive menu and command-line interface
for various security testing tasks.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 默认运行交互模式
		runInteractive()
	},
}

func Execute(version, commit, date string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fcapital/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// 添加子命令
	rootCmd.AddCommand(depsCmd)
	rootCmd.AddCommand(reconCmd)
	rootCmd.AddCommand(subdomainCmd)
	rootCmd.AddCommand(portscanCmd)
	rootCmd.AddCommand(webscanCmd)
	rootCmd.AddCommand(vulnscanCmd)
	rootCmd.AddCommand(workflowCmd)
}

// getExeDir 获取可执行文件所在目录
func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

func initConfig() {
	// 1. 加载 .env 文件（按优先级顺序）
	// 优先级：当前目录 > 可执行文件目录 > 用户主目录
	envFiles := []string{}

	// 当前工作目录
	cwd, _ := os.Getwd()
	if cwd != "" {
		envFiles = append(envFiles,
			filepath.Join(cwd, ".env"),
			filepath.Join(cwd, ".env.local"),
		)
	}

	// 可执行文件所在目录
	exeDir := getExeDir()
	if exeDir != "" {
		// 可执行文件目录本身
		envFiles = append(envFiles,
			filepath.Join(exeDir, ".env"),
			filepath.Join(exeDir, ".env.local"),
		)
		// 可执行文件目录的上级目录（build -> 项目根目录）
		parentDir := filepath.Dir(exeDir)
		if parentDir != "" && parentDir != exeDir {
			envFiles = append(envFiles,
				filepath.Join(parentDir, ".env"),
				filepath.Join(parentDir, ".env.local"),
			)
		}
	}

	// 用户主目录
	home, _ := os.UserHomeDir()
	if home != "" {
		envFiles = append(envFiles,
			filepath.Join(home, ".fcapital", ".env"),
			filepath.Join(home, ".fcapital", ".env.local"),
		)
	}

	// 尝试加载每个 .env 文件（第一个成功的即可）
	loaded := false
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err == nil {
				loaded = true
				break // 只加载第一个找到的
			}
		}
	}

	// 如果没有找到 .env 文件，给出提示
	if !loaded && verbose {
		fmt.Fprintln(os.Stderr, "⚠️  未找到 .env 文件，请确保已配置 API 密钥")
	}

	// 2. 加载配置文件
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		if home != "" {
			viper.AddConfigPath(home + "/.fcapital")
		}
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// 3. 自动读取环境变量
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
