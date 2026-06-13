# Eight Two Five — Agent Implementation Plan

Implementation plan for the **Eight Two Five Online Storefront** (made-to-measure womenswear, Accra). Written for coding agents working in this monorepo: each epic is independently buildable, traced to the client-approved scope, and ends with verifiable acceptance criteria.

**Sources of truth** (repo root, confidential):

- `Eight-Two-Five-Scope_v3.pdf` — Project Scope v3.0, June 2026 (sections cited below as §)
- `Eight-Two-Five-Investment-Summary.pdf` — GHS 20,000 build, 5 components; Paystack named as provider

**Already built** (do not rebuild): monorepo (pnpm + Turborepo), `apps/web` (Vite + React 19 + MUI v9, Oh Polly-style theme in `src/theme.ts`), `services/api` (Go, hexagonal: `domain` → `service` → `adapter` → `transport/httpapi`, composition root in `internal/app`), MongoDB via `mongostore`, Resend email adapter, Cloudinary upload signing, waitlist feature (end-to-end), CI (GitHub Actions: turbo + golangci-lint with ALL linters + testcontainers), deploy configs (Vercel for web with `/api` rewrite to Render for first-party cookies, `render.yaml` for API).

**Slice 4 + Slice 6 (2026-06-12)** — F2 Paystack adapter, E3 measurement chooser, E4 booking calendar, E5a standard orders, E5b custom-request path, E5c manual mark-as-paid, E6 delivery rates, E7 customer account area, E8b admin orders inbox, and E8c admin settings UI are DONE. `domain.PaymentProvider` port + Paystack adapter (`init transaction`, `verify webhook`, `payment link`, payment-event audit); orders domain/state machine (`pending_payment` → `booked` via webhook, invariant: unpaid orders cannot enter `in_production`); standard checkout API (`POST /api/v1/orders`, `POST /payments/webhook`, `GET /orders`, `GET /orders/{ref}`, `GET /admin/orders`); custom request API (`POST /api/v1/orders/request`, `CreateCustomRequest` service); admin order operations (`GET /admin/orders/:ref`, `PUT /admin/orders/:ref/quote`, `POST /admin/orders/:ref/payment-link`, `POST /admin/orders/:ref/mark-paid`, `POST /admin/orders/:ref/status`); delivery rates added to settings and auto-applied at checkout; customer account pages (`/account`, `/account/orders/:ref`) with orders list, order detail, customer-facing stages, and booked-visit call card; `/admin/orders` inbox with three sorted buckets, detail panel, WhatsApp deep link, quote editor, payment link, manual mark-paid, and status transitions; `/admin/settings` full editor (deposit, WhatsApp, visit location, delivery rates table); E4 calendar (`slots` + `visits` domain/service/repo/handlers, `POST /slots/:id/book` with deposit payment, admin slot/visit management, reschedule/cancel reopen slots); E3 measurement chooser on `DesignPage` (band/self/home_visit/workplace + design-change textarea, price-hide rule). E8d admin analytics, F3 notification templates, and E2a OG meta are DONE. E9 launch code (real Home page) is DONE. Next: E9 client content load, domain + env cutover (client/deployment tasks).

**Slice 2 complete (2026-06-12)** — E1 + E8 catalog part are DONE: collections and designs (size bands with per-band chart + pesewas price, photos as Cloudinary public IDs), immutable unique slugs with auto-suffix (`velvet-2`), live/retired lifecycle with cascade (retire/restore collection ⇒ its designs; design restore blocked while collection retired), permanent delete (collection delete cascades), Mongo text-index search, public storefront endpoints (`/collections`, `/collections/{slug}`, `/designs?collection=&q=`, `/designs/{slug}` — retired = 404) and full admin CRUD under `/api/v1/admin` (upload signing moved admin-side). Admin UI: `/admin` shell (Designs | Collections | Settings tabs), collections table + dialogs, designs list (filter/search/bulk retire-restore/thumbnails), design editor with Cloudinary direct upload + size-band/chart editor. Next: Slice 3 — storefront read-side pages (E2).

**Slice 1 complete (2026-06-12)** — F1, F4, F5, F6 are DONE: passwordless email-link auth (`service.Auth`, hashed single-use login tokens + revocable sessions in Mongo with TTL indexes, `e25_session` HttpOnly cookie, `ADMIN_EMAILS` allowlist promotes to admin), store settings (GHS 500 default deposit, public GET + admin PUT), route groups (public / authed `/auth/me` / admin `/api/v1/admin/*`; waitlist listing moved under admin), and the web router shell (react-router 7: `/`, `/login`, `/auth/verify`, `/account`, `/admin`, guards on the me-query). Next: Slice 2 — catalog write-side + admin shell (E1, E8 catalog part).

---

## Feature board — canonical status

