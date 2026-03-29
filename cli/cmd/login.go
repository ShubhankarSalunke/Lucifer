package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with an existing API token",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Print("Enter your Chaos API Token: ")
		reader := bufio.NewReader(os.Stdin)
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		if token == "" {
			fmt.Println("Token cannot be empty")
			return
		}

		// Set temporarily so doRequest uses it
		os.Setenv("CHAOS_TOKEN", token)

		// Test token using the secure /agents endpoint
		resp, err := doRequest("GET", config.GetServerURL()+"/agents", nil)
		if err != nil {
			fmt.Println("Failed to verify token:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Invalid token provided.")
			return
		} else if resp.StatusCode != 200 {
			fmt.Printf("⚠️ Verification failed with status %d (Are you sure the server is healthy?)\n", resp.StatusCode)
			// Decide if we still want to save, let's just abort to be safe
			return
		}

		// Clean up the temp env
		os.Unsetenv("CHAOS_TOKEN")

		// If successful, save token
		cfg := config.Config{
			Token:     token,
			ServerURL: config.GetServerURL(),
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			fmt.Println("Failed to save config:", err)
			return
		}
		fmt.Println("✅ Login successful")
		fmt.Println("Token saved to ~/.chaos/config.json")
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
