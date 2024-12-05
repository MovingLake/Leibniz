package lib

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
)

type HttpHandler struct {
	cfg *LaunchConfig                 // LaunchConfig is a struct defined in config.go
	eps map[string]LeibnizHTTPHandler // Map of paths to http.Handler.
	am  map[string]map[string]bool    // Map of paths to allowed methods.
	db  *sql.DB
}

type LeibnizHTTPHandler interface {
	Serve(context.Context, *sql.DB, http.ResponseWriter, *http.Request)
}

// NewHandler creates a new HttpHandler.
func NewHandler(cfg *LaunchConfig, db *sql.DB, eps map[string]LeibnizHTTPHandler, am map[string]map[string]bool) *HttpHandler {
	return &HttpHandler{cfg: cfg, eps: eps, db: db, am: am}
}

func (h *HttpHandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request) {
	ctx := context.Background()
	e, ok := h.eps[rq.URL.Path]
	if !ok {
		http.Error(rw, fmt.Sprintf("Path not supported %s", rq.URL.Path), http.StatusNotFound)
		return
	}
	allowedMethods, ok := h.am[rq.URL.Path]
	if !ok {
		http.Error(rw, fmt.Sprintf("No HTTP methods allowed for path %s", rq.URL.Path), http.StatusNotFound)
		return
	}
	if v, ok := allowedMethods[rq.Method]; !ok || !v {
		http.Error(rw, fmt.Sprintf("HTTP method %s for path %s not supported", rq.Method, rq.URL.Path), http.StatusMethodNotAllowed)
		return
	}
	NewLogger("http-handler").Info("%s - %s %s\n", rq.Method, rq.URL.Path, rq.RemoteAddr)
	e.Serve(ctx, h.db, rw, rq)
}
