# AGENTS.md

This project is a **starting template for new business systems**: it retains only correct, clearly structured final forms, and does not keep compatibility layers, deprecated aliases, dual code paths, or wrappers that serve an outdated structure. When changing the architecture, change it directly to the target state; do not "add compatibility first and migrate later."

## Technology Stack and Entry Points

- Go 1.26 + `net/http` + `log/slog` + `github.com/uptrace/bun` on top of `modernc.org/sqlite` (pure Go, no CGO). The module path is intentionally `main`, and internal packages uniformly use `main/internal/...`; do not change it to a domain-based path.
- Vue 3 + TypeScript + Vite 8 + Tailwind 4 + shadcn-vue (reka-ui) + Pinia + vue-router + vue-i18n + zod + ky.
- Production is a single binary: `frontend_embed.go` embeds the frontend via `//go:embed web/dist`; `main.go` is responsible only for dependency assembly and process lifecycle.

## Commands

```powershell
# Frontend
pnpm lint
pnpm typecheck
pnpm test
pnpm format
pnpm build

# Backend
go generate ./internal/contractgen     # regenerate both frontend contract modules
go test ./...
golangci-lint run
go build .
```

- Adding or renaming a capability, role, user status or managed user action requires `go generate ./internal/contractgen` before the checks above; `internal/contractgen` compares both committed modules byte for byte in `go test ./...`. The generated files are committed on purpose so `cd web && pnpm build` never needs a Go toolchain.

- Compiling the root package includes `//go:embed web/dist`, while `web/dist` is gitignored. **A freshly cloned repository must run `cd web && pnpm build` first**; otherwise `go build` / `go test ./...` fails immediately due to the embed directive.
- `pnpm dev` starts Vite only, with `/api` proxied to `http://localhost:8080`. To exercise real API endpoints, run `go run .` in a separate terminal.
- Frontend tests run on Vitest (`pnpm test`, `environment: 'jsdom'`) and are **limited to pure logic and state machines** — currently `lib/polling`, pure helpers under `lib/api`, and `utils/`. There is no component-testing setup: views and composables are still verified by `lint + typecheck + build`, and locales/error codes by Go tests (`internal/httpapi/locales_test.go`, `error_codes_test.go`).
- The repository has no CI or pre-commit hooks. Before considering work complete, the following must be run manually in order: `pnpm lint` → `pnpm typecheck` → `pnpm test` → `pnpm build` → `go test ./...` (build runs first to ensure the artifacts required by the embed directive are present).

## Layering and Boundaries

### Backend

- `main.go` only assembles dependencies and starts or gracefully shuts down the server; business logic never belongs there.
- `internal/<domain>/service.go` defines both the `Service` and the `Repository` interface. The Service is responsible for validation, orchestration, and domain errors; **transaction boundaries belong to the Repository implementation** (`internal/database/store`, via `(*bun.DB).RunInTx`). Do not assemble transactions across repositories inside a service.
- `internal/database` has exactly three parts: `dialect` (the `Driver` interface plus one adapter per database: DSN handling, pragmas/pool, SQLite error decoding, migration filesystem), `migrate` (the checksum ledger runner), and `store` (bun row models plus one file per domain repository). `runtime.go` is only the composition root: it selects a dialect, opens it, runs its migrations and exposes a single `*store.Store`. `main.go` builds each domain service from that store, so this package must never import a domain package — adding a domain must not edit it. Domain services receive only their own `Repository` and must not be injected with the `Runtime`.
- `dialect.Driver` is the only place allowed to know database-specific SQL. Repositories must not contain dialect-specific SQL; case-insensitive matching goes through `CaseInsensitiveEqual`/`CaseInsensitiveLike` and constraint decoding through `UniqueViolation`. Adding PostgreSQL means adding one adapter package plus its `migrations/` directory — `store`, `migrate` and the domain packages must not change.
- `internal/httpapi` is strictly the transport layer: `decodeJSON` → call the service → `writeSuccessJSON` / `writeError`. State checks, authorization rules, and business branches must not be written into handlers.
- New routes are registered centrally in `internal/httpapi/server.go`, with the middleware order fixed as `RequireSetupCompleted → RequireAuth → RequireCapability`; `http.ServeMux` uses the `"METHOD /path"` pattern.
- The integration path for a new domain is: `internal/<domain>` → `internal/database/store/<domain>.go` (row models in `models.go`) → construction from `database.Runtime.Store` in `main.go` → `HandlerOptions` → route.
- `internal/httpapi` must not own file storage. Directory layout, allowed types, the size ceiling and path safety belong to the package that owns the stored file — currently `internal/avatar`, injected through `HandlerOptions`. The handler keeps only the transport half: multipart parsing, body limits, and serving an already validated path (that is why `http.ServeFile` may appear in `httpapi`).
- `internal/authorization` is a business-free permission model split by concern: `capability.go` (the capability catalog and its declaration order), `policy.go` (the role → capability matrix), `service.go` (pure resolution). **Never add `CanManageXxx`-style object-level rules there**; the package must stay reusable across projects.
- Object-level rules belong to the domain that owns the object and only encode relationships: `internal/identity/management.go` answers "may this actor act on this target" from role ranks (`internal/identity/role.go`), never from role names. Whether the actor may perform that class of action is `RequireCapability`'s job; the two checks must stay disjoint.
- Adding a role takes two backend edits: one entry in `identity.roleOrder` (its rank) and one entry in `authorization.roleCapabilities` (its grants). A third backend place means the two layers have started duplicating each other. The frontend role enum, the `RoleKey` type and the `ROLE_KEY` handles are generated from `roleOrder` — never hand-write them, and always run `go generate ./internal/contractgen`; a role that is declared but not regenerated makes `/api/auth/me` fail zod parsing and silently logs the role out. What a new role still needs by hand is the label switch (`web/src/lib/auth/roles.ts`), `common.role.*` in both locales, and `ROLE_CAPABILITIES` in the dev mock (its `satisfies Record<RoleKey, …>` catches that one). A user status or managed user action is mirrored the same way — one constant plus one entry in `identity.statusOrder` / `identity.managedUserActionOrder`, then regenerate — but the catalog is not the rule. `normalizeStatus` reads `statusOrder`, so an entry there lets the service accept the value and the frontend see it at once (whether the status offers a toggle is still a branch in `ManagedUserActions`); an action additionally needs its `CanManageUser` case and its `ManagedUserActions` branch. Go cannot reflect constants, so a constant left out of a catalog fails closed at runtime (the service rejects the value). The dangerous direction is the opposite one: `catalog_test.go` fails when `ManagedUserActions` offers an action the catalog does not declare, because that value reaches `/api/management/users` and breaks zod parsing.

