# AGENTS.md

仓库级 AI 协作约定。只记录从代码里不能立刻推断、踩错会浪费一轮构建或破坏契约的事实；通用工程规范不重复。

## 1. 技术栈与运行形态

- 单仓库全栈：Go 1.27（module 名为 `main`）+ Vue 3.5 / Vite 8 / pnpm 11；命令示例使用 PowerShell。
- 生产形态是单二进制：`frontend_embed.go` 通过 `//go:embed web/dist` 嵌入前端，`main.go` 启动 HTTP 服务并托管 SPA。
- 仓库没有 CI 配置，本地命令就是唯一门禁。
- 运行时数据在 `.data/`；SQLite 固定使用 `.data/data.db`，PostgreSQL 连接信息保存在 `.data/database.json`。
- 不提交 `.env`、`.data/`、`web/dist`、`*.exe`；`web/dist` 是构建产物，不要手改。

## 2. 构建顺序：先前端，后 Go

`web/dist` 被 `//go:embed` 引用，目录不存在时根目录的 `go build` 和 `go test` 会直接失败。全新克隆先执行：

```powershell
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
go test ./...
go build .
```

- `.\build.ps1` 已封装正确顺序：前端构建 -> `go mod tidy` -> `go build`，产物为 `app.exe`。
- 只改 Go 且 `web/dist` 已存在时，可只跑 Go 命令；改前端或契约后必须先 `pnpm build` 再构建 Go。
- Vite 会生成 `.gz` 旁路文件，`internal/httpapi/spa.go` 按 `Accept-Encoding` 提供预压缩版本；不要删除或绕开这段逻辑。

## 3. 契约与生成代码

Go 是契约的唯一事实来源，前端生成文件禁止手改：

- `internal/authorization/capability.go`：`capabilityOrder`
- `internal/identity/role.go`：`roleOrder`
- `internal/identity/service.go`：`statusOrder`
- `internal/identity/management.go`：`managedUserActionOrder`

约定：

- 每个 `*Order` 切片必须与同文件 const 块同步。Go 不能反射 const，漏加一项仍可编译、测试也可能通过，但前端永远收不到该值。
- 新值追加到末尾，不要重排；生成枚举的顺序就是切片顺序，重排会产生无法审查的 diff。
- `roleOrder` 必须从高到低声明并保留 rank 间隔；`IsHighestRole` 依赖首项，新增角色时不要重排现有 rank。
- 修改后执行 `go generate ./internal/contractgen`，产物为：
  - `web/src/lib/auth/capabilities.generated.ts`
  - `web/src/lib/api/contract.generated.ts`
- `internal/contractgen/main_test.go` 会逐字节比对；未重新生成会让 `go test ./...` 失败。
- `Roles()`、`Statuses()`、`AllManagedUserActions()` 看似没有生产调用，但生成器和契约测试依赖，勿当死代码删除。
- 只有 capability/role 生成命名句柄；status/action 直接使用生成的字面量类型，不要手写字符串，也不要补一套句柄常量。

## 4. HTTP、认证与安装流程

- 路由集中在 `internal/httpapi/server.go` 的 `newAPIMux`。受保护业务路由按 `RequireSetupCompleted -> RequireAuth -> RequireCapability` 的顺序挂中间件；前端路由守卫不是安全边界。
- setup 前后都必须可用的路由加到 `registerSharedRoutes`，不要在 bootstrap 和完整 mux 各注册一份。
- `Dispatcher` 在安装完成前提供 SPA + bootstrap API，完成后 `Publish` 完整 API；不要绕开它直接挂 `http.Server`。
- JSON 使用 Go 1.27 的 `encoding/json/v2`（别名 `jsonv2`）。HTTP 层统一走 `decodeJSON`（1 MiB 上限、拒绝未知字段、只接受单个 JSON 值）与 `writeSuccessJSON` / `writeError` / `writeDomainError`；不要直接 `json.Unmarshal` 或手写响应包络。
- 响应格式固定：成功 `{success:true,data}`，失败 `{success:false,error:{code,message}}`；handler 不把领域错误原文返回给客户端。
- 新增错误码只在“需要不同文案或不同后续动作”时添加；技术性失败统一 `internal_error`，细节写日志。
- 全局 `WriteTimeout` 是 30s；SSE 路由必须用 `http.ResponseController` 按请求清除写超时，参考 `internal/httpapi/system_logs.go`。
- JWT 签名密钥以数据库为准；`JWT_SECRET` 只是首次种子（≥32 字符）。不要新增写密钥文件，也不要记录 DSN、密码或令牌。

## 5. 错误码与 i18n 联动

新增或删除错误码时，以下位置一起改：

