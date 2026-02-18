-- 002_network.sql
-- Network allowlist configuration (global, not per-team).

CREATE TABLE IF NOT EXISTS network_allowlist (
  domain TEXT PRIMARY KEY
);

-- Default to unrestricted wildcard.
INSERT OR IGNORE INTO network_allowlist(domain) VALUES('*');

