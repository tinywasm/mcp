package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/time"
)

func TestClient_AuthHeader_Sent(t *testing.T) {
	gotAuth := make(chan string, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":null}`))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "my-token")

	// Test Call
	client.Call(nil, "ping", nil, func([]byte, error) {})
	select {
	case a := <-gotAuth:
		if a != "Bearer my-token" {
			t.Errorf("expected 'Bearer my-token', got %q", a)
		}
	case <-timeWait(2000):
		t.Fatal("timed out waiting for Call auth header")
	}

	// Test Dispatch
	client.Dispatch(nil, "ping", nil)
	select {
	case a := <-gotAuth:
		if a != "Bearer my-token" {
			t.Errorf("expected 'Bearer my-token', got %q", a)
		}
	case <-timeWait(2000):
		t.Fatal("timed out waiting for Dispatch auth header")
	}
}

func TestClient_EmptyToken_NoHeader(t *testing.T) {
	gotAuth := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":null}`))
	}))
	defer srv.Close()

	client := mcp.NewClient(srv.URL, "")
	client.Call(nil, "ping", nil, func([]byte, error) {})

	select {
	case a := <-gotAuth:
		if a != "" {
			t.Errorf("expected empty auth header, got %q", a)
		}
	case <-timeWait(2000):
		t.Fatal("timed out")
	}
}

func timeWait(ms int) <-chan bool {
	c := make(chan bool, 1)
	time.AfterFunc(ms, func() {
		c <- true
	})
	return c
}
