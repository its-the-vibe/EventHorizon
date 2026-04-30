package hub_test

import (
	"fmt"
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

func TestBroadcast_FullBuffer(t *testing.T) {
	h := hub.New()
	c := h.Subscribe()
	defer h.Unsubscribe(c)

	// Fill the channel buffer (capacity 16) without draining it.
	for i := range 16 {
		h.Broadcast(fmt.Sprintf("msg-%d", i))
	}

	// This extra broadcast must not block even though the buffer is full.
	h.Broadcast("overflow")

	// Drain and count only the buffered messages.
	count := 0
drain:
	for {
		select {
		case <-c.Channel():
			count++
		default:
			break drain
		}
	}

	if count != 16 {
		t.Errorf("expected 16 buffered messages, got %d", count)
	}
}

func TestLen(t *testing.T) {
	h := hub.New()

	if got := h.Len(); got != 0 {
		t.Errorf("Len() = %d before any subscribes, want 0", got)
	}

	c1 := h.Subscribe()
	if got := h.Len(); got != 1 {
		t.Errorf("Len() = %d after 1 subscribe, want 1", got)
	}

	c2 := h.Subscribe()
	if got := h.Len(); got != 2 {
		t.Errorf("Len() = %d after 2 subscribes, want 2", got)
	}

	h.Unsubscribe(c1)
	if got := h.Len(); got != 1 {
		t.Errorf("Len() = %d after 1 unsubscribe, want 1", got)
	}

	h.Unsubscribe(c2)
	if got := h.Len(); got != 0 {
		t.Errorf("Len() = %d after all unsubscribes, want 0", got)
	}
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
