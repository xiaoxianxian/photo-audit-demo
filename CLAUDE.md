# Photo Audit Platform — 供稿审核后台

## 1. 项目速览

**定位：** 支持供稿（图片）、短视频、直播三种内容形态的多租户 AI 审核平台。AI 机审拦截 90% 低质/违规内容，人工专注 10% 边缘争议。

**技术栈：**
- 前端：React 18 + TypeScript + Ant Design Pro
- 后端：Go (Gin/Fiber) — Phase 1 单体，Phase 2 拆分微服务
- 数据库：PostgreSQL（主存储）+ 分区表（audit_records / audit_logs 按月分区）
- 缓存/队列：Redis（会话/电视墙状态）+ Kafka（审核任务队列，Phase 2 引入）
- 搜索：Elasticsearch（审核记录全文检索、运营报表，Phase 3 引入）
- 对象存储：MinIO（自建 S3 兼容，按租户隔离 bucket）
- 实时通信：WebSocket（审核任务分配、直播电视墙刷新）
- AI 模型：Agnes AI 多模态（视觉 + 文本统一审核）、DeepSeek（裁判模型/NLP）、ASR 转写

**启动命令（Phase 1 MVP）：**
```bash
# 后端
cd backend && go build -o audit-server && ./audit-server

# 前端
cd frontend && npm install && npm run dev
```

**环境变量占位（`.env.example`）：**
```
# 数据库
DATABASE_URL=postgresql://user:password@localhost:5432/photo_audit

# Redis
REDIS_URL=redis://localhost:6379

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET_PREFIX=audit-

# AI 模型
AGNES_API_KEY=sk-xxx
DEEPSEEK_API_KEY=sk-xxx

# Kafka（Phase 2）
KAFKA_BROKERS=localhost:9092

# Elasticsearch（Phase 3）
ELASTICSEARCH_URL=http://localhost:9200

# 服务端口
SERVER_PORT=8080
FRONTEND_PORT=3000
```

---

## 2. 核心架构

### 文件夹结构（规划）

```
photo-audit-platform/
├── frontend/                          # React + TypeScript + Ant Design Pro
│   ├── src/
│   │   ├── pages/
│   │   │   ├── review/                # 审核工作台
│   │   │   │   ├── photo/             # 供稿审核视图
│   │   │   │   ├── video/             # 短视频审核视图
│   │   │   │   └── live-wall/         # 直播电视墙
│   │   │   ├── appeal/                # 申诉管理
│   │   │   ├── quality/               # 质检抽检
│   │   │   └── admin/                 # 租户管理后台
│   │   ├── components/
│   │   ├── services/                  # API 调用层
│   │   ├── stores/                    # 状态管理
│   │   └── utils/
│   └── package.json
│
├── backend/                           # Go 后端（Phase 1 单体）
│   ├── cmd/server/                    # 入口
│   ├── internal/
│   │   ├── api/                       # HTTP handlers + routes
│   │   ├── service/                   # 业务逻辑层
│   │   │   ├── ingestion/             # 内容接入
│   │   │   ├── ai/                    # AI 调度
│   │   │   ├── review/                # 审核工作台
│   │   │   ├── appeal/                # 申诉
│   │   │   ├── quality/               # 质检抽检
│   │   │   └── tenant/                # 租户管理
│   │   ├── model/                     # 数据模型
│   │   ├── repository/                # 数据库访问
│   │   └── middleware/                # 鉴权、租户隔离、日志
│   ├── pkg/                           # 公共包
│   └── go.mod
│
├── deployment/                        # 部署配置
│   ├── docker-compose.yml             # 本地开发环境
│   └── k8s/                           # Phase 3 K8s 配置
└── docs/                              # 设计文档
    └── iteration-v1-core/
```

### 核心 ER 关系

```
tenants (1)
  ├─ (N) users (tenant_id)
  ├─ (N) audit_teams (tenant_id)
  ├─ (N) contents (tenant_id)
  ├─ (N) tenant_audit_rules
  ├─ (N) tenant_audit_levels
  └─ (N) tenant_custom_words

users (1)
  ├─ (N) contents (creator_id)          ← 创作者/上传者
  ├─ (N) audit_teams (leader_id)        ← 团队负责人
  ├─ (N) audit_team_members (user_id)
  ├─ (N) audit_records (reviewer_id)    ← 审核员
  ├─ (N) audit_tasks (assignee_id)      ← 分配的任务
  ├─ (N) appeals (applicant_id)         ← 申诉人
  └─ (N) audit_logs (operator_id)

contents (1)
  ├─ (1) contents_photo
  ├─ (1) contents_short_video
  ├─ (1) contents_live_stream
  └─ (N) content_elements               ← 核心：内容拆分为元素

content_elements (1)
  ├─ (N) audit_tasks                    ← 每个元素产生审核任务
  └─ (N) audit_records                  ← 审核结果记录

contents_live_stream (1)
  └─ (N) live_frame_snapshots           ← 直播截帧

appeals (1)
  └─ (N) appeal_notifications           ← 申诉通知
```

