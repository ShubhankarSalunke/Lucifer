package datamodel

import "time"


type ObservationLog struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
}

type ExperimentResult struct {
	ExperimentID string                 `json:"experiment_id"`
	Type         string                 `json:"type"`
	TargetID     string                 `json:"target_id"`
	Status       string                 `json:"status"`
	CreatedAt    string                 `json:"created_at"`
	Duration     int                    `json:"duration"`
	ExperimentType string               `json:"experiment_type"`
	StartTime    time.Time              `json:"start_time"`

	EndTime      time.Time              `json:"end_time"`
	PreSnapshot  map[string]interface{} `json:"pre_snapshot"`
	PostSnapshot map[string]interface{} `json:"post_snapshot"`
	SnapshotDiff map[string]interface{} `json:"snapshot_diff"`
	Observations []ObservationLog       `json:"observations"`
	Impact       string                 `json:"impact"`
	Restored     bool                   `json:"restored"`

	// Live/Target Metrics
	CPUPercent int `json:"cpu_percent,omitempty"`
	MemoryMB   int `json:"memory_mb,omitempty"`
	NetKBPS    int `json:"net_kbps,omitempty"`
	LatencyMS  int `json:"latency_ms,omitempty"`
}

type Agent struct {
	ID       string `json:"agent_id"`
	Host     string `json:"host"`
	LastSeen string `json:"last_seen,omitempty"`
	Status   string `json:"status,omitempty"`
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
	Timestamp     string        `json:"timestamp"`
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

type ExperimentRecord struct {
	ExperimentID string    `json:"experiment_id"`
	RuleID       string    `json:"rule_id"`
	ExpType      string    `json:"exp_type"`
	Outcome      string    `json:"outcome"`
	Severity     string    `json:"severity"`
	TargetID     string    `json:"target_id"`
	Impact       string    `json:"impact"`
	DurationS    float64   `json:"duration_s"`
	ReportID     string    `json:"report_id"`
	StartTime    time.Time `json:"start_time"`
}
