# Photo Audit Demo — 专项改进方案

> 编制角色：产品经理
> 依据文档：四角色代码评审报告（2026-06-26）
> 项目状态：第一批到第七批开发已完成，核心骨架搭建完毕
> 编制日期：2026-06-26

---

## 一、问题总览与优先级矩阵

以下表格汇总四角色评审报告中识别的全部问题，按严重度分级，标注影响模块和引用来源。

| 编号 | 优先级 | 问题描述 | 影响模块 | 引用角色 | 阻塞场景 |
|------|--------|----------|----------|----------|----------|
| P0-1 | P0 | services.go 中 notifier 在 review_svc 之后才定义（前向引用） | 后端服务层 | 后端架构师 | 运行时可运行但可读性差 |
| P0-2 | P0 | BatchReview 使用 uuid.MustParse(ReviewerID) 无校验，非法值直接 panic | 后端审核服务 | 后端架构师 | 外部传入 reviewer_id 字段即可触发 |
| P0-3 | P0 | 零测试覆盖率，无任何 .test.go / *.spec.ts | 全栈 | 测试工程师 | 无法验证任何变更安全性 |
| P0-4 | P0 | AI 审核引擎完全没有实现代码 —— AIService 已编写但未注入也未接入任何 Handler | 后端 AI 服务 | 产品经理/后端架构师 | 平台核心卖点不成立 |
| P0-5 | P0 | auth.ts persist 与手动 localStorage 双重写入冲突 | 前端认证 | 前端工程师 | 刷新后 token 状态可能不一致 |
| B0-1 | P0 | 并发竞态条件：appeal_service.go 中 findByContentAndApplicant + Create 非原子 | 后端申诉服务 | 测试工程师 | 高频同时申诉同一内容可绕过唯一约束 |
| B0-2 | P0 | parseJudgeResponse 分数提取逻辑脆弱：扫描任意数字，"3 violations" 误解析为 3 分 | 后端 AI 服务 | 测试工程师 | AI 裁判返回文本中包含数字即出错 |
| B0-3 | P0 | BulkInsert 使用 pgx.Batch 不保证原子性，element 插入一半失败产生孤儿数据 | 后端内容入库 | 后端架构师 | 内容创建成功但元素丢失 |
| B0-4 | P0 | 多处 uuid.MustParse() 滥用（review_handlers、live_wall_handlers、quality_audit_handlers） | 后端 Handler 层 | 后端架构师 | 非法 UUID 均可触发 panic |
| B0-5 | P0 | 类型断言无安全检查 c.Locals("tenant_id").(string) | 后端 Handler 层 | 后端架构师 | middleware 未设值即 panic |

| 编号 | 优先级 | 问题描述 | 影响模块 | 引用角色 |
|------|--------|----------|----------|----------|
| P1-1 | P1 | 审核状态机不完整：多元素内容的全局决策逻辑缺失（8 通过 2 打回如何判定？） | 后端业务逻辑 | 产品经理 |
| P1-2 | P1 | ReviewService.ResolveAppeal() 和 AppealService.ResolveAppeal() 两套高度重复逻辑 | 后端审核/申诉 | 后端架构师 |
| P1-3 | P1 | 前端大量 `as any` 类型断言绕过 TypeScript 类型系统 | 前端 API 层 | 前端工程师 |
| P1-4 | P1 | Review.tsx 前端 filter 不调后端 API，只做客户端过滤 | 前端审核页 | 前端工程师 |
| P1-5 | P1 | Login.tsx 注册链接用 `<a href="#register">` 而非 `<Link>` | 前端登录页 | 前端工程师 |
| P1-6 | P1 | Review.tsx 打回操作无二次确认（Popconfirm） | 前端审核页 | 前端工程师 |
| P1-7 | P1 | Dashboard 统计数据无租户隔离（提取了 tenantID 但从不用） | 后端仪表盘 | 后端架构师 |
| P1-8 | P1 | 软删除不一致：tenant_repo.go List 查询不过滤 status=0 记录 | 后端租户仓储 | 测试工程师 |
| P1-9 | P1 | Dashboard 统计逻辑错误：把所有 ai_status 加起来但标签是"待审元素" | 后端仪表盘 | 测试工程师 |
| P1-10 | P1 | 前端 Mock 数据大量存在，API 失败时 fallback | 全前端页面 | 产品经理 |
| P1-11 | P1 | 缺少真正的批量审核 UI（后端有 `/batch` 端点，前端无 checkbox） | 前端审核页 | 产品经理 |
| P1-12 | P1 | 前端 WebSocket callback 无组件卸载防护 | 前端直播电视墙 | 前端工程师 |
| P1-13 | P1 | 双后端并存（Python FastAPI + Go Fiber），无正式废弃声明 | 全架构 | 后端架构师 |
| P1-14 | P1 | Content/element 创建缺少事务保证原子性 | 后端入库服务 | 测试工程师 |
| P1-15 | P1 | 审核日志仅有 append-only 写入，缺少查询 API 和展示页面 | 全栈 | 产品经理 |

