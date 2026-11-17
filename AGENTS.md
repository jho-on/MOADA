# AGENTS.md

MOADA is a self-hosted, end-to-end encrypted cloud storage app. Stack: Go, Templ, HTMX, PostgreSQL, Garage (S3), REST API under `/api/v1/`. Module path: `github.com/jho-on/moada`.

## Non-negotiable constraints

- E2EE is client-side: encryption/decryption and key handling happen in the browser. The backend must never receive plaintext keys or file contents, and stores only encrypted bytes.
- File names and folder metadata are encrypted client-side too.
- Crypto stack: Argon2id, XChaCha20-Poly1305, `crypto_secretstream_xchacha20poly1305`; per-file DEK, user Master Key, password-derived KEK, independent recovery key. Never reuse the password authentication hash as a key.
- Password changes only re-wrap the Master Key; never re-encrypt stored files.
- Each share uses an independent key. The share secret belongs only in the URL fragment (`#`) and must never reach the backend or logs.
- Implement the MVP scope only: do not add V1/V2 features. Changes to cryptography, data formats or public API contracts require explicit approval.
- Do not implement cryptographic primitives from scratch; use established libraries.

## Commands

```sh
make env             # create .env from .env.example
make dev-up          # PostgreSQL + Garage (Docker Compose, waits for health)
make migrate-up      # apply embedded Goose migrations
make run             # generate Templ, run web app on :8080
make check           # go fmt + templ fmt + go vet + go test
make build           # bin/moada and bin/moada-migrate
make test-integration
```

- Single test: `go test ./internal/httpapi -run TestName -v`.
- CI runs `go tool templ generate`, requires a clean `git diff`, then `go vet ./...` and `go test ./... -race -count=1`.
- `.env` is auto-loaded and exported by the Makefile. Do not commit it.

## Layout and wiring

- `cmd/moada`: entrypoint; loads config, opens the pgx pool, creates the S3 client, optionally creates the bucket, serves HTTP, shuts down on SIGINT/SIGTERM.
- `cmd/migrate`: `up`, `down` (one step), `status`, `version`; reads only `MOADA_DATABASE_URL`.
- `internal/httpapi`: routes/middleware; `internal/web/templates`: Templ; `internal/web/static`: embedded CSS/JS/HTMX.
- `internal/database`: pool + Goose; `internal/storage`: Garage S3 only; `internal/health`: readiness checks.
- `migrations/embed.go` embeds `migrations/*.sql`. Add new migrations as sequential Goose files; the app does not auto-migrate.
- Database port in dev is **5434** (not 5432) to avoid local conflicts. Garage S3 is `localhost:3900`.

## Repo-specific rules

- HTMX is vendored (`internal/web/static/vendor`). CSP is strict (`script-src 'self'`); never add CDN or third-party scripts.
- After editing `.templ`, run `go tool templ generate`; generated `*_templ.go` files are committed. `make build`/`make run` regenerate them, `make check` does not.
- Keep handlers thin; business logic belongs in `internal` packages, not HTTP handlers.
- The home page is intentionally plain HTML with white background and black text. Do not add styling or redesign unless explicitly requested.
- Do not leave development-status/debug text (phase labels, “MVP em construção”, TODOs in UI) in templates, README or comments.
- Integration tests (`internal/database`, `internal/storage`) are skipped unless `make test-integration` is used; they need `make dev-up` and use the dev database.
