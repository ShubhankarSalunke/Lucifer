package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"lucifer-cli/config"

	"github.com/spf13/cobra"
)

var (
	expType    string
	agentIDExp string
	duration   int
	cpu        int
	memory     int
	latency    int
	target     string
	bucket     string
	roleArn    string
	kmsKey     string
	prefix     string
	percent    int
	externalID string
)

var createExperimentCmd = &cobra.Command{
	Use:   "create-experiment",
	Short: "Create experiment",
	Run: func(cmd *cobra.Command, args []string) {

		if expType == "" || agentIDExp == "" || duration <= 0 {
			fmt.Println("type, agent and duration are required")
			return
		}

		payload := map[string]interface{}{
			"type":             expType,
			"agent_id":         agentIDExp,
			"duration":         duration,
			"target_container": target,
		}

		switch expType {

		case "cpu_stress":
			if cpu <= 0 || cpu > 100 {
				fmt.Println("cpu must be between 1-100")
				return
			}
			payload["cpu_percent"] = cpu

		case "memory_stress":
			if memory <= 0 {
				fmt.Println("memory must be > 0")
				return
			}
			payload["memory_mb"] = memory

		case "network_latency":
			if latency <= 0 {
				fmt.Println("latency must be > 0")
				return
			}
			payload["latency_ms"] = latency

		case "container_kill":
			if target == "" {
				fmt.Println("target container required")
				return
			}

		case "s3_access_deny":
			if bucket == "" {
				fmt.Println("bucket name required")
				return
			}
			payload["bucket_name"] = bucket
			payload["role_arn"] = roleArn
			payload["external_id"] = externalID

		case "s3_kms_disable":
			if kmsKey == "" {
				fmt.Println("kms-key required")
				return
			}
			payload["kms_key_id"] = kmsKey
			payload["role_arn"] = roleArn
			payload["external_id"] = externalID

		case "s3_object_delete":
			if bucket == "" {
				fmt.Println("bucket name required")
				return
			}
			payload["bucket_name"] = bucket
			payload["prefix"] = prefix
			payload["delete_percent"] = percent
			payload["role_arn"] = roleArn
			payload["external_id"] = externalID

		case "s3_metadata_corrupt":
			if bucket == "" {
				fmt.Println("bucket name required")
				return
			}
			payload["bucket_name"] = bucket
			payload["prefix"] = prefix
			payload["role_arn"] = roleArn
			payload["external_id"] = externalID

		default:
			fmt.Println("invalid experiment type")
			return
		}

		body, err := json.Marshal(payload)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

		resp, err := doRequest("POST", config.GetServerURL()+"/create-experiment", body)
		if err != nil {
			fmt.Println("Request failed:", err)
			return
		}
		defer resp.Body.Close()

		resBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			fmt.Println("Error:", string(resBody))
			return
		}

		var result map[string]interface{}
		json.Unmarshal(resBody, &result)

		fmt.Println("\n✅ Experiment Created")
		fmt.Println("Experiment ID:", result["experiment_id"])
	},
}

func init() {
	createExperimentCmd.Flags().StringVar(&expType, "type", "", "Experiment type")
	createExperimentCmd.Flags().StringVar(&agentIDExp, "agent", "", "Agent ID")
	createExperimentCmd.Flags().IntVar(&duration, "duration", 30, "Duration in seconds")
	createExperimentCmd.Flags().IntVar(&cpu, "cpu", 0, "CPU %")
	createExperimentCmd.Flags().IntVar(&memory, "memory", 0, "Memory MB")
	createExperimentCmd.Flags().IntVar(&latency, "latency", 0, "Latency ms")
	createExperimentCmd.Flags().StringVar(&target, "target", "", "Target container")
	createExperimentCmd.Flags().StringVar(&bucket, "bucket", "", "Target S3 bucket")
	createExperimentCmd.Flags().StringVar(&roleArn, "role-arn", "", "IAM Role ARN to assume")
	createExperimentCmd.Flags().StringVar(&kmsKey, "kms-key", "", "KMS Key ID/ARN")
	createExperimentCmd.Flags().StringVar(&prefix, "prefix", "", "Object prefix")
	createExperimentCmd.Flags().IntVar(&percent, "percent", 10, "Percentage of objects to affect")
	createExperimentCmd.Flags().StringVar(&externalID, "external-id", "", "External ID for role assumption")

	createExperimentCmd.MarkFlagRequired("type")
	createExperimentCmd.MarkFlagRequired("agent")

	rootCmd.AddCommand(createExperimentCmd)
}
