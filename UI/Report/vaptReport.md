# Vulnerability Assessment & Penetration Testing (VAPT) Report
**Date:** Wed, 08 Apr 2026 18:25:07 IST

## Executive Summary
This report summarizes the results of an automated security audit and subsequent chaos engineering experiments. It details identified vulnerabilities, misconfigurations, and validated exploits alongside detailed experiment tracking.

### Summary Statistics
- **Total Rules Evaluated:** 26
- **Passed:** 15
- **Failed:** 11
- **Error:** 0

## Detailed Findings with Verified Exploit Testing

### 1. Security Group Should Not Allow 0.0.0.0/0 on SSH/RDP (Rule ID: A-EC2-1)
- **Severity:** CRITICAL
- **Status:** FAIL
- **Description:** Security group allows unrestricted internet access to SSH or RDP which can expose instances to brute force attacks.
- **Remediation:** Restrict SSH/RDP access to trusted IP addresses or use a bastion host or VPN.

---

### 7. Missing MFA on Privileged IAM User (Rule ID: A-IAM-3)
- **Severity:** HIGH
- **Status:** FAIL
- **Description:** IAM user with admin permissions directly attached does not have MFA enabled exposing privileged access to credential theft.
- **Remediation:** Enforce MFA for all IAM users with admin or elevated permissions using an IAM policy condition.

---

### 9. Cross-Account Role Trust Misconfiguration (Rule ID: A-IAM-5)
- **Severity:** HIGH
- **Status:** FAIL
- **Description:** IAM role has an overly permissive trust policy allowing any principal (*) to assume it enabling unauthorized cross-account access.
- **Remediation:** Restrict role trust policies to specific trusted account IDs and require external ID conditions.

---

### 13. Unauthenticated Lambda URL Invocation Exposure (Rule ID: A-LAM-3)
- **Severity:** CRITICAL
- **Status:** FAIL
- **Description:** Lambda function URL is reachable without credentials, enabling unauthenticated invocation, SSRF payload testing, and high-frequency request abuse that can exhaust resources.
- **Remediation:** Require IAM or authorizer-based authentication for function URLs, enforce WAF/rate limiting, and validate inputs to block SSRF-style payloads.

---

### 14. No DLQ Causes Silent Event Loss on Failure (Rule ID: A-LAM-4)
- **Severity:** HIGH
- **Status:** FAIL
- **Description:** Intentional function failures can silently drop events when no dead-letter queue is configured, creating undetected data pipeline gaps and delayed recovery.
- **Remediation:** Configure DLQ or on-failure destinations for asynchronous invocations, alert on retry exhaustion, and monitor end-to-end pipeline lag and drop rates.

---

### 19. Single-AZ RDS Resilience Gap (AZ Failure Downtime) (Rule ID: A-RDS-4)
- **Severity:** HIGH
- **Status:** FAIL
- **Description:** An AZ-level network blackhole on the DB subnet can cause extended downtime when failover is not optimized; this scenario injects AZ disruption, measures outage and failover duration, and compares outcomes with a Multi-AZ baseline.
- **Remediation:** Enable Multi-AZ deployment for production databases, validate automatic failover behavior in regular game days, and set SLOs for failover duration and application reconnect time.

---

### 20. Lateral DB Access from Non-App EC2 Allowed (Rule ID: A-RDS-5)
- **Severity:** HIGH
- **Status:** FAIL
- **Description:** A non-application EC2 instance in the same VPC can connect to the database, indicating weak east-west segmentation; this scenario attempts internal DB access and measures whether AWS Config rules or GuardDuty detections are triggered.
- **Remediation:** Restrict database security group ingress to only approved application security groups, enforce subnet-level segmentation and NACL controls, and implement alerting for anomalous internal database access.

---

### 21. Block Public Access Disabled (Rule ID: A-S3-1)
- **Severity:** CRITICAL
- **Status:** FAIL
- **Description:** S3 Block Public Access is disabled which may allow public exposure of sensitive data.
- **Remediation:** Enable S3 Block Public Access for the bucket and at the account level.

---

### 23. Versioning Disabled (Rule ID: A-S3-3)
- **Severity:** MEDIUM
- **Status:** FAIL
- **Description:** Bucket versioning is disabled which increases the risk of data loss from accidental deletion or overwrites.
- **Remediation:** Enable versioning for the S3 bucket to maintain object history and allow recovery.

---

### 24. MFA-Delete Disabled (Rule ID: A-S3-4)
- **Severity:** MEDIUM
- **Status:** FAIL
- **Description:** MFA-Delete is disabled for the S3 bucket. This means versioning status cannot be changed and versions cannot be permanently deleted without additional multi-factor authentication.
- **Remediation:** Enable MFA-Delete for the S3 bucket to provide an extra layer of security against malicious deletion.

---

### 26. Access Logging Disabled (Rule ID: A-S3-6)
- **Severity:** MEDIUM
- **Status:** FAIL
- **Description:** S3 access logging is disabled which reduces visibility into bucket access and potential misuse.
- **Remediation:** Enable server access logging for the bucket and store logs in a separate logging bucket.

---

