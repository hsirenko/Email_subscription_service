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
