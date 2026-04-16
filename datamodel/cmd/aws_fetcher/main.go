package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/ShubhankarSalunke/chaos-engineering/datamodel"
	"github.com/joho/godotenv"
)

func fetchMetric(client *cloudwatch.Client, instanceID, metricName string, startTime, endTime time.Time) float64 {
	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String(metricName),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("InstanceId"),
				Value: aws.String(instanceID),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{types.StatisticAverage},
	}

	result, err := client.GetMetricStatistics(context.TODO(), input)
	if err != nil {
		log.Printf("[%s] Error fetching %s: %v", instanceID, metricName, err)
		return 0
	}

	if len(result.Datapoints) > 0 {
		sort.Slice(result.Datapoints, func(i, j int) bool {
			return result.Datapoints[i].Timestamp.After(*result.Datapoints[j].Timestamp)
		})
		return *result.Datapoints[0].Average
	}
	return 0
}

func DiscoverInstances(client *ec2.Client) ([]string, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	}

	result, err := client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	var instanceIDs []string
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				instanceIDs = append(instanceIDs, *instance.InstanceId)
			}
		}
	}
	return instanceIDs, nil
}

func main() {
	// Try multiple paths to find the .env file
	for _, p := range []string{"../../.env", "../.env", ".env", "../../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}


	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	cwClient := cloudwatch.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	dmURL := os.Getenv("DATAMODEL_URL")
	if dmURL == "" {
		dmURL = "http://localhost:8001"
	}

	targetURL := dmURL + "/metrics/compute"
	statusURL := dmURL + "/experiment/status"

	log.Println("Chaos Metrics Fetcher initialized with Auto-Discovery enabled.")

	ticker := time.NewTicker(20 * time.Second) // Slightly relaxed baseline for multi-instance
	defer ticker.Stop()

	var wasTesting bool
	var lastFetch time.Time
	var discoveredInstances []string

	for {
		// Periodically refresh discovered instances (every 2 mins)
		if time.Since(lastFetch) > 120*time.Second || len(discoveredInstances) == 0 {
			ids, err := DiscoverInstances(ec2Client)
			if err != nil {
				log.Printf("Discovery error: %v", err)
				if envID := os.Getenv("AWS_INSTANCE_ID"); envID != "" {
					discoveredInstances = []string{envID}
				}
			} else if len(ids) > 0 {
				discoveredInstances = ids
				log.Printf("Auto-discovered %d instances: %v", len(discoveredInstances), discoveredInstances)
			}
		}

		<-ticker.C

		resp, err := http.Get(statusURL)
		var expStatus struct {
			Active bool `json:"active"`
		}
		if err == nil {
			json.NewDecoder(resp.Body).Decode(&expStatus)
			resp.Body.Close()
		} else {
			log.Printf("Fallback: Orchestrator unreachable on %s. Proceding with baseline monitoring.", statusURL)
		}

		now := time.Now().UTC()

		if expStatus.Active {
			if !wasTesting {
				log.Println("Experiment ACTIVE.")
				wasTesting = true
			}
		} else {
			if wasTesting {
				log.Println("Experiment ENDED.")
				wasTesting = false
			}
			if time.Since(lastFetch) < 60*time.Second {
				continue
			}
		}

		lookback := 5 * time.Minute
		if expStatus.Active {
			lookback = 3 * time.Minute
		}
		startTime := now.Add(-lookback)

		for _, instanceID := range discoveredInstances {
			log.Printf("Fetching metrics for %s...", instanceID)
			cpu := fetchMetric(cwClient, instanceID, "CPUUtilization", startTime, now)
			netIn := fetchMetric(cwClient, instanceID, "NetworkIn", startTime, now)
			netOut := fetchMetric(cwClient, instanceID, "NetworkOut", startTime, now)
			diskR := fetchMetric(cwClient, instanceID, "DiskReadBytes", startTime, now)
			diskW := fetchMetric(cwClient, instanceID, "DiskWriteBytes", startTime, now)

			summary := datamodel.ComputeSummary{
				InstanceID: instanceID,
				Timestamp:  now,
				ComputeMetric: datamodel.ComputeMetric{
					CPUUtilization:  cpu,
					NetworkInBytes:  netIn,
					NetworkOutBytes: netOut,
					DiskReadBytes:   diskR,
					DiskWriteBytes:  diskW,
					StatusFailed:    false,
				},
			}

			payload, _ := json.Marshal(summary)
			res, err := http.Post(targetURL, "application/json", bytes.NewBuffer(payload))
			if err != nil {
				log.Printf("[%s] Failed to POST: %v", instanceID, err)
			} else {
				res.Body.Close()
			}
		}
		lastFetch = now
	}
}
