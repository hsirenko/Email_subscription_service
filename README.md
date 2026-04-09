## Email Subscription Service (GitHub Releases)

API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

The public API contract is defined in `swagger.yaml` and must not be changed.

### Run (Docker)
Start Postgres + app (and optional MailHog):

```bash
docker compose up --build
# with MailHog UI + SMTP sink:
docker compose --profile mailhog up --build
```

App:
- API: `http://localhost:8080`
- Health: `http://localhost:8080/health`

MailHog (when enabled):
- UI: `http://localhost:8025`
- SMTP: `mailhog:1025` (from inside Docker network)

### API (Swagger)
Base path: `/api`

- `POST /api/subscribe` (form-data: `email`, `repo`; optional JSON body supported)
- `GET /api/confirm/{token}`
- `GET /api/unsubscribe/{token}`
- `GET /api/subscriptions?email={email}`

### Environment variables

#### HTTP
- **`PORT`**: HTTP port (default: `8080`)
- **`PUBLIC_URL`**: base URL used for links in emails. Example: `http://localhost:8080`

#### Database
- **`DATABASE_URL`**: Postgres connection string
  - Docker: `postgres://postgres:postgres@postgres:5432/releases?sslmode=disable`
  - Host: `postgres://<user>@localhost:5432/releases?sslmode=disable`

Migrations run automatically on startup via `golang-migrate`.

#### GitHub
- **`GITHUB_TOKEN`** (optional): increases rate limit from 60 req/hour to 5000 req/hour.

#### Email
- **`EMAIL_DRIVER`**: `log` or `smtp` (default: `log`)
- **`SMTP_HOST`**: SMTP host (default: `mailhog`)
- **`SMTP_PORT`**: SMTP port (default: `1025`)
- **`SMTP_FROM`**: from address (default: `noreply@local`)
- **`SMTP_USERNAME`** / **`SMTP_PASSWORD`**: optional auth (MailHog uses none)

#### Scanner
- **`SCAN_INTERVAL`**: how often to scan GitHub for new releases. Example: `10s`, `2m` (default: `2m`).

#### CORS (browser UI on another origin, e.g. Vercel)
- **`CORS_ALLOWED_ORIGINS`**: comma-separated list of allowed `Origin` values (e.g. `https://my-app.vercel.app,http://localhost:5173`). If set (non-empty), CORS middleware is enabled with **`GET`**, **`POST`**, **`OPTIONS`**.
- **`CORS_ALLOW_VERCEL_SUBDOMAINS`**: set to `1` or `true` to allow any origin whose host ends with **`.vercel.app`** (preview + production URLs), in addition to exact matches in **`CORS_ALLOWED_ORIGINS`**.

### Deploy: Fly.io (API) + Vercel (web UI)

