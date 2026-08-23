---
name: photo-audit-project-roadmap
description: 供稿审核平台剩余未完成任务的分步执行计划和当前进展
type: project
---

# Photo Audit Platform — 剩余任务执行节奏

## 当前状态（截至 2026-06-28）

**已完成的核心链路：**
- 用户认证（注册/登录/JWT）✅
- 租户管理 CRUD + 软删除 ✅
- 团队管理 CRUD + 成员管理 ✅
- 内容上传 + 元素拆分 ✅
- 人工审核工作台（单元素/批量审核）✅
- 申诉管理 + 改判闭环 ✅
- 业务看板（真实 DB 计算 + 租户隔离）✅
- 质检抽检（批次 CRUD + QA 记录）✅
- 直播电视墙（流管理 + WebSocket 广播）✅
- AI 机审引擎（Agnes AI + DeepSeek 裁判 + 分歧标记 + 额度检测）✅
- 注册页面 ✅
- API 路径对齐修复 ✅
- 短视频审核视图 ✅
- 租户配置三表 CRUD（规则/等级/敏感词）✅
- MinIO 对象存储集成 ✅
- 申诉通知机制 ✅
- 审核员绩效真实计算 ✅
- Docker Compose 开发环境 ✅
- P1 问题全面修复（事务/DI/租户隔离/冗余文件）✅
- P2 体验优化（键盘快捷键/趋势图表/待办提醒/离线状态等）✅
- **第十九批：WebSocket 审核任务自动分配** ✅
  - 扩展 `websocket_hub.go`：新增 `BroadcastToTenant`/`BroadcastToReviewers`/`BroadcastNewTask`
  - `IngestionService.TriggerAIReview` 增加 `tenantID` 参数，AI 审核完成后广播新任务
  - `ReviewHandler.WebSocket` 端点（`GET /ws/review`），JWT 鉴权 + 用户注册
  - 前端 `Review.tsx` 接入 WebSocket，实时连接状态指示 + 新任务通知自动刷新

**构建验证：**
- `tsc --noEmit` → 0 errors ✅
- `vite build` → 成功 ✅
- Go 后端：需本地 `cd backend && go build ./...`

**第二十批：审核状态机顶层决策逻辑** ✅
  - 从简单 all-human-done 升级为 5 阶段多维决策引擎：
    1. 强制 reject：单个元素 human_rejected + AI 风险分 ≥ 70
    2. 分歧升级：is_conflict=true 且未人工审核 → under_review
    3. 加权投票：cover_image/live_snapshot 权重 2x，多数票 reject → reject
    4. AI 风险阈值：平均 AI 风险分 > 60 → reject（无需人工审核）
    5. 默认：全部 human_done 且无 reject → approve
  - 涉及文件：仅 `ingestion_service.go`，约 80 行替换
- Step 1：清理冗余 .jsx 文件（13 个已清空）✅
- Step 2：统一 Layout 组件（Layout.tsx + 4 页面重构）✅
- Step 3：审核规则/判罚等级/敏感词三表 CRUD ✅
  - SQL: init.sql 追加 tenant_audit_rules / tenant_audit_levels / tenant_custom_words
  - Model: tenant_rule.go / tenant_level.go / tenant_word.go
  - Repo: rule_repo.go / level_repo.go / word_repo.go（Create/FindByID/ListByTenant/Update/Delete）
  - Service: rule_service.go / level_service.go / word_service.go（含参数校验）
  - Handler: rule_handler.go / level_handler.go / word_handler.go
  - Wiring: services.go + handlers.go + routes.go 全部注入
  - API 路径: /audit-rules /audit-levels /custom-words（均走 tenantMW 隔离）
- Step 4：MinIO 对象存储集成 ✅
  - 新建 `backend/internal/storage/minio.go`
  - config.go 新增 MinIOAccessKey/MinIOSecretKey/MinIOBucket
  - services.go 注入 MinIOStorage（可选）
  - content_handlers.go 新增 UploadFile 端点（multipart 上传）
  - routes.go 注册 /upload/file
  - .env.example 更新 MINIO_BUCKET