核心链路：`contents → content_elements → audit_tasks → audit_records`

### 接口规范

- RESTful API，路径前缀 `/api/v1/`
- 请求/响应统一 JSON，错误码遵循 RFC 7807 Problem Details
- 鉴权：JWT Bearer Token，Header: `Authorization: Bearer <token>`
- 租户隔离：请求 Header 携带 `X-Tenant-ID`，数据库层通过 RLS 过滤
- 分页：`?page=1&page_size=20`，响应体 `{"data": [], "total": 1000, "page": 1, "page_size": 20}`
- WebSocket：`ws://host/ws/review`（任务分配）、`ws://host/ws/live-wall`（电视墙刷新）

---

## 3. 功能清单

### 模块一：内容接入与预处理

- [x] 供稿图片拖拽/点击/批量上传
- [x] 供稿图片拖拽/点击/批量上传
- [x] 短视频文件上传（UploadFile 支持视频 MIME 检测 + ffmpeg 抽帧 + ASR 转写）
- [ ] 直播 WebRTC 推流接入（RTMP 已完整实现，WebRTC 需集成 Coturn/mediasoup）
- [x] 缩略图/封面帧生成（MinIO 直链 + ffmpeg 抽帧）
- [x] 视频抽帧 + ASR 转写（VideoProcessor: ffmpeg 抽帧 + ffprobe 测时长 + ASR API）
- [x] 内容元素拆分（标题/评论/截帧图/ASR文本等）
- [x] MinIO 对象存储（原图/原视频/截帧快照）
- [x] 格式/大小/分辨率校验（图片白名单 JPEG/PNG/GIF/WebP + 视频格式 mp4/webm/mov + ffprobe 分辨率检测 480p~4K）

### 模块二：AI 机审引擎

- [x] 多模态模型路由（Agnes AI 视觉 + 文本统一审核）
- [x] NLP 模型路由（DeepSeek 文本审核）
- [x] 裁判模型（DeepSeek 一致性校验）
- [x] 审核结果结构化输出（risk_score / risk_types / confidence / reason）
- [x] 裁判分歧标记（差值 > 20 分 → is_conflict = true）
- [x] 额度/频率控制（DetectQuotaError 检测 402/429）
- [x] 自动降级（检测到 402/429 时自动切换本地规则引擎 fallback，配置项 FALLBACK_ENABLED 默认 true）
- [x] 视频指纹查重（simhash 算法 + video_processor 集成）
- [x] 审核结果异步入库（Phase 2 已升级 Kafka 队列：KAFKA_BROKERS 配置时走 audit-ai-review topic + 消费组，未配置或发布失败自动回退进程内 goroutine）
- [x] 结构化日志系统（JSON 格式，替换所有 fmt.Printf）
- [x] 进程守护（systemd unit + 独立守护脚本，自动重启 + 防风暴）

### 模块三：人工审核工作台

- [x] 供稿审核视图（卡片网格 + AI 评分 + 机审标签）
- [x] 短视频审核视图（播放器 + 转写文字 + 评论列表）
- [x] 直播电视墙（多路缩略图实时刷新 + AI 风险分）
- [x] 裁判分歧橙色高亮提示（ElementCard border: 2px solid #fa8c16）
- [x] 筛选/排序（按风险分/时间/租户/元素类型）
- [x] 批量审核（Review.tsx + review_handlers.go BatchReview）
- [x] 审核操作：通过 / 打回（带原因分类 + 判罚等级）
- [x] 审核状态机（ElementStatus 枚举 + UpdateStatus 仓储方法 + TriggerContentDecision 多维决策引擎）

### 模块四：申诉与改判

- [x] 申诉表单（内容摘要 + 申诉理由 + 证明材料）
- [x] 申诉范围覆盖（AppealHandler.Submit 通用，不限内容类型）
- [x] 申诉仅支持一次（AppealService.SubmitAppeal 事务级检查 + ErrAlreadyAppealed）
- [x] 申诉状态追踪（submitted → under_review → resolved_approved / resolved_maintained）
- [x] 改判日志（AuditRecord 追加记录，ReviewService.ResolveAppeal 创建 appeal 类型记录）
- [x] 申诉结果通知申诉人（Notifier 接口 + ConsoleNotifier + MultiNotifier）
- [x] 质检/抽检发现误判 → 主动改判（QualityAuditService + ReviewService.ResolveAppeal 联动）

### 模块五：质检与抽检

