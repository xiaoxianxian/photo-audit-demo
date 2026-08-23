---
name: photo-audit-remaining-tasks
description: Photo Audit Platform 剩余未完成任务清单，按优先级排列，供新会话接续开发时使用
metadata: 
  node_type: memory
  type: project
  originSessionId: 44d9d6bc-5db7-4c11-b434-d6344c638306
---

# Photo Audit Platform — 剩余未完成任务清单

## 项目速览

- **工作目录：** `/Users/xiaota/Documents/Photo-Audit-Demo`
- **前端：** `frontend/src/` — React 18 + TypeScript + Ant Design 5.x + Vite 5
- **后端：** `backend/` — Go (Fiber) + pgx + MinIO SDK
- **数据库：** `backend/sql/init.sql` — 21 张表
- **构建状态：** `tsc --noEmit` 0 errors, `vite build` 成功, `grep fmt.Printf` 无结果
- **验收状态：** 2026-06-28 最终验收通过，Phase 1 MVP 全部完成

---

## 已完成批次

- 第 1-16 批：基础功能 + P1/P2 全部修复 ✅
- 第 19 批：WebSocket 审核任务自动分配 ✅
- 第 20 批：审核状态机顶层决策逻辑（5 阶段多维决策引擎）✅
- 第 22 批：AI 模型自动降级 + 文件上传格式/分辨率校验 + Python 死代码清理 ✅
- 第 23 批：直播 RTMP 推流管理 + AI 模型配置页面 + 视频指纹查重 + Dashboard 合并查询 ✅
- 第 24 批：结构化日志 + 进程守护 + 集成测试 ✅
- 第 29 批：AI 模型配置后端持久化（ai_configs 表 + CRUD API + 前端对接）✅
- 第 30 批：技术债务清理（删除 AuditCard.tsx 死组件 + CountByReviewer 子查询优化为 LAG() 窗口函数）✅
- 第 31 批：核心 bug 修复（ingestionLog→contentLog + elementRepo 注入 + is_video 对齐 + jsonb 扫描 + AuditLog 筛选器）✅
- 第 32 批：用户申诉提交 + 注册租户选择 + 数据库约束完善 ✅
- 第 33 批：集成测试修复（txConn 去重 + pgx import + AuditLogRepo exported + fmt.Printf 清理 + 视频上传租户上下文）✅
- 最终验收：2026-06-28，10/10 检查项全部 PASS ✅

---

## 未实现模块

### Phase 2/3 独立子系统（后续规划）

#### 1. 直播 WebRTC 信令
- **现状：** RTMP 推流已完整实现（URL 生成 + 截帧调度 + 健康检查 + 前端管理）
- **需求：** WebRTC 低延迟信令需集成 mediasoup SFU + SDP 交换 + 前端 mediasoup-client
- **预估：** 后端 500-800 行 + 前端 300-500 行 + 独立 mediasoup 服务器部署
- **建议：** 作为 Phase 2 独立规划，不纳入当前 MVP

#### 2. Kafka 审核任务队列
- **需求：** 解耦 AI 审核异步处理，替代 goroutine 直接调用
- **建议：** Phase 2 规划

#### 3. Elasticsearch 全文检索
- **需求：** 审核记录搜索 + 运营报表
- **建议：** Phase 3 规划

---

## 技术债务

无遗留。Phase 1 MVP 全部完成，Phase 2/3 独立子系统需单独规划。

---

## 启动指南

```bash
# 1. 启动基础设施
cd deployment && docker compose up -d

# 2. 初始化数据库
psql -h localhost -U postgres -c "CREATE DATABASE photo_audit;"
psql -h localhost -U postgres -d photo_audit -f ../backend/sql/init.sql

# 3. 启动后端
cd ../backend
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/photo_audit
export JWT_SECRET=dev-secret-change-me
go mod tidy && go build ./... && ./audit-server

# 4. 启动前端（另一个终端）
cd ../frontend
npm install && npm run dev
```