- Step 5：申诉通知机制 ✅
  - 新建 `backend/internal/service/notifier.go`（Notifier 接口 + ConsoleNotifier + MultiNotifier）
  - ReviewService.ResolveAppeal / AppealService.SubmitAppeal 增加通知调用
- Step 6：审核员绩效真实计算 ✅
  - LogRepository.CountByReviewer（JOIN users，COUNT FILTER，AVG 耗时）
  - DashboardService.GetReviewerPerformance 替换空返回
  - DashboardHandler.GetReviewerPerformance 简化参数

**剩余任务按执行顺序排列：**

---

## 下一会话执行计划（按优先级）

### 第二十批：审核状态机顶层决策逻辑 ~~已完成~~ ✅
~~**目标：** 改进 `TriggerContentDecision`，从简单的 all-human-done 判断升级为多维决策。~~

**实现内容：**
- 5 阶段决策引擎（按优先级降序执行）：
  1. **强制 reject**：单个元素 human_rejected + AI 风险分 ≥ 70
  2. **分歧升级**：is_conflict=true 且未人工审核 → under_review
  3. **加权投票**：cover_image/live_snapshot 权重 2x，多数票 reject → reject
  4. **AI 风险阈值**：平均 AI 风险分 > 60 → reject（无需人工审核）
  5. **默认**：全部 human_done 且无 reject → approve
- 涉及文件：仅 `ingestion_service.go`，约 80 行替换
- 验证：`tsc --noEmit` 0 errors ✅

### 第二十一批：真实 Agnes AI + DeepSeek 集成（高优先级）
**目标：** 替换 `ai_service.go` 中的 mock 响应为真实 API 调用
**前提：** 需要有效的 API Key（AGNES_API_KEY / DEEPSEEK_API_KEY）
**涉及文件：** `ai_service.go`（~350 行）

### 第二十二批：直播 RTMP/WebRTC 推流接入（中优先级）
**目标：** 集成 SRS/mediaserver 或 WebRTC 信令
**现状：** 后端仅有模拟管理接口
**涉及文件：** 新建 `backend/internal/service/rtmp_service.go` + 修改 `live_wall_handlers.go`

### 第二十三批：AI 模型自动降级（中优先级）
**目标：** 检测到 402/429 时自动切换备用模型
**现状：** `DetectQuotaError` 已返回错误但未实现切换
**涉及文件：** `ai_service.go` + `config.go` + `services.go`

### 第二十四批：AI 模型配置页面（中优先级）
**目标：** 前端页面管理 API Key 和模型切换
**现状：** 无前端页面
**涉及文件：** 新建 `AIConfig.tsx` + `content-api.ts` + `App.tsx` + `Layout.tsx`

### 第二十五批：视频指纹查重 simhash（低优先级）
**目标：** 实现感知哈希算法比对重复视频
**现状：** `video_fingerprint` 字段已定义但从未填充
**涉及文件：** `video_processor.go` + `content_repo.go`

### 第二十六批：格式/分辨率校验（低优先级）
**目标：** 增加图片格式校验（JPEG/PNG/GIF）、视频分辨率校验
**现状：** 仅 100MB 大小校验
**涉及文件：** `content_handlers.go` + `ingestion_service.go`

---

## Step 1：清理冗余 .jsx 文件 ~~已完成~~ ✅

~~**目标：** 删除旧版 .jsx 文件，保持代码整洁，避免混淆。~~

**涉及文件（13 个）：**
- frontend/src/components/Layout.jsx
- frontend/src/main.jsx
- frontend/src/App.jsx
- frontend/src/pages/Dashboard.jsx
- frontend/src/pages/Login.jsx
- frontend/src/pages/Statistics/SafetyStats.jsx
- frontend/src/pages/Supply/Submit.jsx
- frontend/src/pages/Supply/Detail.jsx
- frontend/src/pages/Supply/List.jsx
- frontend/src/pages/Audit/SafetyQueue.jsx
- frontend/src/pages/Audit/QualityQueue.jsx
- frontend/src/pages/Appeal/List.jsx
- frontend/src/router/index.jsx

**操作步骤：**
1. 确认对应的 .tsx/.ts 文件已存在且功能完整
2. 逐一删除 .jsx 文件
3. 检查是否有 import 引用了 .jsx 文件（应该没有）
4. git rm 提交

