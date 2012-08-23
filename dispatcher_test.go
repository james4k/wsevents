package wsevents_test

import (
	"net/http/httptest"
	"testing"

	"github.com/james4k/wsevents"
	"code.google.com/p/go.net/websocket"
)

func TestSimple(t *testing.T) {
	wsevents.OnOpen(func(conn *wsevents.Conn) {
		conn.Data = false
	})

	wsevents.OnClose(func(conn *wsevents.Conn, err error) {
		if err != nil {
			t.Error(err)
		}

		received, ok := conn.Data.(bool)
		if !ok {
			t.Fatal("OnOpen wasn't called")
		} else if !received {
			t.Fatal("echo event wasn't fired")
		}
	})

	wsevents.OnEvent("echo", func(conn *wsevents.Conn, msg string) {
		if msg == "test" {
			conn.Data = true
			conn.Close()
		}
	})

	serv := httptest.NewServer(wsevents.Handler())
	defer serv.Close()

	origin := serv.URL
	url := "ws" + serv.URL[4:]
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Error(err)
	}

	websocket.Message.Send(ws, `{"name": "echo", "args": ["test"]}`)
}

func TestServerSideClose(t *testing.T) {
	wsevents.OnClose(func(conn *wsevents.Conn, err error) {
		if err != nil {
			t.Error(err)
		}
	})

	wsevents.OnEvent("echo", func(conn *wsevents.Conn, msg string) {
	})

	serv := httptest.NewServer(wsevents.Handler())
	defer serv.Close()
	defer serv.CloseClientConnections()

	origin := serv.URL
	url := "ws" + serv.URL[4:]
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Error(err)
	}

	websocket.Message.Send(ws, `{"name": "echo", "args": ["test"]}`)
}