**This table is the single source of truth for what is done, taken, and open.** Statuses: ✅ **done** (implemented, gates green, evidence noted) · 🔧 **taken** (an agent is actively on it — claim line says who/when) · ⬜ **open** (free to pick up).

### Pickup protocol (read before starting)

1. Pick an ⬜ open feature whose **Depends on** column is all ✅. Flip it to 🔧 with a claim note (`taken YYYY-MM-DD, <agent/session>`) **in the same commit/edit you start work**.
2. Build it per **§0 Binding conventions** and the feature's epic section below (data, API, acceptance criteria).
3. Definition of done — all of: `pnpm exec turbo run typecheck lint test build --filter=@eightfivetwo/web --force` green; `cd services/api && golangci-lint run` → 0 issues (ALL linters are enabled — do not weaken `.golangci.yml`); `go test -race ./...` green (testcontainers needs Docker); new behavior covered by tests; acceptance criteria of the epic met.
4. Flip to ✅ with one line of evidence (test names / e2e proof). Update the slice log above if you completed a whole slice.
5. Never invent client-provided content (§5 inputs) — stub with clearly-marked placeholders and note it.

### Board

| ID    | Feature                                                                                   | Status | Depends on | Spec      | Evidence / notes                                                                            |
| ----- | ----------------------------------------------------------------------------------------- | ------ | ---------- | --------- | ------------------------------------------------------------------------------------------- |
| INF-1 | Monorepo, CI (lint-all + testcontainers), Vercel/Render deploy configs                    | ✅     | —          | —         | `.github/workflows/ci.yml`, `render.yaml`, `apps/web/vercel.json`                           |
| INF-2 | Waitlist end-to-end (form → API → Mongo → Resend welcome)                                 | ✅     | INF-1      | pre-scope | `WaitlistForm.test.tsx`, `TestJoinWaitlist`, live Atlas e2e                                 |
| F1    | Passwordless auth: email links, sessions, light accounts, `ADMIN_EMAILS` admin role       | ✅     | INF-1      | §4.8      | `TestAuthFlow`, `TestAdminAccess`, `auth_test.go`, live e2e                                 |
| F4    | Store settings (deposit GHS 500 default, WhatsApp, visit location) + public `cloudName`   | ✅     | F1         | §05       | `TestSettings_Defaults`, `TestSettingsRepository`                                           |
| F5    | API route groups: public / authed / `/api/v1/admin` (waitlist list & upload sign = admin) | ✅     | F1         | —         | `TestListWaitlist_RequiresAdmin`, `TestSignUpload_AdminOnlyAndNotConfigured`                |
| F6    | Web router shell: react-router 7, guards, login/verify/account pages                      | ✅     | F1         | —         | `guards.test.tsx`, `LoginPage.test.tsx`                                                     |
| E1    | Catalog backend: collections/designs, size bands (pesewas), slugs, lifecycle, search      | ✅     | INF-1      | §05–06    | `catalog_test.go`, `TestCatalogRepositories`, `TestCatalogAdminCRUDAndPublicVisibility`     |
| E8a   | Admin shell + catalog UI (lists, editor, Cloudinary upload, bulk retire/restore, deletes) | ✅     | E1, F6     | §05       | 6 admin UI tests; browser smoke                                                             |
| E2    | Storefront pages: /store, /collections/:slug, /designs/:slug, /about, /contact            | ✅     | E1         | §03, §06  | 5 storefront tests; live smoke vs Atlas data. Copy = marked placeholders (client input §09) |
| E2a   | Social link unfurls (OG meta for IG/WhatsApp shares) — needs server-side meta injection   | ✅     | E2         | §06       | `apps/web/middleware.ts` Vercel Edge Middleware; `src/lib/og.ts` + `src/lib/og.test.ts` (18 tests); gates green. Manual: `curl -A "WhatsApp" <url>` |
| F2    | Paystack adapter: init transaction, webhook verify → auto-book, payment links             | ✅     | F1         | §4.5, §06 | `adapter/paystack`, `domain.PaymentProvider`, `payments` audit collection; `TestNewClient_*`, `TestInitTransaction`, `TestVerifyWebhook`, `TestCreatePaymentLink`; `make lint` + `go test -race ./...` green; hardened 2026-06-12 post-review: transition table, webhook amount verification + idempotency, payment-link via transaction/initialize, atomic slot claims |
| E5a   | Orders domain + state machine + standard path (checkout → pay → booked)                   | ✅     | F2, E1, E6 | Fig. 2    | `domain/order`, `service.Order`, `mongostore/order_repository`, `order_handlers`; tests `TestCreateStandardOrder_*`, `TestHandlePaymentWebhook_*`, `TestOrderRepository`, `Test*Order` handlers; `make lint` + `go test -race ./...` green; hardened 2026-06-12 post-review: transition table, webhook amount verification + idempotency, payment-link via transaction/initialize, atomic slot claims |
| E5b   | Custom path: request → dashboard → quote (design/price/timeline) → payment link → booked  | ✅     | E5a, E8b   | §4.5      | `CreateCustomRequest`, `POST /api/v1/orders/request`, frontend `DesignPage` custom submission; `DesignPage.test.tsx`, `TestCreateCustomRequest_*`; gates green. |
| E5c   | Manual mark-as-paid with note (off-platform payments)                                     | ✅     | E5a        | §4.5      | Backend: `service.Order.MarkPaidManually`, `POST /api/v1/admin/orders/:ref/mark-paid`; Frontend: manual-payment note field in `AdminOrdersPage` detail panel; Tests: `TestAdminMarkPaid_*` (httptest), `AdminOrdersPage.test.tsx`; gates green. |
| E3    | Measurement chooser: band / self-measure form / home visit / workplace + design changes   | ✅     | E5a        | §4.2–4.3  | `DesignPage.tsx` measurement chooser with price-hide rule, self-measure placeholder set, home-visit link, workplace note, design-change textarea; `DesignPage.test.tsx`; gates green. |
| E4    | Booking calendar: admin slots, GHS 500 deposit booking, call-to-reschedule, slot reopen   | ✅     | F2, F4     | §4.3, §05 | `domain.Slot`/`Visit`, `service.CalendarSlot`/`CalendarVisit`, Mongo repos, `slot_handlers`; `AdminSlotsPage`, `SlotsPage`; tests `TestSlotRepository`, `TestVisitRepository`, `TestBookSlot_*`, `AdminSlotsPage.test.tsx`; gates green; hardened 2026-06-12 post-review: transition table, webhook amount verification + idempotency, payment-link via transaction/initialize, atomic slot claims |
| E6    | Delivery: rates table (admin-editable), auto-add at checkout, pickup free, off-rate flag  | ✅     | F4         | §4.6      | `domain.DeliveryRate`, settings service/repo/handlers, `TestUpdateSettings_DuplicateArea`, `TestUpdateSettings_InvalidRate`, `TestSettings_Defaults`; checkout auto-adds rate; `make lint` + `go test -race ./...` green |
| E7    | Customer account area: orders list/detail with stages, visit card with call button        | ✅     | E5a        | §4.8, §06 | `AccountPage.test.tsx` (orders list + empty state), `OrderDetailPage.test.tsx` (stage mapping + visit card); `pnpm exec turbo run typecheck lint test build --filter=@eightfivetwo/web --force` green |
| F3    | Notifications: status emails ("order confirmed"/"in production"/"ready") + templates      | ✅     | E5a        | §4.7      | `domain.EmailSender.SendOrderStatusUpdate`; adapters `resend.go` + `noop.go`; `service.Order.notifyStatusChange` called from `HandlePaymentWebhook`, `UpdateOrderStatus`, `MarkPaidManually`; frontend `OrderDetailPage` status alert + email indicator. Tests: `TestHandlePaymentWebhook_BooksOrder`, `TestMarkPaidManually_BooksOrderAndSendsConfirmation`, `TestUpdateOrderStatus_SendsEmailForInProduction`, `TestUpdateOrderStatus_SendsEmailForReady`, `TestUpdateOrderStatus_DoesNotSendEmailForOtherStatuses`, `OrderDetailPage.test.tsx` status alert/timeframe tests. `make test` + `make lint` + web turbo pipeline green. |
| E8b   | Admin orders inbox: 3 types sorted, WhatsApp deep link, quote editor, send payment link   | ✅     | E5a        | §05       | `AdminOrdersPage.tsx` with standard/custom/visit buckets + detail panel; `features/orders/api.ts`/`hooks.ts`; backend `AdminListOrders`, `AdminGetOrder`, `AdminUpdateQuote`, `AdminSendPaymentLink`, `AdminUpdateOrderStatus`; Tests: `AdminOrdersPage.test.tsx`, `TestAdmin*Order` handler tests, service tests; `pnpm exec turbo run typecheck lint test build --filter=@eightfivetwo/web --force` + `make test`/`make lint` green. |
| E8c   | Admin settings UI (deposit, delivery rates, WhatsApp/visit info)                          | ✅     | F4, E6     | §05       | `apps/web/src/pages/admin/AdminSettingsPage.tsx` + `features/settings/` hooks/api; 4 Vitest tests; `pnpm exec turbo run typecheck lint test build --filter=@eightfivetwo/web --force` green |
| E8d   | Admin analytics (sign-ups, orders by status/type, revenue, "online customers")            | ✅     | E5a        | §05       | Backend: `domain.StoreAnalytics`, `domain.AnalyticsRepository`, `service.Analytics`, `mongostore.AnalyticsRepository`, `GET /api/v1/admin/analytics`; Frontend: `AdminAnalyticsPage`, `features/analytics/api.ts`/`hooks.ts`, nav + route; Tests: `TestAnalytics_GetStoreAnalytics`, `TestAdminGetAnalytics_*`, `TestAnalyticsRepository_GetStoreAnalytics`, `AdminAnalyticsPage.test.tsx`; `make test`/`make lint` + web turbo pipeline green. |
| E9    | Launch: client content load, domain + env cutover, replace landing with real Home, E2a    | ✅     | all above  | §06, C5   | Code: `HomePage.tsx`, `WaitlistPage.tsx`, `App.tsx` route swap, `HomePage.test.tsx` (2 tests); `pnpm exec turbo run typecheck lint test build --filter=@eightfivetwo/web --force` green. Content load, domain + env cutover remain client/deployment tasks. |

