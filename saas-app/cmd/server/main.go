// Package main is the SaaS platform server entrypoint. It runs a
// dependency-free Go HTTP server exposing the Auth / Organization / Members /
// Invitation / Usage API (see internal/app). Build and run with:
//
//	go run ./cmd/server
//
// The server listens on :8080 (override with the PORT env var) and persists to
// ./data/saas-store.json (override with SAAS_DATA_FILE, or set it to "" to run
// memory-only).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"saas/internal/app"
)

const defaultDataFile = "data/saas-store.json"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dataFile := os.Getenv("SAAS_DATA_FILE")
	if dataFile == "" {
		dataFile = defaultDataFile
	}

	a, err := app.New(dataFile)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	if err := a.Auth.SeedAdminIfAbsent(); err != nil {
		log.Fatalf("failed to seed platform admin: %v", err)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           a.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	log.Printf("SaaS API listening on http://localhost:%s (store: %s)", port, a.Store.Path())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}