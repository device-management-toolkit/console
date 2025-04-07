# Step 1: Modules caching
FROM golang:1.24-alpine3.20@sha256:00f149d5963f415a8a91943531b9092fde06b596b276281039604292d8b2b9c8 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN apk add --no-cache git
RUN go mod download

# Step 2: Builder
FROM golang:1.24-alpine3.20@sha256:00f149d5963f415a8a91943531b9092fde06b596b276281039604292d8b2b9c8 AS builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o /bin/app ./cmd/app

# Step 3: Final
FROM scratch
COPY --from=builder /app/config /config
COPY --from=builder /app/internal/app/migrations /migrations
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/app"]