package wsevents

import (
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"
)

type jsonObj map[string]interface{}

type dispatcher struct {
	conns       map[*websocket.Conn]*Connection
	handlerType reflect.Type
	eventFns    map[string]reflect.Value
	mu          sync.Mutex
}

// Handler returns an http.Handler which sets up our event handler.
//
// dummy expects a pointer to an empty instance of a struct that implements
// EventHandler. It is used to get the type information of the struct.
//
// onNew is an optional function that is called right after the struct is
// created, and before OnOpen.
//
// If handler has zero event handling methods, than it will panic.
func Handler(dummy EventHandler, onNew func(EventHandler)) http.Handler {
	disp := &dispatcher{}
	disp.conns = make(map[*websocket.Conn]*Connection)
	disp.handlerType = reflect.ValueOf(dummy).Type()
	disp.setupEventFuncs()

	wsHandler := func(ws *websocket.Conn) {
		conn := &Connection{
			ws: ws,
			handler: reflect.New(disp.handlerType.Elem()).Interface().(EventHandler),
			closing: false,
			onClose: make(chan error),
		}
		disp.mu.Lock()
		disp.conns[ws] = conn
		disp.mu.Unlock()

		if onNew != nil {
			onNew(conn.handler)
		}
		conn.handler.OnOpen(conn)

		defer func() {
			var closeErr error
			r := recover()
			if err, ok := r.(error); ok {
				closeErr = err
			}

			disp.mu.Lock()
			delete(disp.conns, ws)
			disp.mu.Unlock()

			// FIXME: Need to avoid the "use of closed network connection" error
			// Or does this only occur in local tests?
			conn.handler.OnClose(closeErr)
		}()

		// TODO: See if a separate goroutine for firing events would be more performant..?
		// Probably not, since that's an extra goroutine per connection. But, it would
		// allow some buffering of events.

		go disp.readUntilClose(conn)
		disp.waitForClose(conn)
	}
	return websocket.Handler(wsHandler)
}

func (disp *dispatcher) readUntilClose(conn *Connection) {
	for {
		deadline := time.Now().Add(2 * time.Minute)
		conn.ws.SetReadDeadline(deadline)

		var obj jsonObj
		err := websocket.JSON.Receive(conn.ws, &obj)
		if err != nil {
			if !conn.closing {
				conn.onClose <- err
			}
			return
		}

		if obj != nil {
			disp.fireEvent(conn, obj)
		}
	}
}

func (disp *dispatcher) waitForClose(conn *Connection) {
	for {
		select {
		case err := <-conn.onClose:
			// FIXME: may need to access/set conn.closing atomically
			conn.closing = true
			if err != nil {
				panic(err)
			} else {
				return
			}
		}
	}
}

func methodIsValidEvent(m *reflect.Method) bool {
	if m.PkgPath != "" || m.Name[:2] != "On" {
		return false
	}

	switch m.Name {
	case "OnOpen", "OnError", "OnClose":
		return false
	}

	return true
}

func (disp *dispatcher) setupEventFuncs() {
	if disp.handlerType.Kind() != reflect.Ptr || disp.handlerType.Elem().Kind() != reflect.Struct {
		panic("wsevents: expected handler to be a pointer to a struct")
	}

	disp.eventFns = make(map[string]reflect.Value)
	count := disp.handlerType.NumMethod()
	for i := 0; i < count; i += 1 {
		method := disp.handlerType.Method(i)
		if methodIsValidEvent(&method) {
			name := strings.ToLower(method.Name[2:])
			disp.eventFns[name] = method.Func
		}
	}
	if len(disp.eventFns) == 0 {
		panic("wsevents: no event methods found")
	}
}

func (disp *dispatcher) fireEvent(conn *Connection, obj jsonObj) {
	name, ok := obj["name"].(string)
	if !ok {
		conn.handler.OnError(ErrMissingEventName)
		return
	}
	args, ok := obj["args"].([]interface{})
	if !ok {
		conn.handler.OnError(ErrMissingEventArgs)
		return
	}

	name = strings.ToLower(name)
	fn, ok := disp.eventFns[name]
	if !ok {
		conn.handler.OnError(ErrUnexpectedEvent)
		return
	}

	// TODO: cache this type? is it expensive to get the type?
	fntype := fn.Type()
	count := fntype.NumIn()
	if len(args) != count-1 {
		conn.handler.OnError(makeArgsMismatchError(fntype, args))
		return
	}

	argvals := make([]reflect.Value, count)
	for i := 1; i < count; i += 1 {
		if !reflect.TypeOf(args[i-1]).AssignableTo(fntype.In(i)) {
			conn.handler.OnError(makeArgsMismatchError(fntype, args))
			return
		}

		argvals[i] = reflect.ValueOf(args[i-1])
	}

	// first arg is the receiver (which is the EventHandler)
	argvals[0] = reflect.ValueOf(conn.handler)
	fn.Call(argvals)
}
