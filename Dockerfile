#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Global build argument for all stages
ARG BUILD_TAGS=""

# Step 1: Modules caching
FROM golang:1.25.7-alpine@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.25.7-alpine@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS builder
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
COPY --from=builder /app/tmp /tmp
COPY --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /.config/device-management-toolkit /.config/device-management-toolkit
CMD ["/app"]