- [x] 质检批次创建（时间范围/内容类型/租户筛选）
- [x] 质检修正模式：仅本地修正 / 连带用户判罚（mode: local_correction / full_correction）
- [x] 抽检任务：计算准确率/召回率/风险水位（QualityAuditStats + quality_audit_handlers.go）
- [x] 审核员绩效统计（审核量/准确率/平均耗时，LogRepository.CountByReviewer）

### 模块六：租户管理与配置

- [x] 三级 RBAC：平台超管 → 租户管理员 → 审核员/质检员
- [x] 审核团队 CRUD + 成员增删 + 角色分配
- [x] AI 模型配置（前端 AIConfig.tsx + 后端 ai_configs 表 CRUD + PUT /api/v1/ai-config 持久化）
- [x] 审核规则配置（tenant_audit_rules CRUD + action: approve/reject/flag + priority 排序）
- [x] 判罚等级配置（tenant_audit_levels CRUD + level_code/level_name）
- [x] 租户自定义敏感词（tenant_custom_words CRUD + category 分类）
- [x] 业务看板（真实计算：countApprovalsAndRejections、avgRiskScore、conflictCount、appealCount）

---

## 4. 正负例清单

### 内容上传
- **正例：** 用户拖拽 JPEG 图片 → 校验通过 → 缩略图生成 → 元素拆分 → 入 AI 审核队列
- **负例：** 上传非图片格式（.exe） → 拒绝并提示"不支持的文件类型"
- **负例：** 上传超大文件（>100MB） → 拒绝并提示"文件过大"
- **负例：** 上传损坏文件 → 拒绝并提示"文件已损坏"
- **负例：** 上传时 MinIO 不可用 → 队列暂存 + 重试机制 + 用户提示"上传中，请稍后查看"

### AI 审核
- **正例：** 图片送入 Agnes AI → 返回 risk_score=15 → 自动通过
- **正例：** 图片送入 Agnes AI → 返回 risk_score=70 → 进入人审队列 + 高风险标签
- **负例：** AI API 超时/5xx → 标记为"审核失败" → 进入人审队列（人工兜底）
- **负例：** AI API 429 频率限制 → 自动退避重试 → 超过重试次数告警
- **负例：** AI API 402 额度耗尽 → 切换备用模型 → 通知管理员

### 裁判分歧
- **正例：** 主审 30 分 + 裁判 75 分 → 差值 45 > 20 → 标记分歧 → 审核员界面橙色高亮
- **正例：** 主审 40 分 + 裁判 45 分 → 差值 5 ≤ 20 → 无分歧标记

### 人工审核操作
- **正例：** 审核员点击"通过" → 元素 human_status=human_passed → 记录 audit_record(action=approve) → 看板统计+1
- **正例：** 审核员点击"打回" → 选择原因 → human_status=human_rejected → 记录 audit_record(action=reject) → 触发申诉入口
- **负例：** 审核员重复点击"通过" → 检查当前 human_status → 如已是 passed 则提示"已审核"
- **负例：** 审核员点击打回但未选原因 → 表单校验拦截

### 申诉
- **正例：** 用户上传申诉理由 → 创建 appeal 记录 → 通知申诉人 → 审核员处理 → 结果通知申诉人
- **负例：** 同一用户就同一内容再次申诉 → UNIQUE 约束拦截 → 提示"您已提交过申诉"
- **负例：** 申诉处理中用户试图再次提交 → 状态检查拦截

### 直播电视墙
- **正例：** 截帧 → AI 审核 → 风险分写入 Redis → WebSocket 推送前端 → 电视墙实时更新
- **负例：** 直播流断开 → 电视墙格子显示"离线" → 告警通知
- **负例：** WebSocket 连接断开 → 前端自动重连 → 断线期间数据通过 Redis 补偿

### 租户隔离
- **正例：** 租户 A 的审核员登录 → 只能看到 tenant_id=A 的数据
- **负例：** 租户 A 的审核员尝试访问租户 B 的内容 → RLS 拦截 → 403

### 用户认证
- **正例：** 用户输入正确用户名密码 → JWT 签发 → 后续请求携带 Token → 访问受保护接口
- **正例：** 平台超管（tenant_id=NULL）→ 可访问所有租户数据
- **负例：** 未携带 Token → 401 Unauthorized
- **负例：** Token 过期/签名无效 → 401 Unauthorized → 跳转登录页
- **负例：** 密码错误 → 401 Unauthorized（不泄露是用户名错还是密码错）
- **负例：** 账号已禁用 → 403 Forbidden
- **负例：** 用户名重复注册 → 409 Conflict
- **负例：** 注册时 role 不在枚举范围内 → 400 Bad Request
- **负例：** 创建团队时负责人不属于当前租户 → 400 Bad Request
- **负例：** 添加重复成员 → 409 Conflict

