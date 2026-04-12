# Local Operations & Compliance Console — Business Logic Questions Log

## Security

**TLS certificate management approach**
Question: The prompt did not specify whether the deployment environment uses a corporate internal CA or self-signed certificates generated at startup.
My Understanding: For an offline-first, air-gapped deployment, relying on a corporate CA introduces an external dependency that may not be available. Self-generated certificates via Caddy's internal CA are more portable.
Solution: Caddy is configured with `tls internal` to auto-generate a local CA and issue certificates at startup. No external CA dependency. Operators can replace with their own certificates by mounting them into the Caddy container.

**Session storage for multi-instance deployments**
Question: The prompt did not clarify whether sessions need to be shared across multiple backend instances or whether single-instance deployment is assumed.
My Understanding: For an internal compliance tool, single-instance deployment is the common case. Shared session storage adds operational complexity without clear benefit at this scale.
Solution: Sessions are stored in PostgreSQL (`sessions` table) with server-side validation on every request. For multi-instance deployments, sticky sessions at the load balancer level would suffice without requiring a separate session store.

**Account lockout policy thresholds**
Question: The prompt mentioned brute-force protection but did not specify lockout thresholds, duration, or whether administrators can manually unlock accounts.
My Understanding: A 30-minute lockout after 5 failed attempts within a 15-minute window is a standard industry baseline that balances security with usability for internal staff.
Solution: Accounts are locked for 30 minutes after 5 failed attempts within a 15-minute rolling window. Locks auto-expire. Administrators can manually unlock via the user management API. Lockout events are recorded in the audit log with source IP.

**CAPTCHA implementation without cloud dependency**
Question: The prompt required CAPTCHA support but did not specify whether a cloud-hosted CAPTCHA service (e.g., reCAPTCHA) is acceptable for an air-gapped deployment.
My Understanding: Cloud-hosted CAPTCHA services require outbound internet access, which is incompatible with offline-first deployment requirements.
Solution: CAPTCHA is implemented as server-generated arithmetic challenges stored in the `captcha_answer` column. Triggered after 3 consecutive failed login attempts. No external service dependency.

**Password complexity rules beyond minimum length**
Question: The prompt specified a 12-character minimum password length but did not define additional complexity requirements such as uppercase, digits, or special characters.
My Understanding: A 12-character minimum without complexity rules is weaker than typical compliance standards (e.g., NIST 800-63B). However, overly strict rules increase user friction for internal staff.
Solution: Implemented 12-character minimum as specified. Additional complexity enforcement (uppercase, digits, special characters) is configurable but not enforced by default. Password history and maximum age are not implemented in v1 — open for future iteration.

**Scope of encryption at rest**
Question: The prompt mentioned sensitive data protection but did not specify which fields require encryption at rest versus which can be stored in plaintext.
My Understanding: Not all data requires encryption — encrypting everything adds overhead without proportional security benefit. Encryption should target fields with regulatory or compliance sensitivity.
Solution: Application-level AES-256-GCM envelope encryption is implemented for the `sensitive_data` table covering bank accounts, routing numbers, tax IDs, SSNs, and credit cards. Each tenant has its own DEK wrapped by a master key with quarterly rotation support. OS/disk-level encryption is a deployment concern left to the operator.

## Scheduling

**30-minute buffer enforcement scope**
Question: The prompt required a 30-minute buffer between consecutive staff assignments but did not specify whether this should be a fixed system-wide constant or configurable per service type or staff member.
My Understanding: A fixed constant is simpler to implement and audit. Per-service configurability adds complexity that is not justified without explicit requirements.
Solution: The 30-minute buffer is enforced as a system-wide constant in the scheduling conflict detection logic. Override capability for urgent appointments is not implemented in v1.

**Staff availability modeling**
Question: The prompt required scheduling with staff availability but did not specify whether availability should support recurring weekly templates, per-day overrides, or both.
My Understanding: Real-world staff schedules require both recurring patterns (standard work weeks) and one-off overrides (holidays, sick days). Supporting only one would make the system impractical.
Solution: Staff availability is modeled via the `staff_availability` table supporting both weekly recurring templates (`is_recurring = TRUE`) and per-day overrides (`specific_date`). Working-hours enforcement at booking time is not currently implemented — open for future iteration.

