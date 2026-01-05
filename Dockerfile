#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Global build argument for all stages
ARG BUILD_TAGS=""

# Step 1: Modules caching
FROM golang:1.25.5-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.25.5-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder
# Build tags control dependencies:
# - Default (no tags): Includes SQLite using modernc.org/sqlite (pure Go)
# - nosqlite: PostgreSQL-only
# - noui: Excludes web UI assets
# Redeclare ARG to make it available in this stage
ARG BUILD_TAGS
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go mod tidy
RUN mkdir -p /app/tmp/
# Use CGO_ENABLED=0 for all builds since modernc.org/sqlite is pure Go
# This allows fully static binaries with scratch base image for all variants
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
COPY --from=builder /app/tmp /tmp
COPY --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /.config/device-management-toolkit /.config/device-management-toolkit
CMD ["/app"]