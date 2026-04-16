package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	// Initialize random seed to make metrics vary across runs
	rand.Seed(time.Now().UnixNano())

	// Baseline values to simulate real fluctuations
	baseRecv := 1024.5
	baseTrans := 512.2
	baseMem := 65.3
	baseCpu := 42.5
	
	agentStart := time.Now()

	// Mock /agents
	http.HandleFunc("/agents", func(w http.ResponseWriter, r *http.Request) {
		agents := map[string]interface{}{}
		
		// Make Agents list truly dynamic over time
		numAgents := 3 + rand.Intn(3) // 3 to 5 agents dynamically
		for i := 1; i <= numAgents; i++ {
			id := fmt.Sprintf("agent-node-%d-%d", i, (int(time.Since(agentStart).Seconds())/10))
			host := fmt.Sprintf("node-k8s-%d", i)
			lastSeen := fmt.Sprintf("%ds ago", rand.Intn(15))
			agents[id] = map[string]string{"host": host, "last_seen": lastSeen}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agents)
	})

	// Mock /experiments
	http.HandleFunc("/experiments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"experiment_id":"exp-auto-001", "status":"running"}`))
			return
		}

		exps := map[string]interface{}{
			"exp-1234": map[string]string{"type": "container_kill", "status": "running"},
			"exp-5678": map[string]string{"type": "memory_stress", "status": "completed"},
			"exp-9101": map[string]string{"type": "cpu_stress", "status": "failed"},
		}
		json.NewEncoder(w).Encode(exps)
	})

	// Mock /api/v1/query for Prometheus
	http.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		
		var valuestr string

		// Add random jitter to baselines
		jitter := func(base float64, percent float64) float64 {
			variation := base * percent
			change := (rand.Float64() * variation * 2) - variation
			return base + change
		}

		if query == "rate(node_network_receive_bytes_total[1m])" {
			valuestr = fmt.Sprintf("%.2f", jitter(baseRecv, 0.5)) // 50% volatility
		} else if query == "rate(node_network_transmit_bytes_total[1m])" {
			valuestr = fmt.Sprintf("%.2f", jitter(baseTrans, 0.5))
		} else if query == "(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100" {
			val := jitter(baseMem, 0.2) // 20% volatility for memory Sparkline
			if val > 100 { val = 100.0 }
			if val < 0 { val = 0.0 }
			valuestr = fmt.Sprintf("%.2f", val)
		} else {
            // CPU usage and others
			val := jitter(baseCpu, 0.3) // 30% volatility
			if val > 100 { val = 100.0 }
			if val < 0 { val = 0.0 }
			valuestr = fmt.Sprintf("%.2f", val)
		}

		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []interface{}{
					map[string]interface{}{
						"metric": map[string]string{},
						"value":  []interface{}{time.Now().Unix(), valuestr},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Mock backend running on :18080...")
	http.ListenAndServe(":18080", nil)
}
