package main

// Message represents a chat message payload exchanged over the websocket.
type Message struct {
	// smt is an alternate field name coming from the client for the sender's name.
	Smt        string `json:"smt,omitempty"`
	ClientName string `json:"client_name"`
	Text       string `json:"text"`
	Room       string `json:"room,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}
