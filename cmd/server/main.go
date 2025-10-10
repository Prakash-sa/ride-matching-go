package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"

	httpapi "github.com/example/ride-matching/internal/http"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	srv := httpapi.NewServerFromEnv()
	// optional migration: run basic migrations/001_create_rides.sql if requested
	if dsn := os.Getenv("PG_DSN"); dsn != "" && os.Getenv("MIGRATE") == "true" {
		if db, err := sql.Open("postgres", dsn); err == nil {
			if b, err := ioutil.ReadFile(filepath.Join("migrations", "001_create_rides.sql")); err == nil {
				if _, err := db.Exec(string(b)); err != nil {
					log.Printf("migration exec error: %v", err)
				} else {
					log.Printf("migration applied: 001_create_rides.sql")
				}
			}
			_ = db.Close()
		} else {
			log.Printf("migration db open error: %v", err)
		}
	}
	log.Printf("ride-matching listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
