# Eight Two Five — System Architecture

A top-to-bottom tour of the system for anyone — engineer, agent, or stakeholder — who needs to understand how it all fits together. For _what to build next_, see [agent_plan.md](agent_plan.md) (feature board + epics). For the product scope, see `Eight-Two-Five-Scope_v3.pdf`.

## 1. What this is

**Eight Two Five (8-2-5)** is a Ghanaian fashion house making made-to-measure corporate womenswear. Today it sells through Instagram/Facebook DMs; this platform gives it an owned storefront where customers **browse collections → pick a size or get measured → pay or send a request → track the order** — and a private dashboard where the merchant runs the whole store. Every garment is custom-made; designs live in limited collections that are _retired_ when fabric runs out.

Two hard business rules shape everything:

1. **Fully standard orders** (listed size band, design as shown) show a price and are paid online to book. **The moment anything is custom, no price is shown** — the order becomes a request the merchant quotes directly (scope §4.4).
2. **No unpaid order ever enters production.** Payment confirmation (Paystack webhook) is what books an order — never a client-side callback (scope §4.5).

## 2. System overview

```text
                        ┌─────────────────────────────────────────────┐
                        │                  Browser                    │
                        └──────────────────────┬──────────────────────┘
                                               │ HTTPS (one origin)
                        ┌──────────────────────▼──────────────────────┐
                        │            Vercel (apps/web)                │
                        │  React 19 SPA (Vite, MUI v9, react-router)  │
                        │  vercel.json rewrites:                      │
                        │   /api/(.*) ──► Render API   (first-party   │
                        │   /(.*)    ──► index.html     cookies work) │
                        └──────────────────────┬──────────────────────┘
                                               │ /api/v1/... (JSON envelope)
                        ┌──────────────────────▼──────────────────────┐
                        │         Render (services/api, Docker)       │
                        │   Go HTTP API — hexagonal architecture      │
                        │   chi router · sessions · all-linters gate  │
                        └───┬──────────────┬──────────────┬───────────┘
                            │              │              │
              ┌─────────────▼──┐  ┌────────▼────────┐  ┌──▼──────────────┐
              │ MongoDB Atlas  │  │ Resend (email)  │  │ Cloudinary      │
              │ data of record │  │ login links,    │  │ design photos — │
              │                │  │ order emails    │  │ browser uploads │
              └────────────────┘  └─────────────────┘  │ direct, API     │
                                                       │ only signs      │
              ┌────────────────┐                       └─────────────────┘
              │ Paystack       │  planned (board F2): checkout, GHS 500
              │ (payments)     │  deposits, payment links, webhook booking
              └────────────────┘
```

Local development mirrors production: Vite's dev server proxies `/api` to `localhost:8080`, so the browser always talks to one origin and the session cookie is always first-party.

## 3. Repository layout

```text
.
├── agent_plan.md            # feature board (done/taken/open) + epics — START HERE for work
├── Architecture.md          # this document
├── Eight-Two-Five-*.pdf     # client scope + investment summary (source of truth)
├── apps/web/                # React storefront + admin (deploys to Vercel)
│   └── src/
│       ├── pages/           # route components (Landing, Store, Design, admin/*, ...)
│       ├── features/        # feature folders: auth/, waitlist/, catalog/, storefront/
│       ├── components/      # shared layout (StorefrontLayout, AnnouncementBar, ...)
│       ├── lib/api.ts       # envelope fetch helper + ApiError (+ auth client)
│       └── theme.ts         # ALL design tokens — no raw hex anywhere else
├── services/api/            # Go API (deploys to Render via render.yaml)
│   ├── cmd/server/          # main.go — entry point only (~50 lines)
│   ├── internal/
│   │   ├── app/             # composition root: wire.go (DI), run.go, server.go, logger.go
│   │   ├── config/          # env loading + validation
│   │   ├── domain/          # entities + ports (interfaces). Zero infrastructure imports.
│   │   ├── service/         # use-cases over ports: Waitlist, Auth, StoreSettings, Catalog
│   │   ├── adapter/
│   │   │   ├── mongostore/  # MongoDB repositories (one file per aggregate) + Connect
│   │   │   ├── email/       # Resend sender + logging no-op fallback
│   │   │   └── media/       # Cloudinary upload signer
│   │   └── transport/httpapi/  # chi router, middleware, handlers, DTOs, error mapping
│   ├── .golangci.yml        # ALL linters enabled (default: all) — the bar for every change
│   ├── Dockerfile           # multi-stage, distroless
│   └── Makefile
├── .github/workflows/ci.yml # web job (turbo) + api job (golangci-lint, race tests)
├── render.yaml              # Render blueprint (Docker service, health checks, env vars)
└── turbo.json / pnpm-workspace.yaml
```

## 4. Backend architecture (Go, hexagonal)

The dependency rule: **everything points inward.** `domain` knows nothing about Mongo, HTTP, or Resend; `service` knows only domain ports; adapters implement ports; the HTTP layer calls services; `internal/app` is the only place where concrete types meet.

