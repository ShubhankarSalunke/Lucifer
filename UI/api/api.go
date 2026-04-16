package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var BASE_URL string
var DATAMODEL_URL string

var client = &http.Client{
	Timeout: 3 * time.Second,
}

type ExperimentCreate struct {
	Type            string `json:"type"`
	TargetContainer string `json:"target_container"`
	Duration        int    `json:"duration"`
	AgentID         string `json:"agent_id"`
	MemoryMB        int    `json:"memory_mb,omitempty"`
	AssignedTo      string `json:"assigned_to,omitempty"`
}

type ExperimentResult struct {
	ExperimentID string                 `json:"experiment_id"`
	Status       string                 `json:"status"`
	Result       map[string]interface{} `json:"result,omitempty"`
}

type Agent struct {
	ID       string `json:"agent_id"`
	Host     string `json:"host"`
	LastSeen string `json:"last_seen,omitempty"`
}

type ComputeMetric struct {
	CPUUtilization  float64 `json:"cpu_utilization"`
	NetworkInBytes  float64 `json:"network_in"`
	NetworkOutBytes float64 `json:"network_out"`
	DiskReadBytes   float64 `json:"disk_read"`
	DiskWriteBytes  float64 `json:"disk_write"`
	StatusFailed    bool    `json:"status_failed"`
}

type ComputeSummary struct {
	InstanceID    string        `json:"instance_id"`
	ComputeMetric ComputeMetric `json:"compute_metric"`
	Timestamp     string        `json:"timestamp"` // Parse strings easily in TUI
}

type ComputeAggregate struct {
	InstanceID      string        `json:"instance_id"`
	Window          string        `json:"window"`
	SampleCount     int           `json:"sample_count"`
	StartedAt       string        `json:"started_at"`
	EndedAt         string        `json:"ended_at"`
	Average         ComputeMetric `json:"average"`
	PeakNetworkBps  float64       `json:"peak_network_bps"`
	PeakCPUPercent  float64       `json:"peak_cpu_percent"`
	LatestTimestamp string        `json:"latest_timestamp"`
}

type FleetAggregate struct {
	ActiveAgents  int     `json:"active_agents"`
	AverageCPU    float64 `json:"avg_cpu"`
	AverageNet    float64 `json:"avg_net"`
	AverageDisk   float64       `json:"avg_disk"`
	TotalInbound  float64       `json:"total_inbound"`
	TotalOutbound float64       `json:"total_outbound"`
}

func init() {
	// Try multiple paths to find the .env file
	for _, p := range []string{"../../.env", "../.env", ".env", "../../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}
	BASE_URL = os.Getenv("URL")
	if BASE_URL == "" {
		BASE_URL = "http://localhost:8000"
	}
	DATAMODEL_URL = os.Getenv("DATAMODEL_URL")
	if DATAMODEL_URL == "" {
		DATAMODEL_URL = "http://localhost:8001"
	}
}

func GetAgents() ([]Agent, error) {
	response, err := client.Get(BASE_URL + "/agents")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Error status code: %d", response.StatusCode)
	}

	var agents map[string]Agent
	if err := json.NewDecoder(response.Body).Decode(&agents); err != nil {
		return nil, err
	}

	var list []Agent
	for id, a := range agents {
		a.ID = id
		list = append(list, a)
	}

	return list, nil
}

func CreateExperiment(exp ExperimentCreate) (ExperimentResult, error) {
	jsonData, err := json.Marshal(exp)
	if err != nil {
		return ExperimentResult{}, err
	}
	response, err := client.Post(BASE_URL+"/experiments", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ExperimentResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return ExperimentResult{}, fmt.Errorf("Error status code: %d", response.StatusCode)
	}

	var result ExperimentResult

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return ExperimentResult{}, err
	}
	return result, nil
}

func GetExperiments() (map[string]interface{}, error) {
	resp, err := client.Get(fmt.Sprintf("%s/experiments", BASE_URL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var exps map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&exps); err != nil {
		return nil, err
	}

	return exps, nil
}

func GetComputeMetrics(instanceID string) (*ComputeSummary, error) {
	resp, err := client.Get(fmt.Sprintf("%s/api/metrics/compute/%s", DATAMODEL_URL, instanceID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var summary ComputeSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, err
	}

	return &summary, nil
}
func SetExperimentStatus(active bool) error {
	payload, _ := json.Marshal(map[string]bool{"active": active})
	resp, err := client.Post(DATAMODEL_URL+"/experiment/status", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func GetComputeHistory(instanceID string, duration time.Duration) ([]ComputeSummary, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/history/%s?duration=%s", DATAMODEL_URL, instanceID, duration.String())
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var history []ComputeSummary
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return history, nil
}

func GetComputeHistoryForScope(instanceID, scope string) ([]ComputeSummary, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/history/%s?scope=%s", DATAMODEL_URL, instanceID, scope)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var history []ComputeSummary
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return history, nil
}



func GetComputeAggregate(instanceID, window string) (*ComputeAggregate, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/aggregate/%s?window=%s", DATAMODEL_URL, instanceID, window)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var aggregate ComputeAggregate
	if err := json.NewDecoder(resp.Body).Decode(&aggregate); err != nil {
		return nil, err
	}

	return &aggregate, nil
}

func GetFleetAggregate() (*FleetAggregate, error) {
	resp, err := client.Get(DATAMODEL_URL + "/api/metrics/fleet/aggregate")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var aggregate FleetAggregate
	if err := json.NewDecoder(resp.Body).Decode(&aggregate); err != nil {
		return nil, err
	}

	return &aggregate, nil
}
func GetDiscoveredAgents() ([]Agent, error) {
	resp, err := client.Get(DATAMODEL_URL + "/api/metrics/discovered")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var instances []string
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return nil, err
	}

	var list []Agent
	for _, id := range instances {
		list = append(list, Agent{
			ID:   id,
			Host: "Autodiscovered (InfluxDB)",
		})
	}
	return list, nil
}
