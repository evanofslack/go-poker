package main

import (
	"fmt"

	"github.com/evanofslack/go-poker/server"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	server.InitServer()
}
