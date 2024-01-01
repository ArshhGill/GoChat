package server

import (
	"fmt"
	"log"
	"net"
)

const (
	SERVER_IP   = "127.0.0.1"
	SERVER_PORT = 4000
)

type EventType = int

const (
	NEW_MESSAGE = iota
	CLIENT_DISCONNECTED
)

type ClientEvent struct {
	client    *Client
	text      string
	eventType EventType
}

type Client struct {
	conn net.Conn
	ip   string
}

type ClientHub struct {
	name    string
	clients []*Client
}

func removeClientFromHub(hub *ClientHub, client *Client) {
	for index, curClient := range hub.clients {
		if curClient.ip == client.ip {
			if len(hub.clients) == 1 {
				hub.clients = []*Client{}
				return
			}

			hub.clients[index] = hub.clients[len(hub.clients)-1]
			hub.clients = hub.clients[:len(hub.clients)-1]
			return
		}
	}

	log.Printf("Could not deleted the client from the hub clients: %s", client.ip)
}

func (hub *ClientHub) serve(eventChan chan ClientEvent) {
	for {
		event := <-eventChan

		switch event.eventType {
		case NEW_MESSAGE:
			log.Printf("message received from client: %s: %s", event.client.ip, event.text)
			for _, client := range hub.clients {
				if client.ip != event.client.ip {
					client.conn.Write([]byte(event.text))
				}
			}
		case CLIENT_DISCONNECTED:
			event.client.conn.Close()
			log.Printf("Client with ip: %s, disconnected", event.client.ip)
			removeClientFromHub(hub, event.client)
		}
	}
}

func handleClient(client *Client, eventChan chan ClientEvent) {
	buffer := make([]byte, 64)
	for {
		n, err := client.conn.Read(buffer)
		if err != nil {
			client.conn.Close()
			eventChan <- ClientEvent{
				eventType: CLIENT_DISCONNECTED,
				client:    client,
			}
			return
		}

		text := string(buffer[0:n])

		event := ClientEvent{
			text:      text,
			eventType: NEW_MESSAGE,
			client:    client,
		}

		eventChan <- event
	}
}

func Serve() {
	connListener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", SERVER_IP, SERVER_PORT))
	if err != nil {
		log.Fatalf("Could not start the server: %s", err)
	}

	log.Printf("Listening to connections on port %d", SERVER_PORT)

	eventChan := make(chan ClientEvent)

	hub := &ClientHub{
		name:    "Main",
		clients: []*Client{},
	}

	go hub.serve(eventChan)

	for {
		conn, err := connListener.Accept()
		if err != nil {
			log.Fatalf("Could not accept the connection to the server: %s", err)
		}

		log.Printf("Successfully connected to client: %s", conn.RemoteAddr().String())
		client := &Client{
			conn: conn,
			ip:   conn.RemoteAddr().String(),
		}

		hub.clients = append(hub.clients, client)

		go handleClient(client, eventChan)
	}
}
