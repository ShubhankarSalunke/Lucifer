# Getting Started

Welcome to Lucifer! This guide will help you get up and running with the DevSecOps platform for Chaos Engineering and Security Auditing.

## Prerequisites

- Go 1.19 or later
- AWS CLI configured (for AWS experiments)
- Docker (optional, for containerized deployment)

## Installation

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/your-org/lucifer.git
cd lucifer
```

2. Build the CLI:
```bash
cd cli
go build -o lucifer-cli .
```

3. Build the orchestrator:
```bash
cd ../chaos-engineering/orchestrator
go build -o orchestrator main.go storage.go
```

4. Build the agent:
```bash
cd ../agent
go build -o agent agent.go
```

### Using Pre-built Binaries

Download the latest releases from the [GitHub releases page](https://github.com/your-org/lucifer/releases).

## Quick Start

1. **Sign up for an account:**
```bash
./lucifer-cli signup
```

2. **Configure AWS credentials:**
```bash
./lucifer-cli aws-connect
```

3. **Start the orchestrator:**
```bash
./lucifer-cli start
```

4. **Create your first chaos experiment:**
```bash
./lucifer-cli create-experiment --type cpu_stress --cpu 80 --duration 30
```

## Architecture Overview

Lucifer consists of several components:

- **CLI**: Command-line interface for user interaction
- **Orchestrator**: Central control plane that manages experiments
- **Agents**: Deployed on target systems to execute chaos experiments
- **Security Audit Module**: Performs vulnerability scanning and penetration testing

## Next Steps

- [Read the CLI Reference](cli-reference.md) for detailed command usage
- [Learn about Chaos Experiments](chaos-experiments.md) to understand available failure injection types
- [Explore Security Auditing](security-auditing.md) for vulnerability assessment features