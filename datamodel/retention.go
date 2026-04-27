package datamodel

import (
	"context"
	"fmt"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)


//RetentionRaw = 3 days covers daily metric data + experiment metric data. 
// RetentionVulnReport = 0 (infinite) because the report is what drives the next experiment — you never want to lose it.

const (
    RetentionRaw          int64 = 3 * 24 * 60 * 60   // 3 days 
    RetentionSummary      int64 = 30 * 24 * 60 * 60   // 30 days 

    MeasurementExperiment = "chaos_experiment"
)


func ApplyRetention(serverURL, token, orgID, bucketID string, retentionSeconds int64) error {
    client := influxdb2.NewClient(serverURL, token)
    defer client.Close()

    bucketsAPI := client.BucketsAPI()

    bucket, err := bucketsAPI.FindBucketByName(context.Background(), bucketID)
    if err != nil {
        return fmt.Errorf("failed to find bucket '%s': %w", bucketID, err)
    }

    ruleType := domain.RetentionRuleType("expire")

    bucket.RetentionRules = domain.RetentionRules{
        {
            Type:         &ruleType,
            EverySeconds: retentionSeconds,
        },
    }

    _, err = bucketsAPI.UpdateBucket(context.Background(), bucket)
    if err != nil {
        return fmt.Errorf("failed to apply retention: %w", err)
    }
    return nil
}
