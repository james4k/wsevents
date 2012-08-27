package wsevents_test

import (
	//"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/james4k/wsevents"
)

type SimpleHandler struct {
	*wsevents.Connection
	MsgChan   chan string
	CloseChan chan bool
}

func (s *SimpleHandler) OnOpen(conn *wsevents.Connection) {
	s.Connection = conn
}

func (s *SimpleHandler) OnError(err error) {
	panic(err)
}

func (s *SimpleHandler) OnClose(err error) {
	if s.CloseChan != nil {
		s.CloseChan <- true
	} else {
		panic(err)
	}
}

func (s *SimpleHandler) OnTestMsg(msg string) {
	s.Close()
	s.MsgChan <- msg
}

func TestSimple(t *testing.T) {
	var simpleHandler *SimpleHandler
	msgChan := make(chan string)
	onNew := func(handler wsevents.EventHandler) {
		var ok bool
		simpleHandler, ok = handler.(*SimpleHandler)
		if !ok {
			t.Fatal("handler was not a *SimpleHandler")
		}

		simpleHandler.MsgChan = msgChan
	}

	serv := httptest.NewServer(wsevents.Handler(&SimpleHandler{}, onNew))
	defer serv.CloseClientConnections()
	defer serv.Close()

	origin := serv.URL
	url := "ws" + serv.URL[4:]
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Error(err)
	}

	websocket.Message.Send(ws, `{"name": "testmsg", "args": ["test 123![]{}@"]}`)

	select {
	case msg := <-msgChan:
		if msg != "test 123![]{}@" {
			t.Fatal("message did not match!")
		}
		if simpleHandler.Connection == nil {
			t.Fatal("OnOpen was not called!")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive message!")
	}
}

func TestClientSideClose(t *testing.T) {
	var simpleHandler *SimpleHandler
	closeChan := make(chan bool)
	onNew := func(handler wsevents.EventHandler) {
		var ok bool
		simpleHandler, ok = handler.(*SimpleHandler)
		if !ok {
			t.Fatal("handler was not a *SimpleHandler")
		}

		simpleHandler.CloseChan = closeChan
	}

	serv := httptest.NewServer(wsevents.Handler(&SimpleHandler{}, onNew))

	origin := serv.URL
	url := "ws" + serv.URL[4:]
	_, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Error(err)
	}

	serv.CloseClientConnections()

	select {
	case <-closeChan:
		if simpleHandler.Connection == nil {
			t.Fatal("OnOpen was not called!")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not get notified of close!")
	}
}