**Recurring appointment support**
Question: The prompt described appointment scheduling but did not specify whether recurring schedules (e.g., weekly inspections) are required.
My Understanding: Recurring appointments are a common scheduling requirement but add significant complexity to conflict detection and cancellation logic. Without explicit requirements, implementing a basic version risks scope creep.
Solution: Recurring appointments are not implemented in v1. Each appointment is created individually. Recurrence patterns can be added in a future iteration once recurrence rules and conflict handling policies are defined.

**Cancellation and rescheduling behavior**
Question: The prompt did not specify whether cancelled appointments should be deleted or retained, and whether rescheduling creates a new record or updates the existing one.
My Understanding: Retaining cancelled appointments is important for audit purposes — compliance systems need a complete history of all scheduling activity, including cancellations.
Solution: Cancelled appointments are retained with `status = 'cancelled'` and excluded from conflict checks. Rescheduling updates the existing record in place. Staff reassignment uses a confirmation workflow (`pending_reassignment` status with `pending_staff_id`). Minimum notice period enforcement is not implemented in v1.

**Time zone handling**
Question: The prompt did not specify whether the system operates in a single fixed time zone or must support multiple time zones for distributed teams.
My Understanding: For an offline-first internal deployment, a single time zone is the most common scenario. Multi-timezone support adds complexity to scheduling conflict detection and display logic.
Solution: The system stores all timestamps in UTC. Display conversion is handled client-side. Multi-timezone staff availability configuration is not implemented in v1 — all availability windows are interpreted as UTC.

## Compliance & Audit

**Audit log retention period**
Question: The prompt required audit logging but did not specify how long audit records must be retained.
My Understanding: Compliance systems typically require 7-year retention to satisfy financial and regulatory audit requirements. Shorter retention periods risk non-compliance.
Solution: Default retention is 7 years for both `audit_logs` and `audit_ledger` tables, seeded in `retention_policies`. Retention is enforced via `EnforceRetention` which logs expired entries to `deletion_log` for DBA archival. Audit ledger entries are never deleted by the application — only flagged for external archival.

**Audit log immutability mechanism**
Question: The prompt required an immutable audit trail but did not specify the technical mechanism for preventing tampering.
My Understanding: Database-level immutability (triggers) is stronger than application-level enforcement because it prevents tampering even if the application code is compromised.
Solution: The `audit_ledger` table is append-only with SHA-256 hash chaining (each entry includes `previous_hash`). Database triggers prevent UPDATE/DELETE operations. Integrity can be verified via `AuditLedger.Verify()`. Critical actions use fail-closed semantics — if the ledger append fails, the action is denied.

**Right to erasure vs. audit log immutability conflict**
Question: The prompt did not address how GDPR-style right-to-erasure requests interact with the immutable audit log requirement.
My Understanding: These two requirements are in direct conflict. Anonymizing actor fields while preserving the action record is the standard approach — it satisfies both the audit requirement and the erasure request.
Solution: Right-to-erasure interaction with the audit ledger is not implemented in v1. The current design prioritizes audit immutability. Anonymization of personal data fields in audit records while preserving the action record is planned for a future iteration.

## Content Governance

**Gray-release rollout configuration**
Question: The prompt required gray-release functionality but did not specify whether rollout percentages and durations should be configurable per policy, per content item, or system-wide.
My Understanding: Per-policy configurability provides the right balance — different content types (e.g., policy documents vs. UI copy) may have different risk profiles requiring different rollout speeds.
Solution: Gray-release duration and percentage are configurable per policy. Automatic escalation (e.g., 10% → 25% → 50% → 100%) is not implemented — promotion is manual only. A hard cap on gray-release duration is not enforced in v1.

**Multi-level review depth limits**
Question: The prompt required multi-level review pipelines but did not specify the maximum number of review levels or whether the same reviewer can appear at multiple levels.
My Understanding: Allowing the same reviewer at multiple levels defeats the purpose of multi-level review. A practical maximum of 5 levels covers the vast majority of real-world approval chains.
Solution: The system supports 1–5 review levels per policy. Each level must have a distinct reviewer — the same user cannot approve at multiple levels for the same content item. Reviewer unavailability handling (auto-escalation after timeout) is not implemented in v1.