```text
transport/httpapi ──► service ──► domain (ports + entities)
        ▲                              ▲
        │           implements         │
   internal/app ◄── adapter/{mongostore,email,media}
 (composition root — all wiring/DI in wire.go)
```

A request walk-through (`POST /api/v1/waitlist`):

1. `router.go` matches the route; middleware adds request ID, logging, recovery, 30s timeout, CORS.
2. `handlers.go` decodes the body (1 MiB cap), calls `service.Waitlist.Join`.
3. The service normalizes/validates, calls the `SubscriberRepository` port, then best-effort emails via the `EmailSender` port (a failed email never fails a signup).
4. The handler maps domain errors to HTTP: `ErrInvalidInput→422`, `ErrDuplicate*→409`, `ErrNotFound→404`, `ErrTokenInvalid→401`, else 500. Every response is the envelope `{"data": ...}` or `{"error": {"code", "message"}}`.

### Authentication (passwordless, scope "light by design")

No passwords exist anywhere. `POST /auth/request-link` upserts the user and emails a single-use link (15-min token, stored **SHA-256 hashed**, atomically consumed). `POST /auth/verify` exchanges it for a 30-day session (also hashed, revocable, TTL-indexed) set as an `e25_session` **HttpOnly SameSite=Lax cookie**. Because Vercel proxies `/api`, the cookie is first-party in every environment. Emails listed in `ADMIN_EMAILS` sign in with the admin role (promote-only). Route groups: public → authed (`RequireAuth`) → admin (`RequireAdmin`, guards `/api/v1/admin/*`).

### Catalog domain rules (implemented)

- Collections contain designs; designs carry **size bands**, each with its own chart (free-form key/value, e.g. `bust: 86 cm`) and its own price.
- **Slugs are immutable** once created (shared Instagram/WhatsApp links must never break); collisions auto-suffix (`velvet`, `velvet-2`).
- Lifecycle: `live ⇄ retired` (retired items 404 publicly but stay in the dashboard). Retiring/restoring a collection cascades to its designs; restoring a design under a retired collection is rejected. **Permanent delete** is a separate deliberate action (collection delete removes its designs).
- Search uses a Mongo text index over design name + note.

### Money

All amounts are **integer pesewas** (`int64`) end to end — GHS 500 is `50000`. Formatting to `GH₵ 500.00` happens only at the UI boundary (`features/catalog/money.ts`). Floats never touch money.

## 5. Frontend architecture (React SPA)

- **Stack**: Vite 8, React 19, MUI v9, react-router 7 (`createBrowserRouter`), TanStack Query 5 for all server state. No global client-state library — the server is the state.
- **Routes**: `/` waitlist landing (pre-launch face) · `/store`, `/collections/:slug`, `/designs/:slug`, `/about`, `/contact` storefront · `/login`, `/auth/verify` auth · `/account` (AuthGuard) · `/admin/*` (AdminGuard → designs/collections/settings tabs).
- **Auth on the client**: one `["me"]` query (`GET /auth/me`, null on 401). Guards render a spinner → redirect to `/login` (or `/` for non-admins). The session lives in the HttpOnly cookie; JS never sees a token.
- **Feature folders** (`src/features/<name>`) own their API functions, hooks, and components; pages compose them. The shared envelope/`ApiError` handling lives once in `lib/api.ts`.
- **Design system**: every color comes from `theme.ts` tokens (noir/clay/sand/stone/moss family). The visual language is editorial fashion e-commerce: white canvas, black announcement bar, uppercase tracked labels, squared black buttons, Fraunces serif display + Archivo body. The repo ships a design-intelligence skill at `.claude/skills/ui-ux-pro-max/` whose checklist (contrast, focus, touch targets, reduced motion) is part of review.
- **Photos**: the browser uploads directly to Cloudinary using a server-issued signature (admin-only endpoint) — file bytes never pass through the API. Public pages build delivery URLs from the `cloudName` exposed by `GET /settings`.

## 6. Data model (MongoDB)

| Collection     | Purpose                                                                            | Key indexes                                        |
| -------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------- |
| `subscribers`  | waitlist                                                                           | unique `email`                                     |
| `users`        | customers + merchant (role field)                                                  | unique `email`                                     |
| `login_tokens` | hashed single-use sign-in tokens                                                   | unique `tokenHash`, TTL `expiresAt`                |
| `sessions`     | hashed revocable sessions                                                          | unique `tokenHash`, TTL `expiresAt`                |
| `settings`     | single doc (`_id: "store"`)                                                        | —                                                  |
| `collections`  | catalog collections                                                                | unique `slug`                                      |
| `designs`      | designs w/ embedded bands + photos                                                 | unique `slug`, `collectionId`, text(`name`,`note`) |
| _planned_      | `orders`, `payments`, `slots`, `visits`, `deliveryRates` — see agent_plan.md E4–E6 |                                                    |

Documents are mapped through per-repo `*Doc` structs (bson tags) — domain entities never carry persistence tags.

## 7. API surface

Envelope everywhere; `/api/v1` prefix; cookie auth.

