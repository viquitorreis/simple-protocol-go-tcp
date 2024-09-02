package main

import (
	"fmt"
	"strings"
)

// hug is the central entity that the clients connect and register with
// it manages available channels ( chat rooms ), broadcasting messages to channels,
// relaying messages (private / direct messages) between clients
type hub struct {
	// channels is a map of all the channels ( chat rooms ) available in the hub
	// it have the name of the channels as the key and the *channel as value
	channels map[string]*channel
	// clients is a map of all the clients connected to the hub
	// it have the username of the clients as the key and the *client as value
	clients map[string]*client
	// commands is a channel to which the clients will send commands to the hub.
	// Hub will further validate and execute the commands
	commands chan command
	// channel of clients that when a client deregisters itself, it will send itself to this channel.
	// Hub will be informed and will remove the client from the clients map
	deregistrations chan *client
	// channel of clients that when a client registers itself, it will send itself to this channel.
	// hub will be informed and after doing the 'username' validation will add the client to the clients map
	registrations chan *client
}

func newHub() *hub {
	return &hub{
		channels:        make(map[string]*channel),
		clients:         make(map[string]*client),
		commands:        make(chan command),
		deregistrations: make(chan *client),
		registrations:   make(chan *client),
	}
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.registrations:
			h.register(client)
		case client := <-h.deregistrations:
			h.deregister(client)
		case cmd := <-h.commands:
			switch cmd.id {
			case JOIN:
				h.joinChannel(cmd.client, cmd.recipient)
			case LEAVE:
				h.leaveChannel(cmd.client, cmd.recipient)
			case MSG:
				h.message(cmd.client, cmd.recipient, cmd.body)
			case USRS:
				h.listUsers(cmd.sender)
			case CHNS:
				h.listChannels(cmd.client)
			default:
				// unknown command
			}
		}
	}
}

func (h *hub) register(c *client) {
	if c.username == "" || c.username == " " || c.username == "@" {
		c.err(fmt.Errorf("[INFO] username cannot be empty"))
		return
	}

	if _, exists := h.clients[c.username]; exists {
		c.err(fmt.Errorf("[INFO] username [%s] already exists", c.username))
		// send a message to the client to choose another username
		c.conn.Write([]byte("Username already exists. Please choose another username\n"))
		// we need to set the username to empty string so that the client can register again
		c.username = ""
	} else {

		h.clients[c.username] = c
		c.conn.Write([]byte("Registered successfully\n"))
		logger(fmt.Sprintf("[INFO] Registered user: %s", c.username))
	}
}

func (h *hub) deregister(c *client) {
	if _, exists := h.clients[c.username]; exists {
		// removing from hub's clients map
		delete(h.clients, c.username)

		// remove the client from all the channels of our clients map
		for _, channel := range h.channels {
			delete(channel.clients, c)
		}

		// send a message to the client that it has been deregistered
		body := fmt.Sprintf("%s: Deregistered successfully\n", getTimeWithMicroseconds())
		c.conn.Write([]byte(body))
		logger(fmt.Sprintf("[INFO] Deregistered user: %s", c.username))
	}
}

func (h *hub) joinChannel(client *client, chanName string) {
	if client.username == "" {
		client.conn.Write([]byte("Please register first\n"))
		return
	}

	// if the client exists in the hub
	if client, ok := h.clients[client.username]; ok {
		if channel, exists := h.channels[chanName]; exists {
			// if it exists, add the client to the channel
			channel.clients[client] = true
		} else {
			// if the channel does not exist, create a new channel and add the client to the channel
			h.channels[chanName] = newChannel(chanName)
			h.channels[chanName].clients[client] = true
		}

		// send a message to the client that it has joined the channel
		body := fmt.Sprintf("%s: Joined channel %s\n", getTimeWithMicroseconds(), chanName)
		client.conn.Write([]byte(body))
		logger(fmt.Sprintf("[INFO] User <%s> joined channel %s", client.username, chanName))
	}
}

func (h *hub) leaveChannel(client *client, chanName string) {
	if client.username == "" {
		client.conn.Write([]byte("Please register first\n"))
		return
	}

	if client, exists := h.clients[client.username]; exists {
		if channel, exists := h.channels[chanName]; exists {
			delete(channel.clients, client)
		}

		// send a message to the client that it has left the channel
		body := fmt.Sprintf("%s: Left channel %s\n", getTimeWithMicroseconds(), chanName)
		client.conn.Write([]byte(body))
		logger(fmt.Sprintf("[INFO] User <%s> left channel %s", client.username, chanName))
	}
}

// func (h *hub) listChannels(username string) {
func (h *hub) listChannels(client *client) {
	if client.username == "" {
		client.conn.Write([]byte("Please register first\n"))
		return
	}

	if client, ok := h.clients[client.username]; ok {
		var names []string

		if len(h.channels) == 0 {
			client.conn.Write([]byte("No channels available\n"))
		}

		for c := range h.channels {
			names = append(names, "#"+c+" ")
		}

		resp := strings.Join(names, ", ")

		client.conn.Write([]byte(resp + "\n"))
		logger(fmt.Sprintf("[INFO] Listing channels for user <%s>", client.username))
	}
}

func (h *hub) listUsers(username string) {
	if client, ok := h.clients[username]; ok {
		var names []string

		if len(h.clients) == 0 {
			client.conn.Write([]byte("No users available\n"))
		}

		for c := range h.clients {
			names = append(names, c)
		}

		resp := strings.Join(names, ", ")

		client.conn.Write([]byte(resp + "\n"))
		logger(fmt.Sprintf("[INFO] Listing users for user <%s>", username))
	}
}
