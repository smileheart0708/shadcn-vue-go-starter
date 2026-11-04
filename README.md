# shadcn-vue-go-starter

`shadcn-vue-go-starter` 是一个面向新业务系统的全栈起始点。它提供一套可以直接继续建设的 Go + Vue 3 基础工程，把认证、授权、数据库、应用壳和常见后台页面先铺好，让新项目从清晰的架构边界开始，而不是从空白脚手架或历史兼容层开始。

## 项目定位

- Go 服务负责配置加载、JWT 会话、角色与 capability 授权、Bun + SQLite/PostgreSQL 持久化、HTTP API 和前端静态资源托管。
- Vue 应用提供登录、注册、初始化设置、Dashboard、个人设置、系统设置、用户管理和系统日志等可运行页面；站点根路径直接进入 Dashboard，未登录或未初始化时由路由守卫接管。
- 前端使用 Pinia、Vue Router、Vue I18n、Tailwind CSS 和 shadcn-vue 风格组件；`MSW` 可通过 `VITE_API_MOCKING=true` 提供登录相关 mock。
- 生产构建会把 `web/dist` 嵌入 Go 二进制，适合从一个仓库继续发展为独立业务应用。

## 技术栈

- Backend: Go 1.27.3, Bun ORM (`uptrace/bun`) + SQLite（`modernc.org/sqlite`，纯 Go 无 CGO）或 PostgreSQL（`pgx` + `pgdialect`）, `net/http`
- Frontend: Vue 3, TypeScript, Vite 8, Tailwind CSS 4
- Tooling: pnpm 11, ESLint, Prettier, `vue-tsc`

## 快速开始

需要 Go 1.27.3、Node.js 和仓库指定版本的 pnpm。首次运行时可按需复制配置模板：

```powershell
Copy-Item .env.example .env
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
go run .
```

服务默认监听 `http://localhost:8080`。首次打开 `/setup` 时先创建管理员，再选择 SQLite 或填写 PostgreSQL 连接；PostgreSQL 必须是已创建的专用空库，后端会验证连接、检查数据库并执行迁移。完成后无需重启。SQLite 直接使用 `.data/data.db`，不产生额外的选择文件；只有 PostgreSQL 的连接信息会保存在 `.data/database.json`。

前端开发服务器需要代理后端 API 时，在另一个终端运行：

```powershell
cd web
pnpm dev
```

## 常用命令

```powershell
cd web
pnpm lint
pnpm typecheck
pnpm format:check
pnpm build

cd ..
go test ./...
go build .
.\build.ps1
```

`build.ps1` 会先构建前端，再生成包含前端资源的 `app.exe`。也可以使用仓库中的 `Dockerfile` 构建 Linux 镜像。

## 目录与扩展方式

- `internal/`：后端领域服务、数据库、HTTP API、认证和基础设施。
- `internal/database/` 分为方言适配、迁移执行器、仓储实现和本机数据库选择文件；`internal/installation/` 负责首次安装协调、候选数据库生命周期和完成后发布完整应用。
- `web/src/views/`：页面；`web/src/components/`：可复用组件；`web/src/lib/api/`：前端 API 客户端；`web/src/locales/`：多语言资源。
- 新领域功能应同时明确后端边界、API 契约、前端状态和页面职责；不要为了兼容旧结构增加包装层。

运行数据默认位于 `.data/`。JWT 签名密钥保存在数据库的 `auth_secrets` 表中：数据库值优先；数据库为空时用 `JWT_SECRET`（至少 32 字符）作为首次种子写入，未配置则自动生成随机密钥并写入，不会写入文件。不要提交 `.env`、`.data/` 或构建产物。

首次安装会按所选方言的迁移目录自动建表。已发布的迁移文件不可修改（启动时会校验 SHA-256）；如果改了迁移，删除本地数据库后重新安装即可。SQLite 不需要 `.data/database.json`，重启时会从 `.data/data.db` 自动恢复；PostgreSQL 的 DSN 只保存在 `.data/database.json`，不会通过 API 返回或写入日志，因此迁移或备份 PostgreSQL 部署时要把该文件一并带上；丢失后 `/setup` 不会接管已完成的 PostgreSQL 安装，需要从备份恢复或手动重建该文件。
