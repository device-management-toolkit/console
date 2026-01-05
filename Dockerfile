#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Step 1: Modules caching
FROM golang:1.25.5-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.25.5-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder
# Build tags control dependencies:
# - Default (no tags): Includes SQLite (requires CGO_ENABLED=1)
# - nosqlite: PostgreSQL-only, enables fully static binaries with CGO_ENABLED=0
# - noui: Excludes web UI assets
ARG BUILD_TAGS=""
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go mod tidy
RUN mkdir -p /app/tmp/
# Use CGO_ENABLED=0 only for nosqlite builds (fully static with pure Go PostgreSQL driver)
# Default builds (with SQLite) require CGO_ENABLED=1
RUN if echo "$BUILD_TAGS" | grep -q "nosqlite"; then \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags="$BUILD_TAGS" -o /bin/app ./cmd/app; \
    elif [ -n "$BUILD_TAGS" ]; then \
      CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags="$BUILD_TAGS" -o /bin/app ./cmd/app; \
    else \
      CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /bin/app ./cmd/app; \
    fi
RUN mkdir -p /.config/device-management-toolkit
# Step 3: Final
FROM scratch
ENV TMPDIR=/tmp
COPY --from=builder /app/tmp /tmp
COPY --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /.config/device-management-toolkit /.config/device-management-toolkit
CMD ["/app"]