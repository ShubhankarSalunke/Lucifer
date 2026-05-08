package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"lucifer-cli/config"

	"github.com/ShubhankarSalunke/chaos-engineering/datamodel"
	"github.com/ShubhankarSalunke/lucifer/connectors"
)

var BASE_URL string
var DATAMODEL_URL string

var client = &http.Client{
	Timeout: 3 * time.Second,
}
var authToken string

func SetAuthToken(tok string) {
	authToken = tok
}
func init() {
	BASE_URL = config.GetServerURL()
	authToken = config.GetToken()

	DATAMODEL_URL = os.Getenv("DATAMODEL_URL")
	if DATAMODEL_URL == "" {
		DATAMODEL_URL = "http://localhost:8001"
	}
}

func doRequest(req *http.Request) (*http.Response, error) {
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	return client.Do(req)
}

func GetAgents() ([]datamodel.Agent, error) {
	req, _ := http.NewRequest("GET", BASE_URL+"/agents", nil)
	response, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("empty response from server")
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Error status code: %d", response.StatusCode)
	}

	var agents map[string]datamodel.Agent
	if err := json.NewDecoder(response.Body).Decode(&agents); err != nil {
		return nil, err
	}

	var list []datamodel.Agent
	for id, a := range agents {
		a.ID = id
		list = append(list, a)
	}

	return list, nil
}

func GetExperiments() (map[string]datamodel.ExperimentResult, error) {
	req, _ := http.NewRequest("GET", BASE_URL+"/experiments", nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("no response from orchestrator")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var exps map[string]datamodel.ExperimentResult
	if err := json.NewDecoder(resp.Body).Decode(&exps); err != nil {
		return nil, err
	}

	return exps, nil
}

var resultsCache map[string]datamodel.ExperimentResult
var lastResultsFetch time.Time

func GetResults() (map[string]datamodel.ExperimentResult, error) {
	if time.Since(lastResultsFetch) < 2*time.Second && resultsCache != nil {
		return resultsCache, nil
	}

	req, _ := http.NewRequest("GET", DATAMODEL_URL+"/api/results", nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("no response from orchestrator")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var results map[string]datamodel.ExperimentResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	resultsCache = results
	lastResultsFetch = time.Now()

	return results, nil
}

func GetComputeMetrics(instanceID string) (*datamodel.ComputeSummary, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/live/%s", DATAMODEL_URL, instanceID)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("datamodel unreachable")
	}
	defer resp.Body.Close()

	var summary datamodel.ComputeSummary
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

func GetComputeHistory(instanceID string, duration time.Duration) ([]datamodel.ComputeSummary, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/history/%s?duration=%s", DATAMODEL_URL, instanceID, duration.String())
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("no response from datamodel")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var history []datamodel.ComputeSummary
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return history, nil
}

func GetComputeHistoryForScope(instanceID, scope string) ([]datamodel.ComputeSummary, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/history/%s?scope=%s", DATAMODEL_URL, instanceID, scope)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("no response from datamodel")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var history []datamodel.ComputeSummary
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return history, nil
}

func GetComputeAggregate(instanceID, window string) (*datamodel.ComputeAggregate, error) {
	url := fmt.Sprintf("%s/api/metrics/compute/aggregate/%s?window=%s", DATAMODEL_URL, instanceID, window)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("no response from datamodel")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var aggregate datamodel.ComputeAggregate
	if err := json.NewDecoder(resp.Body).Decode(&aggregate); err != nil {
		return nil, err
	}

	return &aggregate, nil
}

// SyncAgents performs discovery by reading the AWS credentials from the local CLI config
func SyncAgents() error {
	cfg := config.LoadConfig()
	if cfg.Token == "" {
		return fmt.Errorf("auth required: run 'lucifer login' first")
	}

	awsCfg := connectors.AWSConfig{
		AccessKey:  cfg.AWS.AccessKey,
		SecretKey:  cfg.AWS.SecretKey,
		Region:     cfg.AWS.Region,
		RoleARN:    cfg.AWS.RoleArn,
		ExternalID: cfg.AWS.ExternalId,
	}

	_, err := datamodel.DiscoverAgents(context.Background(), awsCfg)
	return err
}

func GetFleetAggregate() (*datamodel.FleetStats, error) {
	return datamodel.GetFleetStats()
}
func GetDiscoveredAgents() ([]datamodel.Agent, error) {
	req, _ := http.NewRequest("GET", DATAMODEL_URL+"/api/agents", nil)
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("datamodel unreachable")
	}
	defer resp.Body.Close()

	var list []datamodel.Agent
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

func GetHistoricalExperiments(reportID string) ([]datamodel.ExperimentRecord, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/experiments?report_id=%s", DATAMODEL_URL, reportID), nil)
	resp, _ := doRequest(req)
	if resp == nil {
		return nil, fmt.Errorf("datamodel unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("experiments fetch failed (%d)", resp.StatusCode)
	}

	var records []datamodel.ExperimentRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}
	return records, nil
}
