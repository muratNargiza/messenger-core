package messenger

import (
	"encoding/json"
	"testing"
	"time"
)

func testHub(t *testing.T) (*Hub, func()) {
	t.Helper()
	h := NewHub()
	go h.Run()
	return h, h.Stop
}

func testClient(id int64) *Client {
	return &Client{id: id, send: make(chan []byte, 256), done: make(chan struct{})}
}

func TestHub_Run_DirectMessage(t *testing.T) {
	h, cancel := testHub(t)
	defer cancel()

	c1 := testClient(1)
	c1.hub = h
	c2 := testClient(2)
	c2.hub = h

	h.register <- c1
	h.register <- c2
	time.Sleep(50 * time.Millisecond)

	msg := DirectMessage{From: 1, To: 2, Message: "hello"}
	h.direct <- msg

	select {
	case gotBytes := <-c2.send:
		var gotMsg DirectMessage
		json.Unmarshal(gotBytes, &gotMsg)
		if gotMsg.Message != "hello" || gotMsg.From != 1 {
			t.Errorf("got %+v, want %+v", gotMsg, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("message not delivered to client")
	}

	h.unregister <- c1
	h.unregister <- c2
	time.Sleep(50 * time.Millisecond)

	select {
	case _, ok := <-c1.send:
		if ok {
			t.Error("expected closed channel c1")
		}
	case <-time.After(time.Second):
		t.Fatal("channel c1 not closed")
	}
}

func TestHub_Register_ReplacesOldConnection(t *testing.T) {
	h, cancel := testHub(t)
	defer cancel()

	c1Old := testClient(1)
	c1Old.hub = h
	h.register <- c1Old
	time.Sleep(50 * time.Millisecond)

	c1New := testClient(1)
	c1New.hub = h
	h.register <- c1New
	time.Sleep(50 * time.Millisecond)

	select {
	case _, ok := <-c1Old.send:
		if ok {
			t.Error("expected old connection channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("old connection channel not closed")
	}
}

func TestHub_DirectMessage_NonExistentClient(t *testing.T) {
	h, cancel := testHub(t)
	defer cancel()

	msg := DirectMessage{From: 1, To: 999, Message: "hello"}
	h.direct <- msg

	time.Sleep(50 * time.Millisecond)
}

func TestHub_Unregister_NotFound(t *testing.T) {
	h, cancel := testHub(t)
	defer cancel()

	c := testClient(1)
	c.hub = h
	h.unregister <- c

	time.Sleep(50 * time.Millisecond)
}

func TestHub_Shutdown(t *testing.T) {
	h, cancel := testHub(t)

	c := testClient(1)
	c.hub = h
	h.register <- c
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)

	select {
	case _, ok := <-c.send:
		if ok {
			t.Error("expected client channel to be closed on shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("client channel not closed after shutdown")
	}
}
