.PHONY: start

go :
	go run .

redis :
	docker compose up -d redis

next :
	cd client && npm run dev

start :
	make go & make next
