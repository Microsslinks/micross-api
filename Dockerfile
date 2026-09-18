# =============================================================================
#  MicrossAPI / Microsslink 微观互联 — production image
#  Repository: https://github.com/dukaworks/micross-api
#
#  This Dockerfile is the build recipe for THIS repository. It compiles the
#  Go backend and embeds the Rsbuild-built web/dist (Bun stage), then ships
#  a debian-slim runtime image with the `micross-api` binary on PATH.
#
#  Upstream Go module path (github.com/QuantumNous/new-api/...) is preserved
#  unchanged per AGPLv3 §7 attribution, so the ldflags target stays as
#  github.com/QuantumNous/new-api/common.Version even though the produced
#  binary, container image, and source repo belong to dukaworks/micross-api.
#
#  Build:    docker build -t ghcr.io/dukaworks/micross-api:latest -f Dockerfile .
#  Publish:  docker push ghcr.io/dukaworks/micross-api:latest
#  Run:      docker run --rm -p 3000:3000 -v $(pwd)/data:/data \
#                -e TZ=Asia/Shanghai ghcr.io/dukaworks/micross-api:latest
# =============================================================================

# ---- Stage 1: web frontend (Rsbuild via Bun) ---------------------------------
FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder

WORKDIR /build/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY ./web ./
COPY ./VERSION /build/VERSION
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat /build/VERSION) bun run build

# ---- Stage 2: Go backend ------------------------------------------------------
FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2

ENV GO111MODULE=on CGO_ENABLED=0 GOWORK=off

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

# Build-time metadata (passed via `docker build --build-arg`)
# - VERSION     is read from the in-repo VERSION file by default
# - GIT_COMMIT  is informational only; not yet wired into common (kept as ARG
#               so the OCI label below picks it up)
# - BUILD_TIME  is informational only; not yet wired into common (kept as ARG
#               so the OCI label below picks it up)
ARG VERSION=local
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /build

ADD go.mod go.sum ./
# relaykit is a local submodule referenced via replace; its go.mod must be
# present for go mod download to resolve the main module graph.
ADD relaykit/go.mod ./relaykit/go.mod
RUN go mod download

COPY . .
COPY --from=builder /build/web/dist ./web/dist
RUN go build \
    -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=${VERSION}'" \
    -o micross-api

# ---- Stage 3: runtime -------------------------------------------------------
FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

# Re-declare ARG for this stage (ARG scope is per-stage in multi-stage builds)
ARG VERSION=local
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# OCI image metadata
LABEL org.opencontainers.image.title="MicrossAPI" \
      org.opencontainers.image.description="新一代大模型网关与 AI 资产管理系统（Microsslink 微观互联出品）" \
      org.opencontainers.image.source="https://github.com/dukaworks/micross-api" \
      org.opencontainers.image.url="https://github.com/dukaworks/micross-api" \
      org.opencontainers.image.documentation="https://github.com/dukaworks/micross-api/blob/main/README.zh_CN.md" \
      org.opencontainers.image.vendor="MicrossAPI / Microsslink 微观互联" \
      org.opencontainers.image.licenses="AGPL-3.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.authors="dukaworks"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/micross-api /
COPY LICENSE NOTICE THIRD-PARTY-LICENSES.md /licenses/
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/micross-api"]