The **API** runs on [Fly.io](https://fly.io/) using the repo **`Dockerfile`**. The **subscribe UI** is a static [Vite](https://vitejs.dev/) app in **`web/`**, deployed to [Vercel](https://vercel.com/) with **`VITE_API_URL`** pointing at your Fly app.

#### 1) Postgres on Fly

```bash
fly postgres create --name email-subscription-db --region ams
fly postgres attach email-subscription-db -a <your-api-app-name>
```

(`attach` sets **`DATABASE_URL`** on the app.)

#### 2) API app

1. Install the [Fly CLI](https://fly.io/docs/hands-on/install-flyctl/) and log in.
2. Edit **`fly.toml`**: set **`app`** to a unique name (replace `email-subscription-api`).
3. From the repo root:

```bash
fly launch --no-deploy   # if first time; reuse existing fly.toml when prompted
fly secrets set \
  PUBLIC_URL="https://<your-api-app-name>.fly.dev" \
  EMAIL_DRIVER="smtp" \
  SMTP_HOST="smtp.gmail.com" \
  SMTP_PORT="587" \
  SMTP_FROM="you@gmail.com" \
  SMTP_USERNAME="you@gmail.com" \
  SMTP_PASSWORD="<app-password>" \
  GITHUB_TOKEN="<optional-github-pat>" \
  CORS_ALLOW_VERCEL_SUBDOMAINS="true"
```

Add **`CORS_ALLOWED_ORIGINS`** if you use a **custom domain** on Vercel (not `*.vercel.app`), e.g. `https://releases.example.com`.

4. Deploy:

```bash
fly deploy
```

5. Smoke test: `https://<your-app>.fly.dev/health`

**Important:** **`PUBLIC_URL`** must be the **public HTTPS URL** of the API so confirmation and unsubscribe links in emails work.

#### 3) Web UI on Vercel

**Why a root `vercel.json` exists:** If the Vercel **Root Directory** is left as the repository root (`.`), Vercel may detect **Go** (`go.mod`) and try to run a **serverless** runtime—then `/` crashes with **`FUNCTION_INVOCATION_FAILED`**. The repo root **`vercel.json`** forces a static build: `cd web && npm run build` and **`outputDirectory`: `web/dist`**.

You can deploy in either mode:

1. **Recommended:** Vercel → **Settings → General → Root Directory** = **`web`**. Then **`web/vercel.json`** applies (`framework: vite`, output **`dist`**).
2. **Or** leave Root Directory **empty** / **`.`** and rely on the **root `vercel.json`** (builds from **`web/`** automatically).

Under **Environment Variables**, add **`VITE_API_URL`** = `https://<your-api-app-name>.fly.dev` (no trailing slash). Apply to Production (and Preview if you want preview deploys to hit the same API). Redeploy after changing it.

**If you still see `500` / Serverless Function crashed:** In **Settings → General**, clear any **Framework Override** that isn’t **Vite** / static, remove custom **Rewrites** to serverless routes, and confirm **Output Directory** matches **`dist`** (when root is `web`) or leave it unset when using the root **`vercel.json`**.

3. Deploy. Local dev: copy **`web/.env.example`** to **`web/.env`** and run:

```bash
cd web && npm install && npm run dev
```

Open **http://localhost:5173**. The browser will call a **different origin** than the API (e.g. `http://localhost:8080`), so enable CORS on the API: set **`CORS_ALLOWED_ORIGINS=http://localhost:5173`** (e.g. in `docker-compose.yml` for the `app` service). Fly production can use **`CORS_ALLOW_VERCEL_SUBDOMAINS=true`** for `*.vercel.app` plus any custom domain in **`CORS_ALLOWED_ORIGINS`**.

### Optional: real inbox (SMTP, e.g. Gmail)

By default the app uses **`EMAIL_DRIVER=log`** (URLs printed in container logs) or MailHog when you use the MailHog compose profile. To receive confirmation and release mail in a real mailbox:

1. Set **`EMAIL_DRIVER=smtp`** and the SMTP variables under **Email** above (`SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USERNAME`, `SMTP_PASSWORD`).
2. **Gmail (personal):** enable 2-Step Verification, then create an [App Password](https://support.google.com/accounts/answer/185833) and use that 16-character value as **`SMTP_PASSWORD`**. Use **`smtp.gmail.com`**, port **`587`**, and set **`SMTP_USERNAME`** and **`SMTP_FROM`** to the same Gmail address.
3. **`PUBLIC_URL`** must match a URL you can open from the device where you read email (e.g. `http://localhost:8080` only works on the same machine as Docker). For links from a phone, use a tunnel or a deployed base URL.

**Secrets:** do not commit real passwords. Use a **`.env` file** (keep it out of git—add `.env` to `.gitignore`) and wire it with Compose **`env_file`**, or export variables in your shell before `docker compose up`. Rotate any credential that was ever committed to git.

If **`POST /api/subscribe`** returns **500** after switching to SMTP, the subscription row may still be **`pending`**; fix SMTP, remove that row (or wait and use another email/repo), then subscribe again. A duplicate active/pending subscription returns **409 Conflict**.

### Testing
Run unit tests:

```bash
go test ./...
```

Optional MailHog integration test:
```bash
docker compose -f docker-compose.yml -f docker-compose.mailhog.yml --profile mailhog up -d --build
INTEGRATION_MAILHOG=1 go test ./internal/integrations -run TestSubscribeFlow_SendsEmailToMailhog -v -count=1
```

### Verify
```bash
go test ./...
curl -i -X POST -d "email=new@example.com&repo=cli/cli" http://localhost:8080/api/subscribe
```

### Design notes / limitations
- Monolith process: API + scanner/notifier run in one service.
- `last_seen_tag` is stored per repo in `repo_state`, not per subscription.
- Deduping is handled by `notification_log (subscription_id, release_tag)` unique.
- Notifications support retries using `notification_log.status` (`pending/sent/failed`).
- GitHub client retries on 429 using `Retry-After` and `X-RateLimit-Reset` when available.
