package cli

import (
	"fmt"
	"os"

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

func initConfig() {
	// 1. 加载 .env 文件（按优先级顺序）
	// 优先级：当前目录 > 用户主目录 > 项目配置目录
	envFiles := []string{
		".env",                // 当前目录
		".env.local",          // 当前目录本地覆盖（git忽略）
		".env.production",     // 生产环境
		".env.development",    // 开发环境
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		envFiles = append(envFiles,
			home+"/.fcapital/.env",        // 用户配置目录
			home+"/.fcapital/.env.local",  // 用户配置本地覆盖
		)
	}

	// 尝试加载每个 .env 文件
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err == nil {
				if verbose {
					fmt.Fprintln(os.Stderr, "Loaded env file:", envFile)
				}
			}
		}
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
