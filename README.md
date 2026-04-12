# Email subscription service (GitHub releases)

API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

The public API contract is defined in `swagger.yaml` and must not be changed.

---

## How it works

1. **Subscribe** — `POST /api/subscribe` validates email and repo shape, checks the repo exists on GitHub, inserts a **`pending`** row (confirm + unsubscribe tokens), and sends a confirmation email (SMTP or log).
2. **Confirm** — `GET /api/confirm/{token}` activates the subscription.
3. **Notify** — A **scanner** loop (same process as the API) groups active subscriptions by repo, calls GitHub’s **latest release** API, compares to stored **`repo_state`**, and emails subscribers when the tag changes. Sends use the same SMTP/log driver as subscribe.
4. **Dedupe** — Per `(subscription_id, release_tag)` in **`notification_log`**; retries use row status.

**Limits (by design)** — One OS process runs **HTTP + scanner**. `repo_state` is **per repo**, not per subscriber. Private GitHub repos are not supported unless you add per-user GitHub auth later.

---

## Architecture

| Layer | Role |
|--------|------|
| **`cmd/api`** | Loads config, Postgres + migrations, GitHub client, mailer; serves HTTP and starts the scanner goroutine. |
| **`internal/httpapi`** | Chi router: `/health`, `/api/*`. CORS optional (browser UI on another origin). |
| **`internal/service`** | Subscribe / confirm / unsubscribe orchestration. |
| **`internal/store/postgres`** | Subscriptions, repo state, notification log. |
| **`internal/integrations/github`** | REST client with retries (429 / 5xx). |
| **`internal/integrations/email`** | `log` (stdout) or `smtp` sender. |
| **`internal/jobs`** | Release scanner + retry of pending notifications. |
| **`web/`** | Static **Vite** UI: form posts JSON to `VITE_API_URL/api/subscribe`. |

---

## Dependencies

**Backend (Go)** — toolchain in `go.mod` (direct modules):

| Module | Purpose |
|--------|---------|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/go-chi/cors` | CORS for cross-origin browser calls |
| `github.com/golang-migrate/migrate/v4` | SQL migrations on startup |
| `github.com/jackc/pgx/v5` | Postgres driver (`database/sql`) |
| `github.com/jackc/pgerrcode` | Map unique violations to “already subscribed” |

**Runtime** — Postgres 16+ (migrations under `migrations/`).

**Frontend** — Node 18+ for `web/`: **Vite** only (`package.json` devDependencies).

**External** — [GitHub REST API](https://docs.github.com/en/rest), optional SMTP (e.g. Gmail).

---

## Repository layout

```
cmd/api/              # entrypoint
internal/config/    # env-based configuration
internal/domain/    # validation, repo parsing, errors
internal/httpapi/   # router + handlers
internal/service/   # subscription use-cases
internal/store/     # persistence (postgres)
internal/integrations/
internal/jobs/      # release scanner
migrations/         # golang-migrate SQL
web/                # Vite static UI
swagger.yaml        # API contract
Dockerfile          # production-style API image
docker-compose.yml  # local Postgres + app (+ optional MailHog profile)
fly.toml            # Fly.io app config
vercel.json         # static build when Vercel root is repo root
web/vercel.json     # Vite config when Vercel root is web/
```

---

## Run locally

### API from source (`go run`) + Postgres

Typical dev flow: run **Postgres** in Docker, run the **API on the host** with Go (no `npm` required).

1. **Database**

   ```bash
   docker compose up -d postgres
   ```

2. **API** (from repo root; defaults in `internal/config` expect Postgres on `localhost:5432`)

   ```bash
   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/releases?sslmode=disable'
   export PUBLIC_URL='http://localhost:8080'
   go run ./cmd/api
   ```

   This repo has a single `main` package, so **`go run ./...` does the same thing** here.

3. **Smoke check** — `http://localhost:8080/health`, then e.g.:

   ```bash
   curl -i -X POST -H "Content-Type: application/json" \
     -d '{"email":"you@example.com","repo":"cli/cli"}' \
     http://localhost:8080/api/subscribe
   ```

### MailHog (SMTP + inbox UI on :8025)

To exercise the **SMTP** path and read messages in a browser, add MailHog (after Postgres is up):

```bash
docker compose --profile mailhog up -d mailhog
```

- **Web UI:** `http://localhost:8025` (inbox / MIME view)
- **SMTP from the host-run API:** set e.g.  
  `EMAIL_DRIVER=smtp` `SMTP_HOST=localhost` `SMTP_PORT=1025` `SMTP_FROM=noreply@local`  
  (leave `SMTP_USERNAME` / `SMTP_PASSWORD` empty). Restart `go run`.

Inside **Docker Compose’s `app` service**, SMTP is `mailhog:1025`, not `localhost`.

### Full stack in Docker (optional)

```bash
docker compose up --build
```

- API: `http://localhost:8080`, health: `/health`
- Default **`EMAIL_DRIVER=log`** (confirmation URLs in the `app` container logs).

With MailHog profile (starts MailHog alongside the bundled **`app`** image — not `go run` on the host):

```bash
docker compose --profile mailhog up --build
```

### Web UI with Vite (optional)

