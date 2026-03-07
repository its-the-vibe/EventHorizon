package hub_test

import (
	"testing"
	"time"

	"github.com/its-the-vibe/eventhorizon/internal/hub"
)

func TestBroadcast_SingleClient(t *testing.T) {
	h := hub.New()
	c := h.Subscribe()
	defer h.Unsubscribe(c)

	h.Broadcast("hello")

	select {
	case msg := <-c.Channel():
		if msg != "hello" {
			t.Errorf("got %q, want %q", msg, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestBroadcast_MultipleClients(t *testing.T) {
	h := hub.New()
	c1 := h.Subscribe()
	c2 := h.Subscribe()
	defer h.Unsubscribe(c1)
	defer h.Unsubscribe(c2)

	h.Broadcast("multi")

	for _, ch := range []<-chan string{c1.Channel(), c2.Channel()} {
		select {
		case msg := <-ch:
			if msg != "multi" {
				t.Errorf("got %q, want %q", msg, "multi")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for message")
		}
	}
}

func TestUnsubscribe_ClosesChannel(t *testing.T) {
	h := hub.New()
	c := h.Subscribe()
	h.Unsubscribe(c)

	_, open := <-c.Channel()
	if open {
		t.Error("channel should be closed after Unsubscribe")
	}
}

func TestBroadcast_NoClients(t *testing.T) {
	h := hub.New()
	// Should not panic or block.
	h.Broadcast("nobody here")
}

func TestBroadcast_UnsubscribedClientDoesNotReceive(t *testing.T) {
	h := hub.New()
	c := h.Subscribe()
	h.Unsubscribe(c)

	// Drain any buffered messages (there should be none).
	h.Broadcast("gone")

	// The channel is closed; receiving on a closed, empty channel returns zero value + false.
	select {
	case _, open := <-c.Channel():
		if open {
			t.Error("unsubscribed client should not receive messages")
		}
	default:
		// Channel closed with nothing buffered – correct.
	}
}
