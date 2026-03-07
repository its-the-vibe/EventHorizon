# EventHorizon

A kinetic canvas where fleeting messages converge and disappear.

EventHorizon is a Go web service that listens to a Redis Pub/Sub channel and
streams every incoming phrase to connected browsers via **Server-Sent Events**.
Each phrase materialises on screen at a random position, large and glowing,
then drifts toward the centre while shrinking and fading – until it collapses
into a pinpoint and vanishes. Multiple phrases can coexist and overlap on the
canvas at any time.

---

## Architecture

```
Redis channel ──► Go backend (SSE) ──► Browser (kinetic canvas)
```

- **Backend** – Go 1.24, zero-dependency HTTP server, Redis subscriber,
  SSE broadcaster
- **Frontend** – single HTML file; pure vanilla JS with `requestAnimationFrame`
  animations and an `EventSource` connection

---

## Requirements

- [Go 1.24+](https://go.dev/dl/)
- A Redis server (6.x or later)

---

## Configuration

### Config file

Copy the example and edit it:

```bash
cp config.example.yaml config.yaml
```

| Key | Default | Description |
|-----|---------|-------------|
| `server.host` | `""` | Host/IP to listen on (empty = all interfaces) |
| `server.port` | `8080` | HTTP port |
| `redis.host` | `localhost` | Redis hostname |
| `redis.port` | `6379` | Redis port |
| `redis.db` | `0` | Redis database number |
| `redis.channel` | `eventhorizon` | Pub/Sub channel to subscribe to |

> **`config.yaml` is gitignored.** Commit `config.example.yaml` instead.

### Environment variables / `.env`

Sensitive values are read from environment variables (or a `.env` file):

```bash
cp .env.example .env
# edit .env and set your actual Redis password
```

| Variable | Description |
|----------|-------------|
| `REDIS_PASSWORD` | Redis AUTH password (leave empty if not set) |
| `CONFIG_PATH` | Override the config file path (default: `config.yaml`) |

> **`.env` is gitignored.** Commit `.env.example` instead.

---

## Running locally

```bash
# 1. Install dependencies
go mod download

# 2. Configure
cp config.example.yaml config.yaml   # edit redis.host etc.
cp .env.example .env                  # set REDIS_PASSWORD

# 3. Start the server
go run .

# 4. Open your browser
open http://localhost:8080

# 5. Publish a test message
redis-cli publish eventhorizon "Hello, EventHorizon!"
```

---

## Docker / Docker Compose

The production image is built on `scratch` (minimal attack surface, no shell).

```bash
# Build the image
docker compose build

# Start the service  (Redis must be reachable at the host configured in config.yaml)
docker compose up
```

The container runs **read-only** (`read_only: true`). Redis is hosted externally
and is **not** part of the Compose file.

---

## Publishing messages

Send any UTF-8 string to the configured Redis channel:

```bash
redis-cli publish eventhorizon "Your message here"
```

or from within your application using any Redis client library.

