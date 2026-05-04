# Troubleshooting

This guide helps you resolve common issues when using Lucifer.

## Installation Issues

### Go Build Failures

**Problem:** `go build` fails with missing dependencies

**Solution:**
```bash
# Clean module cache
go clean -modcache

# Download dependencies
go mod download

# Tidy modules
go mod tidy

# Try build again
go build
```

### Permission Denied

**Problem:** Cannot execute built binaries

**Solution:**
```bash
# Make binary executable
chmod +x lucifer-cli
chmod +x orchestrator
```

## Authentication Issues

### Invalid Token

**Problem:** API requests return 401 Unauthorized

**Solutions:**
1. Verify token is correct: `lucifer login`
2. Check token hasn't expired
3. Ensure token has Bearer prefix in API calls

### User Not Found

**Problem:** User authentication fails

**Solutions:**
1. Re-run signup: `lucifer signup`
2. Check users.json file exists and is readable
3. Verify orchestrator is running

## Agent Issues

### Agent Not Connecting

**Problem:** Agent fails to register or connect

**Solutions:**
1. Check agent binary is executable
2. Verify orchestrator is running and accessible
3. Check network connectivity between agent and orchestrator
4. Review agent logs for error messages

### Agent Polling Fails

**Problem:** Agent cannot poll for experiments

**Solutions:**
1. Ensure agent ID is correct
2. Check orchestrator API is responding
3. Verify agent is registered with correct user
4. Check agents.json file for agent status

## Experiment Issues

### Experiment Creation Fails

**Problem:** `create-experiment` returns errors

**Common Issues:**
- Invalid experiment type
- Missing required parameters
- Agent not found or offline
- Duration too short/long

**Solutions:**
1. Check experiment parameters: `lucifer create-experiment --help`
2. Verify agent is online: `lucifer agents`
3. Ensure AWS credentials are configured for cloud experiments

### S3 Experiments Fail

**Problem:** S3 chaos experiments don't execute

**Solutions:**
1. Verify AWS credentials: `aws configure`
2. Check IAM permissions for S3 operations
3. Ensure bucket exists and is accessible
4. For `s3_access_deny`, ensure credentials have `s3:DeleteBucketPolicy` permission

### CPU/Memory Stress Not Working

**Problem:** Host-level experiments don't affect system

**Solutions:**
1. Verify agent is deployed on target system
2. Check agent has necessary permissions (root/sudo)
3. Ensure target container/process exists
4. Review agent logs for execution errors

## AWS Integration Issues

### Credentials Not Found

**Problem:** AWS operations fail with credential errors

**Solutions:**
1. Configure AWS CLI: `aws configure`
2. Use `lucifer aws-connect` for Lucifer-specific credentials
3. Check credential file permissions
4. Verify IAM user has required permissions

### Region Mismatch

**Problem:** Resources not found in specified region

**Solutions:**
1. Check AWS region configuration
2. Ensure resources exist in the specified region
3. Update default region if needed

### IAM Permission Errors

**Problem:** AWS API calls fail with access denied

**Solutions:**
1. Review IAM policies attached to user/role
2. Use least-privilege principles
3. Check for explicit denies in policies
4. Verify resource ownership

## Security Audit Issues

### Scan Fails

**Problem:** `audit scan` commands fail

**Solutions:**
1. Check AWS credentials and permissions
2. Verify target services exist
3. Ensure audit module is built correctly
4. Check for network connectivity to AWS APIs

### No Findings Reported

**Problem:** Audit scan returns empty results

**Solutions:**
1. Verify resources exist in the account
2. Check if resources are in supported regions
3. Ensure proper IAM permissions for resource enumeration
4. Review audit rule configurations

## Network Issues

### Connection Refused

**Problem:** Cannot connect to orchestrator API

**Solutions:**
1. Verify orchestrator is running: `ps aux | grep orchestrator`
2. Check port 8000 is not blocked by firewall
3. Ensure correct host/IP address
4. Try localhost vs 0.0.0.0 binding

### Timeout Errors

**Problem:** API calls timeout

**Solutions:**
1. Check network connectivity
2. Increase timeout values if needed
3. Verify orchestrator performance
4. Check for resource constraints

## Performance Issues

### High CPU/Memory Usage

**Problem:** Lucifer components consume excessive resources

**Solutions:**
1. Monitor system resources during experiments
2. Adjust experiment parameters
3. Check for memory leaks in long-running processes
4. Optimize Go garbage collection settings

### Slow Experiment Execution

**Problem:** Experiments take longer than expected

**Solutions:**
1. Check network latency to target systems
2. Verify system resources on target hosts
3. Review experiment parameters
4. Monitor for competing processes

## Logging and Debugging

### Enable Debug Logging

```bash
# Set debug environment variable
export LUCIFER_DEBUG=true

# Run with verbose output
lucifer --verbose command
```

### View Logs

```bash
# Orchestrator logs
tail -f orchestrator.log

# Agent logs
tail -f agent.log

# CLI logs
lucifer --log-level debug command
```

### Common Log Locations

- Orchestrator: `./orchestrator.log`
- Agents: `./agent.log`
- CLI: `~/.lucifer/cli.log`

## Data Issues

### Corrupted JSON Files

**Problem:** JSON storage files become corrupted

**Solutions:**
1. Stop orchestrator before manual editing
2. Use proper JSON formatting
3. Backup files before modifications
4. Use `jq` for safe JSON manipulation

### Missing Experiment Results

**Problem:** Experiment results not showing

**Solutions:**
1. Check experiment completed successfully
2. Verify agent reported results
3. Check results.json file
4. Review API logs for submission errors

## Getting Help

If you can't resolve an issue:

1. Check existing GitHub issues
2. Create a new issue with:
   - Lucifer version
   - Go version
   - Operating system
   - Full error messages
   - Steps to reproduce
   - Relevant logs

3. Join the community Discord/Slack for real-time help

## Emergency Recovery

### Stop All Experiments

```bash
# Kill all running experiments
pkill -f "lucifer"
pkill -f "orchestrator"
pkill -f "agent"
```

### Reset Orchestrator State

```bash
# Backup current state
cp users.json users.json.backup
cp experiments.json experiments.json.backup

# Reset to clean state (CAUTION: loses all data)
rm users.json experiments.json agents.json mapping.json
```

### AWS Resource Cleanup

For S3 experiments that may have left resources in bad states:

```bash
# Remove bucket policies
aws s3api delete-bucket-policy --bucket your-bucket

# Re-enable KMS keys
aws kms enable-key --key-id your-key-id
```