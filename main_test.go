package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/its-the-vibe/eventhorizon/internal/hub"
)

// nonFlusher is a minimal ResponseWriter that does NOT implement http.Flusher,
// used to exercise the "streaming unsupported" branch of sseHandler.
type nonFlusher struct {
	code int
	body string
}

func (nf *nonFlusher) Header() http.Header        { return http.Header{} }
func (nf *nonFlusher) WriteHeader(code int)        { nf.code = code }
func (nf *nonFlusher) Write(b []byte) (int, error) { nf.body += string(b); return len(b), nil }

// flusherRecorder wraps httptest.ResponseRecorder and adds a Flush method so
// that sseHandler can proceed past the http.Flusher type assertion.
// Each call to Flush signals the flushed channel so tests can wait for it.
type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
}

func newFlusherRecorder() *flusherRecorder {
	return &flusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          make(chan struct{}, 1),
	}
}

func (fr *flusherRecorder) Flush() {
	select {
	case fr.flushed <- struct{}{}:
	default:
	}
}

func TestSSEHandler_NonFlusher(t *testing.T) {
	h := hub.New()
	handler := sseHandler(slog.Default(), h)

	w := &nonFlusher{}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	handler.ServeHTTP(w, r)

	if w.code != http.StatusInternalServerError {
		t.Errorf("expected status %d for non-flusher, got %d", http.StatusInternalServerError, w.code)
	}
}

func TestSSEHandler_SetsSSEHeaders(t *testing.T) {
	h := hub.New()
	handler := sseHandler(slog.Default(), h)

	ctx, cancel := context.WithCancel(context.Background())

	fr := newFlusherRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(fr, r)
	}()

	// Cancel the context to make the handler return, then wait for it to
	// finish. Headers are set before the select loop, so they are always
	// written before the handler exits — safe to read after <-done.
	cancel()
	<-done

	if ct := fr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
	if cc := fr.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}
}

func TestSSEHandler_BroadcastDelivered(t *testing.T) {
	h := hub.New()
	handler := sseHandler(slog.Default(), h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fr := newFlusherRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(fr, r)
	}()

	// Poll until the handler goroutine has subscribed to the hub.
	deadline := time.Now().Add(time.Second)
	for h.Len() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("handler did not subscribe within 1s")
		}
		time.Sleep(time.Millisecond)
	}

	h.Broadcast("hello-sse")

	// Wait for the handler to flush the message to the recorder.
	select {
	case <-fr.flushed:
	case <-time.After(time.Second):
		t.Fatal("handler did not flush message within 1s")
	}

	cancel()
	<-done

	// Read body only after the handler has finished to avoid a data race.
	if body := fr.Body.String(); !strings.Contains(body, "data: hello-sse") {
		t.Errorf("response body = %q, want it to contain %q", body, "data: hello-sse")
	}
}