1. `internal/httpapi/error_codes.go`：常量 + `errorDefinitions`。
2. `web/src/locales/zh-CN.json` 和 `en-US.json` 的 `apiError.<code>`。
3. `web/src/lib/api/error-messages.ts`：加一行字面量 `t('apiError.<code>')`。
4. 检查调用方是否需要专属 fallback 文案。

- `TestErrorCodesMatchFrontendLocale` 锁定 Go 注册表与两个语言包；`satisfies Record<ApiErrorCode, ...>` 锁定 TypeScript 端完整性。
- `zh-CN.json` 是 `MessageSchema` 的来源，不要加 `as const`；英文包缺键由类型标注和 ESLint 共同拦截。
- i18n 是硬约束：禁止运行期拼键；数据驱动取键只能写成穷尽 `switch` 或内联到定义处；所有 `t()` 必须使用字面量 key。否则 `no-unused-keys` 看不到键，键会被误删，运行时静默显示原始 key。
- 两个语言包同名 key 的 `{placeholder}` 集合必须一致（Go 测试 `TestLocalePlaceholdersMatch`）。
- 错误提示默认用 `resolveErrorMessageOrInternal(t, error)`；调用方有专属文案时用 `resolveAPIErrorMessage(t, error) ?? t('...')`，不要让通用 `internal_error` 覆盖场景文案。
- 文案分层：`common.*` 承载跨页面复用的动作（`action`）、字段标签与占位（`field`/`placeholder`）、状态（`state`）、通用结果（`feedback`）、校验（`validation`）、头像（`avatar`）与枚举目录（`status`/`role`/`theme`）；`<domain>.*` 只放该域独有的名词和说明。页面不要再自建 `retry`、`loadFailed`、`cancel`、`save` 这类通用键，同一个按钮在不同页面会漂成三种措辞。
- 链接消息（`"@:other.key"`）只用于**必须逐字一致**的成对文案，典型是接口错误码与前端校验提示，例如 `apiError.avatar_invalid_type: "@:common.avatar.invalidType"`。区块标题、路由标题、导航标签、表格列名这类可独立演化的文案不要造链接，需要同词就直接在组件里 `t()` 通用键。`no-unused-keys` 会把整个语言包里所有 `@:` 目标都算作已使用，链接目标不需要再补一处 `t()` 引用。

## 6. 数据库与迁移

- 迁移文件位于 `internal/database/dialect/sqlite/migrations/` 和 `internal/database/dialect/postgresql/migrations/`，命名 `<version>_<name>.sql`。
- 每种方言都要提供对应 SQL；加表、列或索引时两个目录一起改，并保持语义一致。
- 已发布的迁移逐字节不可变：启动时校验 SHA-256，改内容、改名、删文件都会让启动失败。
- 新迁移版本号递增，不要复用或跳回旧号；每个迁移在独立事务中执行。
- 本地需要重新跑迁移时，删除 `.data/data.db` 后重新走 `/setup`；PostgreSQL 部署必须保留 `.data/database.json`，一旦丢失，`/setup` 不会接管已完成的安装。
- 数据库写入边界在 `internal/database` 的仓储层；`internal/installation` 负责首次安装和候选数据库生命周期。领域服务不要自行编排多写事务。

## 7. 前端结构与状态

- 代码按领域分目录并跨层镜像：`components/<domain>/`、`composables/<domain>/`、`utils/<domain>/`、`lib/<domain>/`。
- `views/` 只做页面编排；有状态逻辑放 composable；纯函数放 `utils`；无 UI 的基础能力放 `lib`。
- `stores/` 只放跨路由或需要持久化的状态（auth、locale、theme、polling、system-logs preferences）。列表筛选、分页等页面状态留在 composable，不要复制进 Pinia。
- `components/ui/` 是 shadcn-vue 风格基础组件：只做原语级修改，不写业务逻辑，不导入 store、router 或业务模块。
- `@` 别名的唯一来源是 `web/build/alias.ts`；`tsconfig.json` 与 `tsconfig.app.json` 手工镜像，改别名必须三处同步。配置文件中导入它必须带 `.ts` 扩展名（Vite 原生加载器限制）。
- API 层：`lib/api/*.ts` 使用 ky + zod。认证接口用 `authApi`（401 单飞刷新），公开/安装接口用 `baseApi`；统一 `.json<unknown>()` 后 `schema.parse`，异常交给 `normalizeAPIError(error)`。不要裸用 `fetch`，不要使用 `any`。
- 成功响应用 `successEnvelopeSchema`；`roleKeySchema`、`userStatusSchema`、`managedUserActionSchema`、`capabilitySchema` 是线上词汇表，不要在业务代码中重复字面量。
- ESLint 禁止用 `as` 做类型断言（`as const` 除外）和 `any`：外部输入用 zod / `parseExternalValue` 收窄，不要用断言绕过。
- Vue 组件契约：`defineProps` 用类型声明，`defineEmits` 用类型字面量，自定义事件使用 kebab-case，模板 ref 用 `useTemplateRef`（均为 lint 强制）。
- 新增路由：`router/index.ts` 配 `titleKey`、`requiresAuth`、`requiredCapabilities`（用 `CAPABILITY` 句柄）、`maskUnauthorizedAsNotFound`；同时补 `route-title.ts` 的穷尽 switch 和两个语言包的 `route.*`。
- Tailwind 由 `eslint-plugin-better-tailwindcss` 校验未知、冲突、重复、规范写法和顺序，入口是 `src/style.css`；改模板 class 后必须跑 `pnpm lint`。
- 格式：Prettier 无分号、单引号、LF；`pnpm format:check` 是门禁。
- MSW 仅开发期生效（`VITE_API_MOCKING=true` 且 `import.meta.env.DEV`），不得进入生产包；mock 只补真实后端不可用的场景，不要用它掩盖契约缺口。
- 新增语言时同步 `src/locales/index.ts`、`plugins/i18n/locales.ts`、`eslint.config.mjs` 的 `localeMessageFiles` 与 settings。

