package datamodel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type ComputeMetric struct {
	CPUUtilization  float64 `json:"cpu_utilization"` // Percentage
	NetworkInBytes  float64 `json:"network_in"`      // Bytes
	NetworkOutBytes float64 `json:"network_out"`     // Bytes
	DiskReadBytes   float64 `json:"disk_read"`       // Bytes
	DiskWriteBytes  float64 `json:"disk_write"`      // Bytes
	StatusFailed    bool    `json:"status_failed"`

	// Requires CW Agent, or  custom Go agent
	//MemoryUsagePercent float64 `json:"memory_usage_percent"`
	//Cores              int     `json:"cores"`
}

type StorageMetric struct {
	BucketSizeBytes float64 `json:"bucket_size_bytes"`
	NumberOfObjects int     `json:"number_of_objects"`

	//Require enabling S3 request metrics
	AllRequests    int     `json:"all_requests"`
	GetRequests    int     `json:"get_requests"`
	PutRequests    int     `json:"put_requests"`
	Errors4xx      int     `json:"errors_4xx"`
	Errors5xx      int     `json:"errors_5xx"`
	TotalLatencyMs float64 `json:"total_latency_ms"` // TotalRequestLatency
}

type ComputeSummary struct {
	InstanceID    string `json:"instance_id"`
	ComputeMetric `json:"compute_metric"`
	Timestamp     time.Time `json:"timestamp"`
}

type ComputeAggregate struct {
	InstanceID      string        `json:"instance_id"`
	Window          string        `json:"window"`
	SampleCount     int           `json:"sample_count"`
	StartedAt       time.Time     `json:"started_at"`
	EndedAt         time.Time     `json:"ended_at"`
	Average         ComputeMetric `json:"average"`
	PeakNetworkBps  float64       `json:"peak_network_bps"`
	PeakCPUPercent  float64       `json:"peak_cpu_percent"`
	LatestTimestamp time.Time     `json:"latest_timestamp"`
}

type StorageSummary struct {
	BucketName    string `json:"bucket_name"`
	StorageMetric `json:"storage_metric"`
	Timestamp     time.Time `json:"timestamp"`
}

type AggregatedMetrics struct {
	InstanceID     string        `json:"instance_id"`
	Average        ComputeMetric `json:"average"`
	MaxCPU         float64       `json:"max_cpu"`
	PeakNetworkBps float64       `json:"peak_network_bps"`
	SampleCount    int           `json:"sample_count"`
	Window         string        `json:"window"` // overall or 10m
}

type FleetAggregate struct {
	ActiveAgents  int           `json:"active_agents"`
	AverageCPU    float64       `json:"avg_cpu"`
	AverageNet    float64       `json:"avg_net"`
	AverageDisk   float64       `json:"avg_disk"`
	TotalInbound  float64       `json:"total_inbound"`
	TotalOutbound float64       `json:"total_outbound"`
}

func (dm *DataModel) GetComputeMetrics(instanceID string, start, end time.Time) ([]ComputeSummary, error) {
	query := fmt.Sprintf(`from(bucket: "chaos-engineering")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "compute_metrics" and r.instance_id == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`,
		start.Format(time.RFC3339), end.Format(time.RFC3339), instanceID)

	result, err := dm.DBreader.Query(context.Background(), query)
	if err != nil {
		return nil, NewCustomError(err, http.StatusInternalServerError)
	}
	defer result.Close()

	var summaries []ComputeSummary
	for result.Next() {
		record := result.Record()

		getFloat := func(field string) float64 {
			if v, ok := record.ValueByKey(field).(float64); ok {
				return v
			}
			return 0
		}

		getBool := func(field string) bool {
			if v, ok := record.ValueByKey(field).(bool); ok {
				return v
			}
			return false
		}

		summaries = append(summaries, ComputeSummary{
			InstanceID: instanceID,
			ComputeMetric: ComputeMetric{
				CPUUtilization:  getFloat("cpu_utilization"),
				NetworkInBytes:  getFloat("network_in"),
				NetworkOutBytes: getFloat("network_out"),
				DiskReadBytes:   getFloat("disk_read"),
				DiskWriteBytes:  getFloat("disk_write"),
				StatusFailed:    getBool("status_failed"),
			},
			Timestamp: record.Time(),
		})
	}

	if result.Err() != nil {
		return nil, NewCustomError(result.Err(), http.StatusInternalServerError)
	}

	return summaries, nil
}