---

## Known bugs & issues (found during agent review, 2026-06-12)

These are pre-existing issues in Slices 1–3 that should be fixed in parallel with new feature work. They do not block new slices but degrade robustness, accessibility, or consistency.

### Backend

| # | Issue | Location | Severity | Fix |
| - | ----- | ---------- | -------- | --- |
| B-1 | CORS `AllowedMethods` is missing `DELETE` | `services/api/internal/transport/httpapi/router.go` | ✅ done | Added `http.MethodDelete`; `make lint` + `make test` green. |
| B-2 | `Auth.RequestLink` accepts empty `name` | `services/api/internal/service/auth.go` | ✅ done | Returns `ErrInvalidInput` for empty/whitespace name; added service + HTTP tests. Frontend `LoginPage` now collects name. |
| B-3 | Inconsistent admin update response shapes | `services/api/internal/transport/httpapi/catalog_handlers.go` | ✅ done | `AdminUpdateCollection` now returns the updated collection DTO; added `TestAdminUpdateCollection_ReturnsUpdatedResource`. |
| B-4 | `slugify` strips accented characters instead of transliterating | `services/api/internal/service/catalog.go` | ✅ done | Uses NFD decomposition + mark stripping so `"Été 2026"` → `"ete-2026"`; updated `TestSlugify_TransliteratesAccentsAndStripsSymbols`. |
| B-5 | `Settings` domain does not include delivery rates, despite `F4` wording | `services/api/internal/domain/settings.go` | ✅ done | Added `DeliveryRate` to `domain.Settings`, persisted in Mongo, exposed in public/admin APIs, editable in admin UI. |
| B-6 | `SignUpload` hardcodes Cloudinary folder | `services/api/internal/transport/httpapi/handlers.go` | ✅ done | Accepts optional `folder` body field, defaulting to `"eightfivetwo"`; added `TestSignUpload_UsesOptionalFolder`. |
| B-7 | No unit tests for email adapters or Cloudinary signer | `services/api/internal/adapter/media/*` | ✅ done | Added `TestCloudinarySigner_DeterministicSignature` (deterministic, fields, folder/timestamp sensitivity) without real credentials. |
| B-8 | `golangci-lint` deprecation warning for `wsl` | `services/api/.golangci.yml` | ✅ done | Disabled deprecated `wsl`, enabled `wsl_v5`; `make lint` → 0 issues. |

