# Chaos Experiments

Chaos Engineering is the practice of intentionally injecting failures into systems to test their resilience and identify weaknesses. Lucifer provides a comprehensive set of chaos experiments for various failure scenarios.

## Experiment Types

### Host-Level Experiments

These experiments run on deployed agents and affect the host system or containers.

#### CPU Stress (`cpu_stress`)
Injects high CPU load to test system performance under stress.

**Parameters:**
- `cpu_percent`: CPU utilization percentage (1-100)

**Example:**
```bash
lucifer create-experiment --agent "agent-01" --type cpu_stress --cpu 90 --duration 60
```

#### Memory Stress (`memory_stress`)
Consumes large amounts of memory to test memory management and OOM handling.

**Parameters:**
- `memory_mb`: Memory to consume in MB

**Example:**
```bash
lucifer create-experiment --agent "agent-01" --type memory_stress --memory 1024 --duration 30
```

#### Network Latency (`network_latency`)
Introduces artificial network delays to test application resilience to slow networks.

**Parameters:**
- `latency_ms`: Network latency to inject in milliseconds

**Example:**
```bash
lucifer create-experiment --agent "agent-01" --type network_latency --latency 500 --duration 120
```

### Cloud-Level Experiments

These experiments operate directly on cloud infrastructure without requiring agents.

#### S3 Access Deny (`s3_access_deny`)
Applies a bucket policy that denies access to an S3 bucket for a specified duration.

**Parameters:**
- `bucket_name`: Target S3 bucket
- `duration`: How long to deny access (seconds)

**Example:**
```bash
lucifer create-experiment --type s3_access_deny --bucket my-bucket --duration 300
```

**Note:** Ensure your AWS credentials have `s3:DeleteBucketPolicy` permission to revert the policy.

#### S3 KMS Disable (`s3_kms_disable`)
Temporarily disables a KMS key used for S3 server-side encryption.

**Parameters:**
- `kms_key_id`: KMS key ID to disable
- `duration`: How long to disable the key (seconds)

**Example:**
```bash
lucifer create-experiment --type s3_kms_disable --kms-key alias/my-key --duration 60
```

#### S3 Object Delete (`s3_object_delete`)
Permanently deletes a percentage of objects from an S3 bucket.

**Parameters:**
- `bucket_name`: Target S3 bucket
- `delete_percent`: Percentage of objects to delete (1-100, default: 10)
- `prefix`: Optional prefix to limit deletion scope

**Example:**
```bash
lucifer create-experiment --type s3_object_delete --bucket my-bucket --percent 5 --prefix logs/
```

**Warning:** This experiment causes permanent data loss. Use with caution!

#### S3 Metadata Corruption (`s3_metadata_corrupt`)
Corrupts the metadata of S3 objects by changing their content type.

**Parameters:**
- `bucket_name`: Target S3 bucket
- `duration`: How long to keep corrupted metadata (seconds)
- `prefix`: Optional prefix to limit scope

**Example:**
```bash
lucifer create-experiment --type s3_metadata_corrupt --bucket my-bucket --duration 300
```

## Best Practices

1. **Start Small**: Begin with short-duration, low-impact experiments
2. **Monitor Closely**: Use monitoring tools to observe system behavior during experiments
3. **Have Rollback Plans**: Ensure you can quickly recover from any unexpected issues
4. **Test in Staging**: Run experiments in non-production environments first
5. **Document Findings**: Record what you learn from each experiment

## Safety Measures

- All experiments include automatic reversion where applicable
- Experiments validate parameters before execution
- Failed experiments are logged for analysis
- The orchestrator maintains experiment history and results

## Monitoring Results

After running experiments, check results with:

```bash
lucifer results
```

This will show experiment status, duration, and any errors encountered.