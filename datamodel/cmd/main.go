package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ShubhankarSalunke/chaos-engineering/datamodel"
	"github.com/joho/godotenv"
)

func main() {
	// Try multiple paths to find the .env file
	for _, p := range []string{"../../.env", "../.env", ".env", "../../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	url := os.Getenv("INFLUX_URL")
	if url == "" {
		url = "http://localhost:8086"
	}
	token := os.Getenv("INFLUX_TOKEN")

	org := os.Getenv("INFLUX_ORG")

	bucket := os.Getenv("INFLUX_BUCKET")

	dm, err := datamodel.NewDataModel(url, token, org, bucket)
	if err != nil {
		log.Fatalf("Failed to initialize DataModel: %v", err)
	}
	log.Printf("Datamodel Connected: InfluxDB at %s (Bucket: %s)", url, bucket)
	defer dm.Close()

	orc := datamodel.NewOrchestrator(dm)

	log.Println("Setting up InfluxDB Retention Policies...")
	if err := datamodel.ApplyRetention(url, token, org, bucket, datamodel.RetentionSummary); err != nil {
		log.Printf("Warning: Failed to apply retention policy: %v\n", err)
	}

	log.Println("Registering Background Aggregation Tasks...")
	datamodel.AggregateTask(url, token, org)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Datamodel Orchestrator Server running on :%s\n", port)
	if err := http.ListenAndServe(":"+port, orc.Router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
