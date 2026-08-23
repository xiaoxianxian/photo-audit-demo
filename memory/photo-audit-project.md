---
name: photo-audit-project
description: Photo Audit Platform — 供稿审核后台，多租户 AI 审核平台（供稿/短视频/直播），React 18 + TypeScript + Ant Design Pro 前端，Go Fiber 后端，PostgreSQL + MinIO + WebSocket
metadata: 
  node_type: memory
  type: project
  originSessionId: 45e1b6ad-f582-4922-8fe6-969fce3c79ab
---

## 项目定位

支持供稿（图片）、短视频、直播三种内容形态的多租户 AI 审核平台。AI 机审拦截 90% 低质/违规内容，人工专注 10% 边缘争议。

## 技术栈

- **前端：** React 18 + TypeScript + Ant Design Pro 5.x + Vite 5 + Zustand + Axios — 12 页面全部完成
- **后端：** Go (Fiber) + pgx + MinIO SDK + JWT + ffmpeg — 22+ 端点全部就绪
- **数据库：** PostgreSQL 15 — 20 张表，`backend/sql/init.sql`
- **构建验证：** `tsc --noEmit` 0 errors, `vite build` 成功

## 项目路径

- 工作目录：`/Users/xiaota/Documents/Photo-Audit-Demo`
- 前端源码：`frontend/src/`
- 后端源码：`backend/internal/`
- 数据库：`backend/sql/init.sql`
- 部署：`deployment/docker-compose.yml`
- 项目文档：`CLAUDE.md`

## 前端页面清单（全部完成 ✅）

| 页面 | 路由 | 文件 | 状态 |
|------|------|------|------|
| 登录 | `/login` | `pages/Login.tsx` | ✅ |
| 注册 | `/register` | `pages/Register.tsx` | ✅ |
| 仪表盘（租户/团队/看板） | `/` | `pages/Dashboard.tsx` | ✅ |
| 审核工作台（通用） | `/review` | `pages/Review.tsx` | ✅ |
| 短视频审核 | `/review/video` | `pages/ShortVideoReview.tsx` | ✅ |
| 申诉管理 | `/appeals` | `pages/Appeal.tsx` | ✅ |
| 直播电视墙 | `/live-wall` | `pages/LiveWall.tsx` | ✅ |
| 租户配置 | `/tenant-config` | `pages/TenantConfig.tsx` | ✅ |
| 质量抽检 | `/quality-audit` | `pages/QualityAudit.tsx` | ✅ |
| 审核操作日志 | `/audit-log` | `pages/AuditLog.tsx` | ✅ |
| 提交申诉 | `/appeal/new/:contentId` | `pages/SubmitAppeal.tsx` | ✅ |

## 后端关键文件

- `cmd/server/main.go` — Fiber 入口 + DB 连接 + 中间件
- `internal/api/routes.go` — 所有路由注册（/api/v1 前缀）
- `internal/service/services.go` — DI 容器
- `internal/service/video_processor.go` — ffmpeg 抽帧 + ASR 转写
- `internal/service/ingestion_service.go` — 内容上传 + 元素拆分 + AI 异步审核 + **TriggerContentDecision 多维决策引擎**
- `internal/service/ai_service.go` — Agnes AI + DeepSeek 裁判 + 配额检测 + **fallback 自动降级**
- `internal/service/fallback_service.go` — 本地规则引擎（关键词匹配 + 垃圾链接检测 + 噪声检测）
- `internal/service/review_service.go` — 人工审核 + 批量审核 + 申诉改判（含事务）
- `internal/service/appeal_service.go` — 申诉提交 + 状态追踪 + 通知
- `internal/service/dashboard_service.go` — 真实 DB 统计 + 每日趋势
- `internal/service/quality_audit_service.go` — 质检批次 + 抽检记录
- `internal/service/live_wall_service.go` — 直播管理 + WebSocket 推送
- `internal/service/notifier.go` — Notifier 接口 + ConsoleNotifier + MultiNotifier
- `internal/service/websocket_hub.go` — Hub 模式 + BroadcastNewTask + BroadcastToReviewers
- `internal/storage/minio.go` — MinIO 对象存储

## 功能清单核对（截至 2026-06-28）

### 已完成 [x]
- 内容接入：元素拆分、MinIO、文件上传（100MB + 格式/分辨率校验）、短视频上传（视频 MIME 检测 + 抽帧 + ASR 转写）
- AI 机审：多模态路由、NLP、裁判模型、结构化输出、分歧标记、额度检测、异步触发(goroutine)、**自动降级 fallback**
- 人工审核：供稿视图、短视频视图、直播电视墙、分歧高亮、筛选排序、批量审核、**5 阶段多维决策引擎**
- 申诉改判：表单、范围覆盖、一次限制、状态追踪、改判日志、通知、抽检联动
- 质检抽检：批次创建、修正模式、统计、审核员绩效
- 租户配置：RBAC、团队 CRUD、审核规则、判罚等级、敏感词、业务看板
- 审核操作日志查询页面
- WebSocket 审核任务自动分配
- **直播 RTMP 推流管理**（自动生成 RTMP URL + 截帧调度 + 健康检查）
- **AI 模型配置页面**（API Key 管理 + 降级策略 + 实时状态指示 + 后端 ai_configs 表持久化）
- **视频指纹查重**（感知哈希 + simhash 算法集成到视频处理管线）
- **Dashboard 合并查询**（CTE 单次查询替代 8 次独立 SELECT）
- **结构化日志系统**（JSON 格式 stderr 输出，替换所有 fmt.Printf）
- **进程守护**（systemd unit + 独立守护脚本，自动重启 + 防风暴）

### 未实现 [ ]
- 直播 WebRTC 信令（RTMP 推流已完整实现，WebRTC 需额外集成 Coturn/mediasoup）

## 已知陷阱
- 模块路径已统一为 `audit-platform`
- 错误判断使用 `errors.Is(err, pgx.ErrNoRows)` 而非 `strings.Contains`
- Ant Design 5.x Spin 不支持 `wrapperRenderProps`
- Ant Design 5.x Badge 不支持 `onClick` → 改用外层 div 包裹
- Ant Design 5.x Select 不支持 `onClose` prop
- 旧 `.jsx` 文件会与 `.tsx` 冲突（空文件 0 字节），必须删除
- `backend/app/` Python FastAPI 代码已删除（iteration-v0 死代码）

## 构建验证
- `tsc --noEmit` → **0 errors** ✅
- `vite build` → **成功** (~4s) ✅
- `grep fmt.Printf` → **无结果** ✅
- `grep txConn` → **仅 1 处声明** ✅
- 最终验收：**2026-06-28，10/10 检查项全部 PASS** ✅

## 当前状态
- **已完成：** 33 批开发（P1 全部修复 + P2 全部修复 + WebSocket 任务分配 + 审核决策引擎 + AI 自动降级 + 文件上传校验 + Python 死代码清理 + 直播推流 + AI 模型配置后端持久化 + 视频指纹查重 + Dashboard 合并查询 + 结构化日志 + 进程守护 + 集成测试 + 技术债务清理 + 核心 bug 修复 + 用户申诉提交 + 注册租户选择 + 数据库约束完善 + 集成测试修复）
- **P0 阻塞：** 无
- **P1 已知问题：** 全部 4 项已修复 ✅
- **P2 体验优化：** 全部 10 项已修复 ✅
- **技术债务：** 全部清理 ✅
- **未实现模块：** 1 项（WebRTC 信令 — 独立子系统，建议 Phase 2 规划）
- **验收状态：** 2026-06-28 最终验收通过，10/10 检查项全部 PASS ✅
