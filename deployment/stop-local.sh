#!/usr/bin/env bash
set -euo pipefail

echo "🛑 停止本地服务..."

# 杀死所有 audit-server 进程
pkill -f "audit-server" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true

echo "✅ 已停止所有服务"
