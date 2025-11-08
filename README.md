# Ride Matching (Uber)

This repository is a small ride-matching prototype (Uber-like) written in Go. It includes pluggable components for production-like subsystems and example deployment artifacts for local development and Kubernetes.

Key components

- HTTP API server (`cmd/server`) — exposes rider/driver endpoints and WebSocket endpoints for drivers.
- Redis Geo (`internal/geo`) — stores driver locations using Redis GEO and per-driver metadata.
- Location ingest (Kafka / Redpanda) — drivers publish GPS messages to Kafka; `cmd/consumer` reads the topic and updates Redis.
- Matcher (`internal/matcher`) — finds nearby drivers, computes pickup ETA and a cost score (pickup ETA + rating penalty + surge placeholder), and offers a match via the Dispatcher.
- Dispatcher (`internal/dispatch`) — WebSocket registry for live driver sessions + HTTP push fallback (FCM example provided).
- ETA service (`internal/eta`) — OSRM HTTP client plus a tiny in-memory TTL cache; the matcher can be configured to use OSRM for realistic ETA lookups.
- Trip store (`internal/storage`) — durable ride persistence with a Postgres-backed `PostgresStore` and an in-memory `MemoryStore` for development.
- Payments (`internal/payments`) — small Stripe wrapper to implement hold (manual capture), capture, and cancel flows.
- Observability — Prometheus metrics exported at `/metrics`; example Grafana + Prometheus compose/dev files included.

High-level data flow

1. Drivers (mobile clients) periodically publish their location messages to Kafka (topic: `driver-locations`).
2. The `cmd/consumer` process (or a consumer deployed alongside services) reads messages from Kafka and:
   - GEOADD into Redis to keep an up-to-date geo index
   - HSET driver metadata (rating, online, updated)
3. Rider calls `POST /api/v1/rides/request` on the HTTP API with origin/destination.
4. The Matcher queries Redis Geo for nearby drivers and obtains pickup ETA for each candidate:
   - If configured, the matcher calls an OSRM service via `internal/eta` for route-based ETA.
   - A small ETA cache is consulted to reduce repeated OSRM requests.
5. The matcher scores candidates using a cost function (ETA + rating penalty + surge factor) and selects the best candidate.
6. The Dispatcher delivers a match offer to the driver via an open WebSocket session, or falls back to HTTP push (FCM example).
7. When a match is accepted, the server persists a Ride to Postgres (`internal/storage.PostgresStore`) and the payments subsystem can place a hold (Stripe PaymentIntent with capture_method=manual).
8. On ride completion the server captures funds; on cancel it cancels the PaymentIntent.

Local development (quick start)

- Start local dependencies (Redis, Postgres, Redpanda, Prometheus, Grafana):

```sh
make compose-up
```

- Run the API server locally (dev mode):

```sh
make build
HTTP_ADDR=:8080 ./bin/ride-matching
```

- Run the consumer locally (reads Kafka and updates Redis):

```sh
make run-consumer
# or: KAFKA_BROKERS=localhost:9092 REDIS_ADDR=localhost:6379 go run ./cmd/consumer
```

- Example API calls:

```sh
curl -XPOST localhost:8080/internal/driver/locations -d '{"id":"d1","loc":{"lat":37.77,"lon":-122.41},"rating":4.7}'
curl -XPOST localhost:8080/api/v1/rides/request -d '{"rider_id":"r1","origin":{"lat":37.7749,"lon":-122.4194},"destination":{"lat":37.7929,"lon":-122.3969}}'
```

Observability

- Prometheus metrics are exposed at `/metrics` on the server (default :8080) and at `:2112` in the consumer process. The compose includes `prometheus` and `grafana` services for local dashboards.
- HTTP middleware now emits structured JSON logs (request id, latency, status) and Prometheus metrics (`ride_matching_http_requests_total`, `ride_matching_http_request_duration_seconds`) for each API route.

Configuration / environment variables

- REDIS_ADDR — Redis host:port (e.g. localhost:6379)
- REDIS_PASSWORD — optional password when Redis auth is enabled
- REDIS_GEO_KEY — Redis key used for driver GEO data (default: `drivers_geo`)
- KAFKA_BROKERS — comma-separated broker list (e.g. localhost:9092)
- KAFKA_TOPIC — topic for driver locations (default: `driver-locations`)
- KAFKA_GROUP — consumer group id for the consumer (default: `ride-matching-consumer`)
- PG_DSN — Postgres DSN for `PostgresStore` (if set, TripStore defaults to Postgres)
- STRIPE_API_KEY — Stripe secret key for payments flows
- HTTP_ADDR — HTTP bind address (default: `:8080`)
- HTTP_READ_TIMEOUT / HTTP_WRITE_TIMEOUT / HTTP_IDLE_TIMEOUT — duration strings to tighten HTTP server timeouts (defaults: `5s`, `10s`, `120s`)
- HTTP_SHUTDOWN_TIMEOUT — graceful shutdown timeout (default: `15s`)
- MATCHER_DEFAULT_SPEED_MPS — fallback driver speed used when computing ETA (default: `10`)
- MATCHER_TOP_N — number of drivers to score per match request (default: `8`)
- LOG_LEVEL — `debug`, `info`, `warn`, or `error` (default: `info`)
- MIGRATE — when `true` and `PG_DSN` is set, the server will run `migrations/001_create_rides.sql` before starting

Kubernetes

- Manifests live in `deploy/k8s/`:
  - `configmap.yaml` — example config map for simple env wiring
  - `deployment.yaml` — API deployment (readiness -> `/ready`, liveness -> `/healthz`, Prometheus annotations)
  - `service.yaml` — ClusterIP service
  - `hpa.yaml` — example HorizontalPodAutoscaler
  - `consumer-job.yaml` — example Job manifest to run the consumer in-cluster

Notes and next steps

- This repo is a prototype and intended as scaffolding: production hardening required for HA, security, secrets management, retries, idempotency, and observability.
- Recommended next steps:
  - Configure TLS/auth for Kafka and Redis in your environment
- Use a robust migration tool (e.g. golang-migrate) instead of the demo initContainer
- Add end-to-end integration tests that run against the compose stack
- Add dashboards and alerting rules in Grafana/Prometheus

## Production hardening roadmap

Refer to `docs/production-grade-roadmap.md` for a detailed plan to evolve this prototype into a production-grade, multi-region platform capable of handling Uber-scale traffic.
