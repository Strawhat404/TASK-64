-- Financial Reconciliation Engine tables

-- Transaction feeds imported from CSV/Excel
CREATE TABLE transaction_feeds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    filename VARCHAR(500) NOT NULL,
    feed_type TEXT NOT NULL CHECK (feed_type IN ('internal', 'external')),
    record_count INT NOT NULL DEFAULT 0,
    imported_by UUID NOT NULL REFERENCES users(id),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX idx_transaction_feeds_tenant ON transaction_feeds(tenant_id);
CREATE INDEX idx_transaction_feeds_status ON transaction_feeds(status);

-- Individual transactions from imported feeds
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feed_id UUID NOT NULL REFERENCES transaction_feeds(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transaction_date TIMESTAMPTZ NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    counterparty VARCHAR(500),
    counterparty_account VARCHAR(255),
    memo TEXT,
    reference_number VARCHAR(255),
    source TEXT NOT NULL CHECK (source IN ('internal', 'external')),
    matched BOOLEAN NOT NULL DEFAULT FALSE,
    match_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_feed ON transactions(feed_id);
CREATE INDEX idx_transactions_tenant ON transactions(tenant_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);
CREATE INDEX idx_transactions_amount ON transactions(amount);
CREATE INDEX idx_transactions_matched ON transactions(matched);
CREATE INDEX idx_transactions_match_id ON transactions(match_id);
CREATE INDEX idx_transactions_counterparty ON transactions(counterparty);

-- Matched transaction pairs
CREATE TABLE transaction_matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    internal_tx_id UUID NOT NULL REFERENCES transactions(id),
    external_tx_id UUID NOT NULL REFERENCES transactions(id),
    match_confidence NUMERIC(5,2) NOT NULL,
    match_method TEXT NOT NULL CHECK (match_method IN ('exact', 'fuzzy', 'manual')),
    amount_variance NUMERIC(15,2) NOT NULL DEFAULT 0,
    time_variance_minutes INT NOT NULL DEFAULT 0,
    matched_by UUID REFERENCES users(id),
    matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'disputed', 'reversed'))
);

CREATE INDEX idx_matches_tenant ON transaction_matches(tenant_id);
CREATE INDEX idx_matches_internal ON transaction_matches(internal_tx_id);
CREATE INDEX idx_matches_external ON transaction_matches(external_tx_id);

-- Exception items (unmatched, duplicates, variances)
CREATE TABLE reconciliation_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES transactions(id),
    match_id UUID REFERENCES transaction_matches(id),
    exception_type TEXT NOT NULL CHECK (exception_type IN ('unmatched', 'duplicate_suspect', 'variance_over_threshold', 'amount_mismatch')),
    severity TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    amount NUMERIC(15,2),
    variance_amount NUMERIC(15,2),
    description TEXT,
    assigned_to UUID REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'dismissed')),
    disposition TEXT,
    resolution_notes TEXT,
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exceptions_tenant ON reconciliation_exceptions(tenant_id);
CREATE INDEX idx_exceptions_type ON reconciliation_exceptions(exception_type);
CREATE INDEX idx_exceptions_status ON reconciliation_exceptions(status);
CREATE INDEX idx_exceptions_assigned ON reconciliation_exceptions(assigned_to);
CREATE INDEX idx_exceptions_severity ON reconciliation_exceptions(severity);
