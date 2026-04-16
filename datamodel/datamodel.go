package datamodel

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type DataModel struct {
	client   influxdb2.Client
	DBwriter api.WriteAPI
	DBreader api.QueryAPI
	Cache    *QueryCache
	bucket   string
}

func NewDataModel(serverURL, token, org, bucket string) (*DataModel, error) {
	client := influxdb2.NewClient(serverURL, token)
	writer := client.WriteAPI(org, bucket)

	// Start background error logger for the async writer
	go func() {
		for err := range writer.Errors() {
			fmt.Printf("[InfluxDB Error] %v\n", err)
		}
	}()

	reader := client.QueryAPI(org)
	return &DataModel{
		client:   client,
		DBwriter: writer,
		DBreader: reader,
		Cache:    NewQueryCache(),
		bucket:   bucket,
	}, nil
}

func (dm *DataModel) Close() {
	dm.DBwriter.Flush()
	dm.client.Close()
}

// This only pushes data over HTTP, it does NOT interact with AWS APIs directly and writes a single data point
func (dm *DataModel) PushCloudWatchMetric(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error {
	p := influxdb2.NewPoint(measurement, tags, fields, timestamp)

	// Async batched append. Errors are processed in a background Go routine by the client.
	dm.DBwriter.WritePoint(p)
	return nil
}

// GetDiscoveredInstances queries InfluxDB for a list of unique instance IDs that have pushed metrics in the last 24h.
func (dm *DataModel) GetDiscoveredInstances() ([]string, error) {
	val, err := dm.Cache.Fetch("discovered_instances", 15*time.Second, func() (interface{}, error) {
		// instance_id is a TAG, so we group by it to get one series per instance,
		// then take just the last() row from each series to get a minimal result set.
		query := fmt.Sprintf(`from(bucket: "%s")
			|> range(start: -24h)
			|> filter(fn: (r) => r._measurement == "compute_metrics" and r._field == "cpu_utilization")
			|> group(columns: ["instance_id"])
			|> last()
			|> keep(columns: ["instance_id"])`, dm.bucket)

		result, err := dm.DBreader.Query(context.Background(), query)
		if err != nil {
			return nil, err
		}
		defer result.Close()

		seen := make(map[string]bool)
		instances := make([]string, 0)
		for result.Next() {
			// Tags are accessible via ValueByKey on the grouped record
			instanceID := fmt.Sprintf("%v", result.Record().ValueByKey("instance_id"))
			if instanceID != "" && instanceID != "<nil>" && !seen[instanceID] {
				seen[instanceID] = true
				instances = append(instances, instanceID)
			}
		}

		if err := result.Err(); err != nil {
			return nil, err
		}

		return instances, nil
	})
	if err != nil {
		return nil, err
	}

	return val.([]string), nil
}
