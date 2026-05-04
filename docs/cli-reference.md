# CLI Reference

The `lucifer` CLI is the primary interface for managing your DevSecOps infrastructure, triggering Chaos Engineering experiments, and running VAPT security audits.

## Installation & Setup

Before running the CLI, ensure it is built from the `cli/` directory:

```bash
cd cli
go build -o lucifer-cli .
```

*Note: In the examples below, the binary is referred to as `lucifer` or `./lucifer-cli`.*

## Auth & Configuration Commands

To use the Orchestrator safely in a multi-tenant or enterprise environment, you must authenticate and link your cloud credentials.

### `signup`
Create a new user account and store your generated API token locally.
```bash
lucifer signup
```

### `login`
Authenticate with an existing API token to resume managing your experiments.
```bash
lucifer login
```

### `aws-connect`
Securely configure your AWS credentials to ensure the agent and orchestrator can interact with your AWS infrastructure securely. This adheres to IAM least-privilege principles.
```bash
lucifer aws-connect
```

## Control Plane Commands

Commands used to manage the centralized orchestrator Server and API gateway connections.

### `start`
Boot up the central control plane (Orchestrator) locally for debugging or testing.
```bash
lucifer start
```

### `expose`
Expose the orchestrator control plane via an API Gateway or local tunnel to allow remote agents to ping back results securely.
```bash
lucifer expose
```

## Chaos Engineering Commands

These commands manage your failure injection framework.

### `create-agent`
Registers a new chaos agent. The Orchestrator will issue instructions to this agent.
```bash
lucifer create-agent --id <agent-id>
```
**Flags:**
* `--id string` - The unique identifier you assign to this target agent.

### `create-experiment`
Creates and triggers a chaos experiment on a target agent. This is the core command for stress testing.

```bash
lucifer create-experiment --agent "agent-01" --type "network-latency" --latency 500 --duration 60
```

**Common Flags:**
* `--type string` - Experiment type (e.g., `cpu_stress`, `memory_stress`, `network_latency`, `s3_access_deny`).
* `--agent string` - The ID of the target agent to strike (not required for cloud-level experiments like S3).
* `--duration int` - How long the experiment should run in seconds. (Default: 30)

**Resource Stress Flags:**
* `--cpu int` - Inject extreme load peaking at this CPU % (1-100).
* `--memory int` - Hog the specified Memory in MB.
* `--target string` - Target a specific container ID or process.

**Network Flags:**
* `--latency int` - Introduce this latency (in ms) into network traffic.

**AWS S3 Flags:**
* `--bucket string` - Target S3 bucket name.
* `--percent int` - Percentage of objects to affect (1-100, default: 10).
* `--prefix string` - Limit operations to objects with this prefix.
* `--kms-key string` - KMS key ID for encryption-related experiments.

**IAM Flags:**
* `--role-arn string` - The IAM Role ARN to assume.
* `--external-id string` - The External ID for role assumption.

**Available Experiment Types:**
- `cpu_stress` - CPU load injection
- `memory_stress` - Memory exhaustion
- `network_latency` - Network delay injection
- `s3_access_deny` - Deny access to S3 bucket
- `s3_kms_disable` - Disable KMS key for S3
- `s3_object_delete` - Delete percentage of S3 objects
- `s3_metadata_corrupt` - Corrupt S3 object metadata

### `results`
Fetch, parse, and display the detailed results and blast radii from previously run chaos experiments.
```bash
lucifer results
```

## VAPT & Security Audit Commands

These commands interface with the dedicated Go Security Audit module.

### `audit scan`
Execute a comprehensive security posture scan against a targeted environment or component to spot misconfigurations before a Chaos attack.
```bash
lucifer audit scan aws ec2
lucifer audit scan aws s3
lucifer audit scan aws all
```

### `audit rules`
List, fetch, and describe the internal auditing rules dictating the scan engine. Use this to verify which AWS controls are actively enforced.
```bash
lucifer audit rules
```

## Global Flags
* `-h, --help` - Output detailed helper text for any command sequence.

*Usage Example:*
`lucifer create-experiment --help`