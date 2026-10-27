default:
    @just --list

# install web deps if missing
install:
    cd web && [ -d node_modules ] || npm ci

# go server on :8080, accepting websockets from the web dev server
backend:
    cd backend && ALLOWED_ORIGINS=http://localhost:3000 go run ./cmd/go-poker

# next dev server on :3000, connecting to the go server
web: install
    cd web && NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws npm run dev

# backend and web together; ctrl-c stops both
start:
    #!/usr/bin/env bash
    trap 'kill 0' EXIT
    just backend & just web & wait

# production-like: static export served by the go server on :8080
serve: install
    cd web && npm run build
    cd backend && STATIC_DIR=../web/out go run ./cmd/go-poker

# format go code
fmt:
    cd backend && gofmt -w .

# generate the web's protocol types from the go code
types:
    cd backend && go tool tygo generate

# run go tests
test:
    cd backend && go test -race ./...

# browser tests against a static build on :8099: all, or the named scripts (just e2e host stats).
# See e2e/README.md
e2e *scripts: install
    #!/usr/bin/env bash
    set -euo pipefail
    ! curl -s localhost:8099 >/dev/null || { echo "port 8099 is already in use"; exit 1; }
    (cd e2e && [ -d node_modules ] || npm ci)
    (cd web && npm run build)
    bin="$(mktemp -d)/go-poker"
    (cd backend && go build -o "$bin" ./cmd/go-poker)
    # short timers, which the scripts read too
    export RECONNECT_GRACE=20s BOT_DELAY=100ms
    # every browser connects from one address; seated players' tables outlive a script by the reconnect grace
    PORT=8099 STATIC_DIR=web/out MAX_TABLES_PER_IP=200 MAX_CONNS_PER_IP=200 METRICS_ADDR= "$bin" >/dev/null 2>&1 &
    # restart.mjs runs its own server from the same binary
    export BIN="$bin"
    server=$!
    trap 'kill $server' EXIT
    for _ in $(seq 50); do curl -sf localhost:8099 >/dev/null && break; sleep 0.2; done
    cd e2e && node run.mjs {{ scripts }}

# same checks as ci
check: install
    cd backend && test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)
    cd backend && go vet ./...
    cd backend && ./types-current.sh
    cd backend && go test -race ./...
    cd web && npm run type-check
    cd web && npm run build
