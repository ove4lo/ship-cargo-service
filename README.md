# Ship Cargo Service

Ship cargo space booking service

## Problem

Several supply officers are simultaneously vying for the vessel's limited capacity (weight and volume).
Currently, allocation is handled verbally—there is no tracking of remaining capacity or prioritization, and conflicts can arise.

## Start

```bash
cp .env.example .env
go run ./cmd/api
```

## Check

```bash
curl http://localhost:8081/healthz
```