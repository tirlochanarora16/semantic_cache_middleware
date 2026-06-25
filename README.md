# Semantic Cache Middleware

A high-performance Go middleware that adds **semantic caching** to LLM API calls. It caches OpenAI responses based on the _meaning_ of prompts using vector embeddings and Redis Search, dramatically reducing latency and API costs for semantically similar queries.

---

## Overview

Traditional caching relies on exact string matching. This project uses **semantic similarity** — powered by OpenAI embeddings and Redis Vector Search (RediSearch) — to recognize when two different phrasings of the same question should return the same answer.

### How It Works

1. **Receive Prompt** → A user sends a prompt to `POST /v1/generate`.
2. **Embed** → The prompt is converted into a 1536-dimensional vector using OpenAI `text-embedding-3-small`.
3. **Search Cache** → Redis performs a KNN vector search to find the most semantically similar cached prompt.
4. **Cache Hit** → If the similarity score is within the threshold (`≤ 0.08`), the cached response is returned instantly.
5. **Cache Miss** → If no close match is found, the request is forwarded to OpenAI (`gpt-4.1-mini`), the response is stored in both Redis (for future semantic hits) and PostgreSQL (for audit logging), and then returned to the user.

---

## Features

- **Semantic Caching** — Caches responses by meaning, not just exact text match.
- **Vector Search** — Uses Redis Search (HNSW) with cosine distance for fast KNN lookups.
- **Dual Storage** — Redis for hot vector cache; PostgreSQL for persistent request logs.
- **Type-Safe SQL** — Auto-generated Go code from SQL queries using [sqlc](https://sqlc.dev/).
- **Database Migrations** — Managed with [golang-migrate](https://github.com/golang-migrate/migrate).
- **Structured Logging** — Logs cache hits, misses, and latency metrics.
- **Modular Design** — Clean separation between API handlers, LLM services, and database layers.

---

## Tech Stack

| Layer            | Technology                                    |
| ---------------- | --------------------------------------------- |
| Language         | Go 1.25                                       |
| LLM Provider     | OpenAI (GPT-4.1-mini, text-embedding-3-small) |
| Vector Cache     | Redis with RediSearch (HNSW index)            |
| Persistent Store | PostgreSQL (pgx/v5)                           |
| SQL Generation   | sqlc                                          |
| Migrations       | golang-migrate                                |
| Config           | godotenv                                      |

---

## Project Structure

```
.
├── docker-compose.yml      # Local orchestration: app, postgres, redis, migrate
├── Dockerfile              # Multi-stage image build for app
├── api/
│   └── handler.go          # HTTP handlers (POST /v1/generate)
├── database/
│   ├── postgres.go         # PostgreSQL connection & logging
│   └── redis.go            # Redis connection, vector index, cache ops
├── db/
│   ├── migrations/         # golang-migrate SQL files
│   └── queries/            # sqlc query definitions
├── llm/
│   ├── client.go           # OpenAI client initialization
│   └── service.go          # Core logic: embed → search → generate → cache
├── internal/db/            # sqlc-generated Go code (gitignored)
├── main.go                 # Application entry point
├── sqlc.yaml               # sqlc configuration
├── go.mod / go.sum         # Go dependencies
└── .gitignore
```

---

## Prerequisites

- **Go** 1.25+
- **PostgreSQL** 14+
- **Redis** 7.x with the **RediSearch** module (or Redis Stack)
- **OpenAI API Key**

### Redis Setup

You need Redis with the Search module enabled. The easiest way is to use Redis Stack:

```bash
docker run -d --name redis-stack -p 6379:6379 redis/redis-stack:latest
```

Or ensure your Redis instance supports `FT.CREATE` and `FT.SEARCH` commands.

---

## Installation

### Option A: Run with Docker Compose (Recommended)

```bash
# Set your OpenAI key in .env
echo 'OPENAI_API_KEY=sk-...' > .env

# Build and start all services
docker compose up --build
```

This starts:

- `postgres` (persistent DB)
- `redis` (Redis Stack with RediSearch)
- `migrate` (runs SQL migrations and exits)
- `app` (starts only after dependencies are healthy and migrations succeed)

API endpoint:

- `http://localhost:3000/v1/generate`

### Option B: Run Locally (Without Docker)

```bash
# Clone the repository
git clone https://github.com/tirlochanarora16/semantic_cache_middleware.git
cd semantic_cache_middleware

# Download dependencies
go mod download

# Run database migrations (requires migrate CLI)
migrate -path db/migrations -database "$PSQL_CONN_STR" up

# Generate SQL code (if you modify queries)
sqlc generate

# Build and run
go build -o bin/server .
./bin/server
```

---

## Configuration

Create a `.env` file in the project root:

```env
OPENAI_API_KEY=sk-...
```

| Variable         | Description         |
| ---------------- | ------------------- |
| `OPENAI_API_KEY` | Your OpenAI API key |

Docker Compose notes:

- In Compose, the app uses container-network endpoints from `docker-compose.yml`:
  - `REDIS_URL=redis:6379`
  - `PSQL_CONN_STR=postgres://postgres:postgres@postgres:5432/semantic_cache?sslmode=disable`
- `.env` is still used for `OPENAI_API_KEY`.
- If `.env` is missing inside the container filesystem, startup continues using injected environment variables.

---

## API Usage

### Generate Response

```bash
curl -X POST http://localhost:3000/v1/generate   -H "Content-Type: application/json"   -d '{"prompt": "What is the capital of France?"}'
```

**Response (Cache Miss):**

```json
{
  "response": "The capital of France is Paris.",
  "cache_hit": false,
  "similarity_score": 0
}
```

**Response (Cache Hit):**

```bash
curl -X POST http://localhost:3000/v1/generate   -H "Content-Type: application/json"   -d '{"prompt": "Tell me the capital city of France"}'
```

```json
{
  "response": "The capital of France is Paris.",
  "cache_hit": true,
  "similarity_score": 0.0523
}
```

---

## Database Schema

### PostgreSQL (`llm_logs`)

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE llm_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt TEXT NOT NULL,
    response TEXT NOT NULL,
    latency_ms INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);
```

### Redis Vector Index (`idx:cache`)

- **Type:** HNSW (Hierarchical Navigable Small World)
- **Dimension:** 1536 (matches OpenAI embedding size)
- **Distance Metric:** Cosine
- **Data Structure:** Redis Hash with prefix `cache:`

---

## How the Cache Works

### Similarity Threshold

The project uses a **cosine distance threshold of `0.08`** to determine a cache hit. This means if the closest cached prompt is within 0.08 distance of the new prompt, the cached response is returned.

### Cache Flow

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Client    │────▶│   Handler    │────▶│  Generate Embed │
└─────────────┘     └──────────────┘     └─────────────────┘
                                                  │
                                                  ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Return    │◀────│  Cache Hit?  │◀────│  Redis KNN      │
│   Response  │     │  (≤ 0.08)    │     │  Vector Search  │
└─────────────┘     └──────────────┘     └─────────────────┘
                            │ Yes
                            ▼
                     ┌──────────────┐
                     │ Return Cached│
                     │  Response    │
                     └──────────────┘
                            │ No
                            ▼
                     ┌──────────────┐     ┌─────────────────┐
                     │  Call OpenAI │────▶│  Save to Redis  │
                     │  GPT-4.1-mini│     │  & PostgreSQL   │
                     └──────────────┘     └─────────────────┘
```

---

## Development

### Generate SQL Code

After modifying files in `db/queries/` or `db/migrations/`:

```bash
sqlc generate
```

### Run Migrations

```bash
# Up
migrate -path db/migrations -database "$PSQL_CONN_STR" up

# Down
migrate -path db/migrations -database "$PSQL_CONN_STR" down
```

With Docker Compose:

```bash
# Re-run migrations container manually if needed
docker compose run --rm migrate
```

### Running Tests

```bash
go test ./...
```

---

## Environment Variables Summary

| Variable         | Required | Default | Description                                      |
| ---------------- | -------- | ------- | ------------------------------------------------ |
| `OPENAI_API_KEY` | Yes      | —       | OpenAI API key                                   |
| `REDIS_URL`      | Yes      | —       | Redis connection string (e.g., `localhost:6379`) |
| `PSQL_CONN_STR`  | Yes      | —       | PostgreSQL connection string                     |

---

## License

[MIT](LICENSE)

---

## Author

**Tirlochan Arora** — [GitHub](https://github.com/tirlochanarora16)
