# go-poker

Real time multiplayer poker application powered by go, react, websockets ans redis pubsub

## demo

https://poker.evanslack.dev

## ui

<img width="800" alt="gopoker-screenshot" src="https://github.com/evanofslack/go-poker/assets/51209817/08c93fd3-0814-40e8-ab10-74d613ad996a">

## getting started

Can run from a prebuilt container: `evanofslack/go-poker:latest`

First copy `.env.example` to `.env` and change values if needed

```yaml
services:
  go-poker:
    image: evanofslack/go-poker:latest
    container_name: go-poker
    restart: unless-stopped
    ports:
      - 8080:8080
    environment:
      REDIS_URL: ${REDIS_URL}
    depends_on:
      - redis

  redis:
    container_name: go-poker-redis
    image: redis:latest
    restart: unless-stopped
    entrypoint: redis-server --appendonly yes
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD}
    volumes:
      - redis:/data
      - /etc/timezone:/etc/timezone:ro
      - /etc/localtime:/etc/localtime:ro

volumes:
  redis:
```

## development

Can run application built from local code

```bash
docker-compose -f dev/docker-compose.yaml up --build
```
