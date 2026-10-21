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

# run go tests
test:
    cd backend && go test -race ./...

# browser tests against a static build on :8099, see e2e/README.md
e2e: install
    #!/usr/bin/env bash
    set -euo pipefail
    ! curl -s localhost:8099 >/dev/null || { echo "port 8099 is already in use"; exit 1; }
    (cd e2e && [ -d node_modules ] || npm ci)
    (cd web && npm run build)
    bin="$(mktemp -d)/go-poker"
    (cd backend && go build -o "$bin" ./cmd/go-poker)
    PORT=8099 STATIC_DIR=web/out "$bin" 2>/dev/null &
    server=$!
    trap 'kill $server' EXIT
    for _ in $(seq 50); do curl -sf localhost:8099 >/dev/null && break; sleep 0.2; done
    failed=0
    for t in session betting showdown turn seat forms ux layout; do
        echo "== $t"
        (cd e2e && node "$t.mjs") || failed=1
    done
    exit $failed

# same checks as ci
check: install
    cd backend && test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)
    cd backend && go vet ./...
    cd backend && go test -race ./...
    cd web && npm run type-check
    cd web && npm run build
