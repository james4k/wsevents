package wsevents

import (
	"code.google.com/p/go.net/websocket"
	"net"
	"time"
)

// EventHandler is what a struct must implement to receive events. All of the
// struct's methods that have the "On" prefix, except those that are a member of
// EventHandler, may receive events of that name. For example, for a named event
// "msg" that passes the arguments (int, string), it may be implemented as:
//
//	func (e *ExampleHandler) OnMsg(id int, text string) {
//		...
//	}
//
type EventHandler interface {
	// OnOpen is called when a new WebSocket connection is accepted.  A new
	// instance of the event handler was created just before now.
	OnOpen(*Connection)

	// OnError is called when a recoverable error was encountered.
	OnError(error)

	// OnClose is called when the connection is closed or an unrecoverable error
	// was encountered.
	OnClose(error)
}

// Connection holds the underlying WebSocket connection. It is passed through
// when OnOpen is called, and is what is used to Send events to the client, as
// well as Close the connection. 
//
// To make things simple, add the Connection as an anonymous field for easy
// access to Send and Close.
//
//	type ExampleHandler struct {
//		*wsevents.Connection
//		...
//	}
//	
//	func (e *ExampleHandler) OnOpen (conn *wsevents.Connection) {
//		e.Connection = conn
//		e.Send("msg", arg1, arg2, etc)
//	}
//
type Connection struct {
	ws      *websocket.Conn
	handler EventHandler
	closing bool
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