### Frontend

| # | Issue | Location | Severity | Fix |
| - | ----- | ---------- | -------- | --- |
| W-1 | `Content-Type` header merge order overwrites caller value | `apps/web/src/lib/api.ts` | ✅ done | Caller headers now win; added `api.test.ts` coverage. |
| W-2 | Raw `rgba` color outside `theme.ts` | `apps/web/src/theme.ts` | ✅ done | Added `noirAlpha50`/`noirAlpha70` tokens; `DesignPage` and `theme` components reference them. |
| W-3 | `CopyLinkButton` silently swallows clipboard errors | `apps/web/src/pages/DesignPage.tsx` | ✅ done | Shows transient "Could not copy link" error state; added test. |
| W-4 | `AccountPage` is not wrapped in `StorefrontLayout` | `apps/web/src/pages/AccountPage.tsx` | ✅ done | Wrapped in `StorefrontLayout` for consistent nav/footer. |
| W-5 | `AdminLayout` has no sign-out affordance | `apps/web/src/pages/admin/AdminLayout.tsx` | ✅ done | Added logout button using existing `logout` API + query invalidation. |
| W-6 | No skip-to-content link | `apps/web/src/components/StorefrontLayout.tsx` | ✅ done | Added focusable "Skip to content" link targeting `#main-content`. |
| W-7 | Landing-page nav items are decorative, not links | `apps/web/src/pages/LandingPage.tsx` | ✅ done | Labels already grouped with `aria-hidden`; added per-item `aria-hidden` for explicitness. |
| W-8 | Active nav state relies on color alone | `apps/web/src/components/StorefrontLayout.tsx` | ✅ done | Added underline + font-weight indicator to active nav link. |
| W-9 | Admin tables are not responsive | `apps/web/src/pages/admin/AdminDesignsPage.tsx`, `AdminCollectionsPage.tsx` | ✅ done | Wrapped tables in `overflowX: auto` container with `minWidth`. |
| W-10 | Production bundle is a single 651 kB chunk | `apps/web/vite build` | ⬜ open | Consider route-based code splitting to reduce initial load. |