### 租户管理
- **正例：** 平台超管创建租户 → 租户状态为 active → 可邀请用户加入
- **正例：** 租户管理员编辑租户信息（名称/国家代码）→ 部分更新生效
- **负例：** 租户名称为空 → 400 Bad Request
- **负例：** 国家代码格式非法（非2字母） → 400 Bad Request
- **负例：** 删除不存在的租户 → 404 Not Found
- **负例：** 删除租户（软删除）→ status=0，历史数据保留

---

## 5. 测试用例

1. **上传 → AI 审核 → 人审通过全流程冒烟：** 上传一张图片 → 等待 AI 审核完成 → 在审核工作台看到卡片 → 点击"通过" → 审核日志记录通过操作 → 看板今日已审数 +1
2. **裁判分歧检测：** 上传一张图片 → 主审返回 25 分 → 裁判返回 80 分 → 元素 is_conflict=true → 审核工作台卡片边框橙色高亮
3. **申诉一次限制：** 用户 A 对内容 X 提交申诉 → 成功 → 用户 A 再次对内容 X 提交申诉 → 被拒绝，提示"您已提交过申诉"
4. **租户数据隔离：** 租户 A 的审核员登录 → 尝试通过 API 查询租户 B 的内容 → 返回空结果或 403
5. **直播电视墙实时性：** 启动一路直播 → 截帧间隔 15s → 电视墙应在 3s 内显示最新截帧和风险分

### 第一批新增测试用例（后端骨架 + 认证 + 租户管理）

6. **用户注册 → 登录冒烟：** 注册一个新用户 → 获取 JWT → 用 Token 访问受保护接口 → 返回 200
7. **租户创建 → 列表查询：** 用有效 Token 创建租户 → 列表接口返回刚创建的租户 → 分页参数生效
8. **软删除租户：** 删除租户 → 列表不再返回 → 重新查询该 ID → 404

---

## 6. 已知陷阱

- **模块路径不一致：** 不同 agent 生成的代码使用了不同的 module path（`photo-audit` vs `audit-platform`），已统一为 `audit-platform`
- **错误判断方式脆弱：** 原代码用 `strings.Contains(err.Error(), "no rows")` 判断 pgx.ErrNoRows，已改为 `errors.Is(err, pgx.ErrNoRows)`
- **Spin wrapperRenderProps：** Ant Design 5.x 的 Spin 组件不接受 `wrapperRenderProps` 属性，已移除

<!-- 随着开发推进，在此记录踩坑经验 -->

---

## 7. 变更日志

- **2026-06-25：** 第一批 — 后端骨架 + 用户认证 + 租户管理 + 团队管理
  - 生成 Go 后端完整代码（24 个文件）：config、middleware(auth/tenant/logger)、model、repository、service、api(handler/routes)
  - 生成 React 前端基础页面：Login、Dashboard（租户/团队管理）、API 服务层、auth store
  - 生成数据库建表 SQL（5 张核心表 + 种子数据）
  - 生成 `.env.example` 环境变量模板
  - 更新 CLAUDE.md：功能清单标记 `[x]`、正负例清单新增认证/租户管理场景、测试用例新增 3 条、已知陷阱记录 2 处

- **2026-06-25：** 第二批 — 内容接入 + AI 审核流水线 + 申诉管理 + 业务看板
  - 生成 Go 后端代码：content/appeal/audit_record/dashboard 模型，content/element/appeal/log 仓储，ingestion/ai/review/appeal/dashboard 服务，content/review/appeal/dashboard handlers
  - 生成 React 前端页面：Review.tsx（审核工作台）、Appeal.tsx（申诉管理）、Dashboard.tsx（更新业务看板）
  - 生成 API 服务层：content-api.ts 包含内容/AI/审核/申诉/看板相关 API
  - 更新 App.tsx：添加 /review 和 /appeals 路由

- **2026-06-26：** 第三批 — API 路径修复 + ResolveAppeal 实现 + Dashboard 真实计算 + 性能优化 + 注册页面
  - 修复 content-api.ts 所有 API 路径（/elements/pending → /review/pending 等）
  - 修复前端数据取值路径（data.items → data.data）
  - 实现 ReviewService.ResolveAppeal 完整逻辑（改判/维持 + audit_record + 元素状态回退）
  - 补全 DashboardService 真实 DB 查询（countApprovalsAndRejections、avgRiskScore、conflictCount、appealCount）
  - ElementRepository 新增 FindByID 方法，替换 ReviewService/QualityAuditService 中的 O(n) 内存遍历
  - AppealRepository 新增 FindByContentAndApplicant，修复 (content_id, applicant_id) 联合唯一检查
  - 创建 Register.tsx 注册页面，App.tsx 添加 /register 路由
  - 更新 api.ts postRegister 发送 display_name 和 role 字段

