// Package main is the SaaS platform server entrypoint. It runs a
// dependency-free Go HTTP server exposing the Auth / Organization / Members /
// Invitation / Usage API (see internal/app). Build and run with:
//
//	go run ./cmd/server
//
// The server listens on :8080 (override with the PORT env var).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"saas/internal/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	a := app.New()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           a.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	log.Printf("SaaS API listening on http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}