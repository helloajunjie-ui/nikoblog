# ===== 阶段 1：构建前端 =====
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ===== 阶段 2：编译 Go 后端 =====
# 注意：项目使用 gorm.io/driver/sqlite（底层 mattn/go-sqlite3，CGO 依赖），
# 必须启用 CGO 并安装 gcc/musl-dev 编译工具链，不能用 CGO_ENABLED=0。
FROM golang:1.26-alpine AS backend
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 前端 dist 已由阶段 1 输出到 web/dist，embed 会打包进二进制
COPY --from=frontend /app/frontend/../web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o /nikoblog ./cmd/server

# ===== 阶段 3：运行镜像 =====
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata libc6-compat
WORKDIR /app
COPY --from=backend /nikoblog /app/nikoblog
# 数据目录（SQLite + 上传图片）通过卷持久化
RUN mkdir -p /app/data
ENV NIKOBLOG_DATA_DIR=/app/data
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["/app/nikoblog"]
