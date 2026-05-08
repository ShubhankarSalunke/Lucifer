package datamodel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ShubhankarSalunke/lucifer/connectors"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var (
	cache     *QueryCache
	cacheOnce sync.Once
)

var (
	AgentsFile      = "chaos-engineering/orchestrator/agents.json"
	ExperimentsFile = "chaos-engineering/orchestrator/experiments.json"
	MappingFile     = "chaos-engineering/orchestrator/mapping.json"
	VaptResultsFile = "security-audit/vapt_results.json"
)

func resolvePath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if _, err := os.Stat("../" + path); err == nil {
		return "../" + path
	}
	if _, err := os.Stat("../../" + path); err == nil {
		return "../../" + path
	}
	if _, err := os.Stat("../../../" + path); err == nil {
		return "../../../" + path
	}
	return path 
}

func getCache() *QueryCache {
	cacheOnce.Do(func() {
		cache = NewQueryCache()
	})
	return cache
}

type VAPTResult struct {
	RuleID      string   `json:"rule_id"`
	RuleName    string   `json:"rule_name"`
	Severity    string   `json:"severity"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	Remediation string   `json:"remediation"`
	Experiments []ExperimentResult `json:"experiments"`
}

func GetVAPTFindings() []VAPTResult {
	path := resolvePath(VaptResultsFile)
	f, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var list []VAPTResult
	json.Unmarshal(f, &list)
	return list
}

type FleetStats struct {
	ActiveAgents   int     `json:"active_agents"`
	AverageCPU     float64 `json:"avg_cpu"`
	AverageLatency float64 `json:"avg_latency"`
	AverageMemory  float64 `json:"avg_memory"`
	VaptScore      float64 `json:"vapt_score"`
}


func GetFleetStats() (*FleetStats, error) {
	val, err := getCache().Fetch("fleet_stats", 2*time.Second, func() (interface{}, error) {
		agentsData, _ := readJSON(AgentsFile)
		experiments, _ := readJSON(ExperimentsFile)
		vaptResults := GetVAPTFindings()
		stats := &FleetStats{
			ActiveAgents: len(agentsData),
			VaptScore:    calculateVaptScoreFromList(vaptResults),
		}

		var totalCPU, totalLat, totalMem float64

		var cpuCount, latCount, memCount int

		for _, v := range experiments {
			exp, ok := v.(map[string]interface{})
			if !ok { continue }
			
			// Extract from top level (where local agent results are mapped)
			if cpu, ok := exp["cpu_percent"].(float64); ok && cpu > 0 { 
				totalCPU += cpu
				cpuCount++
			}
			if mem, ok := exp["memory_mb"].(float64); ok && mem > 0 { 
				totalMem += mem
				memCount++
			}

			if lat, ok := exp["latency_ms"].(float64); ok && lat > 0 { 
				totalLat += lat
				latCount++
			}

			if res, ok := exp["result"].(map[string]interface{}); ok {
				if cpu, ok := res["cpu_spike"].(float64); ok && cpu > 0 { totalCPU += cpu; cpuCount++ }
				if lat, ok := res["latency_spike"].(float64); ok && lat > 0 { totalLat += lat; latCount++ }
				if mem, ok := res["memory_spike"].(float64); ok && mem > 0 { totalMem += mem; memCount++ }
			}
		}

		if cpuCount > 0 { stats.AverageCPU = totalCPU / float64(cpuCount) }
		if latCount > 0 { stats.AverageLatency = totalLat / float64(latCount) }
		if memCount > 0 { stats.AverageMemory = totalMem / float64(memCount) }

		return stats, nil
	})

	if err != nil {
		return nil, err
	}
	return val.(*FleetStats), nil
}


func DiscoverAgents(ctx context.Context, awsCfg connectors.AWSConfig) ([]Agent, error) {
	return GetAgents(ctx, awsCfg)
}

func GetAgents(ctx context.Context, awsCfg connectors.AWSConfig) ([]Agent, error) {
	val, err := getCache().Fetch("discovered_agents", 10*time.Second, func() (interface{}, error) {
		agents, _ := readJSON(AgentsFile)
		experiments, _ := readJSON(ExperimentsFile)

		allAgents := make(map[string]Agent)

		//Add agents from live heartbeats
		for id, v := range agents {
			agentData, ok := v.(map[string]interface{})
			if !ok { continue }
			host, _ := agentData["host"].(string)
			lastSeen := "Unknown"
			if ls, ok := agentData["last_seen"].(string); ok {
				lastSeen = ls
			}
			allAgents[id] = Agent{
				ID:       id,
				Host:     host,
				LastSeen: lastSeen,
				Status:   "Active",
			}
		}

		fmt.Printf("[Discovery] Found %d agents in heartbeats. Starting AWS scan...\n", len(allAgents))

		//AWS Discovery Logic (EC2)
		if awsCfg.Region == "" {
			awsCfg.Region = os.Getenv("AWS_REGION")
			if awsCfg.Region == "" {
				awsCfg.Region = os.Getenv("AWS_DEFAULT_REGION")
			}
			if awsCfg.Region == "" {
				awsCfg.Region = "us-east-1"
			}
		}
		
		cfg, err := connectors.ConnectAws(ctx, awsCfg)
		if err == nil {
			client := ec2.NewFromConfig(cfg)
			output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
			if err == nil {
				count := 0
				for _, reservation := range output.Reservations {
					for _, instance := range reservation.Instances {
						id := *instance.InstanceId
						count++
						if _, exists := allAgents[id]; !exists {
							name := "AWS Instance"
							for _, tag := range instance.Tags {
								if *tag.Key == "Name" {
									name = *tag.Value
									break
								}
							}
							host := "N/A"
							if instance.PublicDnsName != nil && *instance.PublicDnsName != "" {
								host = *instance.PublicDnsName
							}
							allAgents[id] = Agent{
								ID:       id,
								Host:     fmt.Sprintf("%s (%s)", host, name),
								LastSeen: "Cloud Discovery",
								Status:   string(instance.State.Name),
							}
						}
					}
				}
				fmt.Printf("[Discovery] AWS Scan completed. Found %d EC2 instances.\n", count)
			} else {
				fmt.Printf("[Discovery] AWS Scan failed: %v\n", err)
			}
		} else {
			fmt.Printf("[Discovery] Failed to connect to AWS: %v\n", err)
		}

		for _, v := range experiments {
			exp, ok := v.(map[string]interface{})
			if !ok { continue }
			
			agentID, _ := exp["agent_id"].(string)
			if agentID != "" {
				if _, exists := allAgents[agentID]; !exists {
					allAgents[agentID] = Agent{
						ID:       agentID,
						Host:     "Historical Resource",
						LastSeen: "Experiment Record",
						Status:   "Offline",
					}
				}
			}
		}

		var agentList []Agent
		for _, a := range allAgents {
			agentList = append(agentList, a)
		}
		return agentList, nil
	})

	if err != nil {
		return nil, err
	}
	return val.([]Agent), nil
}







func readJSON(file string) (map[string]interface{}, error) {
	cacheKey := "json_" + file
	if val, found := getCache().Get(cacheKey); found {
		return val.(map[string]interface{}), nil
	}

	f, err := os.ReadFile(resolvePath(file))
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	json.Unmarshal(f, &data)

	getCache().Set(cacheKey, data, 10*time.Second)
	return data, nil
}


func GetMappedDescription(idOrType string) string {
	mapping, _ := readJSON(MappingFile)
	experiments, _ := readJSON(ExperimentsFile)

	if exp, ok := experiments[idOrType].(map[string]interface{}); ok {
		ruleID, _ := exp["rule_id"].(string)
		if ruleID == "" {
			ruleID, _ = exp["type"].(string)
		}
		if desc, ok := mapping[ruleID].(string); ok {
			return desc
		}
		return fmt.Sprintf("Simulating %s disruption on target infrastructure.", strings.Title(strings.ReplaceAll(ruleID, "_", " ")))
	}

	if strings.HasPrefix(idOrType, "audit_") {
		ruleID := strings.TrimPrefix(idOrType, "audit_")
		findings := GetVAPTFindings()
		for _, f := range findings {
			if f.RuleID == ruleID {
				return fmt.Sprintf("%s\n[REMEDIATION:](fg:green) %s", f.RuleName, f.Remediation)
			}
		}
	}

	if desc, ok := mapping[idOrType].(string); ok {
		return desc
	}

	return "Monitoring system resilience and auditing security baseline."
}

func GetResults() (map[string]ExperimentResult, error) {
	allResults := make(map[string]ExperimentResult)

	mapToStruct := func(m map[string]interface{}, target interface{}) {
		b, _ := json.Marshal(m)
		json.Unmarshal(b, target)
	}

	exps, _ := readJSON(ExperimentsFile)
	for id, val := range exps {
		if expMap, ok := val.(map[string]interface{}); ok {
			var res ExperimentResult
			res.ExperimentID = id
			
			if status, ok := expMap["status"].(string); ok {
				res.Status = status
			}
			if aid, ok := expMap["agent_id"].(string); ok {
				res.TargetID = aid
			}
			if cat, ok := expMap["created_at"].(string); ok {
				res.CreatedAt = cat
			}
			if dur, ok := expMap["duration"].(float64); ok {
				res.Duration = int(dur)
			}
			if etype, ok := expMap["experiment_type"].(string); ok {
				res.ExperimentType = etype
			}


			if resultData, ok := expMap["result"].(map[string]interface{}); ok {
				mapToStruct(resultData, &res)
				
				if cpu, ok := resultData["cpu_spike"].(float64); ok { res.CPUPercent = int(cpu) }
				if lat, ok := resultData["latency_spike"].(float64); ok { res.LatencyMS = int(lat) }
				if mem, ok := expMap["memory_mb"].(float64); ok { res.MemoryMB = int(mem) }
				if l, ok := expMap["latency_ms"].(float64); ok { res.LatencyMS = int(l) }
				if c, ok := expMap["cpu_percent"].(float64); ok { res.CPUPercent = int(c) }
				if restored, ok := resultData["restored"].(bool); ok { res.Restored = restored }
			}



			
			allResults[id] = res
		}
	}

	vaptFindings := GetVAPTFindings()
	for _, finding := range vaptFindings {
		resID := "audit_" + finding.RuleID
		allResults[resID] = ExperimentResult{
			ExperimentID: finding.RuleID,
			Type:         "VAPT_FINDING",
			Status:       finding.Status,
			Impact:       finding.Severity,
			Observations: []ObservationLog{{Timestamp: time.Now(), Message: finding.Message}},
		}

		for _, exp := range finding.Experiments {
			id := exp.ExperimentID
			if id == "" { id = finding.RuleID }
			allResults["audit_"+id] = exp
		}
	}

	return allResults, nil
}


func GetActiveExperimentCount() (int, []string) {
	experiments, _ := readJSON(ExperimentsFile)
	count := 0
	var ids []string
	for id, v := range experiments {
		exp, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if _, hasResult := exp["result"]; !hasResult {
			count++
			ids = append(ids, id)
		}
	}
	return count, ids
}

func GetExperimentIDsByAgent(agentID string) []string {
	experiments, _ := readJSON(ExperimentsFile)
	var ids []string
	cleanID := strings.Trim(agentID, "\"")
	for id, v := range experiments {
		exp, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		aid, _ := exp["agent_id"].(string)
		if strings.Trim(aid, "\"") == cleanID {
			ids = append(ids, id)
		}
	}

	vaptFindings := GetVAPTFindings()
	for _, finding := range vaptFindings {
		ids = append(ids, "audit_"+finding.RuleID)
		
		for _, exp := range finding.Experiments {
			if exp.ExperimentID != "" && exp.ExperimentID != finding.RuleID {
				ids = append(ids, "audit_"+exp.ExperimentID)
			}
		}
	}

	return ids
}

func calculateVaptScoreFromList(findings []VAPTResult) float64 {
	if len(findings) == 0 {
		return 0
	}
	passed := 0
	for _, f := range findings {
		if strings.ToUpper(f.Status) == "PASS" {
			passed++
		}
	}
	return (float64(passed) / float64(len(findings))) * 100.0
}

func RecordVAPTResult(ruleID, status, resourceID string) {
	path := resolvePath(VaptResultsFile)
	data, _ := readJSON(VaptResultsFile)
	if data == nil {
		data = make(map[string]interface{})
	}

	res := map[string]interface{}{
		"rule_id":     ruleID,
		"status":      status,
		"resource_id": resourceID,
		"score":       0.0, 
		"timestamp":   time.Now().Format(time.RFC3339),
	}
	
	if status == "PASS" {
		res["score"] = 100.0
	}

	key := fmt.Sprintf("%s_%s", ruleID, resourceID)
	if resourceID == "" {
		key = ruleID
	}
	data[key] = res

	b, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, b, 0644)
}

