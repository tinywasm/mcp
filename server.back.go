//go:build ignore

package mcp

import (
	stdctx "context"
	"io"
	"net/http"
	"time"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)

// HTTPHandler returns an http.Handler that serves the MCP endpoint.
func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var ctx context.Context
		if auth := r.Header.Get("Authorization"); auth != "" {
			ctx.Set(CtxKeyAuthToken, auth)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		response := s.HandleMessage(&ctx, body)

		w.Header().Set("Content-Type", "application/json")
		if response != nil {
			var b []byte
			if f, ok := response.(fmt.Fielder); ok {
				json.Encode(f, &b)
			}
			w.Write(b)
		}
	})
}

// Serve starts the HTTP server and blocks until exitChan is closed.
func (s *Server) Serve(exitChan chan bool, port string) {
	mux := http.NewServeMux()
	mux.Handle("/mcp", s.HTTPHandler())

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("MCP server error:", err)
		}
	}()

	<-exitChan
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func (s *Server) SetLog(f func(message ...any)) {
	s.log = f
}
