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

# same checks as ci
check: install
    cd backend && test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)
    cd backend && go vet ./...
    cd backend && go test -race ./...
    cd web && npm run type-check
    cd web && npm run build
