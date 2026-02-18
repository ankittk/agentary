CREATE TABLE IF NOT EXISTS network_allowlist (
  domain TEXT PRIMARY KEY
);
INSERT INTO network_allowlist(domain) VALUES('*') ON CONFLICT (domain) DO NOTHING;
