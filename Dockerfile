#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Global build argument for all stages
ARG BUILD_TAGS=""

# Step 1: Modules caching
FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS builder
# Build tags control dependencies:
# - Default (no tags): Full build with UI
# - noui: Excludes web UI assets
# Redeclare ARG to make it available in this stage
ARG BUILD_TAGS
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go mod tidy
RUN mkdir -p /app/tmp/
# Convert hyphens to commas for Go build tags, keep hyphens for Docker stage names
RUN BUILD_TAGS_GO=$(echo "$BUILD_TAGS" | tr '-' ','); \
    if [ -n "$BUILD_TAGS" ]; then \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags="$BUILD_TAGS_GO" -o /bin/app ./cmd/app; \
    else \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/app ./cmd/app; \
    fi
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