# API Reference

The Lucifer Orchestrator provides a REST API for programmatic access to chaos engineering and security auditing features.

## Base URL

```
http://localhost:8000
```

## Authentication

All API requests require authentication via Bearer token in the Authorization header:

```
Authorization: Bearer <your-token>
```

## Endpoints

### User Management

#### POST /create-user

Create a new user account.

**Request Body:**
```json
{
  "user_id": "optional-custom-id"
}
```

**Response:**
```json
{
  "user_id": "generated-or-provided-id",
  "token": "authentication-token"
}
```

### Agent Management

#### POST /register

Register an agent with the orchestrator.

**Request Body:**
```json
{
  "verification_token": "token-from-create-agent",
  "host": "agent-hostname-or-ip"
}
```

**Response:**
```json
{
  "agent_id": "agent-identifier",
  "user_id": "associated-user-id"
}
```

#### POST /create-agent

Create a new agent registration.

**Request Body:**
```json
{
  "agent_id": "unique-agent-identifier"
}
```

**Response:**
```json
{
  "agent_id": "agent-identifier",
  "verification_token": "token-for-agent-registration"
}
```

#### GET /poll/:agent_id

Agent polling endpoint for receiving experiment instructions.

**Response (when experiment available):**
```json
{
  "id": "experiment-id",
  "type": "experiment-type",
  "duration": 60,
  "parameters": {
    "cpu_percent": 80,
    "memory_mb": 1024
  }
}
```

**Response (no experiments):**
```json
{}
```

#### GET /agents

List all agents for the authenticated user.

**Response:**
```json
{
  "agent-1": {
    "host": "host1.example.com",
    "status": "online",
    "last_seen": "2024-01-01T12:00:00Z"
  },
  "agent-2": {
    "host": "host2.example.com",
    "status": "offline",
    "last_seen": "2024-01-01T11:45:00Z"
  }
}
```

### Experiment Management

#### POST /create-experiment

Create and start a new chaos experiment.

**Request Body:**
```json
{
  "type": "cpu_stress",
  "agent_id": "target-agent-id",
  "duration": 60,
  "cpu_percent": 80,
  "memory_mb": 1024,
  "latency_ms": 500,
  "bucket_name": "s3-bucket",
  "kms_key_id": "kms-key-id",
  "delete_percent": 10,
  "prefix": "object-prefix",
  "role_arn": "iam-role-arn",
  "external_id": "external-id",
  "access_key": "aws-access-key",
  "secret_key": "aws-secret-key",
  "region": "aws-region"
}
```

**Response:**
```json
{
  "experiment_id": "generated-experiment-id"
}
```

#### GET /experiments

List all experiments for the authenticated user.

**Response:**
```json
{
  "exp-1": {
    "type": "cpu_stress",
    "agent_id": "agent-1",
    "status": "completed",
    "created_at": "2024-01-01T12:00:00Z",
    "duration": 60,
    "result": {
      "success": true,
      "metrics": {...}
    }
  }
}
```

#### GET /results

Get results for completed experiments.

**Response:**
```json
{
  "exp-1": {
    "experiment_id": "exp-1",
    "status": "completed",
    "result": {
      "success": true,
      "details": "Experiment completed successfully"
    }
  }
}
```

### Result Submission

#### POST /result

Submit experiment results (called by agents).

**Request Body:**
```json
{
  "experiment_id": "experiment-id",
  "status": "completed|failed",
  "result": {
    "custom": "result data"
  }
}
```

**Response:**
```json
{
  "message": "result recorded"
}
```

## Experiment Types

### Host-Level Experiments

- `cpu_stress`: CPU load injection
  - Parameters: `cpu_percent` (1-100)
- `memory_stress`: Memory consumption
  - Parameters: `memory_mb` (MB to consume)
- `network_latency`: Network delay
  - Parameters: `latency_ms` (delay in ms)

### Cloud-Level Experiments

- `s3_access_deny`: Deny S3 bucket access
  - Parameters: `bucket_name`, `duration`
- `s3_kms_disable`: Disable KMS key
  - Parameters: `kms_key_id`, `duration`
- `s3_object_delete`: Delete S3 objects
  - Parameters: `bucket_name`, `delete_percent`, `prefix`
- `s3_metadata_corrupt`: Corrupt S3 metadata
  - Parameters: `bucket_name`, `duration`, `prefix`

## Error Responses

All endpoints return standard HTTP status codes:

- `200`: Success
- `400`: Bad Request (invalid parameters)
- `401`: Unauthorized (invalid/missing token)
- `500`: Internal Server Error

Error response format:
```json
{
  "error": "error description"
}
```

## Rate Limiting

- API requests are rate-limited per user
- Experiment creation is limited to prevent abuse
- Failed authentication attempts are tracked

## WebSocket Support

For real-time experiment monitoring, WebSocket connections are available at:

```
ws://localhost:8000/ws/experiments
```

## SDKs and Libraries

Official SDKs are available for:
- Python
- JavaScript/Node.js
- Go

Community-contributed libraries:
- Ruby
- Java
- .NET