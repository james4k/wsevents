package wsevents

import (
	//"errors"
	"fmt"
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
// Note that if handler has zero event handling methods, than it will panic.
func Handler(handler EventHandler) http.Handler {
	disp := &dispatcher{}
	disp.conns = make(map[*websocket.Conn]*Connection)
	disp.handlerType = reflect.Indirect(reflect.ValueOf(handler)).Type()
	disp.setupEventFuncs()

	wsHandler := func(ws *websocket.Conn) {
		var closing bool
		onReceive := make(chan jsonObj, 2)
		onSend := make(chan jsonObj, 2)
		onClose := make(chan error, 1)

		conn := &Connection{ws, reflect.New(disp.handlerType).Interface().(EventHandler), onReceive, onSend, onClose}
		disp.mu.Lock()
		disp.conns[ws] = conn
		disp.mu.Unlock()
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
			conn.handler.OnClose(closeErr)
			ws.Close()
			close(onReceive)
			close(onSend)
		}()

		go func() {
			for {
				select {
				case obj, ok := <-onReceive:
					if !ok {
						return
					}

					disp.fireEvent(conn, obj)
				}
			}
		}()

		go func() {
			for {
				deadline := time.Now().Add(2 * time.Minute)
				ws.SetReadDeadline(deadline)

				var obj jsonObj
				fmt.Println("read")
				err := websocket.JSON.Receive(ws, &obj)
				if err != nil {
					fmt.Printf("err: %v\n", err)
					if !closing {
						onClose <- err
					}
					return
				}
				fmt.Printf("%v\n", obj)
				if obj != nil {
					onReceive <- obj
				}
			}
		}()

		for {
			select {
			case obj := <-onSend:
				deadline := time.Now().Add(10 * time.Second)
				ws.SetWriteDeadline(deadline)

				err := websocket.JSON.Send(ws, obj)
				if err != nil {
					panic(err)
				}

			case err := <-onClose:
				closing = true
				if err != nil {
					panic(err)
				} else {
					return
				}
			}
		}
	}
	return websocket.Handler(wsHandler)
}

func (disp *dispatcher) setupEventFuncs() {
	disp.eventFns = make(map[string]reflect.Value)
	for i := 0; i < disp.handlerType.NumMethod(); i += 1 {
		method := disp.handlerType.Method(i)
		if method.PkgPath == "" && method.Name[:2] == "On" && method.Name != "OnError" && method.Name != "OnClose" {
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

	fn, ok := disp.eventFns[name]
	if !ok {
		conn.handler.OnError(ErrUnexpectedEvent)
		return
	}

	// TODO: cache this type? is it expensive to get the type?
	fntype := fn.Type()
	count := fntype.NumIn()
	if len(args) != count-1 {
		conn.handler.OnError(MakeArgsMismatchError(fntype, args))
		return
	}

	argvals := make([]reflect.Value, count)
	for i := 1; i < count; i += 1 {
		if !reflect.TypeOf(args[i-1]).AssignableTo(fntype.In(i)) {
			conn.handler.OnError(MakeArgsMismatchError(fntype, args))
			return
		}

		argvals[i] = reflect.ValueOf(args[i-1])
	}

	// first arg is the receiver (which is the EventHandler)
	argvals[0] = reflect.ValueOf(conn.handler)
	reflect.ValueOf(fn).Call(argvals)
}
