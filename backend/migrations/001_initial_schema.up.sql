-- 001_initial_schema.up.sql
-- Initial schema: all tables for the photo audit platform

CREATE TABLE tenants (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(128) NOT NULL,
    country_code VARCHAR(8)  NOT NULL,
    status       SMALLINT NOT NULL DEFAULT 1
                 CHECK (status IN (0, 1)),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   UUID
);

COMMENT ON TABLE tenants IS
  'Organizational tenants (departments / regions). Each tenant has isolated users, teams, and audit data.';

COMMENT ON COLUMN tenants.status IS
  '0 = disabled, 1 = active.';


CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID,
    username      VARCHAR(64) NOT NULL,
    display_name  VARCHAR(128) NOT NULL,
    password_hash VARCHAR(256) NOT NULL,
    role          VARCHAR(32) NOT NULL
                  CHECK (role IN ('platform_admin', 'tenant_admin', 'reviewer', 'quality_checker')),
    email         VARCHAR(256),
    phone         VARCHAR(32),
    languages     TEXT[] NOT NULL DEFAULT '{}',
    status        SMALLINT NOT NULL DEFAULT 1
                  CHECK (status IN (0, 1)),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE users IS
  'Application users. tenant_id = NULL indicates a platform-level admin.';

COMMENT ON COLUMN users.role IS
  'platform_admin = global admin, tenant_admin = tenant owner, reviewer = auditors, quality_checker = QA reviewers.';

ALTER TABLE users ADD CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username);

COMMENT ON COLUMN users.languages IS
  'Array of BCP-47 language tags, e.g. ''{"en-US","zh-CN"}''.';

CREATE INDEX idx_users_tenant_id ON users (tenant_id);


CREATE TABLE audit_teams (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name      VARCHAR(128) NOT NULL,
    leader_id UUID NOT NULL REFERENCES users(id),
    status    SMALLINT NOT NULL DEFAULT 1
              CHECK (status IN (0, 1))
);


CREATE TABLE audit_team_members (
    team_id     UUID    NOT NULL REFERENCES audit_teams(id) ON DELETE CASCADE,
    user_id     UUID    NOT NULL REFERENCES users(id),
    member_role VARCHAR(32) NOT NULL
                CHECK (member_role IN ('reviewer', 'senior_reviewer', 'quality_checker')),
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX idx_audit_team_members_team_id ON audit_team_members (team_id);
CREATE INDEX idx_audit_team_members_user_id ON audit_team_members (user_id);


CREATE TABLE system_configs (
    id            SERIAL PRIMARY KEY,
    config_key    VARCHAR(128) NOT NULL UNIQUE,
    config_value  TEXT,
    description   TEXT,
    encrypted     BOOLEAN NOT NULL DEFAULT false,
    updated_by    UUID REFERENCES users(id),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE TABLE quality_audit_batches (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    created_by     UUID NOT NULL REFERENCES users(id),
    name           VARCHAR(256) NOT NULL,
    mode           VARCHAR(32) NOT NULL
                   CHECK (mode IN ('local_correction', 'full_correction')),
    filter_status  VARCHAR(32) NOT NULL,
    sample_size    INT NOT NULL CHECK (sample_size > 0),
    status         VARCHAR(16) NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'in_progress', 'completed')),
    reviewed_count INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ
);

CREATE INDEX idx_quality_audit_batches_tenant_id ON quality_audit_batches (tenant_id);
CREATE INDEX idx_quality_audit_batches_status ON quality_audit_batches (status);


CREATE TABLE quality_audit_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id      UUID NOT NULL REFERENCES quality_audit_batches(id) ON DELETE CASCADE,
    element_id    UUID NOT NULL,
    original_score INT NOT NULL,
    qa_score      INT NOT NULL CHECK (qa_score >= 0 AND qa_score <= 100),
    qa_level      VARCHAR(16) NOT NULL
                  CHECK (qa_level IN ('pass', 'minor_issue', 'major_issue', 'critical')),
    disagree      BOOLEAN NOT NULL DEFAULT false,
    comment       TEXT,
    created_by    UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quality_audit_records_batch_id ON quality_audit_records (batch_id);
CREATE INDEX idx_quality_audit_records_element_id ON quality_audit_records (element_id);


