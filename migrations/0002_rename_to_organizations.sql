ALTER TABLE tenants RENAME TO organizations;

ALTER INDEX tenants_pkey RENAME TO organizations_pkey;
