module github.com/evanofslack/go-poker

// +heroku goVersion go1.18
go 1.18

// replace github.com/alexclewontin/riverboat => /Users/Evan/Documents/Github/riverboat

require (
	github.com/alexclewontin/riverboat/eval v0.2.2
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/joho/godotenv v1.4.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/chehsunliu/poker v0.1.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
)
