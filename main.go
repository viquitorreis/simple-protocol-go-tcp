package main

import (
	"log"
	"net"
)

func main() {
	logger("[INFO] Starting server on :8081")
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Printf("Error starting server: %v", err)
	}

	hub := newHub()
	go hub.run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
		}

		logger("[INFO] New client connected")

		client := newClient(conn, hub.commands, hub.registrations, hub.deregistrations)

		go client.read()
	}
}
