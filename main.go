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

	_ "github.com/mattn/go-sqlite3"
)

var (
	dsn  = flag.String("dsn", "", "datasource name")
	addr = flag.String("addr", ":8080", "bind address")
)

//go:embed index.html main.js
var fs embed.FS

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

	http.Handle("/", http.FileServer(http.FS(fs)))
	return http.ListenAndServe(*addr, nil)
}

// isPrimary returns true if primary region matches the fly.io region or if primary is unset.
func isPrimary() bool {
	switch os.Getenv("PRIMARY_REGION") {
	case "", os.Getenv("FLY_REGION"):
		return true
	default:
		return false
	}
}