| Group            | Endpoints                                                                                                                                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------- | ----------------- |
| Public           | `GET /healthz` · `POST /waitlist` · `GET /settings` · `GET /collections` · `GET /collections/{slug}` · `GET /designs?collection=&q=` · `GET /designs/{slug}` · `POST /auth/request-link` · `POST /auth/verify` · `POST /auth/logout` |
| Authed           | `GET /auth/me`                                                                                                                                                                                                                       |
| Admin (`/admin`) | `GET /waitlist` · `PUT /settings` · `POST /uploads/sign` · collections CRUD + `/{id}/retire                                                                                                                                          | restore`+`DELETE`· designs CRUD +`GET /{id}`+ bulk`retire | restore`+`DELETE` |

## 8. Configuration

All config is environment variables, loaded in `internal/config` (godotenv reads `services/api/.env` locally — gitignored, holds real secrets).

| Variable                                   | Required | Purpose                                                             |
| ------------------------------------------ | -------- | ------------------------------------------------------------------- |
| `MONGODB_URI`                              | yes      | Atlas connection string                                             |
| `MONGODB_DB`, `PORT`, `ENV`                | no       | defaults: `eightfivetwo`, `8080`, `development`                     |
| `WEB_URL`                                  | no       | base for emailed sign-in links (default `http://localhost:5173`)    |
| `ADMIN_EMAILS`                             | no       | comma-separated; these sign in as admin                             |
| `CORS_ALLOWED_ORIGINS`                     | no       | needed only for cross-origin setups; prod uses the Vercel rewrite   |
| `RESEND_API_KEY`, `EMAIL_FROM`             | no       | absent → emails are logged, not sent (dev reads links from the log) |
| `CLOUDINARY_CLOUD_NAME/API_KEY/API_SECRET` | no       | absent → upload signing returns 503; UI degrades gracefully         |

Web: `VITE_API_URL` (empty in dev and prod — proxy/rewrite keep one origin).

## 9. Testing & quality gates

| Layer           | Tooling                                                                 | What it proves                                          |
| --------------- | ----------------------------------------------------------------------- | ------------------------------------------------------- |
| Go services     | unit tests with in-memory fakes                                         | business rules (validation, cascades, token single-use) |
| Go HTTP         | `httptest` end-to-end (real router + middleware + fakes)                | status mapping, auth guards, cookie flow                |
| Go repositories | **testcontainers** spinning real `mongo:8.0`                            | indexes, duplicate keys, TTL filters, text search       |
| Web             | Vitest + Testing Library (mocked fetch)                                 | components, validation, guards, payload shapes          |
| Visual          | agent-browser smoke screenshots                                         | pages render against the real API                       |
| Lint            | golangci-lint with **`default: all`** (every linter) · ESLint 10 strict | the bar for merging anything                            |

CI (`.github/workflows/ci.yml`) runs the web turbo pipeline and the Go job (golangci-lint action + `go test -race ./...`) on every push/PR — testcontainers run on the runner's Docker.

## 10. Deployment

- **Web → Vercel.** Root directory `apps/web`, auto-detected Vite. `vercel.json` rewrites `/api/(.*)` to the Render URL (keeps cookies first-party) and falls back to `index.html` for SPA routes.
- **API → Render.** `render.yaml` blueprint: Docker (multi-stage → distroless), health check `/healthz`, `sync: false` env vars filled in the dashboard. Graceful shutdown handles SIGTERM (10s drain).
- **Operating costs** (investment summary): ~$7–15/mo hosting, Paystack 1.95%/sale, Resend within 3k emails/mo.

## 11. Architecture decisions

- **REST, not gRPC/GraphQL** — one service, one first-party client; full rationale + revisit triggers in agent_plan.md §6.
- **Sessions in Mongo, not JWTs** — revocable, simple, one store; tokens always hashed at rest.
- **Email-link auth, no passwords** — fits the scope's "light accounts", kills reset flows.
- **Direct-to-Cloudinary uploads** — the API signs, the browser uploads; no file proxying.
- **Same-origin via proxy/rewrite everywhere** — avoids third-party-cookie breakage (Safari ITP) without a custom cookie domain.
- **Integer pesewas for money** — floats never touch amounts.
- **Composition root in `internal/app`** — `main.go` stays a thin entry point; each lifecycle concern in its own file.

## 12. Glossary

| Term                 | Meaning                                                                                        |
| -------------------- | ---------------------------------------------------------------------------------------------- |
| **Size band**        | A standard size (e.g. "8") with its own chart and set price                                    |
| **Chart**            | Key/value measurements for a band (`bust: 86 cm`)                                              |
| **Retire / restore** | Take a design/collection off the shop (reversible) — distinct from permanent delete            |
| **Custom path**      | Any order with custom size or design change: no shown price, becomes a merchant-quoted request |
| **Deposit**          | Fixed GHS 500 home-visit booking fee; counts toward the garment price; merchant-configurable   |
| **Booked**           | Paid-and-confirmed order state — set automatically by payment confirmation                     |