CREATE TABLE contents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    content_type    VARCHAR(32) NOT NULL
                    CHECK (content_type IN ('photo', 'short_video', 'live_stream')),
    review_policy   VARCHAR(32) NOT NULL DEFAULT 'post_then_review'
                    CHECK (review_policy IN ('post_then_review', 'review_before_post')),
    ai_risk_score   INT NOT NULL DEFAULT 0,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'ai_reviewing', 'ai_passed', 'ai_rejected', 'pending_human', 'in_human_review', 'approved', 'rejected')),
    creator_id      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contents_tenant_id ON contents (tenant_id);
CREATE INDEX idx_contents_content_type ON contents (content_type);
CREATE INDEX idx_contents_status ON contents (status);
CREATE INDEX idx_contents_creator_id ON contents (creator_id);
CREATE INDEX idx_contents_created_at ON contents (created_at DESC);


CREATE TABLE tenant_audit_rules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    rule_name    VARCHAR(128) NOT NULL,
    rule_expression TEXT,
    action       VARCHAR(32) NOT NULL,
    priority     INT NOT NULL DEFAULT 0,
    status       SMALLINT NOT NULL DEFAULT 1
                 CHECK (status IN (0, 1)),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_audit_rules_tenant_id ON tenant_audit_rules (tenant_id);


CREATE TABLE tenant_audit_levels (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    level_code  VARCHAR(32) NOT NULL,
    level_name  VARCHAR(64) NOT NULL,
    description TEXT,
    status      SMALLINT NOT NULL DEFAULT 1
                CHECK (status IN (0, 1)),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Per-tenant uniqueness: two tenants may both define 'warning' etc.
    CONSTRAINT uq_tenant_audit_levels_tenant_level UNIQUE (tenant_id, level_code)
);

CREATE INDEX idx_tenant_audit_levels_tenant_id ON tenant_audit_levels (tenant_id);