| 编号 | 优先级 | 问题描述 | 影响模块 | 引用角色 |
|------|--------|----------|----------|----------|
| P2-1 | P2 | 响应格式不统一：部分用 h.Error()/h.OK()，大部分直接 c.JSON | 后端 Handler 层 | 后端架构师 |
| P2-2 | P2 | TenantHandler.Create 传入 uuid.Nil 作为 createdBy，丢失审计追踪 | 后端租户 Handler | 后端架构师 |
| P2-3 | P2 | 重复的 stringPtr() 函数在 ai_service.go 和 review_service.go 各定义一次 | 后端工具函数 | 后端架构师 |
| P2-4 | P2 | 权限控制未完全落地：未区分 reviewer 和 quality_checker 的接口访问权限 | 后端中间件 | 产品经理 |
| P2-5 | P2 | 数据库 users.username 全局 UNIQUE，多租户应改为 (tenant_id, username) 联合唯一 | 数据库 schema | 产品经理 |
| P2-6 | P2 | Dashboard GetStats 顺序执行 8 次独立 SELECT，N+1 查询 | 后端仪表盘 | 测试工程师 |
| P2-7 | P2 | CountByReviewer CTE 相关子查询，大数据量下性能差 | 后端日志仓储 | 测试工程师 |
| P2-8 | P2 | 缺少关键索引：audit_records.element_id, action, appeals.content_id 等无索引 | 数据库 schema | 测试工程师 |
| P2-9 | P2 | Appeal.tsx Modal 两条操作路径校验不一致 | 前端申诉页 | 前端工程师 |
| P2-10 | P2 | 缺少键盘快捷键（方向键导航、Enter 快速通过） | 前端审核页 | 产品经理 |
| P2-11 | P2 | 图片预览太小，卡片高度 120px，应支持放大/全屏 | 前端审核页 | 产品经理 |
| P2-12 | P2 | 筛选器缺少"风险分排序"（高风险优先最重要） | 前端审核页 | 产品经理 |
| P2-13 | P2 | 申诉列表只查 submitted，看不到已处理记录 | 前端申诉页 | 产品经理 |
| P2-14 | P2 | 电视墙缺少点击跳转审核快捷入口 | 前端直播页 | 产品经理 |
| P2-15 | P2 | Dashboard 缺少待办提醒和行动号召 | 前端仪表盘 | 产品经理 |
| P2-16 | P2 | go.sum 缺失，依赖锁定文件丢失 | 后端构建 | 后端架构师 |

---

## 二、专项改进目标

### 总体目标

将 Photo Audit Demo 从一个"骨架演示项目"升级为"可运行的 MVP 产品"，使六大模块中的核心链路（上传 -> AI 审核 -> 人审 -> 申诉 -> 质检）能够端到端跑通。

### 量化目标

| 目标 | 基线值 | 目标值 | 测量方式 |
|------|--------|--------|----------|
| 后端编译成功率 | 未知（go.sum 缺失） | 100%，`go build ./...` 零错误 | CI 脚本 |
| 测试覆盖率 | 0% | P0/P1 相关代码 >= 60% 行覆盖 | go test -cover |
| AI 引擎可用性 | 代码存在但未注入 | Agnes AI 端到端可调通 | 集成测试 |
| 前端 Mock 数据残留 | 至少 4 处 fallback | 0 处（仅保留开发环境 mock server） | grep 检查 |
| P0 问题修复 | 7 个 | 0 个遗留 | 本方案逐项关闭 |
| 类型安全率 | 至少 8 处 `as any` | `as any` 数量降为 0 | TypeScript strict 模式 |

---

## 三、分阶段实施计划

### 第一阶段（紧急修复）— 解决 P0 阻塞问题

