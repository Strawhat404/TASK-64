-- Content Governance tables

-- Moderation rules for auto-blocking
CREATE TABLE moderation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('keyword_block', 'regex_block', 'manual_review')),
    pattern TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_rules_tenant ON moderation_rules(tenant_id);
CREATE INDEX idx_moderation_rules_active ON moderation_rules(is_active);

-- Content items that go through moderation
CREATE TABLE content_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'article' CHECK (content_type IN ('article', 'resource', 'announcement', 'policy')),
    subject VARCHAR(255),
    grade VARCHAR(50),
    tags TEXT[],
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_review', 'in_review', 'approved', 'rejected', 'gray_release', 'published', 'archived')),
    gray_release_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    current_version INT NOT NULL DEFAULT 1,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_items_tenant ON content_items(tenant_id);
CREATE INDEX idx_content_items_status ON content_items(status);
CREATE INDEX idx_content_items_gray_release ON content_items(gray_release_at) WHERE status = 'gray_release';
CREATE INDEX idx_content_items_tags ON content_items USING gin(tags);

-- Content versions for rollback (keep last 10)
CREATE TABLE content_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    subject VARCHAR(255),
    grade VARCHAR(50),
    tags TEXT[],
    release_notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(content_id, version_number)
);

CREATE INDEX idx_content_versions_content ON content_versions(content_id);

-- Moderation queue (multi-level review)
CREATE TABLE moderation_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    reviewer_id UUID REFERENCES users(id),
    review_level INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'escalated')),
    decision_notes TEXT,
    auto_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    blocked_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ
);

CREATE INDEX idx_moderation_reviews_content ON moderation_reviews(content_id);
CREATE INDEX idx_moderation_reviews_reviewer ON moderation_reviews(reviewer_id);
CREATE INDEX idx_moderation_reviews_status ON moderation_reviews(status);
CREATE INDEX idx_moderation_reviews_level ON moderation_reviews(review_level);
