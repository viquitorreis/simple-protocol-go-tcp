package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/google/uuid"
)

var (
	DELIMITER = []byte(`\r\n`)
)

// wrapper around the TCP connection
type client struct {
	// the tcp connection itself
	conn net.Conn
	// a send-only channel to send commands from the client to the server ( hub )
	outbound chan<- command
	// a send-only channel to which the client will send itself when it wants to register with the hub
	register chan<- *client
	// a send-only channel to which the client will send itself when it wants to deregister with the hub
	deregister chan<- *client
	// the username of the user / client connected behing the TCP connection
	username string
	// the id of the client
	id string
}

func newClient(conn net.Conn, outbound chan<- command, register chan<- *client, deregister chan<- *client) *client {
	return &client{
		conn:       conn,
		outbound:   outbound,
		register:   register,
		deregister: deregister,
		id:         uuid.New().String(),
	}
}

func (c *client) read() error {
	// read messages from the client and send them to the hub
	for {
		msg, err := bufio.NewReader(c.conn).ReadBytes('\n')
		if err == io.EOF {
			// connection closed. So we must deregister the client
			c.deregister <- c
			logger("[INFO] Client disconnected")
			return nil
		}

		if err != nil {
			return err
		}

		c.handle(msg)
	}
}

// handle the message received from the client, and route based on the command
func (c *client) handle(msg []byte) {
	cmd := bytes.ToUpper(bytes.TrimSpace(bytes.Split(msg, []byte(" "))[0]))
	args := bytes.TrimSpace(bytes.TrimPrefix(msg, cmd))

	switch string(cmd) {
	case "REG":
		if err := c.reg(args); err != nil {
			c.err(err)
		}

	case "JOIN":
		if err := c.join(args); err != nil {
			c.err(err)
		}

	case "LEAVE":
		if err := c.leave(args); err != nil {
			c.err(err)
		}

	case "MSG":
		if err := c.msg(args); err != nil {
			c.err(err)
		}

	case "CHNS":
		c.chns()
	case "USRS":
		c.usrs()
	default:
		c.err(fmt.Errorf("unknown command: %s", cmd))
	}
}

func (c *client) err(err error) {
	c.conn.Write([]byte("ERR " + err.Error() + "\n"))
}

func (c *client) reg(args []byte) error {
	u := bytes.TrimSpace(args)

	if len(u) == 0 {
		return fmt.Errorf("username cannot be empty")
	}

	if u[0] != '@' {
		return fmt.Errorf("invalid username: %s. Must start with @", u)
	}

	c.username = string(u)
	c.register <- c

	return nil
}

func (c *client) msg(args []byte) error {
	args = bytes.TrimSpace(args)
	if len(args) == 0 || (args[0] != '#' && args[0] != '@') {
		return fmt.Errorf("invalid recipient: %s.\n Must be a channel('#name') or user ('@user')", args)
	}

	recipient := bytes.Split(args, []byte(" "))[0]
	if len(recipient) == 0 {
		return fmt.Errorf("recipient must have a name")
	}

	// remove the recipient from the args so we get the message
	args = bytes.TrimSpace(bytes.TrimPrefix(args, recipient))
	l := bytes.Split(args, DELIMITER)[0]
	// here we are converting the length of the message to an integer
	length, err := strconv.Atoi(string(l))
	if err != nil {
		return fmt.Errorf("body length must be present | Invalid message length: %s", l)
	}

	if length == 0 {
		return fmt.Errorf("body length must be at least 1")
	}

	// the padding will be the length of the message + the length of the delimiter
	// we need padding to know where the length part of the message ends
	padding := len(l) + len(DELIMITER)
	// the body of the message will be the padding to the padding + the length of the message
	body := []byte(getTimeWithMicroseconds() + " " + c.username + ": " + string(args[padding:padding+length]))

	logger(fmt.Sprintf("[INFO] %s is sending message to %s", c.username, recipient))
	c.outbound <- command{
		recipient: string(recipient),
		sender:    c.username,
		body:      body,
		id:        MSG,
		client:    c,
	}

	return nil
}

func (h *hub) message(client *client, recipient string, msg []byte) {
	// check if the sender exists
	if client.username == "" {
		client.err(fmt.Errorf("user not registered"))
		client.conn.Write([]byte("User not registered. Please register first\n"))
		return
	}

	if sender, ok := h.clients[client.username]; ok {
		switch recipient[0] {
		// if the recipient is a channel
		case '#':
			// we then check if the channel exists
			if channel, exists := h.channels[recipient]; exists {
				// if the sender is in the channel, we broadcast the message to all the clients in the channel
				if _, exists := channel.clients[sender]; exists {
					channel.broadcast(sender.username, msg)
				}
			}
		case '@':
			if user, ok := h.clients[recipient]; ok {
				user.conn.Write(append(msg, '\n'))
			}
		}
	}
}

func (c *client) join(args []byte) error {
	channelID := bytes.TrimSpace(args)
	if channelID[0] != '#' {
		return fmt.Errorf("invalid channel name: %s. Must start with #", channelID)
	}

	// send the join command to the hub
	logger(fmt.Sprintf("[INFO] User <%s> is joining channel %s", c.username, channelID))
	c.outbound <- command{
		recipient: string(channelID),
		sender:    c.username,
		id:        JOIN,
		client:    c,
	}

	return nil
}

func (c *client) leave(args []byte) error {
	channelID := bytes.TrimSpace(args)
	if channelID[0] != '#' {
		return fmt.Errorf("invalid channel name: %s. Must start with #", channelID)
	}

	// send the leave command to the hub
	logger(fmt.Sprintf("[INFO] Leaving channel %s", channelID))
	c.outbound <- command{
		recipient: string(channelID),
		sender:    c.username,
		id:        LEAVE,
		client:    c,
	}

	return nil
}

func (c *client) chns() {
	logger(fmt.Sprintf("[INFO] Listing channels for user <%s>", c.username))
	c.outbound <- command{
		sender: c.username,
		id:     CHNS,
		client: c,
	}
}

func (c *client) usrs() {
	logger(fmt.Sprintf("[INFO] Listing users for user <%s>", c.username))
	c.outbound <- command{
		sender: c.username,
		id:     USRS,
		client: c,
	}
}
