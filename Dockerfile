# ── build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26.1-alpine AS builder

WORKDIR /src

# Cache dependency downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully static binary with no CGO dependencies.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /eventhorizon .

# ── runtime stage ─────────────────────────────────────────────────────────────
FROM scratch

# Copy TLS CA certificates so the binary can verify TLS connections (e.g. Redis TLS).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary and static web assets.
COPY --from=builder /eventhorizon /eventhorizon
COPY --from=builder /src/static /static

EXPOSE 8080

ENTRYPOINT ["/eventhorizon"]
