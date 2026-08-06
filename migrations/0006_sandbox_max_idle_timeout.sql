-- The ceiling a sandbox creator may ask for, separate from the default they get
-- when they ask for nothing. One field serving as both would make the default
-- the most expensive option on offer.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS sandbox_max_idle_timeout TEXT NOT NULL DEFAULT '24h';
