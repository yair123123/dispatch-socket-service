# dispatch-socket-service

`dispatch-socket-service` is a standalone Go microservice that manages realtime driver sockets and Redis-based dispatch coordination.

## What this service does
- Authenticates driver WebSocket sessions with JWT.
- Tracks driver connection state (online/offline), availability, and last seen.
- Stores latest driver location + state in Redis (fast path only).
- Receives offer batches from core-service and pushes `ride_offer_created` to connected drivers.
- Handles immediate accept/reject from drivers.
- Uses Redis Lua atomic winner-claim to ensure one winner.
- Sends immediate winner/loser events over WebSocket.
- Syncs winner assignment back to core-service through internal HTTP.
- Retries failed core sync jobs from a Redis-backed queue.

## What this service does NOT do
- Does not decide dispatch radius, rounds, or candidate selection.
- Does not create rides.
- Does not persist location stream into PostgreSQL.
- Does not own pricing/identity/telephony/geocoding/business rules.

## Architecture
- **Fast path:** WebSocket + Redis.
- **Durable truth:** core-service + PostgreSQL.
- **Consistency flow:** atomic winner in Redis first, then async sync to core, with retry queue.

## WebSocket protocol
### Connect
`GET /ws/drivers/connect?token=...`

### Client -> server
- `set_availability`
- `location_update`
- `accept_ride`
- `reject_ride`

### Server -> client
- `connected`
- `ride_offer_created`
- `ride_assigned`
- `ride_offer_canceled`
- `ride_accept_rejected`

## Internal HTTP APIs
- `GET /health`
- `POST /internal/dispatch/offer`
- `POST /internal/dispatch/cancel`
- `POST /internal/dispatch/start-round`

## Redis keys overview
- `drivers:locations` (GEO)
- `driver:state:{driver_id}` (HASH)
- `ride:{ride_id}:dispatch` (HASH)
- `ride:{ride_id}:offer:{driver_id}` (HASH)
- `ride:{ride_id}:winner` (STRING)
- `ride:{ride_id}:round:{round}:drivers` (SET)
- `core:sync:retry:queue` (LIST)

## Immediate accept flow
1. Driver sends `accept_ride`.
2. Service runs Lua script with checks:
   - ride exists + open
   - round matches
   - pending offer exists
   - offer not expired
   - winner not already set
3. On success:
   - winner key set
   - offer marked accepted
   - ride dispatch marked assigned
   - `ride_assigned` sent to winner
   - `ride_offer_canceled` sent to others
4. Sync request to core-service. If failed, queue retry in Redis.

## Configuration
- `APP_PORT`
- `LOG_LEVEL`
- `JWT_SECRET` or `JWT_PUBLIC_KEY`
- `REDIS_URL`
- `CORE_SERVICE_BASE_URL`
- `INTERNAL_SERVICE_SECRET`
- `CORE_SERVICE_TIMEOUT_SECONDS`
- `CORE_CALLBACK_MAX_RETRIES`
- `CORE_CALLBACK_BACKOFF_SECONDS`
- `WS_PING_INTERVAL_SECONDS`
- `WS_READ_TIMEOUT_SECONDS`
- `WS_WRITE_TIMEOUT_SECONDS`
- `DRIVER_STATE_TTL_SECONDS`
- `DRIVER_LOCATION_TTL_SECONDS`
- `REDIS_ACCEPT_LOCK_TTL_SECONDS`
- `CORE_SYNC_RETRY_INTERVAL_SECONDS`
- `CORE_SYNC_MAX_RETRIES`

## Run locally
```bash
go mod tidy
go test ./...
go run ./cmd/server
```

## Testing
Includes tests for:
- connection manager
- availability + location writes to Redis
- offer delivery connected vs non-connected
- atomic accept success and rejection cases
- retry queue behavior
- websocket message routing
