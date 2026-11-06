// Command api is the Wormix API (Go port of the Laravel wormix-api).
//
// Environment:
//
//	DB_DATABASE   SQLite file (default /data/database.sqlite)
//	PHOTOS_DIR    uploaded photos (default: "photos" next to the database, i.e. /data/photos)
//	APP_KEY       accepted for compatibility; nothing the API persists is encrypted
//	LISTEN_ADDR   listen address (default 0.0.0.0:8000)
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/app"
	"github.com/andrij06062003/Wormix/services/api/internal/db"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	dbPath := env("DB_DATABASE", "/data/database.sqlite")
	dataDir := filepath.Dir(dbPath)
	photos := env("PHOTOS_DIR", filepath.Join(dataDir, "photos"))
	addr := env("LISTEN_ADDR", "0.0.0.0:8000")

	if err := os.MkdirAll(filepath.Join(photos, "users"), 0o755); err != nil {
		log.Fatalf("photos dir: %v", err)
	}
	// Keep /data/app_key like the old entrypoint did (lets the PHP API be started on the same volume).
	if os.Getenv("APP_KEY") == "" {
		keyFile := filepath.Join(dataDir, "app_key")
		if st, err := os.Stat(keyFile); err != nil || st.Size() == 0 {
			b := make([]byte, 32)
			rand.Read(b)
			os.WriteFile(keyFile, []byte("base64:"+base64.StdEncoding.EncodeToString(b)+"\n"), 0o600)
		}
	}

	_, statErr := os.Stat(dbPath)
	fresh := os.IsNotExist(statErr)

	d, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	ctx := context.Background()
	created, err := d.Migrate(ctx)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	// game:init only into a new database (running it twice reports duplicates).
	if fresh || created {
		if err := d.Seed(ctx); err != nil {
			log.Fatalf("game:init: %v", err)
		}
	}

	srv := &http.Server{Addr: addr, Handler: app.NewApp(d, photos), ReadHeaderTimeout: 30 * time.Second}
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Printf("wormix-api listening on %s (db %s)", addr, dbPath)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	d.Close()
}