---

## 0. Binding conventions

Agents MUST follow these; they are established in the codebase:

- **Backend**: new business concepts get a `domain` entity + port interface, a `service` use-case, an `adapter` implementation, and `httpapi` handlers. No infrastructure imports in `domain`/`service`.
- **API shape**: REST under `/api/v1`, envelope `{"data": ...}` / `{"error": {"code", "message"}}`. Domain errors map to status codes in handlers (`ErrInvalidInput`→422, `ErrDuplicate*`→409, `ErrNotFound`→404, `ErrForbidden`→403).
- **Money**: all amounts are GHS, stored as **integer pesewas** (`int64`), never floats. Display formatting client-side (`GH₵ 1,250.00`).
- **IDs/slugs**: Mongo `ObjectID` internally; URL-facing resources (collections, designs) also get unique human slugs for shareable links (§06 "Design Pages, Notes & Links").
- **Tests**: every repository gets a testcontainers (`mongo:8.0`) suite gated by `testing.Short()`; every service gets unit tests with fakes; every handler gets `httptest` coverage. Web features get Vitest + Testing Library tests.
- **Web**: feature folders under `src/features/<name>`; server state via TanStack Query; MUI components themed from `src/theme.ts` tokens only (no new raw hex); design language per the ui-ux-pro-max checklist in `.claude/skills/ui-ux-pro-max/`.
- **Email**: all transactional email goes through the `domain.EmailSender` port (Resend adapter; no-op fallback). Budget: 3,000 emails/month (Investment Summary).
- **Media**: design photos upload browser→Cloudinary using the existing signing endpoint pattern; the API stores only public IDs/URLs.
- **"Solid, not flashy"** (§07): proven building blocks, no over-engineering. Everything must serve the mandate — browse, customise, pay or request, track, fulfil. If a feature isn't traceable to an epic below, don't build it.

---

## 1. Cross-cutting foundations (build first)

### F1 — Authentication & accounts ("light by design", §4.8)

- Customers browse and start orders **without** an account; the account is created at the **last step of completing any order** (§4.1, §4.8) — that includes paymentless custom-request submission, since §4.5 requires the customer to later see the merchant's quote (design, price, timeline, status) "in their own account" before paying the link. Submitting a custom request therefore creates the account too.
- Customer dashboard scope is fixed: see orders, follow production stages, see a booked visit (with call button), get matching emails. **Nothing more** (§4.8 constraint).
- Admin (merchant) is a separate role with its own login guarding `/admin`.
- Recommended: passwordless email links via Resend (fits "light", avoids password support burden) + HTTP-only session cookie; admin gets password + same session mechanism. **Open decision — confirm with client.**
- New: `users` collection (role: `customer|admin`), session middleware in `httpapi`, auth context helper.

### F2 — Payments (Paystack)

- Provider is **Paystack** (Investment Summary: 1.95%/sale, no setup/monthly fees, next-working-day settlement). Mobile money + cards **including international** from day one (§06).
- Three payment surfaces, one provider/account (§4.5): (a) standard checkout, (b) fixed **GHS 500** home-visit deposit at booking, (c) merchant-generated payment links for custom balances.
- **Auto-booking rule** (§4.5, verbatim): "The moment payment clears, the order books itself." Implement via Paystack webhook → verify signature → mark order paid → transition to `booked`. Never trust client-side success callbacks alone.
- **Invariant**: no unpaid order may ever be in production (§4.5). Enforce in the order state machine.
- Manual **mark-as-paid with a short note** for off-platform payments, e.g. cash on collection (§4.5).
- The website never collects variable custom prices, design-change prices, or off-rate delivery fees — those are agreed and collected directly by the merchant; **the only fixed online amount besides garment prices is the GHS 500 deposit** (§08).
- New: `domain.PaymentProvider` port; `adapter/paystack` (init transaction, create payment link, verify webhook); `payments` collection (provider refs, amounts, status, raw event audit).

### F3 — Notifications (§4.7)

- Every order event notifies by **email + on-site status**. Customer-facing statuses, verbatim: **"order confirmed", "in production", "ready"** — always with clear timeframes.
- Templates needed: waitlist welcome (done), account/order confirmation, status change (×3), visit booked (+ deposit receipt), custom request received, payment link, order updated after WhatsApp agreement (§4.5).
- New: `notifications` service hooked into order state transitions; template catalog in `adapter/email`.

### F4 — Store settings (§05)

