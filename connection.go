package wsevents

import (
	"net"
	"time"
	"code.google.com/p/go.net/websocket"
)

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
	ws      *websocket.Conn
	handler EventHandler
	onClose chan error
}

// Send sends a named event with a series of arguments.
// Any type supported by the encoding/json marshaller may be used.
func (c *Connection) Send(name string, args ...interface{}) {
	// TODO: custom timeout with a Config struct or something
	deadline := time.Now().Add(10 * time.Second)
	c.ws.SetWriteDeadline(deadline)

	err := websocket.JSON.Send(c.ws, jsonObj{"name": name, "args": args})
	if err != nil {
		if neterr, ok := err.(net.Error); ok && neterr.Temporary() {
			// TODO: return the error instead?
			c.handler.OnError(err)
		} else {
			c.handler.OnClose(err)
		}
	}
}

// Close closes the WebSocket connection.
func (c *Connection) Close() {
	c.onClose <- nil
}
