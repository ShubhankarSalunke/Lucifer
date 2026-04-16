package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Fetch and display experiment results",
	Run: func(cmd *cobra.Command, args []string) {

		resp, err := doRequest("GET", config.GetServerURL()+"/results", nil)
		if err != nil {
			fmt.Println("Request failed:", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			fmt.Println("Error:", string(body))
			return
		}

		var results map[string]interface{}
		json.Unmarshal(body, &results)

		displayResults(results)
	},
}

func displayResults(results map[string]interface{}) {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	// Sort by experiment ID
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("# Experiment Results")
	fmt.Println()

	for _, id := range keys {
		exp := results[id].(map[string]interface{})

		fmt.Printf("## Experiment: %s\n", id)
		fmt.Printf("- **Type**: %s\n", exp["type"])
		fmt.Printf("- **Status**: %s\n", exp["status"])
		fmt.Printf("- **Agent ID**: %s\n", exp["agent_id"])
		if completedAt, ok := exp["completed_at"]; ok {
			fmt.Printf("- **Completed At**: %s\n", completedAt)
		}

		if result, ok := exp["result"].(map[string]interface{}); ok {
			fmt.Println("- **Details**:")
			if expType, ok := result["experiment_type"]; ok {
				fmt.Printf("  - Type: %s\n", expType)
			}
			if executedAt, ok := result["executed_at"]; ok {
				fmt.Printf("  - Executed At: %s\n", executedAt)
			}
			if preMetrics, ok := result["pre_metrics"].(map[string]interface{}); ok {
				fmt.Println("  - Pre-Experiment Metrics:")
				for k, v := range preMetrics {
					fmt.Printf("    - %s: %s\n", k, formatMetric(v))
				}
			}
			if postMetrics, ok := result["post_metrics"].(map[string]interface{}); ok {
				fmt.Println("  - Post-Experiment Metrics:")
				for k, v := range postMetrics {
					fmt.Printf("    - %s: %s\n", k, formatMetric(v))
				}
			}
		}

		fmt.Println()
	}
}

func formatMetric(v interface{}) string {
	if s, ok := v.(string); ok {
		// Clean up multi-line output
		lines := strings.Split(strings.TrimSpace(s), "\n")
		if len(lines) > 1 {
			return "\n      " + strings.Join(lines, "\n      ")
		}
		return s
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	rootCmd.AddCommand(resultsCmd)
}
