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

## development

Run the go server and next dev server with hot reload (needs go, node and [just](https://github.com/casey/just)):

```bash
just start
```

Then open http://localhost:3000. Run `just check` for the same checks as CI, or `just` to list all recipes. Or build and run the production image from local code:

```bash
docker compose -f docker-compose-dev.yaml up --build
```
