-- P1 fix (2026-08-26): level_code was globally UNIQUE, so two tenants could
-- not both define e.g. 'warning'. Scope uniqueness to the tenant instead.
ALTER TABLE tenant_audit_levels
    DROP CONSTRAINT tenant_audit_levels_level_code_key;

-- Drop the old index if a deployment created it under a different name.
DROP INDEX IF EXISTS tenant_audit_levels_level_code_idx;

ALTER TABLE tenant_audit_levels
    ADD CONSTRAINT uq_tenant_audit_levels_tenant_level
    UNIQUE (tenant_id, level_code);