**范围：** 编译风险、panic 漏洞、认证状态冲突、事务一致性基础。

**交付物：**
- 所有 P0/B0 问题修复的 commit
- 基础测试套件（编译验证 + 关键函数单元测试）
- go.sum 锁定文件

**验收标准：**
1. `go build ./...` 零错误
2. `go vet ./...` 零警告
3. 所有 uuid.Parse 调用均有错误处理
4. auth.ts persist 与 localStorage 双重写入消除
5. appeal 提交竞态条件修复并通过并发测试验证

**预计周期：** 5 个工作日

---

### 第二阶段（核心补全）— 解决 P1 高优先级问题

**范围：** AI 引擎实际接入、前端 Mock 清除、批量审核 UI、审核状态机、租户隔离补全、filter 对接后端。

**交付物：**
- AIService 接入 IngestionService 完整调用链
- 前端所有页面去除 fallback mock 数据
- 批量审核交互实现
- 审核状态机顶层决策逻辑
- 租户隔离补全到所有查询

**验收标准：**
1. 上传一张图片 -> 自动调用 Agnes AI -> 得到 risk_score -> 写入 audit_record
2. 审核工作台 filter 下拉变化后立即触发后端重新查询
3. 批量勾选 >= 3 个元素 -> 点击"批量通过" -> 一次性提交
4. 同一内容多个元素，有人审通过有人审打回 -> 内容级别给出全局决策
5. Dashboard 统计只返回当前租户数据

**预计周期：** 12 个工作日

---

### 第三阶段（体验优化）— 解决 P2 及以下问题

**范围：** 键盘快捷键、全屏预览、审核日志查询页、Dashboard 趋势图、索引优化、N+1 查询合并、交互细节打磨。

**交付物：**
- 完整的审核工作台交互体验
- 数据库索引补充和查询性能优化
- 审核日志查询页面
- 前端所有交互细节完善

**验收标准：**
1. 左右方向键切换卡片、Enter 快速通过、Shift+Enter 快速打回
2. 点击图片全屏查看
3. 管理员可在/logs 页面按时间/审核员/操作类型查询
4. Dashboard 首页展示近 7 天通过率趋势折线图
5. Dashboard N+1 查询合并为单次聚合 SQL

**预计周期：** 8 个工作日

---

## 四、各阶段详细任务清单

### 第一阶段：紧急修复（5 个工作日）

#### 任务 1.1：修复 uuid.MustParse() panic 漏洞

- **涉及文件/模块：**
  - `backend/internal/service/review_service.go` line 137
  - `backend/internal/api/review_handlers.go` lines 179, 180
  - `backend/internal/api/live_wall_handlers.go` line 86
  - `backend/internal/api/quality_audit_handlers.go` lines 104, 105, 182
  - `backend/internal/service/appeal_service.go` lines 96, 116, 140
- **工作量：** 1 人日
- **验收标准：**
  - 所有 `uuid.Parse()` 调用均检查 error 并返回 HTTP 400
  - `BatchReview` 中 reviewer_id 非法 UUID 返回 400 而非 panic
  - 编写单元测试验证非法 UUID 输入路径

#### 任务 1.2：修复类型断言安全检查

- **涉及文件/模块：**
  - `backend/internal/api/live_wall_handlers.go` line 86: `c.Locals("tenant_id").(string)`
  - 所有 `c.Locals("user_id").(string)` 类似断言
- **工作量：** 0.5 人日
- **验收标准：**
  - 改为 `tenantID, ok := c.Locals("tenant_id").(string); !ok { return c.Status(500).SendString("tenant not set") }`
  - 或统一抽取为 helper 函数 `getTenantID(c fiber.Ctx) (string, error)`

#### 任务 1.3：修复 auth persist 双重写入冲突

- **涉及文件/模块：**
  - `frontend/src/stores/auth.ts`
- **工作量：** 0.5 人日
- **验收标准：**
  - 删除 `localStorage.setItem('token', token)` 和 `localStorage.removeItem('token')`
  - 由 Zustand persist 中间件统一管理（`name: 'auth-storage'`）
  - `isAuthenticated` 改为 computed 属性而非手动同步

#### 任务 1.4：services.go _notifier_前向引用修复

- **涉及文件/模块：**
  - `backend/internal/service/services.go` lines 80-81
