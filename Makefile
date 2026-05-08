.PHONY: run-all gateway datamodel ui ui-help metrics agent

agent:
	@echo "Starting Chaos Agent..."
	@cd chaos-engineering/agent && go run agent.go

ui:
	@echo "Starting CLI UI..."
	@sleep 2
	@cd UI && go run cmd/main.go $(ARGS)

ui-help:
	@cd UI && go run cmd/main.go --help

build:
	@echo "Building Lucifer Toolset..."
	@mkdir -p bin
	@echo " -> Compiling CLI (lucifer)..."
	@cd cli && go build -o ../bin/lucifer main.go
	@echo " -> Compiling Datamodel Server..."
	@cd datamodel && go build -o ../bin/lucifer-datamodel cmd/server/main.go
	@echo " -> Compiling UI..."
	@cd UI && go build -o ../bin/lucifer-ui cmd/main.go
	@echo " Build complete! Binaries are in the 'bin/' directory."

install: build
	@sudo cp bin/lucifer /usr/local/bin/lucifer
	@sudo cp bin/lucifer-datamodel /usr/local/bin/lucifer-datamodel
	@sudo cp bin/lucifer-ui /usr/local/bin/lucifer-ui
	@echo " Installation complete"

clean:
	@echo "Cleaning up ghost processes on ports 8000, 8001..."
	@lsof -ti:8000,8001 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run" || true
	@echo "All services stopped."
