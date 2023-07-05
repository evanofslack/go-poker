.PHONY: start

go :
	cd backend && go run .

redis :
	docker compose up -d redis

next :
	cd web && npm run dev

start :
	make go & make next
