-- Revert to the original global UNIQUE on level_code.
-- NOTE: fails if data already violates the global constraint
-- (two tenants using the same level_code) — deduplicate first.
ALTER TABLE tenant_audit_levels
    DROP CONSTRAINT IF EXISTS uq_tenant_audit_levels_tenant_level;

ALTER TABLE tenant_audit_levels
    ADD CONSTRAINT tenant_audit_levels_level_code_key
    UNIQUE (level_code);
