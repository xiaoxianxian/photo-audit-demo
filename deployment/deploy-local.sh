#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.full.yml"

echo "========================================"
echo "  Photo Audit Platform — 本地部署"
echo "========================================"
echo ""

# --- 检查 Docker ---
if ! command -v docker &>/dev/null; then
  echo "❌ 未找到 docker，请先安装 Docker Desktop"
  exit 1
fi
if ! docker info &>/dev/null; then
  echo "❌ Docker 守护进程未运行，请启动 Docker Desktop"
  exit 1
fi

# --- 检查 docker compose ---
if ! docker compose version &>/dev/null 2>&1; then
  echo "❌ docker compose 不可用，请升级 Docker Desktop"
  exit 1
fi

# --- 清理旧容器 ---
echo "🧹 清理旧容器..."
docker compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true

# --- 启动服务 ---
echo "🚀 启动所有服务..."
docker compose -f "$COMPOSE_FILE" up -d --build

# --- 等待后端就绪 ---
echo "⏳ 等待后端就绪 (最多 60 秒)..."
for i in $(seq 1 60); do
  if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    echo "✅ 后端已就绪"
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "⚠️  后端尚未就绪，检查日志..."
    docker compose -f "$COMPOSE_FILE" logs backend
    exit 1
  fi
  sleep 1
done

echo ""
echo "========================================"
echo "  ✅ 全部就绪！"
echo "========================================"
echo ""
echo "   前端:  http://localhost:3000"
echo "   后端:  http://localhost:8080"
echo "   MinIO: http://localhost:9001  (minioadmin/minioadmin)"
echo ""
echo "   常用命令:"
echo "   查看日志: docker compose -f deployment/docker-compose.full.yml logs -f"
echo "   停止服务: docker compose -f deployment/docker-compose.full.yml down"
echo "   重启服务: docker compose -f deployment/docker-compose.full.yml restart"
echo ""

# 自动打开浏览器
if command -v open &>/dev/null; then
  open "http://localhost:3000"
elif command -v xdg-open &>/dev/null; then
  xdg-open "http://localhost:3000"
fi
