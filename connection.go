package wsevents

import "code.google.com/p/go.net/websocket"

// Conn holds the Websocket connection with a user-data field
type Conn struct {
	ws        *websocket.Conn
	onReceive <-chan jsonObj
	onSend    chan<- jsonObj
	onClose   chan error

	Data interface{}
}

// Send sends a named event with a number of arguments.
// Any type supported by the encoding/json marshaller may
// be used.
func (c *Conn) Send(name string, args ...interface{}) {
	c.onSend <- jsonObj{
		"name": name,
		"args": args,
	}
}

// Close closes the Websocket connection.
func (c *Conn) Close() {
	c.onClose <- nil
}
