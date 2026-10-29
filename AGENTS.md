# AGENTS.md

This project is a **starting template for new business systems**: it retains only correct, clearly structured final forms, and does not keep compatibility layers, deprecated aliases, dual code paths, or wrappers that serve an outdated structure. When changing the architecture, change it directly to the target state; do not "add compatibility first and migrate later."

## Technology Stack and Entry Points

- Go 1.26 + `net/http` + `log/slog` + `modernc.org/sqlite` (pure Go driver). The module path is intentionally `main`, and internal packages uniformly use `main/internal/...`; do not change it to a domain-based path.
- Vue 3 + TypeScript + Vite 8 + Tailwind 4 + shadcn-vue (reka-ui) + Pinia + vue-router + vue-i18n + zod + ky.
- Production is a single binary: `frontend_embed.go` embeds the frontend via `//go:embed web/dist`; `main.go` is responsible only for dependency assembly and process lifecycle.

## Commands

```powershell
# Frontend
pnpm lint
pnpm typecheck
pnpm format
pnpm build

# Backend
go test ./...
golangci-lint run
go build .
```

- Compiling the root package includes `//go:embed web/dist`, while `web/dist` is gitignored. **A freshly cloned repository must run `cd web && pnpm build` first**; otherwise `go build` / `go test ./...` fails immediately due to the embed directive.
- `pnpm dev` starts Vite only, with `/api` proxied to `http://localhost:8080`. To exercise real API endpoints, run `go run .` in a separate terminal.
- There is no frontend test framework. Locales and error codes are validated by Go tests (`internal/httpapi/locales_test.go`, `error_codes_test.go`); therefore frontend verification consists of `lint + typecheck + build`, and backend verification consists of `go test ./...`.
- The repository has no CI or pre-commit hooks. Before considering work complete, the following must be run manually in order: `pnpm lint` → `pnpm typecheck` → `pnpm build` → `go test ./...` (build runs first to ensure the artifacts required by the embed directive are present).

## Layering and Boundaries

### Backend

- `main.go` only assembles dependencies and starts or gracefully shuts down the server; business logic never belongs there.
- `internal/<domain>/service.go` defines both the `Service` and the `Repository` interface. The Service is responsible for validation, orchestration, and domain errors; **transaction boundaries belong to the Repository implementation** (`internal/database/sqlite`). Do not assemble transactions across repositories inside a service.
- `internal/database/runtime.go` is only a composition-root handle that distributes repositories to domain services; domain services receive only their own `Repository` and must not be injected with the entire `Runtime`.
- `internal/httpapi` is strictly the transport layer: `decodeJSON` → call the service → `writeSuccessJSON` / `writeError`. State checks, authorization rules, and business branches must not be written into handlers.
- New routes are registered centrally in `internal/httpapi/server.go`, with the middleware order fixed as `RequireSetupCompleted → RequireAuth → RequireCapability`; `http.ServeMux` uses the `"METHOD /path"` pattern.
- The integration path for a new domain is: `internal/<domain>` → `internal/database/sqlite/<domain>.go` → `database.Runtime` field → explicit construction in `main.go` → `HandlerOptions` → route.

### Frontend

- The data flow is fixed: `views/` page composition → `composables/<domain>/` state and asynchronous flows → `lib/api/<domain>.ts` (ky requests + zod parsing) → backend. `utils/<domain>/` contains pure functions only.
- `lib/api/client.ts` is the only ky entry point: use `baseApi` for ordinary requests and `authApi` for requests that require an access token. Single-flight 401 refresh and session-invalidation handling live here; business layers must not implement retries or token logic.
- Every API response must be validated with zod (`successEnvelopeSchema`); ESLint forbids `as` type assertions (except `as const`). Values from external inputs such as `localStorage` must be funneled through `parseExternalValue*` in `lib/external-input.ts`.
- Session state belongs only in `stores/auth.ts`; page-local state stays in the corresponding composable. Do not put it into a global store for convenience.
- `components/ui/**` consists of shadcn-vue CLI output, while project-owned components belong in `components/<domain>/`; generate new base components with the CLI rather than hand-writing a parallel set of variants.

## Contracts That Must Change Together

- Response envelope: success is `{success:true,data}`, failure is `{success:false,error:{code,message}}`.
- Adding or removing an error code requires changes in four places simultaneously: `internal/httpapi/error_codes.go`, `apiError.*` in `web/src/locales/zh-CN.json`, `en-US.json`, and `web/src/lib/api/error-messages.ts` (`satisfies` enforces this). A missing change fails `error_codes_test.go` or `pnpm typecheck`.
- i18n: `zh-CN.json` is the sole source of `MessageSchema`; all user-visible copy must be added to both locale files. ESLint forbids runtime key construction—keys must either be written as `t('literal')` or narrowed through an exhaustive `switch`; otherwise the key is treated as unused.
- Capabilities: Go-side constants plus `roleCapabilities` (`internal/authorization/service.go`), synchronized with the frontend `CAPABILITY` map (`lib/auth/roles.ts`) and the zod enum (`lib/api/auth.ts`). **The server-side `RequireCapability` is the actual authorization boundary**; the frontend `requiredCapabilities` is responsible only for route interception and hiding entry points.
- Authentication: the access token is held in memory only (`lib/auth/token.ts`), and the refresh token is stored in an `HttpOnly` cookie; never write the access token to `localStorage` or `sessionStorage`.
- Database migrations: `internal/database/sqlite/migrations/NNNN_name.sql` are append-only. Applied migrations carry a SHA-256 checksum; changing a single byte prevents startup.
- Tests: HTTP-related tests use `newTestContext` (`internal/httpapi/server_test.go`) to set up a real SQLite database plus `httptest`; do not introduce mock repositories or a second database.

## Known Pitfalls

- The startup order in `main.ts` must not change: mock → pinia → `authStore.bindRouter()` → `authStore.initialize()` → `app.use(router)`. Mounting the router earlier causes the initial route guard to read session state before it is ready.
- `VITE_API_MOCKING=true` takes effect only in dev and directly injects a mock owner session (`mocks/dev-session.ts`), bypassing the login and guest flows; unmatched API requests `bypass` to the real backend, so the Go server must still be running.
- The SSE log stream depends on `http.ResponseController` and `statusRecorder.Unwrap()`. Any new `ResponseWriter` wrapper must implement `Unwrap`; otherwise streaming endpoints fail silently.
- `.data/` stores both `data.db` and the auto-generated `.jwt_secret`; deleting the directory invalidates all sessions. `.env`, `.data/`, `web/dist`, and `*.exe` are not tracked.
- For non-obvious pitfalls (framework behavior, protocol boundaries, concurrency/timeouts, third-party quirks), write a one-line comment explaining **why** next to the code that triggers it, including when it can be removed; do not pile this knowledge into this file. Do not delete existing pitfall comments.

## Definition of Done

- The minimum closed loop for a new business capability: domain service + repository → SQLite implementation → handler + route → error codes/bilingual locales → frontend api + composable + view + route meta (including capability).
- Non-trivial logic (branches, parsing, authorization, money, concurrency) requires a small test; purely presentational or copy changes do not.
- Do not add compatibility branches, `deprecated` aliases, dual-write logic, or configuration for "possible future use"; delete code with no callers outright.
