// Package hub manages SSE client subscriptions and message broadcasting.
package hub

import (
	"sync"
)

// Client represents a connected SSE client.
type Client struct {
	ch chan string
}

// Channel returns the channel on which messages are delivered to this client.
func (c *Client) Channel() <-chan string { return c.ch }

// Hub manages a set of SSE clients and broadcasts messages to all of them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// New returns a ready-to-use Hub.
func New() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

// Subscribe registers a new client and returns it.
func (h *Hub) Subscribe() *Client {
	c := &Client{ch: make(chan string, 16)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

// Unsubscribe removes a client and closes its channel.
func (h *Hub) Unsubscribe(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.ch)
}

// Broadcast sends msg to every registered client.
// Clients whose buffers are full are skipped to avoid blocking the broadcaster.
func (h *Hub) Broadcast(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.ch <- msg:
		default:
		}
	}
}