- **工作量：** 0.5 人日
- **验收标准：**
  - notifier 在 review_svc 和 appeal_svc 之前创建
  - 代码阅读顺序与实际依赖顺序一致

#### 任务 1.5：申诉提交竞态条件修复（事务包裹）

- **涉及文件/模块：**
  - `backend/internal/service/appeal_service.go` lines 59-63
  - 需在 `FindByContentAndApplicant` + `Create` 之间加事务
- **工作量：** 1 人日
- **验收标准：**
  - 使用 pgx.Begin 包裹检查 + 创建逻辑
  - 并发提交同一内容的申诉，只有一个成功，另一个返回 409
  - 编写并发测试验证

#### 任务 1.6：parseJudgeResponse 分数提取逻辑加固

- **涉及文件/模块：**
  - `backend/internal/service/ai_service.go` lines 309-338
- **工作量：** 0.5 人日
- **验收标准：**
  - 改为先解析 JSON 再提取 score 字段（如果 DeepSeek 返回 JSON 格式）
  - 或至少用正则 `\bscore[":]+\s*(\d+)\b` 精确匹配
  - 编写 5+ 个边界测试用例

#### 任务 1.7：BulkInsert 和 Ingestion 事务包裹

- **涉及文件/模块：**
  - `backend/internal/repository/element_repo.go` BatchInsert
  - `backend/internal/service/ingestion_service.go` UploadContent
  - 使用 pgx 事务确保 content + elements 要么全成功要么全回滚
- **工作量：** 1 人日
- **验收标准：**
  - 元素插入中途失败 -> content 也被回滚删除
  - 无孤儿数据产生

#### 任务 1.8：生成 go.sum 和基础编译验证

- **涉及文件/模块：**
  - `backend/go.sum`（新生成）
  - `backend/go.mod`（确认无冗余依赖）
- **工作量：** 0.5 人日
- **验收标准：**
  - `go mod tidy` 通过
  - `go build ./...` 零错误
  - 提交 go.sum 到版本控制

---

### 第二阶段：核心补全（12 个工作日）

#### 任务 2.1：AIService 实际接入审核流水线

- **涉及文件/模块：**
  - `backend/internal/service/services.go` — 创建 AIService 实例并注入 Services
  - `backend/internal/service/ingestion_service.go` — UploadContent 完成后异步调用 AIService.ReviewElement
  - `backend/internal/service/review_service.go` — JudgeReview 在适当时机调用裁判模型
  - 配置项：config.go 新增 AgnesAPIKey / DeepSeekAPIKey 字段
- **工作量：** 3 人日
- **验收标准：**
  - 上传一张图片 -> MinIO 存储原图 -> 异步调用 Agnes AI -> 得到 risk_score -> 写入 audit_record
  - 主审和裁判分差 > 20 -> is_conflict = true
  - AI API 超时/5xx -> 标记为审核失败进入人审队列
  - AI API 429/402 -> 自动降级重试 + 告警通知

#### 任务 2.2：前端 content-api.ts 类型安全改造

- **涉及文件/模块：**
  - `frontend/src/services/content-api.ts` — 移除所有 `as any`
  - 定义统一的 API 响应类型 `ApiResponse<T>`
  - `frontend/src/services/api.ts` — 同样检查
- **工作量：** 1 人日
- **验收标准：**
  - 0 个 `as any` 剩余
  - TypeScript strict 编译零错误
  - 所有 API 函数返回类型安全（IDE 可自动补全）

#### 任务 2.3：Review.tsx 前端 filter 对接后端 API

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — filterType 和 riskRange 需传给后端
  - `frontend/src/services/content-api.ts` — getPendingElements 增加 element_kind 和 risk_min/max 参数
  - `backend/internal/api/review_handlers.go` — handler 透传新参数
  - `backend/internal/repository/element_repo.go` — 查询增加过滤条件
- **工作量：** 2 人日
- **验收标准：**
  - 选择"封面图"筛选 -> 后端只返回 element_kind = cover_image 的记录
  - 调整风险分滑块 -> 后端按 risk_score BETWEEN ? AND ? 过滤
  - 筛选条件变化时立即重新加载（不用手动点刷新按钮）

#### 任务 2.4：批量审核 UI 实现

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — 卡片添加 checkbox，顶部显示已选数量
  - `frontend/src/pages/Review.tsx` — 批量操作栏（批量通过/批量打回）
  - `frontend/src/services/content-api.ts` — batchReview 函数调用
  - `backend/internal/service/review_service.go` — BatchReview 已有的逻辑已完善
