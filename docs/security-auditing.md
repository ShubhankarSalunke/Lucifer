# Security Auditing (VAPT)

Lucifer includes comprehensive Vulnerability Assessment and Penetration Testing (VAPT) capabilities to identify security misconfigurations and vulnerabilities in your infrastructure.

## Overview

The security audit module scans AWS resources against a comprehensive set of security rules and best practices. It checks for common misconfigurations, insecure permissions, and potential security risks.

## Supported Services

### Amazon EC2
Scans EC2 instances for:
- Security group configurations
- IAM instance profiles
- Public IP exposure
- Encryption settings

### Amazon S3
Audits S3 buckets for:
- Bucket policies and permissions
- Public access settings
- Encryption configurations
- Versioning and logging

### Amazon IAM
Evaluates IAM resources for:
- Policy permissions
- Role configurations
- User access patterns
- Least privilege violations

### Amazon RDS
Checks RDS instances for:
- Public accessibility
- Encryption settings
- Backup configurations
- Security group rules

### Amazon Lambda
Scans Lambda functions for:
- Runtime security
- Environment variable exposure
- IAM permissions
- VPC configurations

## Running Security Scans

### Scan Specific Services

```bash
# Scan EC2 instances
lucifer audit scan aws ec2

# Scan S3 buckets
lucifer audit scan aws s3

# Scan IAM resources
lucifer audit scan aws iam
```

### Scan All AWS Services

```bash
lucifer audit scan aws all
```

## Viewing Audit Rules

To see all available audit rules:

```bash
lucifer audit rules
```

This displays:
- Rule IDs and descriptions
- Severity levels (Critical, High, Medium, Low)
- Affected services
- Remediation guidance

## Understanding Results

Audit results include:
- **Passed**: Resources that meet security standards
- **Failed**: Resources with security issues
- **Warnings**: Potential security concerns requiring review

Each finding includes:
- Resource ARN
- Rule violated
- Severity level
- Detailed description
- Remediation steps

## Integration with Chaos Engineering

Security audits work seamlessly with chaos experiments:

1. **Pre-Experiment Auditing**: Run security scans before chaos experiments to establish a baseline
2. **Post-Experiment Validation**: Verify that chaos experiments didn't introduce new vulnerabilities
3. **Continuous Monitoring**: Regular audits ensure ongoing security compliance

## Custom Rules

The audit engine supports custom rules written in YAML. Rules are stored in `security-audit/rules/` with filenames indicating the service (e.g., `ec2.yaml`, `s3.yaml`).

### Rule Structure

```yaml
rules:
  - id: "EC2_PUBLIC_IP"
    title: "EC2 instance has public IP"
    description: "EC2 instances should not have public IPs in production"
    severity: "HIGH"
    service: "ec2"
    condition: "public_ip_address != null"
    remediation: "Remove public IP or use NAT gateway"
```

## Best Practices

1. **Regular Scanning**: Run audits weekly or after infrastructure changes
2. **Address Critical Issues First**: Prioritize fixing high-severity findings
3. **Automate Remediation**: Use Infrastructure as Code to automatically fix common issues
4. **Monitor Trends**: Track security posture improvements over time
5. **Combine with Chaos**: Use audits to validate system resilience after chaos experiments

## Output Formats

Audit results can be exported in multiple formats:
- JSON (default)
- CSV
- HTML reports
- JUnit XML (for CI/CD integration)

## Compliance Frameworks

The audit rules align with:
- AWS Well-Architected Framework
- CIS AWS Benchmarks
- NIST Cybersecurity Framework
- ISO 27001 standards