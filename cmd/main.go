package main

import (
	"chatApp/internals/client"
	"chatApp/internals/server"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Insufficient args to start the application: %v\nUsage: [server | client] IPAddress(defaults to 127.0.0.1)", os.Args)
	}

	defaultIp := "127.0.0.1"
	ipaddr := defaultIp

	if len(os.Args) == 3 {
		providedIp := os.Args[2]
		if !isValidIpStructure(providedIp) {
			log.Fatalf("Provided IP for running the server is not valid: %s", providedIp)
		} else {
			ipaddr = providedIp
		}
	}

	env := os.Args[1]

	switch env {
	case "server":

		server.Serve(ipaddr)
	case "client":
		client.Render(ipaddr)
	default:
		log.Fatalf("Unexpected argument: %s", env)
	}
}

func isValidIpStructure(ipAddr string) bool {
	// valid structure example: 192.168.1.23
	split := strings.Split(ipAddr, ".")

	// there should be four elements in the array after splitting. Eg:- 192, 168, 1, 23
	if len(split) != 4 {
		return false
	}

	for _, bytePart := range split {
		val, ok := strconv.Atoi(bytePart)

		if ok != nil {
			return false
		}

		if val > 255 {
			return false
		}
	}

	return true
}
