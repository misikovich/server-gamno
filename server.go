package main

import (
	"fmt"
	"go3/env"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Hello, World!")
	godotenv.Load()
	fmt.Println(env.HOST.GetValue())
}
