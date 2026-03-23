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
Response:
```
{
    "search_criteria": {
        "origin": "CGK",
        "destination": "DPS",
        "departure_date": "2025-12-15",
        "passengers": 1,
        "cabin_class": "economy"
    },
    "metadata": {
        "total_results": 12,
        "providers_queried": 4,
        "providers_succeeded": 4,
        "providers_failed": 0,
        "search_time_ms": 361,
        "cache_hit": false
    },
    "flights": [
        {
            "id": "QZ532_AirAsia",
            "provider": "AirAsia",
            "airline": {
                "name": "AirAsia",
                "code": "QZ"
            },
            "flight_number": "QZ532",
            "departure": {
                "airport": "CGK",
                "city": "Jakarta",
                "datetime": "2025-12-15T19:30:00+07:00",
                "timestamp": 1765801800
            },
            "arrival": {
                "airport": "DPS",
                "city": "Denpasar",
                "datetime": "2025-12-15T22:10:00+08:00",
                "timestamp": 1765807800
            },
            "duration": {
                "total_minutes": 100,
                "formatted": "1h 40m"
            },
            "stops": 0,
            "price": {
                "amount": 595000,
                "currency": "IDR",
                "formatted": "Rp 595.000"
            },
            "available_seats": 72,
            "cabin_class": "economy",
            "aircraft": null,
            "amenities": [],
            "baggage": {
                "carry_on": "Cabin baggage only",
                "checked": "Additional fee"
            },
            "score": 0.05699481865284974
        },
    ]
}
```

### Filtering & Sorting

#### Available Filters

Filters are provided as URL query parameters and applied by all providers. The following filters are supported:

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `priceMin` | number | Minimum flight price (in ticket currency), it will filter totalPrice if round trip/multi city | `priceMin=500000` |
| `priceMax` | number | Maximum flight price (in ticket currency), it will filter totalPrice if round trip/multi city | `priceMax=2000000` |
| `maxStops` | integer | Maximum number of stops allowed | `maxStops=1` |
| `durationMin` | integer | Minimum flight duration in minutes, it will filter totalDuration if round trip/multi city | `durationMin=60` |
| `durationMax` | integer | Maximum flight duration in minutes, it will filter totalDuration if round trip/multi city | `durationMax=300` |
| `airlines` | string (comma-separated) | Filter by airline codes (e.g., QZ for AirAsia) | `airlines=QZ,GA,BT` |
| `departureTimeMin` | string (HH:MM format) | Earliest departure time (24-hour format), not available on multi city | `departureTimeMin=06:00` |
| `departureTimeMax` | string (HH:MM format) | Latest departure time (24-hour format), not available on multi city | `departureTimeMax=20:00` |
| `arrivalTimeMin` | string (HH:MM format) | Earliest arrival time (24-hour format), not available on round trip/multi city | `arrivalTimeMin=08:00` |
| `arrivalTimeMax` | string (HH:MM format) | Latest arrival time (24-hour format), not available on round trip/multi city | `arrivalTimeMax=23:00` |

#### Sorting Options

Results can be sorted using URL query parameters:
`Multi city search only sort by score`
| Parameter | Valid Values | Default | Description |
|-----------|--------------|---------|-------------|
| `sortBy` | `price`, `duration`, `departure`(only available on one way trip), `arrival`(only available on one way trip), or omitted | `score` | Field to sort results by. When omitted or any other value, results are sorted by "best value" score |
| `sortOrder` | `asc`, `desc` | `asc` | Sort order: `asc` for ascending, `desc` for descending |

**Scoring**: When `sortBy` is not specified, results use a best-value score combining:
- 50% price score (normalized against price range)
- 30% duration score (normalized against duration range)
- 20% stops score (stops * 0.5)

### Multi-city search

`POST /search/multi-city`

Search for flights across multiple segments to build complex itineraries (e.g., Medan → Jakarta → Denpasar). Each segment in the request is treated as one leg of a continuous journey. The API automatically validates that each segment departs from the previous segment's destination, ensuring proper flight chaining.

**Request Parameters:**

| Field | Type | Description |
|-------|------|-------------|
| `segments` | array | Array of search segments representing each leg of the journey. Each segment requires `origin`, `destination`, and `departureDate`. Segment connections are validated: each next segment must depart from the previous segment's destination. |
| `passengers` | integer | Number of passengers (required, minimum 1) |
| `cabinClass` | string | Cabin class for all segments (e.g., "economy", "business") |

Example (multi-city itinerary):
```bash
curl -s -X POST "http://localhost:3000/multi-search" \
  -H "Content-Type: application/json" \
  -d '{
    "segments": [
        {
            "origin": "KNO",
            "destination": "CGK",
            "departureDate":  "2025-12-13"
        },
        {
            "origin": "CGK",
            "destination": "DPS",
            "departureDate":  "2025-12-15"
        },
        {
            "origin": "DPS",
            "destination": "CGK",
            "departureDate":  "2025-12-17"
        }
    ],
    "passengers": 1,
    "cabinClass": "economy"
}'
```

