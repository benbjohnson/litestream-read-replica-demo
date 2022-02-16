package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
)

var (
	dsn  = flag.String("dsn", "", "datasource name")
	addr = flag.String("addr", ":8080", "bind address")
)

var notify struct {
	mu sync.Mutex
	ch chan struct{}

	value   int
	latency time.Duration
}

var primaryRegion = "ord"

var regions = []*Region{
	{Code: "ord", Primary: true},
	{Code: "sjc"},
	{Code: "ams"},
	{Code: "nrt"},
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

	notify.ch = make(chan struct{})

	// Default region to primary if not specified.
	if os.Getenv("FLY_REGION") == "" {
		os.Setenv("FLY_REGION", primaryRegion)
	}

	log.Printf("region: %s", os.Getenv("FLY_REGION"))
	log.Printf("opening database: %s", *dsn)

	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	} else if _, err := db.Exec(`PRAGMA journal_mode = wal`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	} else if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS t (id INTEGER PRIMARY KEY, value INTEGER, timestamp TEXT)`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	} else if _, err := db.Exec(`INSERT INTO t (id, value) VALUES (1, 0) ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("set journal mode: %w", err)
	}
	defer db.Close()

	// Monitor database file for changes.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("file watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(*dsn); err != nil {
		return fmt.Errorf("watch db file: %w", err)
	} else if err := watcher.Add(*dsn + "-wal"); err != nil {
		return fmt.Errorf("watch wal file: %w", err)
	}

	go func() {
		if err := monitor(ctx, db, watcher); err != nil {
			log.Fatalf("watcher: %s", err)
		}
	}()

	log.Printf("listening on %s", *addr)

	return http.ListenAndServe(*addr, &handler{db: db})
}

// monitor runs in a separate goroutine and monitors the main DB & WAL file.
func monitor(ctx context.Context, db *sql.DB, watcher *fsnotify.Watcher) error {
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("database file modified: %s", event.Name)
				if err := readDB(ctx, db); err != nil {
					log.Printf("read db: %s", err)
				}
			}
		case err := <-watcher.Errors:
			return err
		}
	}
}

func readDB(ctx context.Context, db *sql.DB) error {
	notify.mu.Lock()
	defer notify.mu.Unlock()

	var value int
	var timestamp string
	if err := db.QueryRowContext(ctx, `SELECT value, timestamp FROM t WHERE id = 1`).Scan(&value, &timestamp); err != nil {
		return fmt.Errorf("query: %w", err)
	}

	// Ignore if the value is the same.
	if notify.value == value {
		return nil
	}

	notify.value = value

	if timestamp != "" {
		t, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return fmt.Errorf("parse timestamp: %w", err)
		}
		notify.latency = time.Since(t)
	} else {
		notify.latency = 0
	}

	// Record latency
	log.Printf("update: value=%d latency=%s", notify.value, notify.latency)

	// Notify watchers.
	close(notify.ch)
	notify.ch = make(chan struct{})

	return nil
}

type Region struct {
	Code    string `json:"code"`
	Primary bool   `json:"primary"`
}
