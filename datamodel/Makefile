.PHONY: build tidy up run-server run-fetcher trigger-chaos stop-chaos view-metrics


build:
	go build ./...

tidy:
	go mod tidy


# Start the core orchestrator API server
run-server:
	@echo "Starting Datamodel Orchestrator on :8080..."
	go run ./cmd/main.go

# Start the AWS CloudWatch 10s/120s fetching script
run-fetcher:
	@echo "Starting AWS CloudWatch Fetcher Simulator..."
	@if [ -z "$$AWS_INSTANCE_ID" ]; then \
		echo "ERROR: Please set AWS_INSTANCE_ID environment variable."; \
		echo "Example: export AWS_INSTANCE_ID='i-01234abcd5678'"; \
		exit 1; \
	fi
	go run ./cmd/aws_fetcher/main.go


# Instantly trigger high-speed 10s real-time fetching for a live experiment
trigger-chaos:
	@echo "Triggering ACTIVE chaos experiment status..."
	curl -X POST http://localhost:8080/experiment/status \
		 -H "Content-Type: application/json" \
		 -d '{"active": true}'
	@echo ""

# Fall back gracefully into the 2m sleepy baseline interval
stop-chaos:
	@echo "Stopping chaos experiment status..."
	curl -X POST http://localhost:8080/experiment/status \
		 -H "Content-Type: application/json" \
		 -d '{"active": false}'
	@echo ""

# Pull the cache's live view of your EC2 metrics
view-metrics:
	@if [ -z "$$AWS_INSTANCE_ID" ]; then \
		echo "ERROR: Please set AWS_INSTANCE_ID environment variable to view."; \
		exit 1; \
	fi
	curl -s http://localhost:8080/api/metrics/compute/$$AWS_INSTANCE_ID | jq .