- **工作量：** 1.5 人日
- **验收标准：**
  - 支持单选/全选/反选元素卡片
  - 批量选择 >= 1 个元素 -> 点击"批量通过" -> 调用 /review/batch API
  - 批量打回时弹出原因选择对话框
  - 操作完成后列表刷新，已审核元素消失

#### 任务 2.5：审核状态机顶层决策逻辑

- **涉及文件/模块：**
  - `backend/internal/model/content.go` — 新增 ContentLevelDecision 类型
  - `backend/internal/service/review_service.go` — HumanReview 结束后触发内容级别决策
  - 决策规则：所有内容元素都通过 -> 内容 approved；任意元素被打回 -> 内容 pending 完整审核；分歧元素 -> 标记橙色高亮
- **工作量：** 1.5 人日
- **验收标准：**
  - 单元素内容：人审通过 -> 内容状态变为 approved
  - 多元素内容（5 个）：3 通过 2 打回 -> 内容状态保持 pending，展示待审进度
  - 裁判分歧元素 -> 审核工作台橙色高亮标记
  - 所有内容审核完毕 -> 自动生成 content_level 决策

#### 任务 2.6：Dashboard 租户隔离补全

- **涉及文件/模块：**
  - `backend/internal/service/dashboard_service.go` — GetStats 接收 tenantID 并传递到所有查询
  - `backend/internal/repository/*.go` — 所有 COUNT/SELECT 查询加上 WHERE tenant_id = $N
  - `backend/internal/api/dashboard_handlers.go` — 确认 tenantID 从 middleware 正确传递
- **工作量：** 1 人日
- **验收标准：**
  - 租户 A 的审核员登录 -> Dashboard 只显示租户 A 的统计数据
  - 租户 B 的数据完全不泄漏
  - 平台超管（tenant_id=NULL）可以查看所有租户汇总

#### 任务 2.7：软删除一致性修复

- **涉及文件/模块：**
  - `backend/internal/repository/tenant_repo.go` — List 增加 status != 0 过滤
  - 全局检查所有涉及软删除字段的查询是否过滤已删除记录
  - 涉及表：tenants, users, audit_teams, contents
- **工作量：** 0.5 人日
- **验收标准：**
  - 已删除租户不在列表返回
  - 已删除用户不参与关联查询

#### 任务 2.8：Dashboard 统计逻辑修复

- **涉及文件/模块：**
  - `backend/internal/service/dashboard_service.go` — countPendingElements 不应把所有 ai_status 累加
  - `backend/internal/repository/element_repo.go` — CountByStatus 增加按状态分类返回值
- **工作量：** 0.5 人日
- **验收标准：**
  - "待审元素"计数 = 仅 ai_status = pending_human 的元素数
  - "机审通过" / "机审拒绝" 分别独立统计
  - Dashboard 展示的数字与审核工作台筛选结果一致

#### 任务 2.9：统一 ResolveAppeal 去重

- **涉及文件/模块：**
  - `backend/internal/service/review_service.go` — ResolveAppeal
  - `backend/internal/service/appeal_service.go` — ResolveAppeal
  - 合并为 AppealService.ResolveAppeal，ReviewService 中调用
- **工作量：** 0.5 人日
- **验收标准：**
  - 仅有一个 ResolveAppeal 实现
  - AppealService.ResolveAppeal 完成后自动更新关联元素的 human_status

#### 任务 2.10：前端 Mock 数据全面清除

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — try/catch 中的 mock fallback 改为 error toast
  - `frontend/src/pages/Dashboard.tsx` — 若有 mock 数据替换
  - `frontend/src/pages/Appeal.tsx` — 若有 mock 数据替换
  - `frontend/src/pages/LiveWall.tsx` — 若有 mock 数据替换
  - 开发环境 mock 改为 Vite mock server 或 MSW（如有需要）
- **工作量：** 1 人日
- **验收标准：**
  - grep `'mock-'` 在 TSX/TS 文件中无匹配
  - API 请求失败 -> 显示友好的 error toast，不显示伪造数据
  - 生产环境行为一致

#### 任务 2.11：LiveWall WebSocket 组件卸载防护