**验证：** `find frontend/src -name "*.jsx"` 应返回空

---

## Step 2：统一 Layout 组件 ~~已完成~~ ✅

~~**目标：** 抽取共享的侧边栏导航组件，消除各页面重复的 Sider 代码。~~

**现状：** Review.tsx、Appeal.tsx、Dashboard.tsx、LiveWall.tsx 各自写了独立的 Sider，样式不一致。

**操作步骤：**
1. 创建 `frontend/src/components/Layout.tsx`，包含：
   - 侧边栏菜单（仪表盘/审核工作台/申诉管理/直播电视墙）
   - 用户信息展示
   - 退出登录按钮
   - 当前选中路由高亮
2. 各页面移除内联 Sider，改用 `<Layout>` 包裹内容区域
3. 保持深色主题风格（#0a0a0a 背景）

**涉及修改：**
- 新建：frontend/src/components/Layout.tsx
- 修改：Dashboard.tsx、Review.tsx、Appeal.tsx、LiveWall.tsx

**验证：** 四个页面的侧边栏样式一致，点击菜单可跳转，退出登录正常

---

## Step 3：补充数据库表 — 审核规则 / 判罚等级 / 敏感词 ~~已完成~~ ✅

~~**目标：** 建表并实现 CRUD，让租户配置真正可用。~~

