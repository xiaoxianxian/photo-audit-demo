#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Photo Audit Platform — 本地快速启动脚本 (无需 Docker)
# =============================================================================
# 前提条件:
#   1. Go 1.23+ 已安装
#   2. Node.js 20+ 已安装
#   3. PostgreSQL 15+ 已安装且正在运行
#   4. Redis 7+ 已安装且正在运行
#   5. MinIO (可选，用于文件上传)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR/.."

echo "========================================"
echo "  Photo Audit Platform — 本地启动"
echo "========================================"
echo ""

# --- 检查依赖 ---
MISSING=0

if ! command -v go &>/dev/null; then
  echo "❌ Go 未安装，请先安装 Go 1.23+"
  MISSING=1
else
  echo "✅ Go $(go version | awk '{print $3}')"
fi

if ! command -v node &>/dev/null; then
  echo "❌ Node.js 未安装，请先安装 Node.js 20+"
  MISSING=1
else
  echo "✅ Node.js $(node --version)"
fi

if ! command -v psql &>/dev/null; then
  echo "⚠️  psql 未找到，跳过数据库检查"
else
  if psql -h localhost -U postgres -c "SELECT 1" &>/dev/null; then
    echo "✅ PostgreSQL 已连接"
  else
    echo "❌ PostgreSQL 连接失败，请检查 postgres 是否运行"
    MISSING=1
  fi
fi

if [ "$MISSING" -ne 0 ]; then
  echo ""
  echo "请安装缺失的依赖后重试"
  exit 1
fi

# --- 初始化数据库 ---
echo ""
echo "📦 初始化数据库..."
DB_EXISTS=$(psql -h localhost -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='photo_audit'" 2>/dev/null || echo "0")
if [ "$DB_EXISTS" = "0" ]; then
  psql -h localhost -U postgres -c "CREATE DATABASE photo_audit;" 2>/dev/null || true
  echo "✅ 创建了 photo_audit 数据库"
else
  echo "✅ photo_audit 数据库已存在"
fi

if [ -f "$PROJECT_DIR/backend/sql/init.sql" ]; then
  psql -h localhost -U postgres -d photo_audit -f "$PROJECT_DIR/backend/sql/init.sql" 2>/dev/null && echo "✅ 数据库表已初始化" || echo "⚠️  数据库初始化跳过（可能表已存在）"
fi

# --- 启动后端 ---
echo ""
echo "🚀 启动后端服务..."
cd "$PROJECT_DIR/backend"

export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/photo_audit?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET="audit-platform"
export FALLBACK_ENABLED="true"
export SERVER_PORT="8080"

# 安装依赖
go mod download 2>/dev/null || true

# 编译并启动
echo "▶ 编译后端..."
go build -o audit-server ./cmd/server/ 2>&1 || {
  echo "❌ 后端编译失败，请检查 Go 代码"
  exit 1
}

echo "▶ 启动后端 (端口 8080)..."
./audit-server &
BACKEND_PID=$!
echo "  后端 PID: $BACKEND_PID"

# --- 等待后端就绪 ---
echo ""
echo "⏳ 等待后端就绪..."
for i in $(seq 1 30); do
  if kill -0 $BACKEND_PID 2>/dev/null; then
    # 检查是否还在监听
    if lsof -i :8080 &>/dev/null 2>&1 || ss -tlnp | grep 8080 &>/dev/null 2>&1; then
      echo "✅ 后端已就绪"
      break
    fi
  fi
  sleep 2
done

# --- 启动前端 ---
echo ""
echo "🚀 启动前端服务..."
cd "$PROJECT_DIR/frontend"

# 安装依赖
if [ ! -d "node_modules" ]; then
  echo "▶ 安装前端依赖..."
  npm install
fi

echo "▶ 启动前端 (端口 3000)..."
npm run dev &
FRONTEND_PID=$!
echo "  前端 PID: $FRONTEND_PID"

echo ""
echo "========================================"
echo "  ✅ 全部就绪！"
echo "========================================"
echo ""
echo "   前端:  http://localhost:3000"
echo "   后端:  http://localhost:8080"
echo "   MinIO: http://localhost:9001"
echo ""
echo "   进程:"
echo "   后端 PID: $BACKEND_PID"
echo "   前端 PID: $FRONTEND_PID"
echo ""
echo "   停止服务: kill $BACKEND_PID $FRONTEND_PID"
echo "   或: ./stop-local.sh"
echo ""

# 自动打开浏览器
if command -v open &>/dev/null; then
  open "http://localhost:3000"
elif command -v xdg-open &>/dev/null; then
  xdg-open "http://localhost:3000"
fi

# 等待进程结束
wait