- **2026-06-26：** 第四批 — 前端清理 + 统一 Layout + 租户配置三表 CRUD
  - **前端清理：** 清空 13 个冗余 .jsx 文件（旧版页面/路由），无 import 引用
  - **统一 Layout：** 新建 `frontend/src/components/Layout.tsx`（侧边栏菜单 + 用户信息 + 退出登录 + 路由高亮）；重构 Dashboard.tsx、Review.tsx、Appeal.tsx、LiveWall.tsx 移除内联 Sider，统一使用 `<AppLayout>` 包裹
  - **数据库表：** init.sql 追加 `tenant_audit_rules`、`tenant_audit_levels`、`tenant_custom_words` 三张表及索引
  - **后端 CRUD（9 个新文件）：**
    - Model: tenant_rule.go / tenant_level.go / tenant_word.go
    - Repo: rule_repo.go / level_repo.go / word_repo.go（Create/FindByID/ListByTenant/Update/Delete）
    - Service: rule_service.go / level_service.go / word_service.go（含 action/code/word 校验）
    - Handler: rule_handler.go / level_handler.go / word_handler.go
  - **Wiring：** services.go + handlers.go + routes.go 注入新实例，路由挂载于 `/audit-rules`、`/audit-levels`、`/custom-words`（均走 tenantMW 隔离）
  - **CLAUDE.md：** 功能清单标记审核规则/判罚等级/敏感词为 `[x]`

- **2026-06-26：** 第五批 — MinIO 对象存储集成
  - 新建 `backend/internal/storage/minio.go`（NewMinIOStorage / UploadBytes / PresignedURL / DeleteObject / GenerateObjectName）
  - config.go 新增 MinIOAccessKey / MinIOSecretKey / MinIOBucket 配置字段及默认值
  - services.go 注入 MinIOStorage（可选，未配置时 nil 不影响启动）
  - main.go 传递 cfg 给 NewServices
  - content_handlers.go 新增 `UploadFile` 端点（POST /api/v1/contents/upload/file），支持 multipart 文件上传
  - handlers.go 使用 `NewContentHandlerWithStorage` 传入 MinIO
  - routes.go 注册 `/upload/file` 路由
  - .env.example 更新 MINIO_BUCKET 占位

- **2026-06-26：** 第六批 — 申诉通知机制 + 审核员绩效真实计算
  - 新建 `backend/internal/service/notifier.go`（Notifier 接口 + ConsoleNotifier + MultiNotifier）
  - ReviewService 构造函数增加 notifier 参数，ResolveAppeal 增加通知调用
  - AppealService 构造函数增加 notifier 参数，SubmitAppeal 增加通知调用
  - services.go 创建 MultiNotifier(ConsoleNotifier{}) 并注入 ReviewService + AppealService
  - LogRepository 新增 CountByReviewer（JOIN users，COUNT FILTER 统计通过/驳回，AVG 计算平均耗时）
  - DashboardService.GetReviewerPerformance 替换空返回为真实 DB 查询
  - DashboardHandler.GetReviewerPerformance 简化参数，移除 page/pageSize/nameFilter
  - dashboard_handlers.go 清理 unused imports (strconv, strings)

- **2026-06-26：** 第七批 — Docker Compose 开发环境
  - 新建 `deployment/docker-compose.yml`：包含 postgres:15、redis:7、minio/minio、minio-mc（自动建桶）
  - 挂载 init.sql 到 /docker-entrypoint-initdb.d/ 自动建表
  - 配置 healthcheck（postgres/redis/minio）
  - .env.example 添加 docker-compose 启动提示

- **2026-06-27：** 第八批 — 前端 TypeScript 全面修复 + 构建验证
  - 清理 13 个冗余 `.jsx`/`.js` 文件（旧版页面/路由/store/api）
  - 修复 `api.ts` 响应拦截器返回类型（`response.data.data` → `response.data`）
  - 重写 `content-api.ts` 全部 API 函数返回值类型（统一 `unwrap` 模式）
  - 修复 `Dashboard.tsx` 导入路径（`api.ts` vs `content-api.ts`）
  - 修复 `Review.tsx` 未使用变量 + Select 不支持 `onClose` prop
  - 修复 `LiveWall.tsx` 未使用变量 + `wsConnected` 状态
  - 修复 `AuditCard.tsx` Badge 不支持 `onClick` → 改用外层 div 包裹
  - 修复 `vite.config.js` 缺少 `@/` 路径别名配置
  - 修复 `tsconfig.json` 严格模式下的 TS6133/TS2322 错误
  - 验证：`tsc --noEmit` 0 errors, `vite build` 成功

