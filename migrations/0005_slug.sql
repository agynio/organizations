-- Organizations gain a cluster-wide unique slug. It appears in identifiers that
-- must resolve without an organization already in context: an app's address and
-- an image proxy reference.
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS slug TEXT;

-- Backfill from the display name. Two passes: derive a candidate, then break
-- any collisions with a fragment of the id, so the unique index below cannot
-- fail on existing data.
WITH candidate AS (
    SELECT id,
           COALESCE(
               NULLIF(left(trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')), 64), ''),
               'org'
           ) AS slug
    FROM organizations
    WHERE slug IS NULL
)
UPDATE organizations o SET slug = candidate.slug
FROM candidate WHERE o.id = candidate.id;

WITH duplicated AS (
    SELECT id, slug,
           row_number() OVER (PARTITION BY slug ORDER BY created_at, id) AS position
    FROM organizations
)
UPDATE organizations o
SET slug = left(duplicated.slug, 55) || '-' || left(replace(o.id::text, '-', ''), 8)
FROM duplicated
WHERE o.id = duplicated.id AND duplicated.position > 1;

CREATE UNIQUE INDEX organizations_slug_key ON organizations (slug);

ALTER TABLE organizations ALTER COLUMN slug SET NOT NULL;
