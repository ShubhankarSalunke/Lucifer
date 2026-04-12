package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit commands for scanning and attacking infrastructure",
}

var scanCmd = &cobra.Command{
	Use:   "scan [provider] [service_name]",
	Short: "Run security audit scan",
	Long:  "Run security audit scan. Specify a cloud provider (e.g. aws) and optionally a service name (e.g. s3, ec2) or use --full for a comprehensive scan.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: Cloud provider argument is required (e.g., aws)")
			return
		}
		provider := args[0]
		service := "all"
		if len(args) > 1 {
			service = args[1]
		}
		full, _ := cmd.Flags().GetBool("full")
		if full {
			service = "all"
		}

		fmt.Printf("Starting security audit scan for %s on %s...\n", service, provider)

		// Executing standalone security-audit
		command := exec.Command("go", "run", "main.go", provider, service)
		command.Dir = "../../security-audit"
		command.Stdout = cmd.OutOrStdout()
		command.Stderr = cmd.ErrOrStderr()

		if err := command.Run(); err != nil {
			fmt.Println("Error running scan:", err)
		}
	},
}

var rulesCmd = &cobra.Command{
	Use:   "rules [provider] [service]",
	Short: "List and describe available auditing rules",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: Cloud provider argument is required (e.g., aws)")
			return
		}
		provider := args[0]
		service := "all"
		if len(args) > 1 {
			service = args[1]
		}
		fmt.Printf("Displaying rules for %s on %s...\n", service, provider)
		rulesPath := fmt.Sprintf("../../security-audit/rules/%s", provider)
		fmt.Printf("Rules are loaded dynamically from %s/\n", rulesPath)

		entries, err := os.ReadDir(rulesPath)
		if err != nil {
			fmt.Println("Error reading rules directory:", err)
			return
		}

		found := false
		for _, entry := range entries {
			if service != "all" && entry.Name() != service+".yaml" {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".yaml") {
				found = true
				fmt.Printf("\n=== %s ===\n", strings.ToUpper(strings.TrimSuffix(entry.Name(), ".yaml")))
				content, err := os.ReadFile(fmt.Sprintf("%s/%s", rulesPath, entry.Name()))
				if err == nil {
					lines := strings.Split(string(content), "\n")
					var id, name string
					for _, line := range lines {
						tLine := strings.TrimSpace(line)
						if strings.HasPrefix(tLine, "- id: ") {
							id = strings.TrimPrefix(tLine, "- id: ")
						} else if strings.HasPrefix(tLine, "name: ") {
							name = strings.TrimPrefix(tLine, "name: ")
							if id != "" {
								fmt.Printf(" - %s: %s\n", strings.Trim(id, `"`), strings.Trim(name, `"`))
								id = ""
								name = ""
							}
						}
					}
				}
			}
		}

		if !found {
			fmt.Println("No rules found for the specified service & provider.")
		}
	},
}

// var reportCmd = &cobra.Command{
// 	Use:   "report [output-format]",
// 	Short: "Manually triggers report generation from the last scan result",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		format := "markdown"
// 		if len(args) > 0 {
// 			format = args[0]
// 		}
// 		fmt.Printf("Generating report in %s format...\n", format)
// 		fmt.Println("Please run 'audit scan' to generate a fresh vapt_report.md.")
// 	},
// }

// var remediateCmd = &cobra.Command{
// 	Use:   "remediate",
// 	Short: "Print AWS CLI or Terraform commands to fix failed policies",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		fmt.Println("Generating remediation strategies for last scan...")
// 		fmt.Println("Review vapt_report.md for explicit remediation details.")
// 	},
// }

func init() {
	scanCmd.Flags().Bool("full", false, "Run a full comprehensive scan")

	auditCmd.AddCommand(scanCmd)
	auditCmd.AddCommand(rulesCmd)
	// auditCmd.AddCommand(reportCmd)
	// auditCmd.AddCommand(remediateCmd)

	rootCmd.AddCommand(auditCmd)
}