### Frontend

- The data flow is fixed: `views/` page composition → `composables/<domain>/` state and asynchronous flows → `lib/api/<domain>.ts` (ky requests + zod parsing) → backend. `utils/<domain>/` contains pure functions only.
- `lib/api/client.ts` is the only ky entry point: use `baseApi` for ordinary requests and `authApi` for requests that require an access token. Single-flight 401 refresh and session-invalidation handling live here; business layers must not implement retries or token logic.
- Every API response must be validated with zod (`successEnvelopeSchema`); ESLint forbids `as` type assertions (except `as const`). Values from external inputs such as `localStorage` must be funneled through `parseExternalValue*` in `lib/external-input.ts`.
- Session state belongs only in `stores/auth.ts`; page-local state stays in the corresponding composable. Do not put it into a global store for convenience.
- `components/ui/**` holds the base primitives. They started as shadcn-vue CLI output and are now maintained locally: changing classes and the variant set is expected and normal, but primitives must stay business-free (no store, router or API imports). `components.json` is deliberately removed — upstream sync is not a goal, so `shadcn-vue add` must not be reintroduced; write a new primitive in the same shape as its neighbours instead. Project-owned components belong in `components/<domain>/`.

## Contracts That Must Change Together

- Response envelope: success is `{success:true,data}`, failure is `{success:false,error:{code,message}}`.
- Adding or removing an error code requires changes in four places simultaneously: `internal/httpapi/error_codes.go`, `apiError.*` in `web/src/locales/zh-CN.json`, `en-US.json`, and `web/src/lib/api/error-messages.ts` (`satisfies` enforces this). A missing change fails `error_codes_test.go` or `pnpm typecheck`.
- New domain errors resolve through `domainErrorMappings` in `error_codes.go` (`writeDomainError`, matched with `errors.Is`), never through a fresh `switch` in a handler. Two documented exceptions stay out of the table because the same error needs a different code by context: session invalidation (`auth_handlers.go`) and the own-account endpoints (`writeOwnAccountError`, where a missing row means a dead session rather than a 404).
- i18n: `zh-CN.json` is the sole source of `MessageSchema`; all user-visible copy must be added to both locale files. ESLint forbids runtime key construction—keys must either be written as `t('literal')` or narrowed through an exhaustive `switch`; otherwise the key is treated as unused.
- Capabilities have exactly one source: `internal/authorization/capability.go` (catalog + `roleCapabilities` in `policy.go`). The frontend module `web/src/lib/auth/capabilities.generated.ts` (schema, `Capability` type, `CAPABILITY` handles) is generated from it — never edit it by hand, and never reintroduce a hand-written capability list. Adding a capability is: declare it, grant it in the matrix or add it to `expectedPolicyDerived` in `capability_test.go`, run `go generate ./internal/contractgen`. `internal/contractgen/main_test.go` fails on drift, `capability_test.go` fails on a capability that is declared but unreachable or granted but undeclared. **The server-side `RequireCapability` is the actual authorization boundary**; the frontend `hasCapability` / `hasAllCapabilities` (`lib/auth/authorize.ts`) is the single frontend decision point, responsible only for route interception and hiding entry points.
- Authentication: the access token is held in memory only (`lib/auth/token.ts`), and the refresh token is stored in an `HttpOnly` cookie; never write the access token to `localStorage` or `sessionStorage`.
- Database migrations: `internal/database/dialect/<dialect>/migrations/NNNN_name.sql` are append-only and belong to their dialect. Applied migrations carry a SHA-256 checksum in `schema_migrations`; changing a single byte or dropping a file that the ledger already records prevents startup, so fix forward with a new file.
- Tests: HTTP-related tests use `newTestContext` (`internal/httpapi/server_test.go`) to set up a real SQLite database plus `httptest`; do not introduce mock repositories or a second database.

