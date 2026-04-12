-- Pricing tiers by duration/headcount
CREATE TABLE pricing_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    tier_type TEXT NOT NULL CHECK (tier_type IN ('duration', 'headcount')),
    min_value INT NOT NULL,
    max_value INT NOT NULL,
    price_usd NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pricing_tiers_service ON pricing_tiers(service_id);

-- Staff credentials
CREATE TABLE staff_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    staff_id UUID NOT NULL REFERENCES staff_roster(id) ON DELETE CASCADE,
    credential_name VARCHAR(255) NOT NULL,
    issuing_authority VARCHAR(255),
    credential_number VARCHAR(255),
    issued_date DATE,
    expiry_date DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked', 'pending')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_staff_credentials_staff ON staff_credentials(staff_id);
CREATE INDEX idx_staff_credentials_status ON staff_credentials(status);
CREATE INDEX idx_staff_credentials_expiry ON staff_credentials(expiry_date);

-- Staff availability windows
CREATE TABLE staff_availability (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    staff_id UUID NOT NULL REFERENCES staff_roster(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_recurring BOOLEAN NOT NULL DEFAULT TRUE,
    specific_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_staff_availability_staff ON staff_availability(staff_id);
CREATE INDEX idx_staff_availability_day ON staff_availability(day_of_week);

-- Resource relationships
CREATE TABLE resource_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_content_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    target_content_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL CHECK (relationship_type IN ('dependency', 'substitute', 'bundle')),
    notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source_content_id, target_content_id, relationship_type)
);
CREATE INDEX idx_resource_rel_tenant ON resource_relationships(tenant_id);
CREATE INDEX idx_resource_rel_source ON resource_relationships(source_content_id);
CREATE INDEX idx_resource_rel_target ON resource_relationships(target_content_id);
CREATE INDEX idx_resource_rel_type ON resource_relationships(relationship_type);

-- Backup staff assignments
CREATE TABLE backup_staff_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    primary_staff_id UUID NOT NULL REFERENCES staff_roster(id),
    backup_staff_id UUID NOT NULL REFERENCES staff_roster(id),
    reason_code TEXT NOT NULL CHECK (reason_code IN ('sick_leave', 'emergency', 'no_show', 'schedule_conflict', 'requested', 'other')),
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'declined', 'active')),
    confirmed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backup_staff_schedule ON backup_staff_assignments(schedule_id);
CREATE INDEX idx_backup_staff_primary ON backup_staff_assignments(primary_staff_id);
CREATE INDEX idx_backup_staff_backup ON backup_staff_assignments(backup_staff_id);

-- Reassignment reason codes on schedules
ALTER TABLE schedules ADD COLUMN IF NOT EXISTS reassignment_reason_code TEXT CHECK (
    reassignment_reason_code IS NULL OR reassignment_reason_code IN ('sick_leave', 'emergency', 'no_show', 'schedule_conflict', 'requested', 'skill_mismatch', 'other')
);

-- Capacity calendar
CREATE TABLE capacity_calendar (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    calendar_date DATE NOT NULL,
    max_capacity INT NOT NULL,
    booked_count INT NOT NULL DEFAULT 0,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(service_id, calendar_date)
);
CREATE INDEX idx_capacity_calendar_tenant ON capacity_calendar(tenant_id);
CREATE INDEX idx_capacity_calendar_service ON capacity_calendar(service_id);
CREATE INDEX idx_capacity_calendar_date ON capacity_calendar(calendar_date);

-- Add metadata column to content_items
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Fix: Add pending_reassignment to schedules status CHECK constraint
ALTER TABLE schedules DROP CONSTRAINT IF EXISTS schedules_status_check;
ALTER TABLE schedules ADD CONSTRAINT schedules_status_check CHECK (
    status IN ('pending', 'confirmed', 'in_progress', 'completed', 'cancelled', 'pending_reassignment')
);

-- Fix: encrypted_fields FK to reference encryption_keys by id instead of key_alias
ALTER TABLE encrypted_fields DROP CONSTRAINT IF EXISTS encrypted_fields_key_alias_fkey;
ALTER TABLE encrypted_fields ADD COLUMN IF NOT EXISTS key_id UUID REFERENCES encryption_keys(id);

-- Fix: sensitive_data FK - drop the invalid FK and keep key_alias as a logical reference
-- validated at application layer (already tenant-scoped in queries)
ALTER TABLE sensitive_data DROP CONSTRAINT IF EXISTS sensitive_data_key_alias_fkey;

-- Fix: Add counterparty_account column to transactions
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS counterparty_account VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_transactions_counterparty_acct ON transactions(counterparty_account);

-- Fix: Add nonce column to encryption_keys for proper envelope encryption
-- The nonce is required to decrypt the DEK stored in encrypted_key
ALTER TABLE encryption_keys ADD COLUMN IF NOT EXISTS nonce BYTEA;

-- Fix: Add full_name column to users for frontend contract alignment
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name VARCHAR(255) NOT NULL DEFAULT '';
