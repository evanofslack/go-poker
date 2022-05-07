.PHONY: start

go :
	go run .

next :
	cd client && npm run dev

start :
	make go & make next