func (dm *DataModel) GetComputeAggregate(instanceID string, window string) (*ComputeAggregate, error) {
	cacheKey := fmt.Sprintf("agg_%s_%s", instanceID, window)

	val, err := dm.Cache.Fetch(cacheKey, 30*time.Second, func() (interface{}, error) {
		rangeStart := "-10m"
		if window == "overall" {
			rangeStart = "0"
		}

		// Use mean() per field - simpler and more reliable than reduce() with time tracking
		query := fmt.Sprintf(`
			from(bucket: "chaos-engineering")
			|> range(start: %s)
			|> filter(fn: (r) => r._measurement == "compute_metrics" and r.instance_id == "%s")
			|> group(columns: ["instance_id", "_field"])
			|> mean()
			|> pivot(rowKey: ["instance_id"], columnKey: ["_field"], valueColumn: "_value")
		`, rangeStart, instanceID)

		result, err := dm.DBreader.Query(context.Background(), query)
		if err != nil {
			return nil, err
		}
		defer result.Close()

		getFloat := func(record interface{ ValueByKey(string) interface{} }, k string) float64 {
			if v, ok := record.ValueByKey(k).(float64); ok {
				return v
			}
			return 0
		}

		if result.Next() {
			rec := result.Record()
			return &ComputeAggregate{
				InstanceID: instanceID,
				Window:     window,
				Average: ComputeMetric{
					CPUUtilization:  getFloat(rec, "cpu_utilization"),
					NetworkInBytes:  getFloat(rec, "network_in"),
					NetworkOutBytes: getFloat(rec, "network_out"),
					DiskReadBytes:   getFloat(rec, "disk_read"),
					DiskWriteBytes:  getFloat(rec, "disk_write"),
				},
			}, nil
		}
		return &ComputeAggregate{InstanceID: instanceID, Window: window}, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*ComputeAggregate), nil
}

func (dm *DataModel) GetFleetAggregate() (*FleetAggregate, error) {
	val, err := dm.Cache.Fetch("fleet_aggregate", 10*time.Second, func() (interface{}, error) {
		// Get the latest reading per instance using last(), then aggregate across fleet
		query := fmt.Sprintf(`
			import "experimental/array"
			cpu = from(bucket: "chaos-engineering")
				|> range(start: -5m)
				|> filter(fn: (r) => r._measurement == "compute_metrics" and r._field == "cpu_utilization")
				|> group(columns: ["instance_id"])
				|> last()
				|> group()
				|> mean()
			cpu
		`)
		_ = query

		// Simpler approach: query all fields in last 5m, pivot, then aggregate
		q2 := fmt.Sprintf(`
			from(bucket: "%s")
			|> range(start: -5m)
			|> filter(fn: (r) => r._measurement == "compute_metrics")
			|> group(columns: ["instance_id", "_field"])
			|> last()
			|> group(columns: ["_field"])
			|> mean()
			|> pivot(rowKey: ["_field"], columnKey: ["_field"], valueColumn: "_value")
		`, dm.bucket)

		// Actually just query instances and count them + get averages per field
		q3 := fmt.Sprintf(`
			from(bucket: "%s")
			|> range(start: -5m)
			|> filter(fn: (r) => r._measurement == "compute_metrics")
			|> group(columns: ["instance_id", "_field"])
			|> last()
		`, dm.bucket)
		_ = q2

		result, err := dm.DBreader.Query(context.Background(), q3)
		if err != nil {
			return nil, err
		}
		defer result.Close()

		instanceCPU := make(map[string]float64)
		instanceNetIn := make(map[string]float64)
		instanceNetOut := make(map[string]float64)
		instanceDiskR := make(map[string]float64)
		instanceDiskW := make(map[string]float64)

		for result.Next() {
			rec := result.Record()
			iid := fmt.Sprintf("%v", rec.ValueByKey("instance_id"))
			field := rec.Field()
			v, _ := rec.Value().(float64)
			switch field {
			case "cpu_utilization":
				instanceCPU[iid] = v
			case "network_in":
				instanceNetIn[iid] = v
			case "network_out":
				instanceNetOut[iid] = v
			case "disk_read":
				instanceDiskR[iid] = v
			case "disk_write":
				instanceDiskW[iid] = v
			}
		}
		if result.Err() != nil {
			return nil, result.Err()
		}

		count := len(instanceCPU)
		if count == 0 {
			return &FleetAggregate{}, nil
		}

		var sumCPU, sumNetIn, sumNetOut, sumDiskR, sumDiskW float64
		for id := range instanceCPU {
			sumCPU += instanceCPU[id]
			sumNetIn += instanceNetIn[id]
			sumNetOut += instanceNetOut[id]
			sumDiskR += instanceDiskR[id]
			sumDiskW += instanceDiskW[id]
		}
		n := float64(count)
		return &FleetAggregate{
			ActiveAgents:  count,
			AverageCPU:    sumCPU / n,
			AverageNet:    (sumNetIn + sumNetOut) / n / 1024,
			AverageDisk:   (sumDiskR + sumDiskW) / n / 1024,
			TotalInbound:  sumNetIn / 1024,
			TotalOutbound: sumNetOut / 1024,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return val.(*FleetAggregate), nil
}

func (dm *DataModel) GetStorageMetrics(bucketName string, start, end time.Time) ([]StorageSummary, error) {
	query := fmt.Sprintf(`from(bucket: "chaos-engineering")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "storage_metrics" and r.bucket_name == "%s")
		|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`,
		start.Format(time.RFC3339), end.Format(time.RFC3339), bucketName)

	result, err := dm.DBreader.Query(context.Background(), query)
	if err != nil {
		return nil, NewCustomError(err, http.StatusInternalServerError)
	}
	defer result.Close()

	var summaries []StorageSummary
	for result.Next() {
		record := result.Record()

		getFloat := func(field string) float64 {
			if v, ok := record.ValueByKey(field).(float64); ok {
				return v
			}
			return 0
		}

		getInt := func(field string) int {
			// Influxdb numeric values could be int64 depending on input
			switch v := record.ValueByKey(field).(type) {
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
			return 0
		}

		summaries = append(summaries, StorageSummary{
			BucketName: bucketName,
			StorageMetric: StorageMetric{
				BucketSizeBytes: getFloat("bucket_size_bytes"),
				NumberOfObjects: getInt("number_of_objects"),
				AllRequests:     getInt("all_requests"),
				GetRequests:     getInt("get_requests"),
				PutRequests:     getInt("put_requests"),
				Errors4xx:       getInt("errors_4xx"),
				Errors5xx:       getInt("errors_5xx"),
				TotalLatencyMs:  getFloat("total_latency_ms"),
			},
			Timestamp: record.Time(),
		})
	}

	if result.Err() != nil {
		return nil, NewCustomError(result.Err(), http.StatusInternalServerError)
	}

	return summaries, nil
}

func (dm *DataModel) FindbyInstanceID(instanceID string) (ComputeSummary, error) {
	cacheKey := "compute_" + instanceID

	val, err := dm.Cache.Fetch(cacheKey, 5*time.Second, func() (interface{}, error) {
		query := fmt.Sprintf(`from(bucket: "chaos-engineering")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "compute_metrics" and r.instance_id == "%s")
			|> last()
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`, instanceID)

		result, err := dm.DBreader.Query(context.Background(), query)
		if err != nil {
			return nil, NewCustomError(err, http.StatusBadRequest)
		}
		defer result.Close()

		if result.Next() {
			record := result.Record()
			getFloat := func(field string) float64 {
				if v, ok := record.ValueByKey(field).(float64); ok {
					return v
				}
				return 0
			}
			getBool := func(field string) bool {
				if v, ok := record.ValueByKey(field).(bool); ok {
					return v
				}
				return false
			}

			return ComputeSummary{
				InstanceID: instanceID,
				ComputeMetric: ComputeMetric{
					CPUUtilization:  getFloat("cpu_utilization"),
					NetworkInBytes:  getFloat("network_in"),
					NetworkOutBytes: getFloat("network_out"),
					DiskReadBytes:   getFloat("disk_read"),
					DiskWriteBytes:  getFloat("disk_write"),
					StatusFailed:    getBool("status_failed"),
				},
				Timestamp: record.Time(),
			}, nil
		}

		if result.Err() != nil {
			return nil, NewCustomError(result.Err(), http.StatusInternalServerError)
		}

		// return dummy if no data is found recently
		return ComputeSummary{InstanceID: instanceID, Timestamp: time.Now()}, nil
	})

	if err != nil {
		return ComputeSummary{}, err
	}
	return val.(ComputeSummary), nil
}



func (dm *DataModel) FindbyBucketName(bucketName string) (StorageSummary, error) {
	cacheKey := "storage_" + bucketName

	val, err := dm.Cache.Fetch(cacheKey, 5*time.Second, func() (interface{}, error) {
		query := fmt.Sprintf(`from(bucket: "chaos-engineering")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "storage_metrics" and r.bucket_name == "%s")
			|> last()
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`, bucketName)

		result, err := dm.DBreader.Query(context.Background(), query)
		if err != nil {
			return nil, NewCustomError(err, http.StatusBadRequest)
		}
		defer result.Close()

		if result.Next() {
			record := result.Record()
			getFloat := func(field string) float64 {
				if v, ok := record.ValueByKey(field).(float64); ok {
					return v
				}
				return 0
			}
			getInt := func(field string) int {
				switch v := record.ValueByKey(field).(type) {
				case int64:
					return int(v)
				case float64:
					return int(v)
				}
				return 0
			}

			return StorageSummary{
				BucketName: bucketName,
				StorageMetric: StorageMetric{
					BucketSizeBytes: getFloat("bucket_size_bytes"),
					NumberOfObjects: getInt("number_of_objects"),
					AllRequests:     getInt("all_requests"),
					GetRequests:     getInt("get_requests"),
					PutRequests:     getInt("put_requests"),
					Errors4xx:       getInt("errors_4xx"),
					Errors5xx:       getInt("errors_5xx"),
					TotalLatencyMs:  getFloat("total_latency_ms"),
				},
				Timestamp: record.Time(),
			}, nil
		}

		if result.Err() != nil {
			return nil, NewCustomError(result.Err(), http.StatusInternalServerError)
		}

		return StorageSummary{BucketName: bucketName, Timestamp: time.Now()}, nil
	})

	if err != nil {
		return StorageSummary{}, err
	}
	return val.(StorageSummary), nil
}

func ComputeSummaryToPoint(summary ComputeSummary) *write.Point {
	tags := map[string]string{
		"instance_id": summary.InstanceID,
	}
	fields := map[string]interface{}{
		"cpu_utilization": summary.CPUUtilization,
		"network_in":      summary.NetworkInBytes,
		"network_out":     summary.NetworkOutBytes,
		"disk_read":       summary.DiskReadBytes,
		"disk_write":      summary.DiskWriteBytes,
		"status_failed":   summary.StatusFailed,
	}
	return influxdb2.NewPoint("compute_metrics", tags, fields, summary.Timestamp)
}

func StorageSummaryToPoint(summary StorageSummary) *write.Point {
	tags := map[string]string{
		"bucket_name": summary.BucketName,
	}
	fields := map[string]interface{}{
		"bucket_size_bytes": summary.BucketSizeBytes,
		"number_of_objects": summary.NumberOfObjects,
		"all_requests":      summary.AllRequests,
		"get_requests":      summary.GetRequests,
		"put_requests":      summary.PutRequests,
		"errors_4xx":        summary.Errors4xx,
		"errors_5xx":        summary.Errors5xx,
		"total_latency_ms":  summary.TotalLatencyMs,
	}
	return influxdb2.NewPoint("storage_metrics", tags, fields, summary.Timestamp)
}

func (dm *DataModel) PushExperimentResult(
	experimentID, ruleID, expType, outcome, severity, targetID, impact string,
	durationS float64,
	reportID string,
	startTime time.Time,
) error {
	tags := map[string]string{
		"exp_type": expType,
		"outcome":  outcome,
		"rule_id":  ruleID,
		"severity": severity, // denormalized — avoids a join just to get severity
	}
	fields := map[string]interface{}{
		"experiment_id": experimentID,
		"target_id":     targetID,
		"impact":        impact,
		"duration_s":    durationS,
		"report_id":     reportID,
	}
	return dm.PushCloudWatchMetric(MeasurementExperiment, tags, fields, startTime)
}

func (dm *DataModel) FindUntestedRules(reportID string) ([]string, error) {
	query := fmt.Sprintf(`
from(bucket: "chaos-engineering")
  |> range(start: -30d)
  |> filter(fn: (r) => r._measurement == "%s")
  |> filter(fn: (r) => r._field == "report_id" and r._value == "%s")
  |> filter(fn: (r) => r.outcome != "confirmed")
  |> keep(columns: ["rule_id"])
  |> distinct(column: "rule_id")
`, MeasurementExperiment, reportID)
	result, err := dm.DBreader.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer result.Close()
	var ruleIDs []string
	for result.Next() {
		if v, ok := result.Record().ValueByKey("rule_id").(string); ok {
			ruleIDs = append(ruleIDs, v)
		}
	}
	return ruleIDs, result.Err()
}