**Request Format Notes:**
- **segments**: Array of search legs. In this example: Medan (KNO) → Jakarta (SIN) → Denpasar (DPS)
- Each segment requires `origin`, `destination`, and `departureDate`
- The API validates segment connections: each segment must depart from the previous segment's destination
- All segments must use the same `cabinClass` and `passengers` count

**Response Format:**

The response contains complete multi-leg itineraries (`trips`) where each leg is properly connected:

**Response Fields:**
- **`trips`**: Array of multi-city itineraries
  - **`segments`**: Array of flight segments in order, each with the same flight details as single-leg results
  - **`total_price`**: Sum of all legs' prices
  - **`total_duration_minutes`**: Sum of all legs' flight durations
  - **`combined_score`**: Best-value score for the entire itinerary
- **`metadata`**: Search statistics including providers queried and response time

Response:
```json
{
    "metadata": {
        "total_results": 10,
        "providers_queried": 12,
        "providers_succeeded": 12,
        "providers_failed": 0,
        "search_time_ms": 327,
        "cache_hit": false
    },
    "trips": [
        {
            "segments": [
                {
                    "id": "QZ7350_AirAsia",
                    "provider": "AirAsia",
                    "airline": {
                        "name": "AirAsia",
                        "code": "QZ"
                    },
                    "flight_number": "QZ7350",
                    "departure": {
                        "airport": "KNO",
                        "city": "Medan",
                        "datetime": "2025-12-13T15:00:00+07:00",
                        "timestamp": 1765612800
                    },
                    "arrival": {
                        "airport": "CGK",
                        "city": "Jakarta",
                        "datetime": "2025-12-13T16:45:00+07:00",
                        "timestamp": 1765619100
                    },
                    "duration": {
                        "total_minutes": 87,
                        "formatted": "1h 27m"
                    },
                    "stops": 0,
                    "price": {
                        "amount": 1485000,
                        "currency": "IDR",
                        "formatted": "Rp 1.485.000"
                    },
                    "available_seats": 88,
                    "cabin_class": "economy",
                    "aircraft": null,
                    "amenities": [],
                    "baggage": {
                        "carry_on": "Cabin baggage only",
                        "checked": "Additional fee"
                    },
                    "score": 0
                },
                {
                    "id": "QZ532_AirAsia",
                    "provider": "AirAsia",
                    "airline": {
                        "name": "AirAsia",
                        "code": "QZ"
                    },
                    "flight_number": "QZ532",
                    "departure": {
                        "airport": "CGK",
                        "city": "Jakarta",
                        "datetime": "2025-12-15T19:30:00+07:00",
                        "timestamp": 1765801800
                    },
                    "arrival": {
                        "airport": "DPS",
                        "city": "Denpasar",
                        "datetime": "2025-12-15T22:10:00+08:00",
                        "timestamp": 1765807800
                    },
                    "duration": {
                        "total_minutes": 100,
                        "formatted": "1h 40m"
                    },
                    "stops": 0,
                    "price": {
                        "amount": 595000,
                        "currency": "IDR",
                        "formatted": "Rp 595.000"
                    },
                    "available_seats": 72,
                    "cabin_class": "economy",
                    "aircraft": null,
                    "amenities": [],
                    "baggage": {
                        "carry_on": "Cabin baggage only",
                        "checked": "Additional fee"
                    },
                    "score": 0.05699481865284974
                },
                {
                    "id": "QZ5325_AirAsia",
                    "provider": "AirAsia",
                    "airline": {
                        "name": "AirAsia",
                        "code": "QZ"
                    },
                    "flight_number": "QZ5325",
                    "departure": {
                        "airport": "DPS",
                        "city": "Denpasar",
                        "datetime": "2025-12-17T19:30:00+07:00",
                        "timestamp": 1765974600
                    },
                    "arrival": {
                        "airport": "CGK",
                        "city": "Jakarta",
                        "datetime": "2025-12-17T22:10:00+08:00",
                        "timestamp": 1765980600
                    },
                    "duration": {
                        "total_minutes": 100,
                        "formatted": "1h 40m"
                    },
                    "stops": 0,
                    "price": {
                        "amount": 595000,
                        "currency": "IDR",
                        "formatted": "Rp 595.000"
                    },
                    "available_seats": 72,
                    "cabin_class": "economy",
                    "aircraft": null,
                    "amenities": [],
                    "baggage": {
                        "carry_on": "Cabin baggage only",
                        "checked": "Additional fee"
                    },
                    "score": 0
                }
            ],
            "total_price": 2675000,
            "total_duration_minutes": 287,
            "combined_score": 0.05699481865284974
        }
    ]
}
```

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

