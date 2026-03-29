package cmd

import (
	"fmt"
	"os"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lucifer",
	Short: "Chaos Engineering CLI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "login" || cmd.Name() == "signup" || cmd.Name() == "help" {
			return
		}
		
		token := config.GetToken()
		if token == "" {
			fmt.Println("❌ Authentication required!")
			fmt.Println("Please login using 'lucifer login' or create an account with 'lucifer signup'.")
			fmt.Println("Alternatively, you can set the CHAOS_TOKEN environment variable.")
			os.Exit(1)
		}

		fmt.Println("✅ Token found, authenticated.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// func init() {
// 	rootCmd.AddCommand(createAgentCmd)
// 	rootCmd.AddCommand(createExperimentCmd)
// }
