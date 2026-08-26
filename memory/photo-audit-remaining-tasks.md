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

### Phase 3 规划（见 docs/PHASE3-PLAN.md）

#### 1. ES 运营报表聚合（T3-1）
- **触发条件：** 审核记录 > 100 万条 + 报表 PRD 定稿（多维下钻确认）
- **现状：** ES 全文检索已就绪（Phase 2 `836b599d`），dashboard/stats 走 PG 2.7ms 无瓶颈
- **预估：** 后端 200-300 行 + 前端 150 行
- **建议：** 暂不启动，等数据量增长

#### 2. K8s 部署配置（T3-2）
- **触发条件：** 稳定版决定 + CI/CD 流水线搭建
- **现状：** docker-compose 本地开发环境完备
- **预估：** yaml 模板 ~400 行 + Helm chart（可选）

#### 3. 媒体面 SRS 接入（T3-3）
- **触发条件：** 真实用户并发观看到 50+ + 录像回放需求
- **现状：** WHIP/WHEP 信令已解耦（Phase 2 `be5b375f`），媒体面 P2P
- **预估：** 独立 SRS 进程 + 信令适配 ~150 行 + 前端播放器改造 ~100 行
- **建议：** 信令面已预留升级路径，随时可接 SRS/mediamtx

#### 4. Redis 缓存层（T3-4）
- **触发条件：** 高 QPS 场景（> 100 QPS）
- **现状：** Redis 已用于会话 + 电视墙广播
- **预估：** ~200 行（缓存 key 设计 + 失效策略）

---

### 已完成（Phase 2，2026-08-26）

- Kafka 审核任务队列（`ddaf70e1`）✅
- Elasticsearch 全文检索（`836b599d`）✅
- WebRTC WHIP/WHEP 信令（`be5b375f`）✅
- 租户 RBAC 门禁（`447b6763`）✅

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