CREATE TABLE tenant_custom_words (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    word       VARCHAR(256) NOT NULL,
    category   VARCHAR(32),
    status     SMALLINT NOT NULL DEFAULT 1
               CHECK (status IN (0, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_custom_words_tenant_id ON tenant_custom_words (tenant_id);


CREATE TABLE contents_photo (
    content_id     UUID PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    title          TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    original_url   TEXT NOT NULL DEFAULT '',
    thumbnail_url  TEXT NOT NULL DEFAULT '',
    file_name      VARCHAR(256) NOT NULL DEFAULT '',
    file_size      BIGINT NOT NULL DEFAULT 0,
    mime_type      VARCHAR(128) NOT NULL DEFAULT '',
    width          INT NOT NULL DEFAULT 0,
    height         INT NOT NULL DEFAULT 0
);


CREATE TABLE contents_short_video (
    content_id         UUID PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    title              TEXT NOT NULL DEFAULT '',
    description        TEXT NOT NULL DEFAULT '',
    original_url       TEXT NOT NULL DEFAULT '',
    thumbnail_url      TEXT NOT NULL DEFAULT '',
    file_name          VARCHAR(256) NOT NULL DEFAULT '',
    file_size          BIGINT NOT NULL DEFAULT 0,
    mime_type          VARCHAR(128) NOT NULL DEFAULT '',
    duration           INT NOT NULL DEFAULT 0,
    video_fingerprint  VARCHAR(64) NOT NULL DEFAULT '',
    asr_text           TEXT NOT NULL DEFAULT ''
);


CREATE TABLE contents_live_stream (
    content_id  UUID PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    stream_url  TEXT NOT NULL DEFAULT '',
    play_url    TEXT NOT NULL DEFAULT '',
    frame_interval INT NOT NULL DEFAULT 15
);


CREATE TABLE content_elements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id      UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    element_kind    VARCHAR(32) NOT NULL
                    CHECK (element_kind IN ('cover_image', 'video_frame', 'title', 'comment', 'asr_text', 'user_avatar', 'nickname', 'description', 'live_snapshot')),
    element_content TEXT NOT NULL DEFAULT '',
    ai_risk_score   INT NOT NULL DEFAULT 0,
    ai_risk_types   TEXT[] NOT NULL DEFAULT '{}',
    ai_confidence   FLOAT8 NOT NULL DEFAULT 0,
    ai_status       VARCHAR(32) NOT NULL DEFAULT 'pending_ai'
                    CHECK (ai_status IN ('pending_ai', 'ai_processing', 'ai_passed', 'ai_rejected')),
    human_status    VARCHAR(32) NOT NULL DEFAULT 'pending_human'
                    CHECK (human_status IN ('pending_human', 'in_human_review', 'human_passed', 'human_rejected')),
    is_conflict     BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_elements_content_id ON content_elements (content_id);
CREATE INDEX idx_content_elements_ai_status ON content_elements (ai_status);
CREATE INDEX idx_content_elements_human_status ON content_elements (human_status);
CREATE INDEX idx_content_elements_is_conflict ON content_elements (is_conflict) WHERE is_conflict = true;
CREATE INDEX idx_content_elements_element_kind ON content_elements (element_kind);


CREATE TABLE audit_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id             UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    element_id          UUID NOT NULL REFERENCES content_elements(id),
    reviewer_id         UUID REFERENCES users(id),
    review_type         VARCHAR(32) NOT NULL
                        CHECK (review_type IN ('ai_primary', 'ai_judge', 'human', 'quality', 'appeal')),
    action              VARCHAR(32) NOT NULL
                        CHECK (action IN ('approve', 'reject', 'maintain', 'reverse')),
    penalty_level_code  VARCHAR(32),
    reason              VARCHAR(32),
    comment             TEXT,
    ai_score_before     INT,
    ai_score_after      INT,
    is_conflict         BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_records_element_id ON audit_records (element_id);
CREATE INDEX idx_audit_records_reviewer_id ON audit_records (reviewer_id);
CREATE INDEX idx_audit_records_review_type ON audit_records (review_type);
CREATE INDEX idx_audit_records_action ON audit_records (action);
CREATE INDEX idx_audit_records_created_at ON audit_records (created_at DESC);
CREATE INDEX idx_audit_records_is_conflict ON audit_records (is_conflict) WHERE is_conflict = true;


CREATE TABLE appeals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    content_id      UUID NOT NULL REFERENCES contents(id),
    applicant_id    UUID NOT NULL REFERENCES users(id),
    reason          TEXT NOT NULL,
    evidence_urls   TEXT[] NOT NULL DEFAULT '{}',
    status          VARCHAR(32) NOT NULL DEFAULT 'submitted'
                    CHECK (status IN ('submitted', 'under_review', 'resolved_approved', 'resolved_maintained')),
    reviewer_id     UUID REFERENCES users(id),
    resolution      VARCHAR(32),
    penalty_level_code VARCHAR(32),
    comment         TEXT,
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX idx_appeals_content_id ON appeals (content_id);
CREATE INDEX idx_appeals_applicant_id ON appeals (applicant_id);
CREATE INDEX idx_appeals_status ON appeals (status);
CREATE INDEX idx_appeals_submitted_at ON appeals (submitted_at DESC);

ALTER TABLE appeals ADD CONSTRAINT uq_appeals_content_applicant UNIQUE (content_id, applicant_id);


CREATE TABLE live_streams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    content_id  UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    stream_key  VARCHAR(128) NOT NULL,
    stream_url  TEXT,
    play_url    TEXT,
    status      VARCHAR(16) NOT NULL DEFAULT 'idle'
                CHECK (status IN ('idle', 'streaming', 'offline')),
    viewer_count INT NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_live_streams_tenant_id ON live_streams (tenant_id);
CREATE INDEX idx_live_streams_status ON live_streams (status);


CREATE TABLE live_frame_snapshots (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id     UUID NOT NULL REFERENCES live_streams(id) ON DELETE CASCADE,
    snapshot_url  TEXT NOT NULL,
    snapshot_time TIMESTAMPTZ NOT NULL,
    ai_risk_score INT NOT NULL DEFAULT 0,
    ai_risk_types TEXT[] NOT NULL DEFAULT '{}',
    ai_confidence FLOAT8 NOT NULL DEFAULT 0
);

CREATE INDEX idx_live_frame_snapshots_stream_id ON live_frame_snapshots (stream_id);
CREATE INDEX idx_live_frame_snapshots_time ON live_frame_snapshots (snapshot_time DESC);


CREATE TABLE ai_configs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id),
    agnes_api_key           TEXT NOT NULL DEFAULT '',
    agnes_endpoint          TEXT NOT NULL DEFAULT 'https://api.agnes.ai/v1/review',
    agnes_concurrency       INT NOT NULL DEFAULT 10,
    deepseek_api_key        TEXT NOT NULL DEFAULT '',
    deepseek_model          TEXT NOT NULL DEFAULT 'deepseek-chat',
    fallback_enabled        BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_configs_tenant_id ON ai_configs (tenant_id);

ALTER TABLE ai_configs ADD CONSTRAINT uq_ai_configs_tenant UNIQUE (tenant_id);


-- Fix tenants.created_by FK (added after users exists)
DO $$ BEGIN
    ALTER TABLE tenants
        ADD CONSTRAINT fk_tenants_created_by_users
            FOREIGN KEY (created_by) REFERENCES users(id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
