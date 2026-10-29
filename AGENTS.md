# Repository Guidelines

## Project Structure & Module Organization

- `main.go` is the Go composition root. `frontend_embed.go` embeds the built `web/dist` assets into the server binary.
- `internal/` contains Go packages for account policies, authentication, authorization, configuration, database access, HTTP API, identity, logging, and setup. Tests live beside code as `*_test.go`.
- `web/src/` contains the Vue 3 app: `components/`, `views/`, `layouts/`, `router/`, `stores/`, `composables/`, `lib/`, `locales/`, `mocks/`, and `utils/`. Reusable shadcn-vue primitives live under `web/src/components/ui/`.
- `web/dist/`, `.data/`, `node_modules/`, and compiled executables are generated or local-only files.

## Project-Level Constraints

This is a starter, not a compatibility layer. Optimize for a clean, current foundation:

- Do not preserve legacy wrappers, deprecated exports, shims, or historical package structure unless requested.
- Prefer clear boundaries, explicit data flow, and production-safe defaults over speculative flexibility.
- When contracts change, update callers, tests, mocks, and docs together; add no branches for old callers.

### Intentionally unreferenced code

Two categories look like dead code but are deliberate. Do not delete them as part of an
unrelated cleanup:

- **`web/src/components/ui/**`** — the shadcn-vue primitive pool. Components here are
  expected to sit unused until an app feature needs one; that availability is the point of
  the starter. They are tree-shaken out of the bundle, so an unused primitive costs nothing
  at runtime. Remove one only when deliberately trimming the kit.
- **`web/src/components/dashboard/**`** — the dashboard cards and chart. They run on
  static data and demonstrate the intended UI patterns; replace the data sources when
  wiring a real dashboard.

Everything else — unreferenced exports, helpers, i18n keys, dependencies — is fair game and
should be deleted rather than left behind. i18n keys live in three files
(`locales/schema.ts`, `en-US.ts`, `zh-CN.ts`); the schema is what keeps the two locales in
lockstep, so add and remove keys in all three together.

## Build, Test, and Development Commands

Run frontend commands in `web/` with pinned `pnpm`:

```text
pnpm install --frozen-lockfile  # install the locked dependencies
pnpm dev           # start Vite development mode
pnpm lint          # run ESLint
pnpm typecheck     # run vue-tsc
pnpm format:check  # verify Prettier formatting
pnpm build         # create web/dist for embedding
```

From the repository root, use `go test ./...` for all Go tests and `go build .` for the backend. On Windows, `.\build.ps1` builds the frontend and then produces `app.exe`.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep business logic out of Vue views when a composable or service boundary exists. Use `PascalCase.vue` components, `use`-prefixed composables, and `*_test.go` tests. Prettier enforces no semicolons, single quotes, trailing commas, LF endings, and one attribute per line.

## Testing Guidelines

Add focused Go tests beside behavior changes. No frontend test runner is configured; validate UI changes with `pnpm lint`, `pnpm typecheck`, `pnpm format:check`, and `pnpm build`.

## Commit & Pull Request Guidelines

Use scoped Conventional Commit subjects such as `feat(sqlite): ...`, `chore(web): ...`, or `fix(httpapi): ...`. PRs should explain the change, list validation, include relevant screenshots, and call out migrations or configuration changes.

## Security & Configuration Tips

Use `.env.example` for local configuration. Never commit `.env`, `.data/`, secrets, or build outputs. Treat `JWT_SECRET`, database paths, and `VITE_API_MOCKING` as environment-specific.