- Merchant-editable settings, dashboard-managed: **home-visit deposit amount (GHS 500 to start — configurable, §05/§09)**, delivery rates by area, WhatsApp number, visit location/contact details.
- New: `settings` collection (single doc), `GET/PUT /api/v1/admin/settings`.

### F5 — Authorization & routing split

- Public storefront API (no auth), customer API (session), admin API (`/api/v1/admin/*`, admin role). Add chi route groups + middleware now so every epic lands in the right group.

### F6 — Web app routing shell

- Add React Router: public `/`, `/about`, `/contact`, `/store`, `/collections/:slug`, `/designs/:slug`; customer `/account/*`; admin `/admin/*` (separate layout, guarded). Keep the current landing page as `/` until launch content replaces it.

---

## 2. Epics

### E1 — Catalog: collections, designs, size bands (§1.2, §05, §06)

**Requirements**

- Collections: themed, named, with a short note/description; released in batches ("roughly every two months or each quarter", ~10 designs each, §1.2). Designs belong to a collection.
- Designs: photos (multiple), name, details/note, **size bands** — e.g. 8/10/12/14/16/18/20/24 (§4.2) — each band with its **own full size chart** (bust, waist, etc.) and its **own set price in GHS** (§4.2, §06).
- Lifecycle (§05): live → **retired** (leaves shop, stays in dashboard, restorable) → optionally **deleted for good** (separate, deliberate action). Retire single design, several at once, or a whole collection. When a collection's fabric runs out its designs are retired (§1.2).
- Every design and collection has its own **shareable link** for Instagram/Facebook/WhatsApp (§06) → stable slugs + social meta tags (OG image from Cloudinary).
- Search across designs; browse all designs at once or by collection (§4.1, §06).

**Data** — `collections` {name, slug, theme/note, status: live|retired, createdAt}; `designs` {collectionId, name, slug, note, photos[{publicId, order}], sizeBands[{label, chart{measurements...}, priceGHS}], status: live|retired, createdAt, retiredAt}.

**API** — public: `GET /collections`, `GET /collections/{slug}`, `GET /designs?collection=&q=`, `GET /designs/{slug}`; admin: full CRUD + `POST /admin/designs/{id}/retire|restore`, bulk retire, `DELETE` (permanent, confirm-guarded), photo upload signing (exists).

**Acceptance** — retired designs 404 publicly but list in admin; duplicate slug 409; size-band prices render in GHS; share link unfurls with photo; testcontainers suite covers lifecycle + search.

### E2 — Storefront pages (§03, §06)

**Requirements**

- **Four pages** (§03, verbatim: "small, clear set of pages"): Home (welcome **and highlights** — featured collections/designs — leading into the store, §06), About Us (brand story), Contact Us, and the Store ("the heart of it"). Fully custom brand design — "not a stock template" (§07). Static page copy is set during the build, **not merchant-editable** (§05, §08).
- Store: browse all collections, open one collection, see all designs at once, search, open a design page (photos, details, size bands + charts, prices, the measurement chooser, request-to-change option).
- No account needed to browse (§4.8). Mobile-first; the brand sells via Instagram/Facebook today (§1.2).

**Acceptance** — Lighthouse mobile ≥ 90 perf/a11y on store pages; design page renders chart + price per band; ui-ux-pro-max pre-delivery checklist passes.

### E3 — Sizing, measurement & customisation (§4.2, §4.3, Figure 1)

**Requirements**

- On a design page the customer picks a **standard size band** (price shown) or declares nothing fits → chooses one of **three measurement paths** (Figure 1):
  1. **Measure yourself** — fill the measurement form on the site. No visit, no deposit. The standard measurement set is provided by the client (§09).
  2. **We come to you (home visit)** — pick a slot from the calendar, pay **GHS 500 deposit** which **counts toward the garment price** and compensates the trip if the customer changes their mind (§4.3). → E4.
  3. **Come to us (workplace)** — treated like walking in; **no booking, no deposit** (§4.3).
- **Design changes** (§4.2): keep the design exactly as shown, or request a change (e.g. sleeveless) via a **request-to-change option on every design page** (§4.5, §06). Carries no set price.
- **Pricing rule** (§4.4, verbatim): fully standard (listed band + design as shown) → price shown, pay to book. **"The moment anything is custom — the size, the design, or both — no price is shown"** and the order becomes a request.

**Data** — measurement profile embedded in the order (not a separate reusable profile — accounts are "light"); `customisation` {sizeMode: band|self|home_visit|workplace, bandLabel?, measurements?, designChange?: text}.

**Acceptance** — choosing any custom option hides price and routes to request flow; self-measure form validates the client-provided measurement set; deposit ≠ hard-coded (reads settings).

### E4 — Home-visit booking calendar (§4.3, §05)

**Requirements**

