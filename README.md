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

## development

Can run application built from local code

```bash
docker-compose -f dev/docker-compose.yaml up --build
```
