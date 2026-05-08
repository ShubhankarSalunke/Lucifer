package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func resolveAuditDir() (string, error) {
	locations := []string{
		"security-audit",
		"../security-audit",
		"../../security-audit",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			abspath, _ := filepath.Abs(loc)
			return abspath, nil
		}
	}
	return "", fmt.Errorf("security-audit directory not found — ensure it's cloned at the same level as lucifer-cli")
}

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

		auditDir, _ := resolveAuditDir()
		fmt.Printf("Starting security audit scan for %s on %s...\n", service, provider)

		argsList := []string{"run", "main.go", provider}
		if service != "all" {
			argsList = append(argsList, service)
		}
		command := exec.Command("go", argsList...)
		command.Dir = auditDir
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		command.Env = append(os.Environ(),
			"AWS_ROLE_ARN="+getEnvOrDefault("AWS_ROLE_ARN", ""),
			"AWS_EXTERNAL_ID="+getEnvOrDefault("AWS_EXTERNAL_ID", ""),
		)

		if err := command.Run(); err != nil {
			fmt.Printf("Error running scan: %v\n", err)
			fmt.Println("\nTroubleshooting:")
			fmt.Println("1. Ensure AWS credentials are configured (aws-connect or env vars)")
			fmt.Println("2. Check that the IAM role has audit permissions")
			fmt.Println("3. Run 'cd security-audit && go run main.go' directly to see full error")
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

		auditDir, err := resolveAuditDir()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Printf("Displaying rules for %s on %s...\n", service, provider)
		rulesPath := filepath.Join(auditDir, "rules", provider)
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
				content, err := os.ReadFile(filepath.Join(rulesPath, entry.Name()))
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

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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

	rootCmd.AddCommand(auditCmd)
}
