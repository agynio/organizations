ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS sandbox_default_idle_timeout TEXT NOT NULL DEFAULT '30m',
    ADD COLUMN IF NOT EXISTS sandbox_default_ttl TEXT NOT NULL DEFAULT '72h';
