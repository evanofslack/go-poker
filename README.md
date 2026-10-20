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

With nginx, forward the `Host` header (the server only accepts same-origin websockets) and the upgrade headers:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

Set `ALLOWED_ORIGINS` (comma separated) to allow websockets from other origins, e.g. a frontend dev server.

## development

Can run application built from local code

```bash
docker-compose -f dev/docker-compose.yaml up --build
```