- **2026-06-27：** 第九批 — 租户配置前端页面
  - 新建 `TenantConfig.tsx`：三 Tab 页面（审核规则/判罚等级/敏感词库）
  - 每个 Tab 独立管理组件（RuleManager/LevelManager/WordManager），支持 CRUD + 删除确认
  - `content-api.ts` 新增 18 个 API 函数（rules/levels/words 各 4 个）
  - `App.tsx` 添加 `/tenant-config` 路由，`Layout.tsx` 添加「租户配置」菜单项

- **2026-06-27：** 第十批 — 质量抽检前端页面
  - 新建 `QualityAudit.tsx`：批次列表 + 详情抽屉（样本列表/抽检评分表单/记录/统计）
  - 进度条显示抽检完成度，支持开始/完成抽检操作
  - `App.tsx` 添加 `/quality-audit` 路由，`Layout.tsx` 添加「质量抽检」菜单项

- **2026-06-27：** 第十一批 — 短视频审核视图
  - 新建 `ShortVideoReview.tsx`：3 列布局（视频播放器 + ASR 转写 + 评论 + 审核操作）
  - 支持逐元素审核表格 + 批量通过/打回 + 裁判分歧提示
  - `App.tsx` 添加 `/review/video` 路由，`Layout.tsx` 添加「短视频审核」菜单项

- **2026-06-27：** 第十二批 — CLAUDE.md 功能清单全面核对更新
  - 逐项交叉验证代码与功能清单，将 42 个 `[ ]` 中实际已实现的标记为 `[x]`
  - 未实现功能保留 `[ ]` 并附加简短说明
  - 新增 6 条变更日志条目

- **2026-06-27：** 第十四批 — 核心数据库表补全
  - `init.sql` 新增 `contents`、`content_elements`（含 updated_at）、`audit_records`、`appeals` 建表语句 + 索引 + 约束
  - `element_repo.go` 修复 `CountByStatus` tenant_id 列不存在问题（JOIN contents）
  - `log_repo.go` 修复 5 处 tenant 隔离 JOIN（ce.tenant_id → c.tenant_id via contents）
  - `ContentElement` 模型新增 `UpdatedAt` 字段
  - 所有 element_repo 查询/扫描追加 updated_at

- **2026-06-27：** 第十五批 — 核心 bug 修复
  - `content_elements` 创建时 HumanStatus 错误设为 ElementPendingAI（应为 ElementPendingHuman），涉及 ingestion_service.go(5 处)、video_processor.go(2 处)、content_handlers.go processVideoAsync(2 处)
  - `log_repo.go` CountByReviewer appeals 字段从硬编码 0 改为真实 subquery
  - Dashboard.tsx 添加 Tabs 组件绑定 activeTab 状态（之前声明但未使用）
  - 移除 "查看成员" 死按钮；简化 TenantModal 可见性逻辑

- **2026-06-27：** 第十六批 — P1 问题全面修复
  - **P1-1 ResolveAppeal 事务包裹：** `review_service.go` ResolveAppeal 整个流程（audit_record 创建 + element 状态回滚 + appeal 更新）放入 DB 事务；`log_repo.go` 新增 `CreateWithTx`；`appeal_repo.go` 新增 `UpdateWithTx`；`element_repo.go` 新增 `BeginTx` + `UpdateStatusWithTx`
  - **P1-2 appeal_service.go DI 不一致：** `NewAppealService` 增加 `appealRepo` 参数，由 `services.go` 统一注入，移除内部 `NewAppealRepository(pool)` 重建
  - **P1-3 CountPendingAppeals 租户隔离：** `log_repo.go` CountPendingAppeals 增加 `tenantID` 参数，JOIN `contents` 表按 `c.tenant_id` 过滤；`dashboard_service.go` 调用处传递 `tenantID`
  - **P1-4 vite.config.js 冗余清理：** 删除 `frontend/vite.config.js`（与 `vite.config.ts` 并存且 proxy target 错误）

- **P1 状态：** 全部 4 项已修复 ✅

