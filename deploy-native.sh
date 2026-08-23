#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo "  Photo Audit Platform — 本地部署脚本"
echo "  (无 Docker，原生安装)"
echo "========================================"
echo ""

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

# ---------- 0. 前置检查 ----------
MISSING=()

if ! command -v brew &>/dev/null; then
  MISSING+=("Homebrew")
fi

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "❌ 以下依赖未安装，请先安装："
  for m in "${MISSING[@]}"; do echo "   - $m"; done
  echo ""
  echo "安装 Homebrew："
  echo '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
  echo ""
  exit 1
fi

# ---------- 1. 安装 Go ----------
if command -v go &>/dev/null; then
  echo "✅ Go 已安装: $(go version)"
else
  echo "📦 安装 Go..."
  brew install go
  echo "✅ Go 已安装: $(go version)"
fi

# ---------- 2. 安装 PostgreSQL ----------
if command -v psql &>/dev/null; then
  echo "✅ PostgreSQL 已安装: $(psql --version)"
else
  echo "📦 安装 PostgreSQL..."
  brew install postgresql@15
  echo "✅ PostgreSQL 已安装: $(psql --version)"
fi

# ---------- 3. 启动 PostgreSQL ----------
echo ""
echo "🔄 启动 PostgreSQL..."

# 自动检测正确的 brew service 名
PG_SERVICE=""
for svc in postgresql@15 postgresql@14 postgresql postgresql@16; do
  if brew services list | grep -q "$svc"; then
    PG_SERVICE="$svc"
    break
  fi
done

if [ -z "$PG_SERVICE" ]; then
  # 尝试用 pg_ctl 直接启动
  PGDATA=$(ls -d /opt/homebrew/var/postgres 2>/dev/null || echo "")
  if [ -n "$PGDATA" ] && [ -d "$PGDATA" ]; then
    pg_ctl -D "$PGDATA" start -l "$PGDATA/logfile" 2>/dev/null || true
  else
    echo "⚠️  无法自动检测 PostgreSQL 服务，尝试默认名..."
    brew services start postgresql@15 2>/dev/null || brew services start postgresql 2>/dev/null || true
  fi
else
  echo "   检测到服务: $PG_SERVICE"
  brew services start "$PG_SERVICE"
fi

sleep 3

# 检查是否运行
if ! pg_isready &>/dev/null; then
  echo "⚠️  PostgreSQL 未就绪，尝试手动启动..."
  # 尝试用当前用户启动
  PGDATA=$(ls -d /opt/homebrew/var/postgres 2>/dev/null || echo "")
  if [ -n "$PGDATA" ]; then
    initdb "$PGDATA" 2>/dev/null || true
    pg_ctl -D "$PGDATA" start -l "$PGDATA/logfile" 2>/dev/null || true
  fi
  sleep 2
fi

if ! pg_isready &>/dev/null; then
  echo "❌ PostgreSQL 无法启动，请手动运行："
  echo "   brew services start postgresql@15"
  exit 1
fi
echo "✅ PostgreSQL 已运行"

# ---------- 4. 创建数据库 ----------
echo ""
echo "📦 创建数据库 photo_audit..."

# Homebrew PostgreSQL 15 默认使用当前系统用户作为超级用户，而非 "postgres" 角色
CURRENT_USER="$(whoami)"
echo "   使用用户: $CURRENT_USER"

# 删除已存在的数据库（避免冲突）
psql -U "$CURRENT_USER" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='photo_audit' AND pid <> pg_backend_pid();" 2>/dev/null || true
psql -U "$CURRENT_USER" -c "DROP DATABASE IF EXISTS photo_audit;" 2>/dev/null || true
sleep 1
createdb -U "$CURRENT_USER" photo_audit
echo "✅ 数据库创建成功"

# ---------- 5. 初始化表结构 ----------
echo ""
echo "📋 执行初始化 SQL（建表 + 种子数据）..."
psql -U "$CURRENT_USER" -d photo_audit -f backend/sql/init.sql
echo "✅ 表结构初始化完成"

# ---------- 5.5 创建后端 .env 文件（适配 Homebrew PostgreSQL）----------
echo ""
echo "🔧 创建后端 .env 配置文件..."

# 确保 pg_hba.conf 允许本地 TCP 连接（Homebrew 默认可能只有 peer 认证）
PG_DATA=$(ls -d /opt/homebrew/var/postgres 2>/dev/null || echo "")
if [ -n "$PG_DATA" ] && [ -f "$PG_DATA/pg_hba.conf" ]; then
  # 如果还没有 trust 的本地 TCP 规则，添加一条
  if ! grep -q "127.0.0.1/32" "$PG_DATA/pg_hba.conf" 2>/dev/null; then
    echo "host    all             all             127.0.0.1/32            trust" >> "$PG_DATA/pg_hba.conf"
    echo "   已添加 TCP trust 认证规则到 pg_hba.conf"
    brew services restart postgresql@15 2>/dev/null || true
    sleep 2
  fi
fi

cat > backend/.env <<EOF
SERVER_PORT=8080
DATABASE_URL=postgresql://${CURRENT_USER}@localhost:5432/photo_audit
REDIS_URL=
MINIO_ENDPOINT=
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=
MINIO_BUCKET=audit-platform
FALLBACK_ENABLED=true
MIGRATE_AUTO_UP=true
EOF
echo "✅ .env 已创建（数据库用户: $CURRENT_USER）"

# ---------- 6. 安装后端依赖 ----------
echo ""
echo "📦 安装后端 Go 依赖..."
cd backend
go mod download
cd ..
echo "✅ 后端依赖安装完成"

# ---------- 7. 安装前端依赖 ----------
echo ""
echo "📦 安装前端依赖..."
cd frontend
npm install
cd ..
echo "✅ 前端依赖安装完成"

# ---------- 8. 前端构建验证 ----------
echo ""
echo "🔍 前端 TypeScript 验证..."
cd frontend
npx tsc --noEmit
cd ..
echo "✅ TypeScript 编译通过"

# ============================================================
echo ""
echo "========================================"
echo "  ✅ 安装完成！"
echo "========================================"
echo ""
echo "下一步：启动服务（开两个终端窗口）"
echo ""
echo "  【终端 1】启动后端："
echo "    cd $PROJECT_DIR/backend"
echo "    go run ./cmd/server/"
echo ""
echo "  【终端 2】启动前端："
echo "    cd $PROJECT_DIR/frontend"
echo "    npm run dev"
echo ""
echo "  浏览器验收地址: http://localhost:3000"
echo "  登录账号: admin / admin123"
echo "  登录后可在「租户配置」「质量抽检」等页面验收全部功能"
echo ""
echo "  后端健康检查: curl http://localhost:8080/api/v1/health"
echo ""
