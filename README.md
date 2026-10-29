# shadcn-vue-go-starter

`shadcn-vue-go-starter` 是一个面向新业务系统的全栈起始点。它提供一套可以直接继续建设的 Go + Vue 3 基础工程，把认证、授权、数据库、应用壳和常见后台页面先铺好，让新项目从清晰的架构边界开始，而不是从空白脚手架或历史兼容层开始。

## 项目定位

- Go 服务负责配置加载、JWT 会话、角色与 capability 授权、SQLite 持久化、HTTP API 和前端静态资源托管。
- Vue 应用提供登录、注册、初始化设置、Dashboard、个人设置、系统设置、用户管理和系统日志等可运行页面；站点根路径直接进入 Dashboard，未登录或未初始化时由路由守卫接管。
- 前端使用 Pinia、Vue Router、Vue I18n、Tailwind CSS 和 shadcn-vue 风格组件；`MSW` 可通过 `VITE_API_MOCKING=true` 提供登录相关 mock。
- 生产构建会把 `web/dist` 嵌入 Go 二进制，适合从一个仓库继续发展为独立业务应用。

## 技术栈

- Backend: Go 1.26.3, SQLite, `net/http`
- Frontend: Vue 3, TypeScript, Vite 8, Tailwind CSS 4
- Tooling: pnpm 11, ESLint, Prettier, `vue-tsc`

## 快速开始

需要 Go 1.26.3、Node.js 和仓库指定版本的 pnpm。首次运行时可按需复制配置模板：

```powershell
Copy-Item .env.example .env
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
go run .
```

服务默认监听 `http://localhost:8080`。前端开发服务器需要代理后端 API 时，在另一个终端运行：

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
- `web/src/views/`：页面；`web/src/components/`：可复用组件；`web/src/lib/api/`：前端 API 客户端；`web/src/locales/`：多语言资源。
- 新领域功能应同时明确后端边界、API 契约、前端状态和页面职责；不要为了兼容旧结构增加包装层。

运行数据默认位于 `.data/`，JWT secret 会从 `JWT_SECRET` 读取；未配置时应用会在数据目录生成并保存随机 secret。不要提交 `.env`、`.data/`、secret 或构建产物。
