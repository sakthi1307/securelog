# SecureLog

SecureLog is a lightweight security event ingestion and alerting service built in Go.  
It ingests security events at high throughput, correlates them asynchronously, and raises deduplicated alerts using simple detection rules.

This project is intentionally scoped as a **weekend backend system** to demonstrate real-world backend engineering patterns used in SOC (Security Operations Center) platforms.

---

## Features

- **Batch security event ingestion**
  - REST API to ingest events concurrently
  - Bounded worker pool using Go channels
  - Per-item success/failure response

- **Asynchronous alerting**
  - Events are published to a rule engine via channels
  - Detection rule:
    > 5 `login_failed` events from the same IP within 2 minutes

- **Deduplicated alerts**
  - One open alert per fingerprint (rule + entity)
  - Alerts are updated instead of duplicated
  - Alerts can be acknowledged or closed

- **Production-style backend foundations**
  - PostgreSQL with proper indexing
  - Partial unique indexes for alert deduplication
  - Structured logging
  - API key authentication
  - Graceful shutdown

---

## Architecture Overview

HTTP API (chi)
|
v
Event Ingest Handler
|
| (bounded worker pool)
v
PostgreSQL (events)
|
| (publish via channel)
v
Rule Engine Goroutine
|
v
PostgreSQL (alerts)




**Design goal:**  
Ingestion is decoupled from alert evaluation so the write path remains fast even if rule evaluation becomes expensive.

---

## API Endpoints

### Health & Readiness
GET /healthz
GET /v1/readyz


---

### Event Ingestion

**Request**
```json
{
  "events": [
    {
      "ts": "2026-01-20T06:20:00Z",
      "type": "login_failed",
      "severity": "high",
      "src_ip": "9.9.9.9",
      "host": "api-1",
      "username": "bob",
      "msg": "invalid password"
    }
  ]
}
{
  "results": [
    {
      "index": 0,
      "status": "ok",
      "id": "uuid"
    }
  ]
}
```

**Alerts**

```bash
GET  /v1/alerts
POST /v1/alerts/{id}/ack
POST /v1/alerts/{id}/close
```

Alerts are created when the rule threshold is crossed and updated as new events arrive.

Alert Rule (MVP)

**Rule:**

Trigger an alert when 5 login_failed events from the same IP occur within 2 minutes

Alerts are deduplicated using a fingerprint:

```
login_failed_spike:<src_ip>
```

Only one open alert exists per fingerprint

Additional matching events update the existing alert

Closing an alert allows future alerts for the same fingerprint


**Database Design**
Events

Time-series optimized

Indexed on timestamp, type, and source IP

Uses PostgreSQL inet type for IP correctness

Alerts

Partial unique index:

UNIQUE (fingerprint) WHERE state = 'open'


Enables:

No duplicate open alerts

New alerts after closure

Uses PostgreSQL INSERT ... ON CONFLICT upsert pattern



**Start database**
```bash
docker compose up -d
```

**Run migrations**
```bash
migrate -database "$DATABASE_URL" -path migrations up
```

**Run server**
```bash
go run ./cmd/api
```

**Example Test Flow**
```bash
# Send events

curl -H 'X-API-Key: dev-key' -H 'Content-Type: application/json' \
  -d '{"events":[ ... ]}' \
  localhost:8080/v1/events
# View alerts
curl -H 'X-API-Key: dev-key' localhost:8080/v1/alerts
```

SecureLog was built to practice and demonstrate:

Backend API design

Concurrency with Go channels

Worker pool patterns

Asynchronous processing

Database modeling for real-world workloads

Alert deduplication and lifecycle management