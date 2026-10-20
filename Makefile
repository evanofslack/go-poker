.PHONY: start

go :
	cd backend && go run .

next :
	cd web && npm run dev

start :
	make go & make next
