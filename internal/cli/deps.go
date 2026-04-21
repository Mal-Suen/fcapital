package cli

import (
	"fmt"
	"runtime"
	"text/tabwriter"

	"github.com/Mal-Suen/fcapital/internal/core/toolmgr"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Manage tool dependencies",
	Long:  `Check, install, and update external tool dependencies.`,
}

var depsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check tool dependencies status",
	Long:  `Check if all required tools are installed and available.`,
	Run:   runDepsCheck,
}

var depsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all supported tools",
	Long:  `List all tools that fcapital can integrate with.`,
	Run:   runDepsList,
}

var depsInstallCmd = &cobra.Command{
	Use:   "install [tools...]",
	Short: "Install missing tools",
	Long:  `Install specified tools or all missing tools if no arguments provided.`,
	Run:   runDepsInstall,
}

func init() {
	depsCmd.AddCommand(depsCheckCmd)
	depsCmd.AddCommand(depsListCmd)
	depsCmd.AddCommand(depsInstallCmd)
}

func runDepsCheck(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Checking tool dependencies...")
	fmt.Printf("[*] OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	tm := InitToolManager()
	tools := tm.List()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOOL\tCATEGORY\tSTATUS\tSOURCE\tVERSION")

	ready := 0
	missing := 0

	for _, tool := range tools {
		status := "❌ Missing"
		if tool.Status == 1 { // StatusReady
			status = "✅ Ready"
			ready++
		} else {
			missing++
		}

		source := "-"
		if tool.Source == 1 { // SourceSystem
			source = "system"
		} else if tool.Source == 2 { // SourceLocal
			source = "local"
		}

		version := tool.Version
		if version == "" {
			version = "-"
		}
		if len(version) > 30 {
			version = version[:30] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", tool.Binary, tool.Category, status, source, version)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("[*] Ready: %d, Missing: %d\n", ready, missing)

	if missing > 0 {
		yellow.Println("\n  [!] Some tools are missing. Run 'fcapital deps install' for installation instructions.")
	} else {
		green.Println("\n  [+] All tools are ready!")
	}
}

func runDepsList(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Supported tools:")
	fmt.Println()

	tm := GetToolManager()
	tools := tm.List()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TOOL\tCATEGORY\tDESCRIPTION")

	for _, tool := range tools {
		fmt.Fprintf(w, "%s\t%s\t%s\n", tool.Binary, tool.Category, tool.Description)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("[*] Run 'fcapital deps check' to see which tools are installed.")
}

func runDepsInstall(cmd *cobra.Command, args []string) {
	fmt.Println("\n[*] Tool Installation Instructions")
	fmt.Println()

	tm := InitToolManager()

	if len(args) == 0 {
		missing := tm.GetMissingTools()
		if len(missing) == 0 {
			green.Println("  [+] All tools are already installed!")
			return
		}
		fmt.Printf("[*] Found %d missing tools:\n\n", len(missing))

		for _, tool := range missing {
			fmt.Printf("  %s (%s):\n", tool.Binary, tool.Description)
			printInstallInstructions(tool)
			fmt.Println()
		}
	} else {
		for _, name := range args {
			tool, err := tm.Get(name)
			if err != nil {
				red.Printf("  [!] Unknown tool: %s\n", name)
				continue
			}
			fmt.Printf("  %s:\n", tool.Binary)
			printInstallInstructions(tool)
			fmt.Println()
		}
	}
}

func printInstallInstructions(tool *toolmgr.Tool) {
	install := tool.Install

	switch runtime.GOOS {
	case "windows":
		if install.Windows != "" {
			fmt.Printf("    Windows: %s\n", install.Windows)
		}
		if install.Go != "" {
			fmt.Printf("    Go:      %s\n", install.Go)
		}
		if install.Pip != "" {
			fmt.Printf("    Pip:     %s\n", install.Pip)
		}
	case "darwin":
		if install.MacOS != "" {
			fmt.Printf("    macOS:   %s\n", install.MacOS)
		}
		if install.Go != "" {
			fmt.Printf("    Go:      %s\n", install.Go)
		}
		if install.Pip != "" {
			fmt.Printf("    Pip:     %s\n", install.Pip)
		}
	default:
		if install.Linux != "" {
			fmt.Printf("    Linux:   %s\n", install.Linux)
		}
		if install.Go != "" {
			fmt.Printf("    Go:      %s\n", install.Go)
		}
		if install.Pip != "" {
			fmt.Printf("    Pip:     %s\n", install.Pip)
		}
	}
}