**需要新建的表：**
```sql
-- tenant_audit_rules: 租户审核规则
CREATE TABLE tenant_audit_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    rule_name VARCHAR(128) NOT NULL,
    rule_expression TEXT,
    action VARCHAR(32) NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- tenant_audit_levels: 判罚等级配置
CREATE TABLE tenant_audit_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    level_code VARCHAR(32) NOT NULL UNIQUE,
    level_name VARCHAR(64) NOT NULL,
    description TEXT,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- tenant_custom_words: 租户自定义敏感词
CREATE TABLE tenant_custom_words (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    word VARCHAR(256) NOT NULL,
    category VARCHAR(32),
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**操作步骤：**
1. 在 init.sql 末尾追加三张表的 CREATE 语句
2. 创建 model 文件：tenant_rule.go、tenant_level.go、tenant_word.go
3. 创建 repository：rule_repo.go、level_repo.go、word_repo.go
4. 创建 service：rule_service.go、level_service.go、word_service.go
5. 创建 handler：rule_handler.go、level_handler.go、word_handler.go
6. 注册路由到 routes.go

**验证：** 能通过 API 增删改查规则/等级/敏感词

---

## Step 4：MinIO 对象存储集成

**目标：** 让文件上传有实际落盘能力。

**操作步骤：**
1. 创建 `backend/internal/storage/minio.go`
   - 初始化 MinIO 客户端
   - CreateBucketIfNotExists（按租户隔离）
   - UploadObject / DownloadObject / PresignedURL
2. 修改 IngestionService.UploadContent，上传文件到 MinIO 后获取 URL
3. 前端 Upload 组件对接真实的文件上传接口（multipart/form-data）
4. 更新 config.go 添加 MINIO_ENDPOINT 等字段

**涉及修改：**
- 新建：backend/internal/storage/minio.go
- 修改：backend/internal/service/ingestion_service.go
- 修改：前端上传组件

**验证：** 上传图片 → 存入 MinIO → 返回 URL → 审核工作台能加载图片

---

## Step 5：申诉通知机制 ~~已完成~~ ✅

~~**目标：** 申诉提交/处理后通知申诉人。~~

**现状：** AppealService 没有通知逻辑。

**操作步骤：**
1. 创建 `backend/internal/service/notifier.go`
   - 定义 Notifier 接口
   - 实现 ConsoleNotifier（console.log 占位，方便开发调试）
   - 预留 EmailNotifier / PushNotifier 接口
2. 在 AppealService.SubmitAppeal 中调用 notifier.NotifyAppealSubmitted
3. 在 ReviewService.ResolveAppeal 中调用 notifier.NotifyAppealResolved

**涉及修改：**
- 新建：backend/internal/service/notifier.go
- 修改：backend/internal/service/appeal_service.go
- 修改：backend/internal/service/review_service.go

**验证：** 提交申诉和处理申诉时在日志中看到通知记录

---

## Step 6：审核员绩效真实计算 ~~已完成~~ ✅

~~**目标：** 替换 DashboardService.GetReviewerPerformance 的空返回。~~

**操作步骤：**
1. 在 LogRepository 添加按 reviewer_id 分组的统计查询
2. 关联 users 表获取 reviewer_name
3. 返回分页数据 {items, total}

**涉及修改：**
- 修改：backend/internal/repository/log_repo.go（新增 CountByReviewer 方法）
- 修改：backend/internal/service/dashboard_service.go

**验证：** 看板"审核员绩效"表格显示真实数据

---

## Step 7：Docker Compose 开发环境 ~~已完成~~ ✅

~~**目标：** 一键启动 PostgreSQL + Redis + MinIO。~~

**操作步骤：**
1. 创建 deployment/docker-compose.yml
2. 包含服务：postgres:15, redis:7, minio/minio
3. 配置网络和数据卷
4. 更新 .env.example 添加对应环境变量

**涉及文件：**
- 新建：deployment/docker-compose.yml
- 修改：backend/.env.example

---

---

## Step 8：WebSocket 审核任务自动分配 ~~已完成~~ ✅

~~**目标：** 扩展 WebSocket Hub 支持审核任务自动推送给在线审核员。~~

**现状：** `websocket_hub.go` 仅用于直播电视墙广播。

**操作步骤：**
1. 扩展 `websocket_hub.go`：新增 `BroadcastToTenant`/`BroadcastToReviewers`/`BroadcastNewTask` 方法
2. `IngestionService.TriggerAIReview` 增加 `tenantID` 参数 + `wsHub` 引用
3. AI 审核完成后，如果有 pending_human 元素，广播 `new_task` 消息
4. `ReviewHandler.WebSocket` 端点：JWT 鉴权 + 用户注册到 Hub
5. 前端 `Review.tsx` 接入 WebSocket，显示连接状态 + 新任务通知

**涉及修改：**
- 修改：`websocket_hub.go`（重写 Hub 支持 tenant/role 筛选）
- 修改：`ingestion_service.go`（增加 tenantID + wsHub）
- 修改：`review_service.go`（增加 wsHub 引用）
- 修改：`services.go`（调整 DI 顺序，注入 wsHub）
- 修改：`review_handlers.go`（增加 WebSocket 方法 + authSvc 注入）
- 修改：`content_handlers.go`（传递 tenantID 给 TriggerAIReview）
- 修改：`live_wall_handlers.go`（更新 Register 调用）
- 修改：`routes.go`（注册 /ws/review 路由）
- 修改：`handlers.go`（传递 authSvc 给 ReviewHandler）
- 修改：`Review.tsx`（WebSocket 连接 + 新任务通知 UI）

**验证：** `tsc --noEmit` 0 errors ✅

---

## 剩余未实现模块（全新功能，按优先级排列）

### 高优先级
1. **真正的 Agnes AI + DeepSeek 集成** — `AIService` 使用 mock 响应，需要对接真实 API
2. **审核状态机顶层决策逻辑** — ✅ 已完成（多维决策引擎）

### 中优先级
3. **直播 RTMP/WebRTC 推流接入** — 后端仅有模拟管理接口
4. **自动降级（AI 模型切换）** — 检测到 402/429 返回 `DetectQuotaError`，但未实现切换备用模型
5. **AI 模型配置页面** — 无前端页面管理 API Key 和模型切换

### 低优先级
6. **视频指纹查重（simhash）** — `video_fingerprint` 字段已定义但从未填充
7. **格式/大小/分辨率校验** — 仅 100MB 大小校验

## 技术债务
1. `dashboard_service.go GetStats` — 8 次独立 SELECT 顺序执行，可合并为单次查询
2. `log_repo.go CountByReviewer` — 相关子查询在大数据量下性能差，应改用 LAG() 窗口函数
3. `logger.go` 中间件 — 定义了但未使用（main.go 用的是 Fiber 内置 logger）
4. `AuditCard.tsx` — 前端定义了但未引用的组件（Review.tsx 使用内联 ElementCard）
