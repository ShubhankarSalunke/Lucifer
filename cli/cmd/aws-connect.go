package cmd

import (
	"fmt"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var (
	awsAccessKey string
	awsSecretKey string
	awsRegion    string
	awsRoleArn   string
	awsExtID     string
)

var awsConnectCmd = &cobra.Command{
	Use:   "aws-connect",
	Short: "Configure AWS credentials for your user account",
	Run: func(cmd *cobra.Command, args []string) {

		if awsAccessKey == "" || awsSecretKey == "" || awsRegion == "" {
			fmt.Println("Access Key, Secret Key, and Region are required for an explicit connection.")
			return
		}

		cfg := config.LoadConfig()
		cfg.AWS.AccessKey = awsAccessKey
		cfg.AWS.SecretKey = awsSecretKey
		cfg.AWS.Region = awsRegion
		cfg.AWS.RoleArn = awsRoleArn
		cfg.AWS.ExternalId = awsExtID

		if err := config.SaveConfig(cfg); err != nil {
			fmt.Println("Error saving local AWS configuration:", err)
			return
		}

		fmt.Println("\n✅ AWS Configuration Saved Successfully Locally")
	},
}

func init() {
	awsConnectCmd.Flags().StringVar(&awsAccessKey, "access-key", "", "AWS Access Key ID")
	awsConnectCmd.Flags().StringVar(&awsSecretKey, "secret-key", "", "AWS Secret Access Key")
	awsConnectCmd.Flags().StringVar(&awsRegion, "region", "", "AWS Region (e.g. us-east-1)")
	awsConnectCmd.Flags().StringVar(&awsRoleArn, "role-arn", "", "AWS IAM Role ARN (optional)")
	awsConnectCmd.Flags().StringVar(&awsExtID, "external-id", "", "AWS External ID (optional)")

	awsConnectCmd.MarkFlagRequired("access-key")
	awsConnectCmd.MarkFlagRequired("secret-key")
	awsConnectCmd.MarkFlagRequired("region")

	rootCmd.AddCommand(awsConnectCmd)
}