## Adding a Database Dialect

1. Create `internal/database/dialect/<dialect>/` with a `Driver` implementing `dialect.Driver`: `Name`, `Open` (DSN validation, pool, settings verification), `Migrations`, `BunDialect`, `UniqueViolation`, `CaseInsensitiveEqual`, `CaseInsensitiveLike`.
2. Add `internal/database/dialect/<dialect>/migrations/NNNN_name.sql` holding the full schema for that database, plus a `migrations.go` embed holder. Table and column names must match the row models in `internal/database/store/models.go`.
3. Add one case to `selectDriver` in `internal/database/runtime.go` and the driver's DSN to `internal/config` if it needs one. Nothing in `store`, `migrate` or the domain packages changes.
4. Verify with the same tests: `go test ./internal/database/...` plus a store test run against the new dialect before trusting it.

## Known Pitfalls

- `ColumnExpr` on a bun select **replaces** the default column list instead of extending it. A listing that adds a computed column must also spell out the row columns (see `userColumns` in `internal/database/store/identity.go`), otherwise the model scans zeros with no error.
- bun does not fill `autoincrement` primary keys automatically when the dialect supports `RETURNING` (SQLite and PostgreSQL do). Inserts that need the generated id must call `.Returning("id")`.
- `.data/data.db` is created from the current migration set. When a migration file changes, the checksum guard makes startup fail on purpose: delete the local database instead of disabling the guard.
- The startup order in `main.ts` must not change: mock → pinia → `authStore.bindRouter()` → `authStore.initialize()` → `app.use(router)`. Mounting the router earlier causes the initial route guard to read session state before it is ready.
- `VITE_API_MOCKING=true` takes effect only in dev and directly injects a mock owner session (`mocks/dev-session.ts`), bypassing the login and guest flows; unmatched API requests `bypass` to the real backend, so the Go server must still be running.
- The SSE log stream depends on `http.ResponseController` and `statusRecorder.Unwrap()`. Any new `ResponseWriter` wrapper must implement `Unwrap`; otherwise streaming endpoints fail silently.
- `.data/` stores both `data.db` and the auto-generated `.jwt_secret`; deleting the directory invalidates all sessions. `.env`, `.data/`, `web/dist`, and `*.exe` are not tracked.
- `web/src/lib/auth/capabilities.generated.ts` and `web/src/lib/api/contract.generated.ts` are listed in `web/.prettierignore` on purpose: the drift test compares bytes, so letting Prettier reformat them would make "generate → format → test" fail in a loop. Do not move them back into the formatted set without switching that test to semantic comparison.
- `account.delete_self` is checked inside `deleteAccountHandler`, not via middleware, because it is derived from `account_policies` per request and must return `CodeAccountDeleteForbidden`; see the comment on that branch before "unifying" it.
- For non-obvious pitfalls (framework behavior, protocol boundaries, concurrency/timeouts, third-party quirks), write a one-line comment explaining **why** next to the code that triggers it, including when it can be removed; do not pile this knowledge into this file. Do not delete existing pitfall comments.

## Definition of Done

- The minimum closed loop for a new business capability: domain service + repository → store implementation + row model → dialect migration → handler + route → capability/role/enum catalog + `go generate ./internal/contractgen` → error codes/bilingual locales → frontend api + composable + view + route meta (including capability).
- Non-trivial logic (branches, parsing, authorization, money, concurrency) requires a small test; purely presentational or copy changes do not.
- Do not add compatibility branches, `deprecated` aliases, dual-write logic, or configuration for "possible future use"; delete code with no callers outright.