## 8. 测试：锁定行为，拒绝凑数

- Go 测试是主要契约门禁，命令为 `go test ./...`。
  - 单包/单用例：`go test ./internal/httpapi -run TestErrorCodesMatchFrontendLocale`
  - 仓储和安装测试使用临时目录中的真实 SQLite（`t.TempDir()` + `dialect/sqlite` + `migrate`），不要 mock 数据库行为。
  - 测试普遍使用 `t.Parallel()`，新增测试保持同样风格。
  - 跨包契约测试（错误码、语言包占位符、生成文件漂移、角色/状态/动作目录）是有意存在的，不要挪走或删除。
- 前端只测纯逻辑与状态机：`web/src/**/*.spec.ts`，由 `vitest.config.ts` 以 jsdom 运行。
  - 不引入组件挂载测试、快照测试或 E2E。组件行为优先拆成纯函数/composable 后在 spec 中锁定；接口契约由 Go 测试锁定。
  - 单文件：`pnpm vitest run src/lib/api/error-messages.spec.ts`
  - 测试只断言可观察行为（输入 -> 输出、状态迁移、错误码/文案键），不要断言内部调用次数或实现细节。
- 每个 bug 修复先补一个能复现缺陷的测试，再改实现；修复后删除临时调试测试。
- 不写重复测试，不为覆盖率写测试，不为类型系统已保证的约束写运行期断言，不写只验证框架行为的测试。
- 无法运行检查时，在最终回复说明原因和未覆盖风险，不要把“没跑”说成“通过”。

## 9. 代码批注约定

- 只在“看代码猜不到、改错会再踩坑”的位置注释：外部约束、协议/平台怪癖、不可逆操作、安全决策、性能取舍、上游 bug 的 workaround。
- 注释解释 **why**，不解释 **what**；一行能说清就不写段落。禁止分区横幅、教程式旁白、逐行翻译和 `// 处理数据` 这类空注释。
- 踩过的坑必须批注在坑所在的代码位置，写清触发条件、失败表现、为什么不能改成“更自然”的写法、何时可以移除；不要只写在提交信息或 PR 描述里。
- workaround 使用可检索标记：`shortcut: <限制>, <升级/移除条件>`。
- 行为改变时同步更新注释；过期或误导性注释按缺陷处理。删除被改动路径上的注释代码和废弃注释。
- 现有解释“为什么”的注释是承重结构（`json.go`、`spa.go`、`contractgen`、`build/alias.ts`、`vitest.config.ts` 等），重构前先理解，不要顺手删除。

## 10. 验证清单

前端改动：

```powershell
cd web
pnpm lint
pnpm typecheck
pnpm test
pnpm format:check
pnpm build
```

Go 改动（在仓库根目录）：

```powershell
gofmt -w <改动的 .go 文件>
go vet ./...
go test ./...
go build .
```

契约或迁移改动额外执行：

```powershell
go generate ./internal/contractgen
go test ./internal/contractgen ./internal/httpapi ./internal/database/...
```

- 仓库未配置 golangci-lint，不要自行引入或大改 Go lint 规则。
- 前端类型检查是 `vue-tsc --build`，包含 spec 的 `tsconfig.vitest.json`；不要只跑 `tsc`。
- 新增依赖：前端在 `web/` 用 `pnpm add` 并提交 `pnpm-lock.yaml`；Go 在根目录用 `go get` / `go mod tidy`。不要手改 lockfile。
