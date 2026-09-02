# ── build stage ──────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Cache dependency downloads separately from source compilation.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully static binary with no CGO dependencies.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /eventhorizon .

# ── runtime stage (distroless) ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian13:nonroot

# Copy the compiled binary and static web assets.
COPY --from=builder /eventhorizon /eventhorizon
COPY --from=builder /src/static /static

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/eventhorizon"]
