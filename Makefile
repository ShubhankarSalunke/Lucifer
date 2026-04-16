.PHONY: run-all gateway datamodel ui ui-help metrics agent

gateway:
	@echo "Starting API Gateway (Port 8000)..."
	@cd backend-go/orchestrator && PORT=8000 go run .

datamodel:
	@echo "Starting Datamodel Server (Port 8001)..."
	@cd datamodel && PORT=8001 go run cmd/main.go

metrics:
	@echo "Starting CloudWatch Metrics Fetcher..."
	@cd datamodel && go run cmd/aws_fetcher/main.go

agent:
	@echo "Starting Chaos Agent..."
	@cd backend-go/agent && go run agent.go

ui:
	@echo "Starting CLI UI..."
	@sleep 2
	@cd UI && go run cmd/main.go $(ARGS)

ui-help:
	@cd UI && go run cmd/main.go --help

clean:
	@echo "Cleaning up ghost processes on ports 8000, 8001..."
	@lsof -ti:8000,8001 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run" || true
	@echo "All services stopped."
