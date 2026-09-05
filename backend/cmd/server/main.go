// Command server wires the DB connection, pipeline config, and API routes
// together (technical architecture doc section 4-6) and starts listening.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"vasooli/internal/api"
	"vasooli/internal/db"
	"vasooli/internal/guardrail"
	"vasooli/internal/pipeline"
)

func main() {
	ctx := context.Background()

	databaseURL := requireEnv("DATABASE_URL")
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	scenariosPath := os.Getenv("DEMO_SEED_PATH")
	if scenariosPath == "" {
		scenariosPath = "fixtures/demo_dataset.json"
	}
	simulatedHour := envInt("SIMULATED_HOUR", 12)

	conn, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer conn.Close()

	cfg := pipeline.Config{
		Caps:           guardrail.DefaultCaps,
		SimulatedHour:  simulatedHour,
		SimulatedToday: time.Now().UTC(),
		ScenariosPath:  scenariosPath,
	}

	server := api.NewServer(conn, cfg)
	mux := http.NewServeMux()
	server.Routes(mux)

	handler := api.CORSMiddleware(allowedOrigin, mux)

	log.Printf("vasooli backend listening on :%s (allowed origin: %q)", port, allowedOrigin)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("missing required environment variable %s", name)
	}
	return v
}

func envInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid integer for %s: %v", name, err)
	}
	return n
}
