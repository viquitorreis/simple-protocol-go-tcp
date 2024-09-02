package main

// this is not a go channel. It's just chat rooms
type channel struct {
	name string
	// clients is a map of all the clients in the channel
	// having the list of the clients available will allow us to easily broadcast messages to all the clients in the channel
	clients map[*client]bool
}

func newChannel(name string) *channel {
	return &channel{
		name:    name,
		clients: make(map[*client]bool),
	}
}

func (c *channel) broadcast(sender string, msg []byte) {
	message := append([]byte(sender), ": "...)
	message = append(message, msg...)
	message = append(message, '\n')

	for cl := range c.clients {
		cl.conn.Write(message)
	}
}
