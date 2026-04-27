package datamodel

import (
	"context"
	"fmt"
	"log"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type ChaosMode bool
const (
    SteadyState ChaosMode = false 
    ActiveChaos ChaosMode = true  
)

func WindowScript(mode ChaosMode) string {
    if mode == ActiveChaos {
        return AggregateWindow("1m")
    }
    return AggregateWindow("5m")
}

func AggregateTask(serverURL, token, orgID string){
	client := influxdb2.NewClient(serverURL, token)
	taskAPI := client.TasksAPI()
	fluxScript := `from(bucket: "raw_cloudwatch_metrics")
		|> range(start: -5m)
		|> filter(fn: (r) => r._measurement == "compute_metrics" and r._field == "cpu_utilization")
		|> to(bucket: "minimalist_storage")
	`

	_, err := taskAPI.CreateTaskWithEvery(context.Background(), "downsample_cpu", fluxScript, "5m", orgID)
	if err != nil {
		log.Printf("Note: Aggregation task setup (already exists): %v\n", err)
	}
}

func AggregateWindow(every string) string {

	// CPU: mean smooths transient spikes; max() can be used for burst detection.
	computeCPU := fmt.Sprintf(`
// EC2 – compute: cpu_utilization (mean over window)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "compute_metrics" and r._field == "cpu_utilization")
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)
  |> yield(name: "compute_cpu_mean")
`, every)

	// Network: mean throughput is the standard research metric for bandwidth.
	computeNetwork := fmt.Sprintf(`
// EC2 – compute: network_in / network_out (mean throughput per window)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "compute_metrics" and
      (r._field == "network_in" or r._field == "network_out"))
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)
  |> yield(name: "compute_network_mean")
`, every)

	// Disk I/O: mean reflects sustained read/write load per window.
	computeDisk := fmt.Sprintf(`
// EC2 – compute: disk_read / disk_write (mean I/O bytes per window)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "compute_metrics" and
      (r._field == "disk_read" or r._field == "disk_write"))
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)
  |> yield(name: "compute_disk_mean")
`, every)

	// Bucket size & object count are gauges — last() preserves the final state.
	storageGauge := fmt.Sprintf(`
// S3 – storage: bucket_size_bytes & number_of_objects (last gauge value)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "storage_metrics" and
      (r._field == "bucket_size_bytes" or r._field == "number_of_objects"))
  |> aggregateWindow(every: %s, fn: last, createEmpty: false)
  |> yield(name: "storage_gauge_last")
`, every)

	// Request counts: sum() accumulates discrete events within each window.
	storageRequests := fmt.Sprintf(`
// S3 – storage: request counts (sum per window for throughput KPIs)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "storage_metrics" and
      (r._field == "all_requests" or r._field == "get_requests" or r._field == "put_requests"))
  |> aggregateWindow(every: %s, fn: sum, createEmpty: false)
  |> yield(name: "storage_requests_sum")
`, every)

	// Errors: sum() gives total error count; drives error-rate = errors/requests.
	storageErrors := fmt.Sprintf(`
// S3 – storage: error counts (sum per window; used in error-rate SLO)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "storage_metrics" and
      (r._field == "errors_4xx" or r._field == "errors_5xx"))
  |> aggregateWindow(every: %s, fn: sum, createEmpty: false)
  |> yield(name: "storage_errors_sum")
`, every)

	// Latency: mean gives average response time; swap fn: max for worst-case SLO.
	storageLatency := fmt.Sprintf(`
// S3 – storage: total_latency_ms (mean per window; swap max for SLO breach detection)
from(bucket: "chaos-engineering")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "storage_metrics" and r._field == "total_latency_ms")
  |> aggregateWindow(every: %s, fn: mean, createEmpty: false)
  |> yield(name: "storage_latency_mean")
`, every)

	return computeCPU + computeNetwork + computeDisk +
		storageGauge + storageRequests + storageErrors + storageLatency
}