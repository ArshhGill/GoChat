package client

import (
	"net"
	"os"
    "fmt"
)

const (
	HOST = "127.0.0.1"
	PORT = "4000"
	TYPE = "tcp"
)

func Serve() {
	tcpServer, err := net.ResolveTCPAddr(TYPE, HOST+":"+PORT)
	if err != nil {
		fmt.Printf("ResolveTCPAddr failed: %s", err)
		os.Exit(1)
	}

	conn, err := net.DialTCP(TYPE, nil, tcpServer)
	if err != nil {
		fmt.Printf("Dial failed: %s", err)
		os.Exit(1)
	}

    defer conn.Close()

	_, err = conn.Write([]byte("This is a message\n"))
	if err != nil {
		fmt.Printf("Write data failed: %s", err)
		os.Exit(1)
	}

	for {
		// buffer to get data
		received := make([]byte, 1024)
		_, err = conn.Read(received)
		if err != nil {
			fmt.Printf("Read data failed: %s", err)
			os.Exit(1)
		}

		fmt.Printf("Received message: %s", string(received))
	}
}
