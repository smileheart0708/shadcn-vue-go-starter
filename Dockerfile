# Stage 1: Build Frontend
FROM node:24-alpine AS frontend-builder
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm run build
# Stage 2: Build Backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .
# Stage 3: Runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata mailcap
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend-builder /app/main .
EXPOSE 8080
CMD ["./main"]
