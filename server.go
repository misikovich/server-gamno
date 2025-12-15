package main

import (
	"fmt"
	"go3/env"
)

func main() {
	env.LoadEnv()
	fmt.Println("Hello, World!")
	fmt.Println("Host: " + env.Host.Get())
	fmt.Println("Port: " + env.Port.Get())
	fmt.Println("API Host: " + env.APIHost.Get())
	fmt.Println("API Port: " + env.APIPort.Get())
}
