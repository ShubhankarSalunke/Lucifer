package datamodel

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Orchestrator manages the API layer over the DataModel and the background metric push routines.
type Orchestrator struct {
	DataModel *DataModel
	Router    *chi.Mux

	// Channels used for tracking if an experiment is actively running
	// If true, background worker allows metric flushes.
	expStatusChan      chan bool
	isExperimentActive bool
}

func NewOrchestrator(dm *DataModel) *Orchestrator {
	orc := &Orchestrator{
		DataModel:          dm,
		Router:             chi.NewRouter(),
		expStatusChan:      make(chan bool),
		isExperimentActive: false,
	}

	orc.setupRoutes()
	go orc.RunMetricPusher()

	return orc
}

func (o *Orchestrator) setupRoutes() {
	o.Router.Use(middleware.Logger)
	o.Router.Use(middleware.Recoverer)

	// Receive incoming CloudWatch metrics
	o.Router.Post("/metrics/compute", o.handleIngestCompute)
	o.Router.Post("/metrics/storage", o.handleIngestStorage)

	// Used by CLI UI to query the cache/database
	o.Router.Get("/api/metrics/compute/{instanceID}", o.handleGetCompute)
	o.Router.Get("/api/metrics/compute/history/{instanceID}", o.handleGetComputeHistory)
	o.Router.Get("/api/metrics/compute/aggregate/{instanceID}", o.handleGetAggregated)
	o.Router.Get("/api/metrics/fleet/aggregate", o.handleGetFleetAggregate)
	o.Router.Get("/api/metrics/storage/{bucketName}", o.handleGetStorage)
	o.Router.Get("/api/metrics/discovered", o.handleGetDiscovered)

	// Endpoint to toggle experiment status
	o.Router.Post("/experiment/status", o.handleExperimentStatus)
	o.Router.Get("/experiment/status", o.handleGetExperimentStatus)
}

// RunMetricPusher is a background ticker logic that manages flushing/dropping metrics.
// It checks every 60 seconds.
func (o *Orchestrator) RunMetricPusher() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case status := <-o.expStatusChan:
			o.isExperimentActive = status
			if !status {
				// Force a flush when turning off to make sure trailing metrics are saved
				o.DataModel.DBwriter.Flush()
			}
		case <-ticker.C:
			if o.isExperimentActive {
				// Normally async writes happen automatically via InfluxDB library buffers,
				// but we explicitly call Flush every 60s during experiments
				// to ensure the UI sees fresh data quickly.
				o.DataModel.DBwriter.Flush()
			}
		}
	}
}

// handleExperimentStatus toggles the channel
func (o *Orchestrator) handleExperimentStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send status update into the channel
	o.expStatusChan <- req.Active

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (o *Orchestrator) handleGetExperimentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"active": o.isExperimentActive})
}

func (o *Orchestrator) handleIngestCompute(w http.ResponseWriter, r *http.Request) {

	var metric ComputeSummary
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pt := ComputeSummaryToPoint(metric)
	o.DataModel.DBwriter.WritePoint(pt) // This is now purely async batched (Nagling)
	w.WriteHeader(http.StatusAccepted)
}

func (o *Orchestrator) handleIngestStorage(w http.ResponseWriter, r *http.Request) {

	var metric StorageSummary
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pt := StorageSummaryToPoint(metric)
	o.DataModel.DBwriter.WritePoint(pt)
	w.WriteHeader(http.StatusAccepted)
}

func (o *Orchestrator) handleGetCompute(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceID")
	res, err := o.DataModel.FindbyInstanceID(instanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (o *Orchestrator) handleGetComputeHistory(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceID")
	scope := r.URL.Query().Get("scope")
	durStr := r.URL.Query().Get("duration")
	if scope == "" && durStr == "" {
		durStr = "10m"
	}

	end := time.Now()
	start := time.Unix(0, 0)

	switch scope {
	case "", "10m":
		if durStr == "" {
			durStr = "10m"
		}
		duration, err := time.ParseDuration(durStr)
		if err != nil {
			http.Error(w, "invalid duration", http.StatusBadRequest)
			return
		}
		start = end.Add(-duration)
	case "overall":
		start = time.Unix(0, 0)
	default:
		http.Error(w, "invalid scope", http.StatusBadRequest)
		return
	}

	res, err := o.DataModel.GetComputeMetrics(instanceID, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res == nil {
		res = []ComputeSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (o *Orchestrator) handleGetAggregated(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceID")
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "overall"
	}

	res, err := o.DataModel.GetComputeAggregate(instanceID, window)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}


func (o *Orchestrator) handleGetFleetAggregate(w http.ResponseWriter, r *http.Request) {
	res, err := o.DataModel.GetFleetAggregate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (o *Orchestrator) handleGetStorage(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "bucketName")
	res, err := o.DataModel.FindbyBucketName(bucketName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
func (o *Orchestrator) handleGetDiscovered(w http.ResponseWriter, r *http.Request) {
	instances, err := o.DataModel.GetDiscoveredInstances()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}