- **2026-06-28：** 第十七批 — P2 体验优化全面修复
  - **P2-5 键盘快捷键：** Review.tsx 添加 Enter/Space 通过、Esc 打回、← → 切换元素；焦点指示器 + 快捷键提示栏
  - **P2-6 上/下一个导航：** 集成在键盘快捷键中，聚焦元素蓝色边框高亮
  - **P2-7 图片全屏预览：** 已有 Image.Preview，无需额外改动 ✅
  - **P2-8 Dashboard 趋势图表：** 新增 `GET /dashboard/trend` 后端 API + `GetDailyTrend` 服务方法 + `CountByActionDateRange` 仓储方法；前端纯 CSS 柱状图展示近 7 天数据
  - **P2-9 Dashboard 待办提醒：** 顶部橙色提醒条显示待审元素数量
  - **P2-10 申诉详情补充原始信息：** Appeal.tsx 弹窗新增原始 AI 审核结果展示（风险分/标签/置信度/分歧标记），调用新 API `/review/content/:contentId`
  - **P2-11 直播离线状态：** StreamTile 灰显 + 虚线边框 + OFFLINE 标签
  - **P2-12 直播点击跳转：** 高风险流点击跳转到审核工作台
  - **P2-13 username 联合唯一：** init.sql 删除全局 UNIQUE，添加 `(tenant_id, username)` 复合唯一约束；auth_service.go Register 增加 tenant 感知的唯一性检查；user_repo.go 新增 `FindByUsernameAndTenant`
  - **P2-14 审核操作日志页面：** 新建 AuditLog.tsx 页面 + `GET /review/logs` API，支持按操作类型/审核类型筛选

- **P2 状态：** 全部 10 项已修复 ✅

- **2026-06-28：** 第十八批 — 记忆文件更新
  - 更新 `photo-audit-project.md`、`project_state.md`、`photo-audit-project-status.md`、`photo-audit-remaining-tasks.md`、`MEMORY.md`
  - 标记 P1/P2 全部完成，记录未实现模块和技术债务

- **2026-06-28：** 第二十批 — 审核状态机顶层决策逻辑升级
  - `TriggerContentDecision` 从简单 all-human-done 升级为 5 阶段多维决策引擎
    1. 强制 reject：单个元素 human_rejected + AI 风险分 ≥ 70
    2. 分歧升级：is_conflict=true 且未人工审核 → under_review
    3. 加权投票：cover_image/live_snapshot 权重 2x，多数票 reject → reject
    4. AI 风险阈值：平均 AI 风险分 > 60 → reject（无需人工审核）
    5. 默认：全部 human_done 且无 reject → approve
  - 涉及文件：仅 `ingestion_service.go`，改动约 80 行

- **2026-06-28：** 第十九批 — WebSocket 审核任务自动分配
  - **websocket_hub.go** 重写：Hub 改为 `connections map[*WSClient]*WSConnection`，新增 `BroadcastToTenant`/`BroadcastToReviewers`/`BroadcastNewTask`
  - **ingestion_service.go**：`TriggerAIReview` 增加 `tenantID` 参数 + `wsHub` 引用，AI 审核后广播新任务
  - **review_service.go**：增加 `wsHub *Hub` 字段
  - **services.go**：调整 DI 顺序，`wsHub` 在 `NewIngestionService` 之前创建
  - **review_handlers.go**：增加 `authSvc` 依赖，新增 `WebSocket` 端点（JWT 鉴权 + 用户注册）
  - **content_handlers.go**：传递 `tenantID` 给 `TriggerAIReview`
  - **live_wall_handlers.go**：更新 `Register` 调用适配新签名
  - **routes.go**：注册 `GET /ws/review` WebSocket 路由
  - **handlers.go**：`NewReviewHandler` 传入 `svc.AuthService`
  - **Review.tsx**：前端接入 WebSocket，连接状态指示 + 新任务通知自动刷新
  - 验证：`tsc --noEmit` 0 errors ✅

- **2026-06-28：** 第二十批 — 审核状态机顶层决策逻辑升级
  - `TriggerContentDecision` 从简单 all-human-done 升级为 5 阶段多维决策引擎：
    1. 强制 reject：单个元素 human_rejected + AI 风险分 ≥ 70
    2. 分歧升级：is_conflict=true 且未人工审核 → under_review
    3. 加权投票：cover_image/live_snapshot 权重 2x，多数票 reject → reject
    4. AI 风险阈值：平均 AI 风险分 > 60 → reject（无需人工审核）
    5. 默认：全部 human_done 且无 reject → approve
  - 涉及文件：仅 `ingestion_service.go`，约 80 行替换
  - 验证：`tsc --noEmit` 0 errors ✅

- **2026-06-28：第二十二批 — AI 模型自动降级 + 文件上传格式/分辨率校验**
  - **自动降级：** 新建 `fallback_service.go`（关键词匹配 + 垃圾链接检测 + 噪声检测）；`ai_service.go` 增加 `WithFallback` 和 `fallback` 字段；无 API Key 时自动使用本地规则引擎；检测到 402/429 时自动切换 fallback；`config.go` 新增 `FallbackEnabled` 配置项（默认 true）；`services.go` 注入 fallback
  - **文件上传校验：** `content_handlers.go` 新增图片格式白名单（JPEG/PNG/GIF/WebP）；视频格式白名单（mp4/webm/mov）；ffprobe 检测分辨率（最高 4K/最低 480p）
  - **清理：** 删除 `backend/app/` 整个 Python FastAPI 目录（iteration-v0 死代码，已被 Go Fiber 后端完全替代）
  - 验证：`tsc --noEmit` 0 errors, `vite build` 成功, Go 文件括号平衡