**Auto-block false positive handling**
Question: The prompt required auto-block for policy violations but did not specify the process for handling false positives or whether auto-block decisions require administrator approval to reverse.
My Understanding: Auto-block false positives need a clear resolution path to avoid blocking legitimate content indefinitely. A single Reviewer being able to unblock is sufficient for most cases, with Administrator override as a fallback.
Solution: Auto-blocked content is placed in an expedited review queue separate from the normal pipeline. A Reviewer can unblock content after review. Administrator approval is not required for unblocking. A dry-run mode for rule tuning is not implemented in v1.

## Financial Reconciliation

**Auto-match threshold configuration**
Question: The prompt required automated transaction matching but did not specify whether the match score threshold (70) and variance threshold ($1.00) should be fixed or configurable per feed source.
My Understanding: Different counterparties and transaction types have different data quality characteristics. A fixed threshold would produce too many false positives for some feeds and miss matches for others.
Solution: The auto-match score threshold (70) and variance threshold ($1.00) are system-wide defaults. Per-feed-source configurability is not implemented in v1. Threshold tuning based on match accuracy metrics is planned for a future iteration.

**Large CSV import handling**
Question: The prompt required CSV import for reconciliation feeds but did not specify file size limits or how partial failures should be handled.
My Understanding: Rejecting an entire file on a single invalid row is too strict for large financial imports where a small number of malformed rows is expected. Partial import with error reporting is more practical.
Solution: Valid rows are imported and invalid rows are reported with error details. No hard file size limit is enforced in v1. Chunked/streaming processing for very large files is not implemented — batch processing with synchronous response is used.

**Concurrent feed processing isolation**
Question: The prompt did not specify whether multiple reconciliation feeds can be processed simultaneously or must be serialized to prevent matching conflicts.
My Understanding: Concurrent processing of feeds that share transaction references could cause the same transaction to be matched by two simultaneous processes. Row-level locking prevents this.
Solution: Concurrent feed processing is supported. Row-level locking on transactions during matching prevents concurrent resolution of the same exception by multiple reviewers. Cross-feed duplicate detection uses the transaction reference and amount as a composite deduplication key.

## Encryption & Key Management

**HSM integration requirement**
Question: The prompt required encryption key management but did not specify whether a Hardware Security Module (HSM) is required for production deployments.
My Understanding: HSM integration is ideal for high-security production environments but adds significant operational complexity. For an offline-first internal deployment, software-based key management with encrypted key storage is a practical alternative.
Solution: Software-based AES-256 key management is implemented. The master key is loaded from an environment variable (`MASTER_KEY`) or auto-generated on first run with a warning. HSM integration (PKCS#11, KMIP) is not implemented in v1 — planned for future iteration if production requirements demand it.

**Cross-tenant key isolation**
Question: The prompt required multi-tenant support but did not specify whether each tenant's data should be encrypted with a separate key or whether a shared key is acceptable.
My Understanding: Shared encryption keys mean a key compromise exposes all tenants' data. Per-tenant DEKs limit the blast radius of a key compromise to a single tenant.
Solution: Each tenant has its own DEK scoped by `tenant_id` in `encryption_keys`. Tenant key rotation is independent. Cross-tenant isolation is enforced at both the access-control layer (all queries are tenant-scoped) and the encryption layer (each tenant's data is encrypted with a tenant-specific DEK).

## Data Lifecycle

**Per-tenant retention policy overrides**
Question: The prompt required data retention management but did not specify whether individual tenants can override the system-wide retention period.
My Understanding: Different tenants may be subject to different regulatory requirements. A minimum retention floor prevents tenants from setting retention periods that would violate compliance requirements.
Solution: Per-tenant retention overrides are not implemented in v1. The system-wide 7-year default applies to all tenants. Per-tenant configurability with a minimum retention floor is planned for a future iteration.

**Legal hold scope and release authorization**
Question: The prompt required legal hold functionality but did not specify whether holds should be scoped at the table level, record level, or both, and who can release holds.
My Understanding: Record-level holds are more precise and avoid over-retention of unrelated data. Table-level holds are useful for broad litigation scenarios. Only Administrators should be able to place and release holds to prevent unauthorized data deletion.
Solution: Legal holds are implemented via the `legal_holds` table supporting both table-level and record-level holds (`target_table` + optional `target_record_id`), scoped per tenant. Holds prevent deletion — retention enforcement skips held records. Only Administrators can place and release holds. Legal hold interaction with right-to-erasure requests is not implemented in v1.