- Merchant opens days/times from the dashboard; **only those slots are shown**; a taken slot disappears for everyone else (§4.3). Initial availability provided by client (§09).
- Booking confirmed by the **GHS 500 deposit** through Paystack (§4.3, §06).
- Customers **cannot** cancel/reschedule on the site (deposit paid): they see a **button to call** Eight Two Five; merchant reschedules/cancels from the dashboard, which **reopens the slot** (§4.3, §05).
- After measurement (home or workplace), the order continues exactly like any custom order (§4.3).
- Customer account shows the booked visit **with a call button** (§06 "Customer Accounts").

**Data** — `slots` {start, end, status: open|booked|closed}; `visits` {orderId, slotId, depositPaymentId, status: booked|done|cancelled}.

**Acceptance** — double-booking race rejected (unique index on slotId, tested with testcontainers concurrency test); deposit webhook books the visit atomically with slot claim; reschedule reopens old slot.

### E5 — Orders & routing (Figure 2, §4.4, §4.5)

**Requirements**

- One gating question sets the path (Figure 2): standard size AND design as shown?
  - **Standard path**: price shown → pay online → booked automatically. "No back-and-forth needed."
  - **Custom path**: no price shown → measurement choice (E3/E4) → order lands in the **merchant dashboard** as a complete request (every measurement and choice attached) → merchant talks (WhatsApp), then sets **design, price, timeline** → sends **payment link** for balance → customer pays → **books itself**.
- Custom requests **never** go to WhatsApp as the record; the dashboard is "the single, accurate reference" (§4.5). A per-order **WhatsApp button** deep-links to that customer's chat (`wa.me/<phone>?text=<order ref>`). §05 says "open **any** order, tap WhatsApp" — so the customer phone number is **required on every order**, standard ones included.
- Three incoming order types in the dashboard, sorted: **standard bookings, custom requests, visit bookings** (§05).
- Status machine (internal → customer-facing): `pending_payment|requested → quoted → payment_link_sent → booked` ("order confirmed") `→ in_production → ready → fulfilled (delivered|picked_up) | cancelled`. Production takes "roughly two weeks, depending on current bookings" (§4.1) — timeline is set per order by the merchant.
- Merchant updates after the WhatsApp conversation are recorded on the order; customer is emailed and sees design/price/timeline/status in their account (§4.5).
- Manual mark-as-paid with note (§4.5). Invariant: an order can enter `in_production` only when paid.

**Data** — `orders` {ref (human-readable), customerId, designId snapshot (name/photo/price at order time), type: standard|custom_size|design_change|visit, customisation, quote {pricePesewas, timeline, notes}, delivery (E7), payments[], status, statusHistory[{status, at, by}], customerPhone}.

**Acceptance** — full standard path e2e (testcontainers + Paystack stub): pay → webhook → booked → emails; custom path: request → quote → link → webhook → booked; unpaid order rejected from `in_production` transition; all three order types sort correctly in admin inbox.

### E6 — Delivery & fulfilment (§4.6)

**Requirements**

- Chosen at booking; "the garment price is always what secures the booking" (§4.6).
- **Pickup**: free; collect when ready.
- **Dispatch with a set rate**: rate for the area auto-added and billed together at checkout.
- **Dispatch without a set rate**: customer books and pays garment price only; delivery is priced/arranged **off-platform** after the garment is ready (verbatim out-of-scope: "handled between Eight Two Five and the customer, outside the website").
- No live courier integration (§08) — record the choice, add rates where defined; merchant handles actual dispatch.
- Delivery rates by area are merchant-editable data (initial table from client, §09).

**Data** — `deliveryRates` {area, ratePesewas}; on order: `delivery {mode: pickup|dispatch, area?, ratePesewas?|null = arrange_directly}`.

**Acceptance** — checkout totals = garment + rate when rate exists; rate-less area books at garment price and flags "delivery arranged directly"; rates editable in admin.

### E7 — Customer account area (§4.8, §06)

**Requirements** (and nothing more — §4.8): list my orders; order detail with production stage + timeframe; booked visit with **call button**; account created at checkout's last step; email updates mirror everything.

**Acceptance** — a customer sees exactly their orders (authz tested); stages match the three customer statuses; visit card shows tel: link.

### E8 — Merchant dashboard (§05, §06)

**Requirements**

- Private admin area: "run the store without needing us day to day" — kept "clean and uncluttered on purpose".
- Catalog management ~as easy as posting to Instagram (§05): upload designs with photos/details/bands/charts/prices; collection + design notes; retire/restore/bulk/delete-for-good (E1 admin surface).
- Orders inbox: three types sorted, all measurements/choices attached; open order → WhatsApp button → set design/price/timeline/status → generate + send payment link (email or WhatsApp) → automatic customer notification (§05).
- Calendar management (E4 admin surface): open slots, reschedule/cancel (reopens slot).
- Settings (F4): deposit amount, delivery rates, etc.
- **Analytics** (§05): sign-ups, online customers, and store analytics "that genuinely help" — orders by status/type, revenue (booked GHS), waitlist/sign-up counts, design/collection views. Keep modest; charts follow ui-ux-pro-max chart rules. **Open decision: define "online customers" metric with client.**

