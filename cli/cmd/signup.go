package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var signupCmd = &cobra.Command{
	Use:   "signup",
	Short: "Create a new user and store API token",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Print("Enter User ID (Email): ")
		reader := bufio.NewReader(os.Stdin)
		userID, _ := reader.ReadString('\n')
		userID = strings.TrimSpace(userID)

		payload := map[string]string{}
		if userID != "" {
			payload["user_id"] = userID
		}
		
		bodyBytes, _ := json.Marshal(payload)

		resp, err := doRequest("POST", config.GetServerURL()+"/create-user", bodyBytes)
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

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		token := result["token"].(string)

		cfg := config.Config{
			Token:     token,
			ServerURL: config.GetServerURL(),
		}

		err = config.SaveConfig(cfg)
		if err != nil {
			fmt.Println("Failed to save config:", err)
			return
		}
		fmt.Println("Token:", token)
		fmt.Println("✅ Signup successful")
		fmt.Println("Token saved to ~/.chaos/config.json")
	},
}

func init() {
	rootCmd.AddCommand(signupCmd)
}