- **涉及文件/模块：**
  - `frontend/src/pages/LiveWall.tsx` — useEffect cleanup 中关闭 WebSocket 并设置 ref.isMounted = false
  - 所有 setState 调用前检查 ref.isMounted
- **工作量：** 0.5 人日
- **验收标准：**
  - 离开 /live-wall 页面 -> WebSocket 关闭 -> 无 console error
  - 组件卸载后 ws.onmessage 不再调用 setState

#### 任务 2.12：Login.tsx 注册链接修复 + 打回二次确认

- **涉及文件/模块：**
  - `frontend/src/pages/Login.tsx` — `<a href="#register">` -> `<Link to="/register">`
  - `frontend/src/pages/Review.tsx` — 打回按钮增加 Ant Design Popconfirm
- **工作量：** 0.5 人日
- **验收标准：**
  - 点击"去注册"跳转到 /register 路由（而非锚点）
  - 点击打回弹出确认框："确定要打回此元素吗？"，选择原因后才提交

---

### 第三阶段：体验优化（8 个工作日）

#### 任务 3.1：键盘快捷键支持

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — useEffect 绑定键盘事件
  - 快捷键设计：
    - `ArrowLeft` / `ArrowRight`：上一张/下一张卡片
    - `Enter`：快速通过选中卡片
    - `Shift+Enter`：快速打回选中卡片
    - `Escape`：关闭弹窗/取消选择
- **工作量：** 1.5 人日
- **验收标准：**
  - 审核员可用纯键盘操作完成 80% 的审核动作
  - 快捷键不与浏览器默认行为冲突
  - 快捷键提示展示在页面角落

#### 任务 3.2：全屏预览功能

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — ElementCard 点击图片 -> 打开 Ant Design Image.PreviewGroup
  - 支持图片缩放、旋转、全屏
- **工作量：** 1 人日
- **验收标准：**
  - 点击卡片中的图片 -> 放大预览
  - 支持鼠标滚轮缩放、拖拽移动
  - ESC 关闭预览

#### 任务 3.3：审核日志查询页面

- **涉及文件/模块：**
  - `frontend/src/pages/` — 新建 AuditLogs.tsx
  - `backend/internal/api/` — 新建 log_handler.go，提供 GET /logs 查询端点
  - `backend/internal/repository/log_repo.go` — 新增 QueryLogs 方法（按时间/审核员/操作类型筛选）
  - `frontend/src/services/content-api.ts` — 新增 getAuditLogs API
  - `frontend/src/App.tsx` — 添加 /logs 路由
- **工作量：** 1.5 人日
- **验收标准：**
  - 管理员可按日期范围、审核员姓名、操作类型（approve/reject/reverse）查询
  - 表格展示：时间、审核员、元素、操作、原因
  - 支持分页

#### 任务 3.4：Dashboard 趋势图表

- **涉及文件/模块：**
  - `frontend/src/pages/Dashboard.tsx` — 集成 Ant Design Chart 或 ECharts
  - `backend/internal/service/dashboard_service.go` — 新增 GetStatsByDateRange 返回近 N 天统计数据
  - `backend/internal/repository/log_repo.go` — 新增按天分组统计 SQL
- **工作量：** 1.5 人日
- **验收标准：**
  - 首页展示近 7 天/30 天的审核趋势折线图
  - 通过率、拒绝率、风险平均分三指标可选切换
  - 移动端图表自适应缩放

#### 任务 3.5：Dashboard N+1 查询合并

- **涉及文件/模块：**
  - `backend/internal/service/dashboard_service.go` — GetStats 将 8 次独立 SELECT 合并为 1-2 次聚合查询
  - `backend/internal/repository/log_repo.go` — 新增 AggregateStats(ctx, tenantID, startDate, endDate) 方法
- **工作量：** 1 人日
- **验收标准：**
  - Dashboard 首次加载 SQL 执行次数从 8 次降低到 1-2 次
  - 结果与原来逐次查询完全一致
  - 查询耗时 < 100ms（10 万级数据量）

#### 任务 3.6：数据库索引补充

- **涉及文件/模块：**
  - `backend/sql/init.sql` 或新建 `backend/sql/migrations/002_add_indexes.sql`
  - 需新增索引：
    - `audit_records(element_id)`
    - `audit_records(action)`
    - `appeals(content_id)`
    - `appeals(status)`
    - `content_elements(content_id)`
    - `content_elements(ai_status)`
    - `content_elements(human_status)`
    - `audit_logs(operator_id, created_at)`
