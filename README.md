# Ship Cargo Service

Ship cargo space booking service with concurrent access to limited capacity.

## Problem

Several supply officers simultaneously compete for limited vessel capacity (weight + volume). Currently allocation is handled verbally — no tracking of remaining capacity, no priorities, conflicts are possible.

## Stack

- **Go** — API server
- **PostgreSQL** — data storage
- **Redis** — distributed locking
- **Docker Compose** — infrastructure

## Running

```bash
cp .env.example .env
docker compose up -d
make migrate-up
make run
```

## API

All endpoints except register, login and health check require `Authorization: Bearer <token>` header.

### Health
```
GET /healthz
```

### Auth
```
POST /register  — register a new user (supplier or manager)
POST /login     — get JWT token
```

### Vessels (manager only)
```
POST /vessels   — create a vessel (name, max_weight_kg, max_volume_m3)
GET  /vessels   — list all vessels
```

### Voyages
```
POST /voyages   — create a voyage (manager only)
GET  /voyages   — list all voyages with remaining capacity
```

### Bookings
```
POST /bookings  — book cargo space on a voyage
```

Booking supports dual capacity constraints (weight + volume), partial placement, priority levels (urgent/normal/low) and idempotency keys to prevent duplicate bookings.