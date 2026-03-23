# flight-search-service

Simple flight search API that aggregates mock flight data from multiple providers.

## Requirements

- Go 1.25+
- Redis (optional; the service will still run if Redis connection fails, but caching will be disabled)

## Setup

1. (Optional) Create a `.env` file in the repo root:
   - `PORT` (default: `3000`)
   - `REDIS_ADDRESS` (default: `localhost:6379`)
   - `REDIS_POOL_SIZE` (default: `10`)

2. Run the service:

```bash
go run ./cmd/api
```

## Endpoints

### Health

`GET /healthz`

### One-way / Round-trip search

`POST /search`

Example:
```bash
curl -s -X POST "http://localhost:3000/search?sortBy=price&sortOrder=asc" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departureDate": "2025-12-15",
    "returnDate": null,
    "passengers": 1,
    "cabinClass": "economy"
  }'
```

Optional filters are provided as URL query params (applied by all providers), including:
`priceMin`, `priceMax`, `maxStops`, `durationMin`, `durationMax`, `airlines`, `departureTimeMin/Max`, `arrivalTimeMin/Max`.

## Mock Providers, Delays, Retries, and Rate Limiting

This service uses mock provider responses (JSON files under `internal/provider/*/mock-response.json`) and simulates real-world behavior:

- Provider latency: each provider calls `internal/helper.SimulateDelay(...)` with a configured min/max delay (see `cmd/api/main.go` where providers are constructed).
- Rate limiting: providers are wrapped by `internal/service/limiter.NewRatedProvider(...)` to cap how frequently a provider can be queried.
- Retry with exponential backoff: providers that fail will retry up to a max attempt count using a growing backoff delay.

## Scoring (Best Value)

Results are ranked using a "best value" score (price + convenience) to emulate a typical travel search experience.

- For one-way flights, the score uses:
  - 50% price score
  - 30% duration score (total minutes)
  - 20% stops score (stops * 0.5)
- For round-trip itineraries, the score uses:
  - 50% total trip price score
  - 30% total trip duration score
  - 20% total stops score (outbound stops + inbound stops)

When `sortBy` is `price`, `duration`, `departure`, or `arrival`, the service sorts by that field. Otherwise, it falls back to the computed best-value score.

