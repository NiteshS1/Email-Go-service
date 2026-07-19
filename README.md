# Email Service

Microservice that sends emails on behalf of other services. It accepts requests over **HTTP** (sync) or **RabbitMQ** (async), renders HTML templates (or raw bodies), delivers via **SMTP**, and records outcomes in **PostgreSQL**. Optional **OpenTelemetry** export is supported.

## Architecture

| Layer | Technology |
| --- | --- |
| HTTP API | Go 1.24+, [Fiber](https://gofiber.io/) |
| Async | RabbitMQ (`email.send` queue) |
| Persistence | PostgreSQL (+ golang-migrate) |
| Delivery | SMTP |
| Templates | `welcome`, `otp`, `email` (HTML under `backend/templates/`) |
| Observability | Structured JSON logs, `/health`, optional OTLP traces |

```
Other services ──HTTP──► Email Service ──SMTP──► Mail provider
       │                      │
       └── RabbitMQ ──────────┘
                              └── PostgreSQL (send history)
```

For a deeper design write-up, see [`backend/docs/SYSTEM_DESIGN.md`](backend/docs/SYSTEM_DESIGN.md).

## Prerequisites

- Go **1.24+** (see `backend/go.mod`)
- Docker & Docker Compose (for Postgres / RabbitMQ / full stack)
- SMTP credentials (required to actually send mail)

## Quick start

All make targets below are run from `backend/`.

### Option A — Full stack with Docker

```bash
cd backend
# Set SMTP_* in your environment or a .env file next to docker-compose.yml
make docker-up
```

This starts Postgres, RabbitMQ, and the app. The API is published on **http://localhost:3000** (container listens on 8080).

- RabbitMQ management UI: http://localhost:15672 (`guest` / `guest`)
- Swagger UI: http://localhost:3000/swagger

Stop with `make docker-down`.

### Option B — Local app + Docker infra

```bash
cd backend
make up-infra          # Postgres + RabbitMQ
# Create backend/.env using the example in Configuration below
make dev               # go run cmd/main.go
```

API defaults to **http://localhost:8080**.

## Configuration

The service loads a `.env` file from the working directory when present (`godotenv`), then reads the environment.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DB_USER` | yes | — | Postgres user |
| `DB_PASSWORD` | yes | — | Postgres password |
| `DB_NAME` | yes | — | Database name |
| `DB_HOST` | no | `localhost` | Postgres host |
| `DB_PORT` | no | `5432` | Postgres port |
| `APP_PORT` | no | `8080` | HTTP listen port |
| `RABBITMQ_URL` | no | — | If unset, the async consumer is disabled |
| `SMTP_FROM` | no* | — | From address |
| `SMTP_HOST` | no* | — | SMTP host |
| `SMTP_PORT` | no | `587` | SMTP port |
| `SMTP_PASSWORD` | no* | — | SMTP password / app password |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | — | OTLP HTTP endpoint; tracing disabled if empty |

\*Needed for real delivery; without SMTP the service still boots but sends will fail.

**Local `.env` example** (matches Compose defaults):

```env
DB_USER=postgres
DB_PASSWORD=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=email_db
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
APP_PORT=8080
SMTP_FROM=you@example.com
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_PASSWORD=your-smtp-password
```

Migrations run automatically on startup.

## API overview

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Liveness + DB ping (`{"status":"ok"}` or unhealthy) |
| `POST` | `/emails/send` | Template-based send (preferred) |
| `POST` | `/send-email` | Raw body send |
| `GET` | `/swagger` | Swagger UI |
| `GET` | `/swagger.json` | OpenAPI JSON |
| `GET` | `/swagger.yaml` | OpenAPI YAML |

### Template send example

```bash
curl -s -X POST http://localhost:8080/emails/send \
  -H 'Content-Type: application/json' \
  -H 'X-Trace-ID: 11111111-1111-1111-1111-111111111111' \
  -d '{
    "tenant_id": 1,
    "service_id": 101,
    "receiver_email": "user@example.com",
    "template": "welcome",
    "subject": "Welcome",
    "data": { "name": "Ada" }
  }'
```

Built-in templates: `welcome`, `otp`, `email`. Optional `attachments` is a list of `{ "name", "url" }` (URLs are fetched and attached).

Trace ID resolution for `/emails/send`: body `trace_id` → `X-Trace-ID` header → generated UUID.

## Async (RabbitMQ)

Publish JSON with the **same shape** as `POST /emails/send` to the durable queue **`email.send`**. Failed/poison handling uses **`email.send.dlq`** (declared for dead-letter use).

A sample producer lives at [`frontend/test_producer.py`](frontend/test_producer.py).

Full guide: [`backend/docs/RABBITMQ.md`](backend/docs/RABBITMQ.md).  
Running RabbitMQ on a separate host: [`backend/docs/whatTODO.md`](backend/docs/whatTODO.md).

## Project layout

```
.
├── README.md
├── backend/
│   ├── cmd/main.go              # Entrypoint
│   ├── docker-compose.yml       # Postgres, RabbitMQ, backend
│   ├── Dockerfile
│   ├── Makefile
│   ├── migrations/              # SQL migrations
│   ├── templates/               # HTML email templates
│   ├── docs/                    # Design / RabbitMQ / tracing notes
│   └── internal/
│       ├── config/
│       ├── consumer/            # RabbitMQ consumer
│       ├── domain/
│       ├── fetcher/             # Attachment URL fetching
│       ├── handler/
│       ├── infrastructure/      # DB, SMTP, migrate
│       ├── repository/
│       ├── routes/              # HTTP routes + embedded OpenAPI
│       ├── service/
│       └── utils/
└── frontend/
    └── test_producer.py         # Example RabbitMQ publisher
```

## Makefile & CI

From `backend/`:

| Target | Purpose |
| --- | --- |
| `make up-infra` | Postgres + RabbitMQ |
| `make up-rabbitmq` | RabbitMQ only |
| `make dev` | Run the app locally |
| `make test` / `make test-cover` | Unit tests |
| `make build` / `make start` | Build and run binary |
| `make docker-up` / `make docker-down` | Full Compose stack |
| `make lint` | `golangci-lint` |
| `make migrate-up` / `make migrate-down` | Manual migrations (CLI) |

GitLab CI (`.gitlab-ci.yml`): **lint** → **test** → **build** (container image on default branch / tags).

## Observability

- **Health:** `GET /health` — returns `503` if the database is unreachable (useful for load balancers / probes).
- **Logs:** JSON to stdout via `slog`.
- **Tracing:** set `OTEL_EXPORTER_OTLP_ENDPOINT` to export spans. Details: [`backend/docs/TRACING.md`](backend/docs/TRACING.md).

## Further reading

- [`backend/docs/SYSTEM_DESIGN.md`](backend/docs/SYSTEM_DESIGN.md) — system design
- [`backend/docs/RABBITMQ.md`](backend/docs/RABBITMQ.md) — async email queue
- [`backend/docs/TRACING.md`](backend/docs/TRACING.md) — OpenTelemetry / Tempo
- [`backend/docs/whatTODO.md`](backend/docs/whatTODO.md) — separate RabbitMQ host
