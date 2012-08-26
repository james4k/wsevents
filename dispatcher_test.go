package wsevents_test

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/james4k/wsevents"
)

type SimpleHandler struct {
	Conn *wsevents.Connection
	MsgChan chan string
}

func (s *SimpleHandler) OnOpen(conn *wsevents.Connection) {
	s.Conn = conn
}

func (s *SimpleHandler) OnError(err error) {
	fmt.Println(err)
}

func (s *SimpleHandler) OnClose(err error) {
	fmt.Println(err)
}

func (s *SimpleHandler) OnTestMsg(msg string) {
	fmt.Println("before msg send")
	s.MsgChan <- msg
	fmt.Println("after msg send")
}

func TestSimple(t *testing.T) {
	var simpleHandler *SimpleHandler
	msgChan := make(chan string, 1)
	onNew := func(handler wsevents.EventHandler) {
		simpleHandler = handler.(*SimpleHandler)
		simpleHandler.MsgChan = msgChan
	}

	serv := httptest.NewServer(wsevents.Handler(&SimpleHandler{}, onNew))
	defer serv.Close()

	origin := serv.URL
	url := "ws" + serv.URL[4:]
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Error(err)
	}

	websocket.Message.Send(ws, `{"name": "testmsg", "args": ["test 123![]{}@"]}`)

	fmt.Println("selecting")
	select {
	case msg := <-msgChan:
		fmt.Println("received msg")
		if msg != "test 123![]{}@" {
			t.Fatal("message did not match!")
		}
		if simpleHandler.Conn == nil {
			t.Fatal("OnOpen was not called!")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive message!")
	}
	fmt.Println("end")
}
/*
func TestClientSideClose(t *testing.T) {
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

	time.Sleep(100 * time.Millisecond)
	fmt.Println("close")
}
*/
