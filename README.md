# eightfivetwo

Monorepo for the **Eight Two Five** online storefront — a Ghanaian made-to-measure womenswear brand (see `Eight-Two-Five-Scope_v3.pdf` for the product scope). A React (Vite + MUI) frontend on Vercel and a Go API on Render, backed by MongoDB, with Resend for transactional email and Cloudinary for media (design photos).

Current state: production-ready foundation with a waitlist/coming-soon flow exercising every integration end-to-end. The storefront domains from the scope — collections, designs, size bands, measurements, bookings, orders, payments — slot into the same hexagonal structure (`internal/domain` ports → `internal/adapter` implementations).

**Start here:**

- [Architecture.md](Architecture.md) — the whole system explained: diagrams, layers, data model, API surface, decisions
- [agent_plan.md](agent_plan.md) — the feature board (✅ done / 🔧 taken / ⬜ open) with pickup protocol, plus per-epic specs

## Layout

```text
.
├── apps/
│   └── web/              # Vite + React 19 + MUI 9 SPA (deploys to Vercel)
├── services/
│   └── api/              # Go HTTP API, hexagonal architecture (deploys to Render)
├── .github/workflows/    # CI: web (turbo) + api (golangci-lint, testcontainers)
├── render.yaml           # Render blueprint for the API
├── turbo.json            # Turborepo task graph (JS/TS side)
└── pnpm-workspace.yaml
```

## Architecture

The Go API follows hexagonal (ports & adapters) architecture:

- `internal/domain` — entities, domain errors, and ports (`SubscriberRepository`, `EmailSender`, `UploadSigner`). Zero infrastructure imports.
- `internal/service` — use-cases (waitlist join/list) orchestrating the ports.
- `internal/adapter` — infrastructure implementations: `mongostore` (MongoDB), `email` (Resend + no-op fallback), `media` (Cloudinary upload signing).
- `internal/transport/httpapi` — chi router, handlers, domain-error → HTTP status mapping.
- `cmd/server` — composition root: config, wiring, graceful shutdown.

Integrations degrade gracefully: without `RESEND_API_KEY` emails are logged instead of sent; without Cloudinary credentials the signing endpoint returns 503. Uploads are signed server-side and sent browser → Cloudinary directly, so file bytes never transit the API.

## API

| Method | Path                   | Description                             |
| ------ | ---------------------- | --------------------------------------- |
| GET    | `/healthz`             | Liveness (used by Render health checks) |
| GET    | `/api/v1/healthz`      | Liveness (proxied path used by the SPA) |
| POST   | `/api/v1/waitlist`     | Join waitlist `{ "email", "name" }`     |
| GET    | `/api/v1/waitlist`     | List newest subscribers (max 100)       |
| POST   | `/api/v1/uploads/sign` | Signature for direct Cloudinary upload  |

Responses use an envelope: `{ "data": ... }` or `{ "error": { "code", "message" } }`.

## Local development

Prereqs: Node ≥ 22, pnpm ≥ 10, Go ≥ 1.25, Docker (for MongoDB and tests).

```sh
pnpm install

# 1. MongoDB
docker run -d --name eightfivetwo-mongo -p 27017:27017 mongo:8.0

# 2. API (http://localhost:8080)
cd services/api
MONGODB_URI=mongodb://localhost:27017 go run ./cmd/server

# 3. Web (http://localhost:5173 — proxies /api to :8080)
pnpm dev
```

## Testing & linting

```sh
pnpm test                      # web: vitest (Testing Library)
pnpm lint && pnpm typecheck    # web: eslint + tsc

cd services/api
make test          # full suite — spins up real MongoDB via testcontainers
make test-short    # unit tests only (no Docker needed)
make lint          # golangci-lint (config: .golangci.yml)
```

CI (`.github/workflows/ci.yml`) runs both sides on every push/PR: turbo tasks for the web, golangci-lint + `go test -race` (with testcontainers) for the API.

## Deployment

**Web → Vercel.** Import the repo, set the project **Root Directory** to `apps/web` (framework auto-detects as Vite). **Leave `VITE_API_URL` empty in production** — `vercel.json` rewrites `/api/*` to the Render service, so API calls stay same-origin and the `SameSite=Lax` session cookie is sent (a direct cross-origin `VITE_API_URL` would silently break auth). Optional: set the ignored-build-step to `npx turbo-ignore` to skip builds when `apps/web` didn't change.

**API → Render.** Create a Blueprint from this repo; `render.yaml` provisions the Docker service from `services/api`. Fill the `sync: false` env vars in the dashboard: `MONGODB_URI` (MongoDB Atlas), `CORS_ALLOWED_ORIGINS` **and** `WEB_URL` (both your exact Vercel origin, e.g. `https://eighttwofive.vercel.app`), `ADMIN_EMAILS` (comma-separated bootstrap super-admins — required or no one can reach `/admin`), `PAYSTACK_SECRET_KEY`, `RESEND_API_KEY` + `EMAIL_FROM`, and the `CLOUDINARY_*` keys.

### Environment variables (API)

See [services/api/.env.example](services/api/.env.example) for the full list with defaults.