Only if you want the static page from **`web/`** on a dev server (`http://localhost:5173`):

```bash
cp web/.env.example web/.env   # VITE_API_URL=http://localhost:8080
cd web && npm install && npm run dev
```

Then enable CORS on the API, e.g. `CORS_ALLOWED_ORIGINS=http://localhost:5173`.

### Tests

```bash
go test ./...
```

Optional MailHog integration test (stack up first):

```bash
docker compose -f docker-compose.yml -f docker-compose.mailhog.yml up -d --build
INTEGRATION_MAILHOG=1 go test ./internal/integrations -run TestSubscribeFlow_SendsEmailToMailhog -v -count=1
```

---

## Configuration (environment variables)

| Variable | Purpose | Default / notes |
|----------|---------|-----------------|
| `PORT` | Listen port | `8080` |
| `PUBLIC_URL` | Base URL for confirm/unsubscribe links in email | e.g. `http://localhost:8080`; production must be **public HTTPS** |
| `DATABASE_URL` | Postgres DSN | Compose sets `postgres` hostname |
| `GITHUB_TOKEN` | Bearer token for GitHub API | Optional for public repos; **invalid token → 401** and subscribe fails. Omit or use a valid PAT. Fine-grained PATs must allow **all repos you want to support**, not one repo only. |
| `EMAIL_DRIVER` | `log` or `smtp` | `log` |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD` | SMTP | Defaults suit MailHog/local |
| `SCAN_INTERVAL` | Scanner ticker | e.g. `2m` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated `Origin` values | If non-empty, enables CORS for `GET`/`POST`/`OPTIONS` |
| `CORS_ALLOW_VERCEL_SUBDOMAINS` | `1` / `true` | Allows any `https://*.vercel.app` origin (plus `CORS_ALLOWED_ORIGINS` for custom domains) |

Do not commit secrets; use Fly secrets, Vercel env UI, or local `.env` (gitignored).

---

## Deploy

### API on Fly.io

1. Create Postgres (or use existing): `fly postgres create`, then `fly postgres attach <db-app> -a <api-app>`.
2. Set `app` in **`fly.toml`** to your API app name.
3. Set secrets (example):

   ```bash
   fly secrets set \
     PUBLIC_URL="https://<api-app>.fly.dev" \
     GITHUB_TOKEN="<valid-pat-or-omit>" \
     EMAIL_DRIVER="smtp" \
     SMTP_HOST="smtp.gmail.com" SMTP_PORT="587" \
     SMTP_FROM="you@gmail.com" SMTP_USERNAME="you@gmail.com" \
     SMTP_PASSWORD="<app-password>" \
     CORS_ALLOW_VERCEL_SUBDOMAINS="true" \
     -a <api-app>
   ```

   Add `CORS_ALLOWED_ORIGINS` for a **non-`*.vercel.app`** frontend domain.

4. Deploy from machine with your source: **`fly deploy`** (builds local tree; no Git push required unless you use CI).

5. Check **`https://<api-app>.fly.dev/health`**. There is **no** route on **`/`** (404 is normal).

### Web on Vercel

1. **Environment:** `VITE_API_URL` = `https://<api-app>.fly.dev` (no trailing slash), scoped to **Production** (and Preview if needed).
2. **Root directory:** either **`web`** (uses `web/vercel.json`) or repo **root** (uses root `vercel.json` so Vercel does not treat the repo as a Go serverless project).
3. **Redeploy** after env changes; use **“Clear build cache and redeploy”** if the UI still shows “configure VITE_API_URL”.

Vercel builds from **Git** by default: push (or merge) so production gets code changes; env-only updates still need a redeploy.

---

## Operations & troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| **`github: status 401`** in logs (`api error -> 500`) | Bad/expired `GITHUB_TOKEN`, or fine-grained PAT not allowed for that repo. Fix secret or `fly secrets unset GITHUB_TOKEN` for public-only + lower rate limits. |
| **`POST /api/subscribe` → 500** after SMTP misconfiguration | Row may be **`pending`**; fix SMTP, remove row or use another email/repo; duplicate returns **409**. |
| Fly logs missing handler lines | Stdlib `log` and Chi both use **stdout** in current code so errors sit next to access logs. Tail: `fly logs -a <app>`. |

Gmail: 2FA + [App Password](https://support.google.com/accounts/answer/185833), `smtp.gmail.com:587`, same address for `SMTP_FROM` and `SMTP_USERNAME`.

---

## Developing further

| Goal | Where to look |
|------|----------------|
| API behavior / status codes | `internal/httpapi/handlers/subscription_handlers.go` |
| Subscribe / confirm rules | `internal/service/subscription.go` |
| GitHub calls / retries | `internal/integrations/github/http_client.go` |
| DB schema & queries | `migrations/`, `internal/store/postgres/` |
| Scanner cadence & notifications | `internal/jobs/scanner.go` |
| CORS rules | `internal/httpapi/router.go` |
| Contract for external clients | **`swagger.yaml`** |

Keep **`swagger.yaml`** in sync if you change paths, methods, or response semantics. Use the **`curl`** subscribe example under **Run locally** for a quick manual check.
