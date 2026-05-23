# Ouroboros

**Ouroboros** is a read-only endpoint inventory scanner for developer and build machines. It walks configured filesystem roots, detects installed third-party packages and MCP server definitions from lockfiles and install metadata, and emits structured NDJSON records. Optionally, it matches discoveries against an operator-supplied **exposure catalog** to surface policy violations and known-risk components.

The tool does not run package managers, resolve dependencies over the network, or execute installed code. Output is suitable for local inspection, log pipelines, or a generic HTTPS ingest endpoint.

Ouroboros targets security and platform teams who need **repeatable visibility** into what is installed on laptops, CI runners, and shared dev hosts—including **web3 and agentic workflows** where MCP servers and container-pinned tools are part of the attack surface.

## Features

- **Profiles** (`baseline`, `project`, `deep`) — tune scan breadth without changing the core schema
- **Ecosystems** — npm, pnpm, yarn, bun, PyPI, Go modules, Rubygems, Composer, MCP (JSON configs), editor and browser extensions, and more
- **Exposure catalog** — YAML/JSON entries with severity; optional **`--semver-match`** for constraint-style versions (`>=`, `<=`, …), not only exact pins
- **Transports** — stdout (default), local NDJSON file, or HTTP POST with bearer token (env-configurable)
- **Agent mode** — periodic scans with **SQLite** history for drift analysis (`~/.ouroboros/agent.db`)
- **Selftest** — embedded fixtures and catalog; no network required

## Quick start

```bash
git clone https://github.com/daemon-blockint-tech/ouroboros.git
cd ouroboros
go build -o ouroboros ./cmd/ouroboros
./ouroboros scan --profile project
./ouroboros selftest
```

Scan with exposure matching:

```bash
./ouroboros scan --profile project \
  --exposure-catalog ./threat_intel \
  --semver-match
```

Run the local agent once:

```bash
./ouroboros agent --once --profile project --exposure-catalog ./threat_intel
```

## Commands

| Command | Purpose |
|---------|---------|
| `scan` | One-shot inventory to stdout, file, or HTTP sink |
| `roots` | Print resolved scan roots for a profile and exit |
| `agent` | Periodic scans persisted to SQLite |
| `selftest` | Embedded fixtures + catalog smoke test |
| `version` | Print build version |

Run `ouroboros scan --help` for the full flag list (ecosystem filters, excludes, device ID, findings-only mode, and sink options).

## Output model

Records are **NDJSON** with a stable schema (`schema/v0.1.0/`). Package rows describe what was found; finding rows link a package to a catalog entry; `scan_summary` closes each run with counters and timing.

See [docs/inventory-sources.md](docs/inventory-sources.md), [docs/state-model.md](docs/state-model.md), and [docs/transport.md](docs/transport.md) for operator-facing detail.

## Configuration

- **Roots** — derived from profile plus optional explicit paths; see `ouroboros roots`
- **Catalog** — one file or a directory of JSON files; see [threat_intel/](threat_intel/) for examples
- **Secrets** — HTTP token and device ID via flags or environment variables (see `docs/transport.md`)

## Roadmap

Planned work is tracked in [docs/ROADMAP.md](docs/ROADMAP.md) (Postgres store, SARIF/SBOM export, additional MCP config formats, fleet dashboards, and cross-platform releases).

## License

Apache License 2.0 — see [LICENSE](LICENSE). Third-party attributions and notices are in [NOTICE](NOTICE).
