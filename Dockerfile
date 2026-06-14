# syntax=docker/dockerfile:1.7

# ─────────────────────────────────────────────────────────────────────────────
# Ouroboros — read-only endpoint package-inventory scanner.
#
# NOTE ON CONTAINERIZED SCANNING: by default the binary scans the *container's*
# filesystem, which holds no developer packages and is therefore near-empty.
# To inventory a real host, mount the host paths read-only and point a `deep`
# scan at them, e.g.:
#
#   docker run --rm --read-only \
#     -v "$HOME:/scan/home:ro" \
#     ghcr.io/daemon-blockint-tech/ouroboros:latest \
#     scan --profile deep --root /scan/home
#
# The image is built from a pure-Go binary (CGO disabled; modernc.org/sqlite is
# pure Go), so it runs on the distroless `static` base with no libc.
# ─────────────────────────────────────────────────────────────────────────────

# ---- Build stage ------------------------------------------------------------
# Pin the toolchain to the version in go.mod (go 1.25.x). For reproducible
# supply-chain hygiene, pin by digest in CI (golang:1.25-bookworm@sha256:...).
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build

# Cross-compile targets injected by buildx; default to the build platform.
ARG TARGETOS
ARG TARGETARCH
# Version stamped into the binary (-X main.Version). Pass --build-arg VERSION=…
# from CI (git tag); falls back to the repo VERSION file at build time.
ARG VERSION=""

ENV CGO_ENABLED=0 \
    GOFLAGS=-mod=readonly

WORKDIR /src

# Download modules first so the layer is cached unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Build the static binary. Flags mirror .goreleaser.yaml (-trimpath, -s -w).
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    VERSION="${VERSION:-$(cat VERSION 2>/dev/null || echo dev)}"; \
    GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH:-amd64}" \
    go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" \
        -o /out/ouroboros ./cmd/ouroboros

# Smoke-test the freshly built binary against the embedded fixtures so a broken
# build fails here, not in production. Runs only when not cross-compiling.
RUN if [ "${TARGETARCH:-amd64}" = "$(go env GOHOSTARCH)" ]; then \
        /out/ouroboros selftest >/dev/null && /out/ouroboros version; \
    fi

# ---- Runtime stage ----------------------------------------------------------
# distroless `static` carries CA certificates (needed for the HTTPS sink) and a
# /etc/passwd `nonroot` entry (uid 65532, HOME=/home/nonroot) but no shell or
# package manager — minimal attack surface for a tool that may run fleet-wide.
FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION="dev"
LABEL org.opencontainers.image.title="ouroboros" \
      org.opencontainers.image.description="Read-only endpoint package-inventory scanner (NDJSON, exposure catalogs, SQLite agent)" \
      org.opencontainers.image.source="https://github.com/daemon-blockint-tech/ouroboros" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/ouroboros /usr/local/bin/ouroboros
# Bundle the example threat-intel catalogs and license alongside the binary.
COPY --from=build /src/threat_intel /opt/ouroboros/threat_intel
COPY --from=build /src/LICENSE /opt/ouroboros/LICENSE

# uid:gid for the distroless nonroot user; no privileged work is ever needed.
USER 65532:65532
WORKDIR /home/nonroot

ENTRYPOINT ["/usr/local/bin/ouroboros"]
# Default to the offline self-test so `docker run <image>` proves the image works.
CMD ["selftest"]
