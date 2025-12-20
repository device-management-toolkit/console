#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Step 1: Modules caching
FROM golang:1.25.5-alpine@sha256:72567335df90b4ed71c01bf91fb5f8cc09fc4d5f6f21e183a085bafc7ae1bec8 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.25.5-alpine@sha256:72567335df90b4ed71c01bf91fb5f8cc09fc4d5f6f21e183a085bafc7ae1bec8 AS builder
ARG BUILD_TAGS=""
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go mod tidy
RUN mkdir -p /app/tmp/
RUN if [ -z "$BUILD_TAGS" ]; then \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/app ./cmd/app; \
    else \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags="$BUILD_TAGS" -o /bin/app ./cmd/app; \
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