# Ouroboros roadmap

## v0.1 (current dev)

- [x] Ouroboros module and CLI branding
- [x] Semver catalog matching (`--semver-match`)
- [x] SQLite agent store (`ouroboros agent`)
- [x] Drift query API stub in `internal/store`
- [x] `OUROBOROS_*` environment variables for tests and multi-user roots
- [ ] `ouroboros drift` CLI (show version changes since last scan)

## v0.2 — Security / web3 positioning

- [ ] Curated **blockchain-adjacent** exposure catalog samples (npm/pypi/go + MCP docker pins)
- [ ] Document SIEM / SOAR ingestion (NDJSON + HTTP sink unchanged)
- [ ] SARIF export for findings (optional sink)

## v0.3 — Intel & roots

- [ ] Optional workers: VirusTotal / Shodan enrichment (API keys via env, no vendor lock-in)
- [ ] Git remote and network URL roots (clone or fetch lockfiles)
- [ ] Codex `config.toml` MCP server entries (today: JSON MCP configs only)

## v0.4 — Platform

- [ ] Windows CI + release artifacts (`.goreleaser.yaml`)
- [ ] Postgres store driver mirroring SQLite schema
- [ ] SPDX / CycloneDX SBOM emission from scan results

## Open core / commercial (ideas only)

- Hosted drift dashboard and fleet policy
- Private catalog sync and advisory SLA
- Enterprise SSO for HTTP sink — implementation stays generic HTTP + bearer token
