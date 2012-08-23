package wsevents

import (
	//"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"
)

type jsonObj map[string]interface{}

var DefaultDispatcher = &Dispatcher{}

type Dispatcher struct {
	conns    map[*websocket.Conn]*Conn
	openFn   func(*Conn)
	closeFn  func(*Conn, error)
	eventFns map[string]interface{}
	mu       sync.Mutex
}

func OnOpen(fn func(*Conn)) {
	DefaultDispatcher.OnOpen(fn)
}

func OnClose(fn func(*Conn, error)) {
	DefaultDispatcher.OnClose(fn)
}

func OnEvent(name string, fn interface{}) {
	DefaultDispatcher.OnEvent(name, fn)
}

func Handler() http.Handler {
	return DefaultDispatcher.Handler()
}

func (disp *Dispatcher) OnOpen(fn func(*Conn)) {
	disp.mu.Lock()
	defer disp.mu.Unlock()
	disp.openFn = fn
}

func (disp *Dispatcher) OnClose(fn func(*Conn, error)) {
	disp.mu.Lock()
	defer disp.mu.Unlock()
	disp.closeFn = fn
}

func (disp *Dispatcher) OnEvent(name string, fn interface{}) {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func || t.NumIn() < 1 {
		fmt.Printf("wsevents: '%s' not a valid function", t.Name())
		return
	}
	connType := reflect.TypeOf(&Conn{})
	if t.In(0) != connType {
		fmt.Printf("wsevents: type mismatch: %v vs %v\n", t.In(0), connType)
		return
	}

	disp.mu.Lock()
	defer disp.mu.Unlock()
	if disp.eventFns == nil {
		disp.eventFns = make(map[string]interface{})
	}
	disp.eventFns[name] = fn
}

func (disp *Dispatcher) Handler() http.Handler {
	disp.mu.Lock()
	disp.conns = make(map[*websocket.Conn]*Conn)
	disp.mu.Unlock()

	handler := func(ws *websocket.Conn) {
		var closing bool
		onReceive := make(chan jsonObj, 2)
		onSend := make(chan jsonObj, 2)
		onClose := make(chan error, 1)

		conn := &Conn{ws, onReceive, onSend, onClose, nil}
		disp.mu.Lock()
		disp.conns[ws] = conn
		disp.openFn(conn)
		disp.mu.Unlock()

		defer func() {
			var closeErr error
			r := recover()
			if err, ok := r.(error); ok {
				closeErr = err
			}

			disp.mu.Lock()
			delete(disp.conns, ws)
			disp.closeFn(conn, closeErr)
			disp.mu.Unlock()
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
				err := websocket.JSON.Receive(ws, &obj)
				if err != nil {
					if !closing {
						onClose <- err
					}
					return
				}
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
	return websocket.Handler(handler)
}

func (disp *Dispatcher) fireEvent(conn *Conn, obj jsonObj) {
	name, ok := obj["name"].(string)
	if !ok {
		return
	}
	args, ok := obj["args"].([]interface{})
	if !ok {
		return
	}

	disp.mu.Lock()
	fniface, ok := disp.eventFns[name]
	disp.mu.Unlock()
	if !ok {
		fmt.Println("missing func", name)
		return
	}

	fntype := reflect.TypeOf(fniface)
	count := fntype.NumIn()
	if len(args) != count-1 {
		fmt.Printf("arg count mismatch for %s\n", name)
		return
	}
	argvals := make([]reflect.Value, count)
	for i := 1; i < count; i += 1 {
		if fntype.In(i) != reflect.TypeOf(args[i-1]) {
			fmt.Printf("type mismatch: %v vs %v\n",
				fntype.In(i),
				reflect.TypeOf(args[i-1]))
			return
		}

		argvals[i] = reflect.ValueOf(args[i-1])
	}
	argvals[0] = reflect.ValueOf(conn)
	reflect.ValueOf(fniface).Call(argvals)
}
