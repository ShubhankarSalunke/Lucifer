# Architecture

This document describes the high-level architecture of Lucifer, including its components, data flow, and deployment patterns.

## System Overview

Lucifer is a distributed DevSecOps platform consisting of multiple components that work together to provide chaos engineering and security auditing capabilities.

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     CLI         │    │  Orchestrator   │    │     Agents      │
│                 │    │                 │    │                 │
│ • User Interface│◄──►│ • Control Plane │◄──►│ • Chaos Execution│
│ • Command Parser│    │ • API Server    │    │ • Monitoring     │
│ • Result Display│    │ • Experiment Mgmt│    │ • Reporting     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Security Audit  │
                    │                 │
                    │ • Rule Engine   │
                    │ • Scanner       │
                    │ • Report Gen    │
                    └─────────────────┘
```

## Core Components

### CLI (Command Line Interface)

**Location:** `cli/`
**Language:** Go
**Purpose:** User interaction and command processing

The CLI provides:
- User authentication and session management
- Command parsing and validation
- Result formatting and display
- Configuration management

### Orchestrator

**Location:** `chaos-engineering/orchestrator/`
**Language:** Go
**Purpose:** Central control plane and experiment coordination

Key responsibilities:
- User and agent management
- Experiment scheduling and execution
- Result aggregation and storage
- API endpoint management
- Cloud resource interactions (AWS, etc.)

### Agents

**Location:** `chaos-engineering/agent/`
**Language:** Go
**Purpose:** Execute chaos experiments on target systems

Agent capabilities:
- Host-level chaos injection (CPU, memory, network)
- System monitoring and metrics collection
- Result reporting to orchestrator
- Auto-registration and heartbeat

### Security Audit Module

**Location:** `security-audit/`
**Language:** Go
**Purpose:** Vulnerability assessment and penetration testing

Components:
- **Scanner:** Discovers and analyzes cloud resources
- **Rule Engine:** Evaluates resources against security rules
- **Evaluator:** Processes findings and generates reports

## Data Flow

### Experiment Execution Flow

1. **User Request**
   - CLI sends experiment creation request to Orchestrator API

2. **Validation & Scheduling**
   - Orchestrator validates parameters and user permissions
   - Stores experiment metadata in local JSON storage

3. **Agent Assignment**
   - For host-level experiments: Orchestrator polls assigned agent
   - For cloud-level experiments: Orchestrator executes directly

4. **Execution**
   - Agent/Orchestrator applies chaos injection
   - Monitors system behavior during experiment

5. **Reversion & Reporting**
   - Automatic reversion (where applicable)
   - Results sent back to Orchestrator
   - User notified via CLI

### Security Audit Flow

1. **Scan Initiation**
   - CLI requests audit scan for specific services

2. **Resource Discovery**
   - Audit module queries cloud APIs for resources

3. **Rule Evaluation**
   - Each resource evaluated against relevant rules

4. **Report Generation**
   - Findings compiled into structured reports
   - Results returned to CLI

## Storage Architecture

### Local Storage (JSON Files)

The orchestrator uses simple JSON file storage for:
- User accounts and authentication tokens
- Agent registrations and mappings
- Experiment definitions and results

**Files:**
- `users.json` - User accounts and tokens
- `agents.json` - Registered agents and status
- `experiments.json` - Experiment definitions and status
- `mapping.json` - User-agent relationships

### Cloud Integration

- **AWS SDK v2**: Direct integration for cloud experiments
- **Connectors**: Abstraction layer for multi-cloud support
- **Credentials**: Secure credential management via CLI

## Deployment Patterns

### Development Setup

```
Local Machine
├── CLI (built binary)
├── Orchestrator (local server)
└── Agents (deployed to target systems)
```

### Production Setup

```
Cloud Environment
├── Orchestrator (containerized, scalable)
├── Agents (deployed via automation)
├── Load Balancer (API gateway)
└── Monitoring (integrated metrics)
```

### Hybrid Setup

```
Corporate Network
├── Orchestrator (on-premises)
├── Agents (distributed across environments)
└── CLI (developer workstations)
```

## Security Considerations

### Authentication & Authorization

- Token-based authentication for API access
- User-agent mapping for experiment isolation
- AWS IAM integration for cloud access

### Data Protection

- Local encryption for sensitive configuration
- Secure credential handling
- Audit logging for all operations

### Network Security

- HTTPS for API communications
- Agent-orchestrator secure channels
- Least-privilege access patterns

## Scalability

### Horizontal Scaling

- Multiple orchestrator instances behind load balancer
- Agent pools for high-volume experiments
- Distributed result aggregation

### Performance Optimization

- Asynchronous experiment execution
- Efficient polling mechanisms
- Cached audit rule evaluation

## Monitoring & Observability

### Metrics Collection

- Experiment success/failure rates
- System resource utilization
- API response times

### Logging

- Structured logging across all components
- Error tracking and alerting
- Audit trails for compliance

## Future Enhancements

- Database backend for better scalability
- Kubernetes operator for automated deployments
- Multi-cloud support expansion
- Advanced analytics and reporting
- Integration with CI/CD pipelines