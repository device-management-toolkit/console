#*********************************************************************
# * Copyright (c) Intel Corporation 2025
# * SPDX-License-Identifier: Apache-2.0
# **********************************************************************

# Step 1: Modules caching
FROM golang:1.25.5-alpine@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.25.5-alpine@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9 AS builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN go mod tidy
RUN mkdir -p /app/tmp/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o /bin/app ./cmd/app
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