package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dsn  = flag.String("dsn", "", "datasource name")
	addr = flag.String("addr", ":8080", "bind address")
)

//go:embed index.html main.js
var fs embed.FS

var notify struct {
	mu sync.Mutex
	ch chan struct{}
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	flag.Parse()
	if *dsn == "" {
		return fmt.Errorf("flag required: -dsn DSN")
	}

	log.Printf("opening database: %s", *dsn)

	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	} else if _, err := db.Exec(`PRAGMA journal_mode = wal`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	}
	defer db.Close()

	log.Printf("listening on %s", *addr)

	return http.ListenAndServe(*addr, &handler{db: db})
}

type handler struct {
	db *sql.DB
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("region")

	// Redirect to primary if this is a write request.
	if r.Method != "GET" && !isPrimary() {
		w.Header().Set("fly-replay", fmt.Sprintf("region:"+primaryRegion()))
		return
	}

	// Otherwise, redirect to specified region.
	region := r.URL.Query().Get("region")
	if region != "" && region != currentRegion() {
		w.Header().Set("fly-replay", fmt.Sprintf("region:"+currentRegion()))
		return
	}

	switch r.URL.Path {
	case "/stream":
		h.handleStream(w, r)
	default:
		http.FileServer(http.FS(fs)).ServeHTTP(w, r)
	}
}

func (h *handler) handleStream(w http.ResponseWriter, r *http.Request) {
	log.Printf("stream connected %p", r)
	defer log.Printf("stream disconnected %p", r)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/event-stream")

	if _, err := fmt.Fprintf(w, "event: init\n\n"); err != nil {
		log.Printf("cannot write init event: %s", err)
		return
	}
	w.(http.Flusher).Flush()

	notify.mu.Lock()
	notifyCh := notify.ch
	notify.mu.Unlock()

	for {
		// Query from database.
		var n int
		if err := h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
			log.Printf("cannot query: %s", err)
			return
		}

		// Marshal data & write SSE message.
		if _, err := fmt.Fprintf(w, "event: update\ndata: %d\n\n"); err != nil {
			log.Printf("cannot write update event: %s", err)
			return
		}
		w.(http.Flusher).Flush()

		select {
		case <-r.Context().Done():
			return
		case <-notifyCh:
		}

		notify.mu.Lock()
		notifyCh = notify.ch
		notify.mu.Unlock()
	}
}

// currentRegion returns the fly.io region. If unset, returns "local".
func currentRegion() string {
	if v := os.Getenv("FLY_REGION"); v != "" {
		return v
	}
	return "local"
}

// primaryRegion returns the primary region. If unset, returns "local".
func primaryRegion() string {
	if v := os.Getenv("PRIMARY_REGION"); v != "" {
		return v
	}
	return "local"
}

// isPrimary returns true if primary region matches the fly.io region or if primary is unset.
func isPrimary() bool {
	switch primaryRegion() {
	case "", currentRegion():
		return true
	default:
		return false
	}
}