**Acceptance** — non-admin gets 403 on every `/admin` route; an order can be taken from request → quoted → link → booked entirely inside the dashboard; analytics numbers reconcile with collection counts in tests.

### E9 — Launch, testing & handover (Investment component 5)

- Cross-device testing (phones + computers, per commitment), load client content (photos, bands, charts, prices, rates, calendar — all client-provided per §09), domain + production env setup (Vercel + Render + Atlas + Paystack live keys + Resend domain), go-live, handover docs. Replace the waitlist landing page with the real Home page; keep waitlist data for launch announcement.

---

## 3. Build order

Engineering order (dependencies), mapped to the Investment Summary components:

| #   | Slice                                                               | Epics                         | Investment component                             |
| --- | ------------------------------------------------------------------- | ----------------------------- | ------------------------------------------------ |
| 1   | Foundations: auth, settings, route groups, router shell             | F1, F4, F5, F6                | (enables all)                                    |
| 2   | Catalog write-side + admin shell                                    | E1, E8 (catalog part)         | C1 storefront GHS 4,500 / C4 dashboard GHS 3,500 |
| 3   | Storefront read-side: pages, browse, design page, share links       | E2                            | C1                                               |
| 4   | Standard checkout: Paystack, accounts-at-checkout, delivery, emails | F2, E5 (standard), E6, E7, F3 | C2 GHS 5,000                                     |
| 5   | Custom flow: measurement chooser, requests, quotes, payment links   | E3, E5 (custom)               | C2/C3                                            |
| 6   | Visits: calendar, slots, deposit, visit management                  | E4                            | C3 GHS 5,000                                     |
| 7   | Dashboard completion: inbox polish, analytics, settings UI          | E8                            | C4                                               |
| 8   | Launch                                                              | E9                            | C5 GHS 2,000                                     |

Each slice = PR-sized units with tests green (`turbo run lint typecheck test build` + `go test ./...` + `golangci-lint run`) before the next begins.

---

## 4. Out of scope — hard guardrails (§08, verbatim list)

Do NOT build in this phase: ready-to-wear/stock inventory; loyalty/rewards; downloadable iOS/Android app (the site must just work beautifully on phones); live courier integration with real-time tracking; collecting variable custom/design-change/off-rate-delivery prices through the website (only the fixed GHS 500 deposit and listed garment prices are taken online); merchant free-form page editing; content creation/photography/social management.

## 5. Client-provided inputs (§09 — block, don't invent)

Design photos, descriptions, collection names/themes; size bands + charts + prices; the standard measurement set; delivery rates by area; home-visit availability; deposit confirmation (GHS 500 start); brand colours/logo/guidelines. If missing when a slice needs them, stub with clearly-marked fixtures and flag.

## 6. Architecture decision: transport (REST vs gRPC vs GraphQL)

Decision (2026-06-12, after owner asked "use gRPC and GraphQL where necessary"): **plain REST/JSON stays; neither gRPC nor GraphQL is necessary anywhere in this system today.** Rationale, recorded so agents don't re-litigate:

- **gRPC** earns its keep for service-to-service calls. This platform is ONE Go service; there are no internal services to talk to. Browsers cannot speak native gRPC — adding it would mean a gRPC-web proxy (Envoy) in front of Render, breaking the simple Vercel `/api` rewrite and first-party cookie auth, for zero functional gain. **Trigger to revisit:** the API splits into multiple internal Go services (e.g., a separate payments or notifications worker) — then gRPC between them is the right call, keeping REST at the edge.
- **GraphQL** earns its keep with multiple clients whose data needs diverge, or third-party API consumers shaping their own queries. This system has exactly one first-party client whose views the REST endpoints are already shaped for (`/collections/{slug}` returns collection+designs precisely because the page needs both). GraphQL would add a schema/resolver layer, kill plain HTTP caching, and complicate the all-linters Go gate via codegen. **Trigger to revisit:** a native mobile app or partner integrations arrive (both currently OUT of scope, §08) — then a gqlgen gateway over the existing services layer slots in cleanly, because the hexagonal services are transport-agnostic by design.
- The scope's own build principle (§07 "Solid, not flashy — no over-engineering") and the GHS 20,000 budget bind this decision.

## 7. Open decisions (ask before the relevant slice)

1. Auth mechanism for "light" accounts (recommended: passwordless email link).
2. "Online customers" analytics definition (§05).
3. Paystack account setup ownership + webhook URL/domain timing.
4. Domain name + email sending domain (affects Resend DNS setup lead time; note: the Resend account is currently in sandbox mode and can only send to the owner's address until a domain is verified).
5. Whether the current waitlist stays post-launch (recommended: becomes "new collection" notify list).
