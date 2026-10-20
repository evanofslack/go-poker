.PHONY: go next start

# go server on :8080, accepting websockets from next dev
go :
	cd backend && ALLOWED_ORIGINS=http://localhost:3000 STATIC_DIR=../web/out go run ./cmd/go-poker

# next dev server on :3000, connecting to the go server
next :
	cd web && NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws npm run dev

start :
	make go & make next
