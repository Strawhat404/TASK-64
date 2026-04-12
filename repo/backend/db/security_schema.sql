-- Security & Data Lifecycle tables

-- Encryption key registry
CREATE TABLE encryption_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    key_alias VARCHAR(255) NOT NULL,
    encrypted_key BYTEA NOT NULL,
    nonce BYTEA,
    algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'rotated', 'revoked')),
    rotation_number INT NOT NULL DEFAULT 1,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, key_alias)
);

CREATE INDEX idx_encryption_keys_alias ON encryption_keys(key_alias);
CREATE INDEX idx_encryption_keys_status ON encryption_keys(status);
CREATE INDEX idx_encryption_keys_tenant ON encryption_keys(tenant_id);

-- Encrypted field registry (tracks which fields are encrypted with which key)
CREATE TABLE encrypted_fields (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(255) NOT NULL,
    column_name VARCHAR(255) NOT NULL,
    key_id UUID REFERENCES encryption_keys(id),
    key_alias VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(table_name, column_name)
);

-- Sensitive data store (bank accounts, etc encrypted at rest)
CREATE TABLE sensitive_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id),
    data_type TEXT NOT NULL CHECK (data_type IN ('bank_account', 'routing_number', 'tax_id', 'ssn', 'credit_card')),
    encrypted_value BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    key_alias VARCHAR(255) NOT NULL,
    label VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sensitive_data_tenant ON sensitive_data(tenant_id);
CREATE INDEX idx_sensitive_data_owner ON sensitive_data(owner_id);
CREATE INDEX idx_sensitive_data_type ON sensitive_data(data_type);

-- Audit ledger (immutable, append-only with hash chain)
CREATE TABLE audit_ledger (
    id BIGSERIAL PRIMARY KEY,
    entry_hash TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    tenant_id UUID REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    details JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Make audit_ledger append-only (revoke UPDATE and DELETE)
-- This is enforced at the application level as well
CREATE INDEX idx_audit_ledger_tenant ON audit_ledger(tenant_id);
CREATE INDEX idx_audit_ledger_user ON audit_ledger(user_id);
CREATE INDEX idx_audit_ledger_action ON audit_ledger(action);
CREATE INDEX idx_audit_ledger_created ON audit_ledger(created_at);
CREATE INDEX idx_audit_ledger_hash ON audit_ledger(entry_hash);

-- Data retention policy tracking (tenant-scoped)
CREATE TABLE retention_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    table_name VARCHAR(255) NOT NULL,
    retention_years INT NOT NULL DEFAULT 7,
    last_purge_at TIMESTAMPTZ,
    next_purge_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_retention_policies_tenant ON retention_policies(tenant_id);

-- Secure deletion log
CREATE TABLE deletion_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(255) NOT NULL,
    record_id TEXT NOT NULL,
    deletion_reason TEXT NOT NULL,
    deleted_by UUID REFERENCES users(id),
    retention_met BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deletion_log_table ON deletion_log(table_name);
CREATE INDEX idx_deletion_log_date ON deletion_log(deleted_at);

-- Rate limit tracking
CREATE TABLE rate_limits (
    id BIGSERIAL PRIMARY KEY,
    identifier TEXT NOT NULL,
    identifier_type TEXT NOT NULL CHECK (identifier_type IN ('user', 'ip')),
    window_start TIMESTAMPTZ NOT NULL,
    request_count INT NOT NULL DEFAULT 1,
    UNIQUE(identifier, identifier_type, window_start)
);

CREATE INDEX idx_rate_limits_identifier ON rate_limits(identifier, identifier_type);
CREATE INDEX idx_rate_limits_window ON rate_limits(window_start);
