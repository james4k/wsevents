package wsevents

import "code.google.com/p/go.net/websocket"

type Conn struct {
	ws		*websocket.Conn
	onReceive	<-chan jsonObj
	onSend		chan<- jsonObj

	Data	interface{}
}

func (c *Conn) Send(name string, args ...interface{}) {
	c.onSend <- jsonObj{
		"name":	name,
		"args":	args,
	}
}

func (c *Conn) Close() {
	c.ws.Close()
}
