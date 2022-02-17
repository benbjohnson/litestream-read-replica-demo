package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/benbjohnson/litestream-read-replica-demo/assets"
)

type handler struct {
	db *sql.DB
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/api") {
		log.Printf("http: %s %s", r.Method, &url.URL{Path: r.URL.Path, RawQuery: r.URL.RawQuery})
	}

	// Redirect to primary if this is a write request.
	if r.Method != "GET" && primaryRegion != os.Getenv("FLY_REGION") {
		log.Printf("redirecting to primary: %s", "region:"+primaryRegion)
		w.Header().Set("fly-replay", "region="+primaryRegion)
		return
	}

	// Otherwise, redirect to specified region.
	region := r.URL.Query().Get("region")
	if region != "" && region != os.Getenv("FLY_REGION") {
		log.Printf("redirecting to region: %s", "region:"+region)
		w.Header().Set("fly-replay", "region="+region)
		return
	}

	switch r.URL.Path {
	case "/api/regions":
		h.handleRegions(w, r)
	case "/api/inc":
		h.handleInc(w, r)
	case "/api/stream":
		h.handleStream(w, r)
	default:
		http.FileServer(http.FS(assets.FS)).ServeHTTP(w, r)
	}
}

func (h *handler) handleRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(regions)
}

func (h *handler) handleInc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		httpError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, err := h.db.ExecContext(r.Context(), `UPDATE t SET value = value + 1, timestamp = ? WHERE id = 1`, time.Now().Format(time.RFC3339Nano)); err != nil {
		httpError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{}`))
}

func (h *handler) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("stream connected %p", r)
	defer log.Printf("stream disconnected %p", r)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/event-stream")

	notify.mu.Lock()
	notifyCh, value, latency := notify.ch, notify.value, notify.latency
	notify.mu.Unlock()

	for {
		// Marshal data & write SSE message.
		if buf, err := json.Marshal(Event{Value: value, Latency: latency.Seconds()}); err != nil {
			log.Printf("cannot marshal event: %s", err)
			return
		} else if _, err := fmt.Fprintf(w, "event: update\ndata: %s\n\n", buf); err != nil {
			log.Printf("cannot write update event: %s", err)
			return
		}
		w.(http.Flusher).Flush()

		// Wait for change to value.
		select {
		case <-r.Context().Done():
			return
		case <-notifyCh:
		}

		notify.mu.Lock()
		notifyCh, value, latency = notify.ch, notify.value, notify.latency
		notify.mu.Unlock()
	}
}

type Event struct {
	Value   int     `json:"value"`
	Latency float64 `json:"latency"`
}

func httpError(w http.ResponseWriter, r *http.Request, err string, status int) {
	log.Printf("http error: %s %s: %s", r.Method, r.URL, err)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{err})
}
