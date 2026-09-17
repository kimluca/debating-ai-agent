# 🗣️ Multi-Agent Debate System

Give it a yes/no or contested question and three LLM personas — an
**Advocate**, a **Skeptic**, and a **Pragmatist** — argue it out over
multiple rounds, each one seeing the full transcript so far and reacting
to what came before. A neutral moderator pass then synthesizes where they
agreed, where they genuinely disagreed, and what a reasonable person
should take away.

## Stack

- **Go** orchestrator, `net/http`
- **GraphQL** — same hand-rolled, dependency-free resolver approach as the
  Codebase Archaeologist project
- **Postgres** to persist every debate transcript + synthesis
- **TypeScript + React** frontend rendering the live transcript, color-coded
  by persona

## Design choice: pluggable LLM provider

`internal/llm/llm.go` defines a one-method `Provider` interface. Two
implementations exist:

- `AnthropicProvider` — real calls to the Claude API
- `MockProvider` — deterministic, offline responses

If `ANTHROPIC_API_KEY` isn't set, the server automatically falls back to
the mock provider, so the **entire orchestration pipeline — multi-round
turn-taking, shared transcript, moderator synthesis, Postgres
persistence, GraphQL API, React frontend — is runnable and demoable
without any API key or cost.** Swap in a real key and the exact same code
path drives real model calls.

## Running it

```bash
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
export ANTHROPIC_API_KEY="sk-ant-..."   # optional - omit to use the mock provider
go run ./cmd/

cd web && npm install && npm run dev
```

## Deploying to GCP

Same shape as the archaeologist service: single static Go binary on
**Cloud Run**, **Cloud SQL for Postgres** via the Auth Proxy, `ANTHROPIC_API_KEY`
injected as a **Secret Manager** secret rather than a plain env var in
production.

## Project layout

```
cmd/server.go                entrypoint
internal/llm/                pluggable LLM provider (Anthropic + Mock)
internal/orchestrator/       turn-taking logic, persona definitions, synthesis
internal/db/                 Postgres persistence for debates
internal/graphql/            GraphQL HTTP handler
web/                         React/TypeScript frontend
```
