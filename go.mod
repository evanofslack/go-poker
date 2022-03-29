module github.com/evanofslack/go-poker

go 1.18

replace github.com/alexclewontin/riverboat => /Users/Evan/Documents/Github/riverboat

require (
	github.com/alexclewontin/riverboat v0.6.0
	github.com/alexclewontin/riverboat/eval v0.2.2
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.5.0
	github.com/joho/godotenv v1.4.0
)

require github.com/chehsunliu/poker v0.1.0 // indirect