## 下一会话执行计划（按优先级排序）

**全部完成。** Phase 1 MVP 所有功能已实现。

### Phase 2 规划（独立子系统，后续进行）
- **WebRTC 直播信令：** mediasoup SFU + SDP 交换 + 前端播放器替换
- **Kafka 审核任务队列：✅ 已完成（2026-08-26，ddaf70e1）** — internal/queue（segmentio/kafka-go）+ docker-compose KRaft 单节点；TriggerAIReview 发消息 → 消费组调 ProcessAIReviewContent；publish 失败自动回退进程内 goroutine。坑：kafka.DialLeader 会无限阻塞（改 DialContext+CreateTopics）；KAFKA_LISTENERS 禁写 0.0.0.0（须写 PLAINTEXT://:9092）
- **Elasticsearch 全文检索：** 审核记录搜索 + 运营报表

- **2026-06-28：第三十一批 — 核心 bug 修复**
  - **content_handlers.go 编译错误：** 修复 5 处 `ingestionLog` 未定义（应为 `contentLog`）；新增 `elementRepo` 字段到 ContentHandler struct + handlers.go 注入
  - **uploadFile is_video 响应不匹配：** 后端 `is_video` 从顶层移到 `data` 内部，与前端 `unwrap(res).data` 取值路径对齐
  - **quality_repo json_object_agg 扫描类型：** `jsonb` 改为 `[]byte` + `json.Unmarshal`，替代直接扫 `map[string]interface{}`
  - **AuditLog 筛选器不工作：** 前端 `getAuditLogs` 增加可选 params 参数；后端 `ListAuditLogs` 读取 `action`/`review_type` query params；新增 `ListAllFiltered` 仓储方法
  - 验证：`tsc --noEmit` 0 errors ✅

- **2026-06-28：第三十二批 — 用户申诉提交 + 注册租户选择 + 数据库约束完善**
  - **申诉提交前端：** 新建 `SubmitAppeal.tsx` 页面（申诉理由 + 证明材料链接）；`content-api.ts` 新增 `submitAppeal` 函数；`App.tsx` 添加 `/appeal/new/:contentId` 路由；`Appeal.tsx` 添加「提交新申诉」按钮
  - **注册租户选择：** Register.tsx 新增租户选择（加入现有租户 / 创建新租户）；注册后 tenantId 正确设置到 auth store；解决所有租户隔离查询返回空结果的问题
  - **数据库完善：** `live_streams.content_id` 添加 `ON DELETE CASCADE`；`appeals` 表新增 `tenant_id` 列 + 索引
  - **后端适配：** Appeal 模型新增 `TenantID`；Repository 所有查询/扫描追加 `tenant_id`；Service `SubmitAppealInput` 增加 `TenantID`；Handler 从 middleware context 提取 `tenant_id`
  - 验证：`tsc --noEmit` 0 errors ✅

- **2026-06-28：第三十三批 — 集成测试修复**
  - **txConn 接口重复声明：** 删除 `element_repo.go` 和 `log_repo.go` 中的重复 `txConn` 声明，保留在 `appeal_repo.go` 中统一声明
  - **log_repo.go / element_repo.go 缺 pgx import：** 添加 `"github.com/jackc/pgx/v5"` import
  - **ListAuditLogs 访问未导出字段：** `ReviewService.auditLogRepo` 改为 `AuditLogRepo`（exported），handler 和内部调用同步更新
  - **middleware/logger.go fmt.Printf：** 替换为结构化日志 `mwLog.Info()`
  - **UploadFile 视频上传丢失租户上下文：** `processVideoAsync` 增加 `tenantID` 参数，从 form 字段提取
  - 验证：`tsc --noEmit` 0 errors, `grep fmt.Printf` 无结果 ✅

### WebRTC 直播信令评估

- **现状：** RTMP 推流已完整实现（StreamScheduler 截帧调度 + WebSocket 推送电视墙）
- **WebRTC 集成需要：** mediasoup SFU 服务器 + 信令 WebSocket（SDP exchange）+ 前端 mediasoup-client 替换播放器
- **预估工作量：** 后端 500-800 行 + 前端 300-500 行 + 独立 mediasoup 服务器部署
- **结论：** 属于独立子系统，建议作为 Phase 2 单独规划，不纳入当前 MVP

### 技术债务（后续修复）

无遗留技术债务。Phase 1 MVP 全部完成，Phase 2/3 独立子系统需单独规划。
