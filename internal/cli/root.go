package cli

import (
	"fmt"
	"os"

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
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.fcapital")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