- **工作量：** 0.5 人日
- **验收标准：**
  - 索引创建后 `EXPLAIN ANALYZE` 查询计划显示 index scan 而非 sequential scan
  - 审核工作台查询响应时间 < 200ms（10 万级数据）

#### 任务 3.7：CountByReviewer 查询性能优化

- **涉及文件/模块：**
  - `backend/internal/repository/log_repo.go` — Line 215 附近的子查询优化
  - 使用 JOIN 替代相关子查询，或使用 materialized view
- **工作量：** 0.5 人日
- **验收标准：**
  - 性能测试：10 万条 audit_logs 下 CountByReviewer 查询 < 500ms
  - EXPLAIN ANALYZE 显示使用 index + hash join

#### 任务 3.8：Review.tsx 风险分排序 + 筛选器改进

- **涉及文件/模块：**
  - `frontend/src/pages/Review.tsx` — 筛选区域增加排序下拉框（风险分从高到低、时间从新到旧）
  - `frontend/src/services/content-api.ts` — getPendingElements 增加 sort_by / sort_order 参数
  - `backend/internal/api/review_handlers.go` — handler 传递排序参数
  - `backend/internal/repository/element_repo.go` — ORDER BY ai_risk_score DESC
- **工作量：** 1 人日
- **验收标准：**
  - 选择"风险分降序" -> 高risk_score 的元素排在最前面
  - 排序结果与服务端一致（非前端排序）

#### 任务 3.9：Appeal.tsx 校验统一 + 已处理记录查看

- **涉及文件/模块：**
  - `frontend/src/pages/Appeal.tsx` — 统一两条 Modal 操作路径的字段校验
  - Appeal.tsx 增加 Tab 切换（待处理 / 已处理）
  - `backend/internal/service/appeal_service.go` — ListByStatus 支持多状态查询
- **工作量：** 1 人日
- **验收标准：**
  - 两条操作路径都必须经过 validateFields()
  - 申诉列表支持切换到"已处理"Tab 查看历史记录
  - 材料证明文件可预览

---

## 五、风险与依赖

### 阶段间依赖关系

```
阶段一（5天）
  |---> 阶段二（12天）
  |         |---> 阶段三（8天）
  v
[Go 编译通过 + 无 panic]  [AI 引擎可用 + 前端真实 API]  [完整用户体验]
```

- 阶段二是阶段一的前提：只有编译通过、无 panic、基础事务正确的代码，才能在此基础上添加 AI 引擎接入
- 阶段三是阶段二的前提：只有 Mock 数据清除、前端 filter 对接后端后，才有真实的性能瓶颈需要优化

### 潜在风险

| 风险 | 影响阶段 | 风险等级 | 缓解措施 |
|------|----------|----------|----------|
| Agnes AI API 不可用（无 API Key 或限流） | 阶段二 2.1 | 高 | 开发环境增加 AI mock server（Wire Mock），不阻塞前端联调 |
| DeepSeek 裁判模型返回格式不稳定 | 阶段二 2.1 | 中 | parseJudgeResponse 加固已在阶段一 1.6 处理 |
| 双后端（Python + Go）决策延迟 | 阶段二 1.13 | 中 | 方案：明确 Go Fiber 为主，Python 标记 deprecated，README 更新 |
| PostgreSQL 数据量测试不足 | 阶段三 3.5/3.6/3.7 | 低 | 阶段三用 docker-compose 启动 PG，导入 10 万条模拟数据压测 |
| 前端 TypeScript 严格模式编译失败 | 阶段二 2.2 | 中 | 先改 content-api.ts，其余文件逐步配合 |
| 审核状态机决策规则不明确 | 阶段二 2.5 | 高 | 需产品经理明确规则文档后再开发，避免返工 |
| go.sum 缺失导致 CI 构建不稳定 | 阶段一 1.8 | 低 | 修复后立即提交到版本控制 |

---

## 六、验收标准

### 端到端功能测试用例（MVP 验收 Checklist）

