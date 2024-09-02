package main

type ID int

type command struct {
	id ID
	// recipient is who/what is receiver of the command. Can be @user or a #channel
	recipient string
	// the sender command which is the @username of a user
	sender string
	body   []byte
	// client is the client that sent the command
	client *client
}

const (
	REG ID = iota
	JOIN
	LEAVE
	MSG
	CHNS
	USRS
)
