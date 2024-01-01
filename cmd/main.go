package main

import (
	"chatApp/internals/client"
	"chatApp/internals/server"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Insufficient args to start the application: %v", os.Args)
	}

	env := os.Args[1]

	switch env {
	case "server":
		server.Serve()
	case "client":
		client.Serve()
	}
}