#### TC-01：完整审核流程（核心链路）

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 用 admin 账号登录 | 跳转到 Dashboard，显示统计数据 |
| 2 | 创建租户 T1 | 租户创建成功 |
| 3 | 以租户 T1 审核员 A 登录 | 看到 T1 的数据 |
| 4 | 上传一张图片到 T1 | 图片上传成功，MinIO 存储原图 |
| 5 | 触发 AI 审核 | audit_record 写入 ai_status，risk_score 返回 |
| 6 | 进入审核工作台 | 卡片展示该图片，显示 AI 评分和风险标签 |
| 7 | 点击"通过" | audit_record 写入 human_status=passed，Dashboard 已审数 +1 |
| 8 | 验证 Dashboard | 今日已审数、通过率等统计正确 |

#### TC-02：裁判分歧检测

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 上传一张有争议的图片 | 模拟 Agnes AI 返回 risk_score=30 |
| 2 | DeepSeek 裁判模型返回 risk_score=75 | 分差 45 > 20 |
| 3 | 审核工作台查看 | 卡片边框橙色高亮，标签显示"AI 结论分歧" |

#### TC-03：批量审核

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 审核工作台进入 | 展示 20 张待审卡片 |
| 2 | 勾选 5 张卡片 | 底部显示"已选择 5 项" |
| 3 | 点击"批量通过" | 5 条 audit_record 同时写入，卡片从列表消失 |

#### TC-04：申诉与改判

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 审核员 A 打回了一张图片 | audit_record 写入 human_rejected |
| 2 | 用户上传申诉（附证明材料） | appeal 记录创建成功 |
| 3 | 审核员 A 查看申诉列表 | 看到待处理的申诉 |
| 4 | 点击"改判解封" | 元素 human_status 回退为 pending_human，通知提交者 |
| 5 | 用户再次申诉同一内容 | 返回 409，提示"您已提交过申诉" |

#### TC-05：租户数据隔离

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 创建租户 T1 和 T2 | 两个租户独立 |
| 2 | 以 T1 审核员身份登录 | Dashboard 和审核工作台只看到 T1 数据 |
| 3 | 尝试通过 API 直接查询 T2 的数据 | 返回空或 403 |

#### TC-06：键盘快捷键操作

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 审核工作台有 20 张卡片 | 焦点在第 1 张 |
| 2 | 按右方向键 | 焦点切换到第 2 张卡片 |
| 3 | 按 Enter | 第 2 张卡片审核通过 |
| 4 | 按 Shift+Enter | 弹出打回原因选择 |

#### TC-07：软删除租户

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 平台超管删除租户 T1 | T1 status=0，不在列表返回 |
| 2 | T1 的审核员登录 | 仍然可查看历史审核记录（数据保留） |
| 3 | 尝试创建同名新租户 | 用户名在 T1 下仍存在，不能冲突 |

#### TC-08：Dashboard 真实性验证

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 审核 10 张图片（5 通过 5 打回） | Dashboard 今日已审 = 10 |
| 2 | 通过率显示 50%，拒绝率 50% | 数值准确 |
| 3 | 风险平均分与 10 张图片的实际平均分一致 | 无偏差 |

#### TC-09：AI 引擎降级

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 模拟 Agnes AI API 返回 500 | 图片标记为"AI 审核失败" |
| 2 | 直接进入人审队列 | 审核员可以看到该卡片并手动审核 |
| 3 | AI API 恢复后再次触发 | 正常完成审核 |

#### TC-10：并发申诉防护

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 两个请求并发提交同一用户对同一内容的申诉 | 一个成功（201），一个返回 409 |
| 2 | 查看数据库 appeals 表 | 同一 (content_id, applicant_id) 只有一条记录 |

### 代码质量验收

- [ ] `go build ./...` 零错误
- [ ] `go vet ./...` 零警告
- [ ] `gofmt -s -l ./...` 无格式化差异
- [ ] `go test -cover ./...` 行覆盖率 >= 60%（阶段一涉及的代码）
- [ ] `npm run build` 前端 TypeScript 编译零错误
- [ ] 无 `as any` 剩余
- [ ] 无 `uuid.MustParse` 残留
- [ ] 无 console.log 在生产代码中
- [ ] 所有 API handler 均有 error 处理
- [ ] 所有 `c.Locals()` 均有类型安全检查

### 架构验收

- [ ] Go Fiber 为唯一后端框架（Python FastAPI 代码标记 deprecated）
- [ ] 四层架构（handler -> service -> repository -> DB）保持清晰
- [ ] 依赖注入集中在 services.go
- [ ] 中间件复用链式结构（auth -> tenant -> logger）
- [ ] WebSocket Hub 线程安全（register/unregister/map 读写保护）
