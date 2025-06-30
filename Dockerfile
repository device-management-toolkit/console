# Step 1: Modules caching
FROM golang:1.24.4-alpine3.20@sha256:e5c2e59960f8636d02f77029c8f0a7a6b882f87fee8d2e4a9ce6c9ff112ed735 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.24.4-alpine3.20@sha256:e5c2e59960f8636d02f77029c8f0a7a6b882f87fee8d2e4a9ce6c9ff112ed735 AS builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN mkdir -p /app/tmp/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o /bin/app ./cmd/app

# Step 3: Final
FROM scratch
ENV TMPDIR=/tmp
COPY --from=builder /app/tmp /tmp
COPY --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/app"]