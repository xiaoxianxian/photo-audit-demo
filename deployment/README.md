# Photo Audit Platform — 部署指南

## 环境要求

### 方式一：Docker Compose（推荐）

需要安装：
- Docker Desktop（macOS/Windows）或 Docker Engine（Linux）
- Docker Compose v2

启动命令：
```bash
cd deployment
docker compose -f docker-compose.full.yml up -d --build
```

停止命令：
```bash
docker compose -f docker-compose.full.yml down
```

### 方式二：本地启动（需要 Go + Node.js）

需要安装：
- Go 1.23+
- Node.js 20+
- PostgreSQL 15+
- Redis 7+
- MinIO（可选，用于文件上传）

启动命令：
```bash
cd deployment
bash start-local.sh
```

停止命令：
```bash
bash stop-local.sh
```

## 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:3000 | Vite 开发服务器 |
| 后端 API | http://localhost:8080 | Fiber HTTP 服务器 |
| MinIO 控制台 | http://localhost:9001 | 对象存储管理 |
| MinIO API | http://localhost:9000 | S3 兼容 API |

## 环境变量

创建 `.env` 文件（在项目根目录）：

```bash
# 数据库
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/photo_audit

# Redis
REDIS_URL=redis://localhost:6379

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=audit-platform

# AI 模型（可选）
AGNES_API_KEY=sk-xxx
DEEPSEEK_API_KEY=sk-xxx

# 服务端口
SERVER_PORT=8080
FRONTEND_PORT=3000

# 降级开关
FALLBACK_ENABLED=true
```

## 数据库初始化

```bash
psql -h localhost -U postgres -c "CREATE DATABASE photo_audit;"
psql -h localhost -U postgres -d photo_audit -f backend/sql/init.sql
```

## 后端编译

```bash
cd backend
go mod tidy
go build -o audit-server ./cmd/server/
./audit-server
```

## 前端开发

```bash
cd frontend
npm install
npm run dev
```

## 前端构建

```bash
cd frontend
npm run build
# 产出在 dist/ 目录
```

## 常见问题

### 1. 后端启动失败

检查 PostgreSQL 是否运行：
```bash
psql -h localhost -U postgres -c "SELECT 1"
```

检查端口是否被占用：
```bash
lsof -i :8080
```

### 2. 前端无法连接后端

检查 Vite proxy 配置：
```typescript
// frontend/vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
}
```

### 3. MinIO 连接失败

检查 MinIO 是否运行：
```bash
curl http://localhost:9000/minio/health/live
```

### 4. Docker 构建失败

清理缓存后重试：
```bash
docker compose -f docker-compose.full.yml down --rmi all --volumes
docker compose -f docker-compose.full.yml up -d --build --no-cache
```
