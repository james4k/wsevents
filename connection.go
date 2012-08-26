package wsevents

import "code.google.com/p/go.net/websocket"

type EventHandler interface {
	// OnOpen is called when we've accepted a new WebSocket connection
	// A new instance of the event handler was created just before now.
	OnOpen(*Connection)

	// OnError is called when we've encountered a recoverable error
	OnError(error)

	// OnClose is called when the connection is closed or we've encountered
	// an unrecoverable error.
	OnClose(error)
}

type Connection struct {
	ws        *websocket.Conn
	handler   EventHandler
	onReceive <-chan jsonObj
	onSend    chan<- jsonObj
	onClose   chan error
}

// Send sends a named event with a series of arguments.
// Any type supported by the encoding/json marshaller may be used.
func (c *Connection) Send(name string, args ...interface{}) {
	c.onSend <- jsonObj{
		"name": name,
		"args": args,
	}
}

// Close closes the WebSocket connection.
func (c *Connection) Close() {
	c.onClose <- nil
}
