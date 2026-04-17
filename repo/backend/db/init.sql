-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_domain ON tenants (domain);

-- Roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

INSERT INTO roles (name) VALUES
    ('Administrator'),
    ('Scheduler'),
    ('Reviewer'),
    ('Auditor');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id INT NOT NULL REFERENCES roles(id),
    username VARCHAR(150) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    captcha_required BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_users_tenant_id ON users (tenant_id);
CREATE INDEX idx_users_role_id ON users (role_id);
CREATE INDEX idx_users_username ON users (username);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_is_active ON users (is_active);

-- Services table
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    base_price_usd NUMERIC(10,2) NOT NULL,
    tier TEXT NOT NULL CHECK (tier IN ('standard', 'premium', 'enterprise')),
    after_hours_surcharge_pct INT NOT NULL DEFAULT 20,
    same_day_surcharge_usd NUMERIC(10,2) NOT NULL DEFAULT 25.00,
    duration_minutes INT NOT NULL CHECK (
        duration_minutes >= 15
        AND duration_minutes <= 240
        AND duration_minutes % 15 = 0
    ),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_services_tenant_id ON services (tenant_id);
CREATE INDEX idx_services_tier ON services (tier);
CREATE INDEX idx_services_is_active ON services (is_active);

-- Staff roster table
CREATE TABLE staff_roster (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    specialization VARCHAR(255),
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_staff_roster_tenant_id ON staff_roster (tenant_id);
CREATE INDEX idx_staff_roster_user_id ON staff_roster (user_id);
CREATE INDEX idx_staff_roster_specialization ON staff_roster (specialization);
CREATE INDEX idx_staff_roster_is_available ON staff_roster (is_available);

-- Schedules table
CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id),
    staff_id UUID NOT NULL REFERENCES staff_roster(id),
    client_name VARCHAR(255) NOT NULL,
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'confirmed', 'in_progress', 'completed', 'cancelled')
    ),
    requires_confirmation BOOLEAN NOT NULL DEFAULT FALSE,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_schedules_tenant_id ON schedules (tenant_id);
CREATE INDEX idx_schedules_service_id ON schedules (service_id);
CREATE INDEX idx_schedules_staff_id ON schedules (staff_id);
CREATE INDEX idx_schedules_status ON schedules (status);
CREATE INDEX idx_schedules_scheduled_start ON schedules (scheduled_start);
CREATE INDEX idx_schedules_scheduled_end ON schedules (scheduled_end);

-- Exclusion constraint: prevent overlapping schedules for the same staff member
ALTER TABLE schedules
    ADD CONSTRAINT no_overlapping_schedules
    EXCLUDE USING gist (
        staff_id WITH =,
        tstzrange(scheduled_start, scheduled_end) WITH &&
    )
    WHERE (status NOT IN ('cancelled'));

-- Audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id UUID,
    details JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs (tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs (resource_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- Sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_token ON sessions (token);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

-- Seed default tenant
INSERT INTO tenants (id, name, domain) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Default Tenant', 'localhost');

-- Seed demo users for all four roles
-- Password for all: Admin12345!!! hashed with Argon2id (hex-encoded salt and key)
INSERT INTO users (id, tenant_id, role_id, username, email, full_name, password_hash) VALUES
    (
        'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
        (SELECT id FROM roles WHERE name = 'Administrator'),
        'admin',
        'admin@localhost',
        'System Administrator',
        '$argon2id$v=19$m=65536,t=3,p=4$39696f4379d3e27231bd2a0c65ac5216$78df9c4a849315dca91f3b16cae5d827ee7fb2e64b4fe3cc6e1c903668dbb668'
    ),
    (
        'b2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
        (SELECT id FROM roles WHERE name = 'Scheduler'),
        'scheduler_user',
        'scheduler@localhost',
        'Demo Scheduler',
        '$argon2id$v=19$m=65536,t=3,p=4$39696f4379d3e27231bd2a0c65ac5216$78df9c4a849315dca91f3b16cae5d827ee7fb2e64b4fe3cc6e1c903668dbb668'
    ),
    (
        'b3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
        (SELECT id FROM roles WHERE name = 'Reviewer'),
        'reviewer_user',
        'reviewer@localhost',
        'Demo Reviewer',
        '$argon2id$v=19$m=65536,t=3,p=4$39696f4379d3e27231bd2a0c65ac5216$78df9c4a849315dca91f3b16cae5d827ee7fb2e64b4fe3cc6e1c903668dbb668'
    ),
    (
        'b4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55',
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
        (SELECT id FROM roles WHERE name = 'Auditor'),
        'auditor_user',
        'auditor@localhost',
        'Demo Auditor',
        '$argon2id$v=19$m=65536,t=3,p=4$39696f4379d3e27231bd2a0c65ac5216$78df9c4a849315dca91f3b16cae5d827ee7fb2e64b4fe3cc6e1c903668dbb668'
    );
