# go-poker

Real time multiplayer poker application powered by go, react, and websockets

## demo

https://poker.evanslack.dev

## ui

<img width="800" alt="gopoker-screenshot" src="https://github.com/evanofslack/go-poker/assets/51209817/08c93fd3-0814-40e8-ab10-74d613ad996a">

## getting started

Can run from a prebuilt container: `evanofslack/go-poker:latest`

```yaml
services:
  go-poker:
    image: evanofslack/go-poker:latest
    container_name: go-poker
    restart: unless-stopped
    ports:
      - 8080:8080
```

## deploying

Tables, ledgers and hand history live in memory, so they're lost when the server restarts.

Serve it over HTTPS so the websocket runs on `wss://`. The simplest option is [Caddy](https://caddyserver.com), which gets a certificate and proxies websockets automatically:

```
poker.example.com {
    reverse_proxy localhost:8080
}
```

Caddy sets `X-Forwarded-For`, so also set `TRUST_PROXY=true` for the per-address limits to see real client addresses (the limits are listed in `.env.example`).

With nginx, forward the `Host` header (the server only accepts same-origin websockets), the client address and the upgrade headers, and set `TRUST_PROXY=true`:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Set `ALLOWED_ORIGINS` (comma separated) to allow websockets from other origins, e.g. a frontend dev server.

## metrics

Prometheus metrics are served at `/metrics` on a separate port, `METRICS_ADDR` (default `:2112`; set it empty to turn them off), never on the public game port. Don't expose it publicly: firewall it, or bind a private address like `10.0.0.5:2112`.

| metric | labels | |
|---|---|---|
| `poker_connections` | | open websockets |
| `poker_tables` | `state`: active, ended | open tables |
| `poker_players` | `kind`: human, bot | seated players |
| `poker_limit` | `limit`: conns, conns_per_ip, tables, tables_per_ip, messages | configured limits |
| `poker_limit_hits_total` | `limit` | connections, tables or messages refused by a limit |
| `poker_hands_total` | | hands dealt |
| `poker_tables_closed_total` | `reason`: empty, ended, idle, shutdown | tables closed |
| `poker_seat_grace_total` | `outcome`: reconnected, expired | dropped players who came back or lost their seat |
| `poker_slow_client_drops_total` | | clients dropped for falling behind |
| `poker_panics_total` | | panics recovered while handling a message |

Plus the standard `go_*` and `process_*` metrics.

## development

Run the go server and next dev server with hot reload (needs go, node and [just](https://github.com/casey/just)):

```bash
just start
```

Then open http://localhost:3000. Run `just check` for the same checks as CI, or `just` to list all recipes. Or build and run the production image from local code:

```bash
docker compose -f docker-compose-dev.yaml up --build
```
