-- ============================================================================
-- Migration 05: Governance, Reconciliation & Security Hardening
-- Applies after: 01_init.sql, 02_governance.sql, 03_reconciliation.sql, 04_security.sql
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 1. Users: Add rolling-window login tracking, CAPTCHA answer, location support
-- ----------------------------------------------------------------------------
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_failed_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS captcha_answer INT;

-- ----------------------------------------------------------------------------
-- 2. Locations table (ABAC / location-based access)
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_locations_tenant ON locations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_locations_active ON locations(is_active);

ALTER TABLE users ADD COLUMN IF NOT EXISTS location_id UUID REFERENCES locations(id);

-- ----------------------------------------------------------------------------
-- 3. Services: Add headcount, tools, add-ons, daily cap
-- ----------------------------------------------------------------------------
ALTER TABLE services ADD COLUMN IF NOT EXISTS headcount INT NOT NULL DEFAULT 1;
ALTER TABLE services ADD COLUMN IF NOT EXISTS required_tools TEXT[];
ALTER TABLE services ADD COLUMN IF NOT EXISTS add_ons JSONB;
ALTER TABLE services ADD COLUMN IF NOT EXISTS daily_cap INT;

-- ----------------------------------------------------------------------------
-- 4. Schedules: Add reassignment reason code
-- ----------------------------------------------------------------------------
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS reassignment_reason TEXT;

-- ----------------------------------------------------------------------------
-- 5. Content items: Add custom metadata field
-- ----------------------------------------------------------------------------
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS metadata JSONB;

-- ----------------------------------------------------------------------------
-- 6. Legal holds table
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS legal_holds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hold_reason TEXT NOT NULL,
    held_by UUID NOT NULL REFERENCES users(id),
    hold_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hold_end TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    target_table VARCHAR(255) NOT NULL,
    target_record_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_legal_holds_tenant ON legal_holds(tenant_id);
CREATE INDEX IF NOT EXISTS idx_legal_holds_active ON legal_holds(is_active);
CREATE INDEX IF NOT EXISTS idx_legal_holds_target ON legal_holds(target_table, target_record_id);

-- ----------------------------------------------------------------------------
-- 7. Audit ledger immutability: DB-level triggers to prevent UPDATE/DELETE
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION prevent_audit_ledger_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_ledger is append-only: % operations are not allowed', TG_OP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Drop triggers first to make migration idempotent
DROP TRIGGER IF EXISTS audit_ledger_no_update ON audit_ledger;
DROP TRIGGER IF EXISTS audit_ledger_no_delete ON audit_ledger;

CREATE TRIGGER audit_ledger_no_update
    BEFORE UPDATE ON audit_ledger
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_ledger_modification();

CREATE TRIGGER audit_ledger_no_delete
    BEFORE DELETE ON audit_ledger
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_ledger_modification();

-- Functions to temporarily disable/enable triggers for authorized retention cleanup
CREATE OR REPLACE FUNCTION disable_audit_ledger_triggers() RETURNS void AS $$
BEGIN
    ALTER TABLE audit_ledger DISABLE TRIGGER audit_ledger_no_update;
    ALTER TABLE audit_ledger DISABLE TRIGGER audit_ledger_no_delete;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION enable_audit_ledger_triggers() RETURNS void AS $$
BEGIN
    ALTER TABLE audit_ledger ENABLE TRIGGER audit_ledger_no_update;
    ALTER TABLE audit_ledger ENABLE TRIGGER audit_ledger_no_delete;
END;
$$ LANGUAGE plpgsql;

-- ----------------------------------------------------------------------------
-- 7a. Schedules: Add pending_staff_id for reassignment confirmation workflow
-- ----------------------------------------------------------------------------
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS pending_staff_id UUID REFERENCES staff_roster(id);

-- ----------------------------------------------------------------------------
-- 7b. Protect critical audit_logs entries from mutation
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION prevent_critical_audit_log_modification()
RETURNS trigger AS $$
DECLARE
    critical_actions TEXT[] := ARRAY[
        'login', 'logout', 'login_failed',
        'create_user', 'deactivate_user', 'update_user_role',
        'review_decision', 'promote_content', 'rollback_content',
        'create_schedule', 'cancel_schedule', 'confirm_assignment', 'reassign_schedule',
        'store_sensitive_data', 'reveal_sensitive_data', 'rotate_encryption_key',
        'resolve_exception', 'assign_exception'
    ];
BEGIN
    IF OLD.action = ANY(critical_actions) THEN
        RAISE EXCEPTION 'Cannot modify or delete critical audit log entries (action: %)', OLD.action;
        RETURN NULL;
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_logs_protect_critical_update ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_protect_critical_delete ON audit_logs;

CREATE TRIGGER audit_logs_protect_critical_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_critical_audit_log_modification();

CREATE TRIGGER audit_logs_protect_critical_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_critical_audit_log_modification();

-- ----------------------------------------------------------------------------
-- 8. Seed default retention policies
-- ----------------------------------------------------------------------------
INSERT INTO retention_policies (id, table_name, retention_years, is_active)
VALUES
    (uuid_generate_v4(), 'audit_ledger', 7, TRUE),
    (uuid_generate_v4(), 'audit_logs', 7, TRUE)
ON CONFLICT DO NOTHING;
