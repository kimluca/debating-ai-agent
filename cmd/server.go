// Command server starts the Multi-Agent Debate System.
//
// Env vars:
//
//	DATABASE_URL       Postgres DSN
//	ANTHROPIC_API_KEY  if set, personas are powered by the real Claude API;
//	                    if unset, a deterministic MockProvider is used so
//	                    the whole system still runs and is demoable offline.
//	DEBATE_ROUNDS      how many times each persona speaks (default 2)
//	PORT               defaults to 8081
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"debate/internal/db"
	"debate/internal/graphql"
	"debate/internal/llm"
	"debate/internal/orchestrator"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/debate?sslmode=disable"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	rounds := 2
	if v, err := strconv.Atoi(os.Getenv("DEBATE_ROUNDS")); err == nil && v > 0 {
		rounds = v
	}

	store, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("could not connect to postgres: %v", err)
	}
	log.Println("connected to postgres and ensured schema")

	var provider llm.Provider
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		provider = llm.NewAnthropicProvider(key)
		log.Println("using AnthropicProvider (real LLM calls)")
	} else {
		provider = llm.NewMockProvider()
		log.Println("ANTHROPIC_API_KEY not set - using MockProvider (deterministic, offline)")
	}

	orch := orchestrator.New(provider, rounds)
	srv := &graphql.Server{Store: store, Orchestrator: orch}

	http.HandleFunc("/graphql", srv.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	log.Printf("listening on :%s (POST /graphql)\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
