package main

import (
	"fmt"
	"go3/env"
)

func Env() {
	env.LoadEnv()
	fmt.Println("Hello, World!")
	fmt.Println("Host: " + env.Host.Get())
	fmt.Println("Port: " + env.Port.Get())
	fmt.Println("API Host: " + env.APIHost.Get())
	fmt.Println("API Port: " + env.APIPort.Get())
	fmt.Println("Dev Mode: " + env.DevMode.Get())
}

func main() {
	Env()
}
