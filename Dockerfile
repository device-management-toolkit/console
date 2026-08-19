#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# syntax=docker/dockerfile:1.7

# Global build argument for all stages
ARG BUILD_TAGS=""

# Step 1: Modules caching
FROM golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS builder
# Build tags control dependencies:
# - Default (no tags): Full build with UI
# - noui: Excludes web UI assets
# Redeclare ARG to make it available in this stage
ARG BUILD_TAGS
ARG TARGETARCH=amd64
ARG TARGETVARIANT
RUN apk add --no-cache ca-certificates
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN mkdir -p /app/tmp/
# Convert hyphens to commas for Go build tags, keep hyphens for Docker stage names
# Use BuildKit cache mounts for faster iterative builds
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod/cache \
    sh -c 'BUILD_TAGS_GO=$(echo "$BUILD_TAGS" | tr "-" ","); \
    GOARM_VALUE=""; \
    if [ "$TARGETARCH" = "arm" ] && [ -n "$TARGETVARIANT" ]; then \
      GOARM_VALUE=${TARGETVARIANT#v}; \
      export GOARM=$GOARM_VALUE; \
    fi; \
    if [ -n "$BUILD_TAGS" ]; then \
      CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -tags="$BUILD_TAGS_GO" -o /bin/app ./cmd/app; \
    else \
      CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /bin/app ./cmd/app; \
    fi'
RUN mkdir -p /.config/device-management-toolkit

# Step 3: Final - Use scratch for all builds (all are fully static with pure Go)
FROM scratch
ENV TMPDIR=/tmp
ENV XDG_CONFIG_HOME=/.config
COPY --chown=65534:65534 --from=builder /app/tmp /tmp
COPY --chown=65534:65534 --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --chown=65534:65534 --from=builder /.config/device-management-toolkit /.config/device-management-toolkit
USER 65534:65534
CMD ["/app"]