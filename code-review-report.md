# Photo Audit Demo — 四角色代码评审报告

> 评审日期：2026-06-26
> 参与角色：资深后端架构师、资深前端工程师、资深产品经理、资深测试工程师
> 项目定位：多租户 AI 审核平台（供稿/短视频/直播），技术栈 Go(Fiber)+React+PostgreSQL+MinIO+WebSocket

---

## 🧑‍💻 角色一：资深后端架构师（Go / Fiber）

### ✅ 优点

1. **四层架构清晰** — `handler → service → repository → DB` 职责边界明确
2. **依赖注入容器** — `services.go` 中 `Services` 结构体集中装配，良好的 DI 实践
3. **中间件链式复用** — `tenant.go` 内部复用 `Auth()` 提取器，避免重复解析 JWT
4. **错误处理一致** — 所有 repository 层使用 `fmt.Errorf("context: %w", err)` 包装
5. **WebSocket Hub 模式** — register/unregister channels + ping keepalive + tenant-scoped broadcast

### ❌ 问题

**[P0 编译风险] services.go 中 notifier 使用前定义**

`review_svc := NewReviewService(..., notifier)` 引用了下一行才创建的 `notifier`。虽 Go 允许同作用域变量先取零值，但可读性差且易引发困惑。

**[P0 Panic] `BatchReview` 使用 `uuid.MustParse` 无校验**

`review_service.go` line 137: `uuid.MustParse(input.ReviewerID)` — 非法 UUID 直接 panic 崩溃。应改用 `uuid.Parse()` 返回错误。

**[P1] `AIService` 未被任何 Handler 调用**

`ai_service.go` 实现了 `ReviewElement()` 和 `JudgeReview()`，但 `services.go` 中未创建实例，也没有在任何 handler 中被调用。AI 审核流水线空转。

**[P1] 多处 `uuid.MustParse()` 滥用**

- `review_handlers.go` lines 179, 180
- `live_wall_handlers.go` line 86
- `quality_audit_handlers.go` lines 104, 105, 182

**[P1] 类型断言无安全检查**

`live_wall_handlers.go` line 86: `c.Locals("tenant_id").(string)` — middleware 未设值则 panic。

**[P1] 两套几乎相同的 `ResolveAppeal`**

`ReviewService.ResolveAppeal()` 和 `AppealService.ResolveAppeal()` 逻辑高度重复。

**[P2] 响应格式不统一**

部分 handler 使用 `h.Error()` / `h.OK()`，大部分直接 `c.JSON(fiber.Map{...})`。

**[P2] `BulkInsert` 缺少事务保证原子性**

`element_repo.go` 使用 `pgx.Batch` 不保证原子性，content 创建成功但 element 插入一半 → 孤儿数据。

**[P2] Dashboard 统计数据无租户隔离**

`dashboard_handlers.go` line 23 提取了 `tenantID` 但从未在 DB 查询中使用。

**[P2] `TenantHandler.Create` 传入 `uuid.Nil` 作为 `createdBy`**

`handlers.go` line 164 硬编码 `uuid.Nil`，丢失审计追踪。

**[P2] 重复的 `stringPtr()` 函数**

分别在 `ai_service.go` 和 `review_service.go` 定义两次。

---

## 🎨 角色二：资深前端工程师（React / TypeScript）

### ✅ 优点

1. **组件拆分合理** — 各页面有独立 helper 函数和子组件
2. **Zustand + persist 中间件** — token 持久化方案合理
3. **Axios 拦截器** — 正确处理 401 跳登录和 token 注入
4. **Register.tsx 确认密码校验** — 使用 `dependencies` + 自定义 validator，写法规范
5. **LiveWall.tsx 图片 onError 回退** — 有容错处理

### ❌ 问题

**[P0] `stores/auth.ts` — persist 与手动 localStorage 双重写入冲突**

`login` 中既 `localStorage.setItem('token', token)` 又 `set({ token, ... })`（persist 自动存到 `'auth-storage'`），双重写入可能导致刷新后不一致。

**[P0] 派生状态 `isAuthenticated` 由 subscribe 手动同步**

`App.tsx` 用 `!!s.token && !!s.user` 计算，`auth.ts` 也用 subscribe 同步，两个来源可能不一致。

**[P1] `content-api.ts` — 大量 `as any` 类型断言**

第 81、105、170、181 等处 `(data as any)?.data` 绕过类型系统，后端结构变化时编译器不报错。

**[P1] `Review.tsx` — 前端 filter 不调后端 API**

`fetchElements` 依赖数组不包含 `filterType`、`riskRange`，只做了前端过滤，后端 `ai_status` 参数未传递。

**[P1] `Login.tsx` — 注册链接用 `<a href="#register">` 而非 `<Link>`**

硬编码锚点将来页面分离时将失效。

**[P1] `Review.tsx` — 打回操作无二次确认**

选了打回标签就立即提交，无 Popconfirm。

**[Bug] `Appeal.tsx` — Modal 两条操作路径校验不一致**

"改判解封"直接调用 `onResolve` 不需填写备注，"取消+提交"路径先 `validateFields()`。

**[Bug] `LiveWall.tsx` — WebSocket callback 无组件卸载防护**

`ws.onmessage` 触发 `fetchStreams()`，组件卸载后 `setStreams` 仍会调用。

---

## 👔 角色三：资深产品经理

### 产品经理视角

#### ✅ 优点

1. **核心问题抓得准** — "AI 拦截 90%"定位正确，多租户 + RBAC 贴合 B2B SaaS
2. **内容拆分元素模型** — 审核粒度细，便于追溯
3. **裁判分歧机制** — 成熟产品做法
4. **审核规则/判罚等级/敏感词 CRUD** — 给了租户足够自定义能力

#### ❌ 问题

**[P0] 功能完成度差距巨大**

六大模块中只有租户管理基本完成。**AI 审核引擎完全没有实现代码** — 没有 Agnes AI 调用、DeepSeek 裁判模型。

**[P1] 审核状态机不完整**

多元素内容（如视频 10 个元素，8 通过 2 打回）的全局决策逻辑缺失。

**[P1] 前端所有页面有大量 Mock 数据**

API 失败时用 mock 数据 fallback，生产环境不可接受。

**[P1] 缺少真正的批量审核 UI**

API 有 `/review/batch`，前端无 checkbox 多选交互。

**[P2] 缺少审核操作日志查询页面**

设计了 `audit_logs` 但无查询 API 或展示页面。

**[P2] 权限控制未完全落地**

未区分 reviewer 和 quality_checker 的接口访问权限。

**[P2] 数据库 `users.username` 全局 UNIQUE**

多租户场景应改为 `(tenant_id, username)` 联合唯一。

### 普通用户（审核员）视角

#### ✅ 优点

1. **卡片式设计直观** — AI 评分、风险标签一目了然
2. **裁判分歧橙色高亮** — 视觉信号清晰
3. **风险分四级颜色** — 绿/黄/橙/红识别效率高
4. **申诉详情弹窗** — 含证明材料查看，状态标签颜色区分

#### ❌ 痛点

1. **打回无二次确认** — 误触无法回退
2. **没有键盘快捷键** — 一天看几百张图，鼠标点击效率低
3. **图片预览太小** — 卡片高度 120px，应支持放大/全屏
4. **缺少上一个/下一个导航** — 方向键导航缺失
5. **筛选器缺少"风险分排序"** — 高风险优先最重要
6. **申诉列表只查 submitted** — 看不到已处理记录
7. **电视墙缺少点击跳转审核** — 无快捷入口
8. **Dashboard 缺少待办提醒** — 无行动号召

---

## 🔍 角色四：资深测试工程师

### ✅ 优点

1. **正负例清单完整** — CLAUDE.md 覆盖核心链路场景
2. **测试用例设计合理** — 7 条冒烟测试覆盖核心流程

### ❌ 问题

**[P0] 零测试覆盖率**

不存在任何 `.test.go`、`*_test.go`、`*.spec.ts`、`*.test.tsx` 文件。

**[P0] 并发竞态条件**

`appeal_service.go` lines 59-63: `findByContentAndApplicant` + `Create` 两个独立 DB roundtrip，非原子操作。

**[P0] `parseJudgeResponse` 分数提取逻辑脆弱**

`ai_service.go` lines 309-338 — 扫描任意数字，"3 violations" 会被误解析为分数 3。

**[P1] 多步骤服务操作缺少事务**

- `IngestionService.UploadContent` → 孤儿数据
- `ReviewService.ResolveAppeal` → 状态不一致
- `QualityAuditService.SubmitQARecord` → 孤儿记录

**[P1] 软删除不一致**

`tenant_repo.go` List 查询不过滤 `status = 0` 记录。

**[P1] Dashboard 统计逻辑错误**

`dashboard_service.go` line 145-155: 把所有 `ai_status` 加起来，但标签是"待审元素"。

**[P2] 缺少关键索引**

`audit_records.element_id`、`audit_records.action`、`appeals.content_id` 等无索引。

**[P2] N+1 查询问题**

`DashboardService.GetStats` 顺序执行 8 次独立 SELECT。

**[P2] `CountByReviewer` CTE 性能极差**

`log_repo.go` line 215 相关子查询，大数据量下极慢。

### 测试优先级建议

| 优先级 | 测试内容 | 涉及文件 |
|--------|----------|----------|
| Critical | Login/Register 并发、JWT 有效性 | `auth_service.go` |
| Critical | HumanReview 状态机（重复提交防护） | `review_service.go` |
| Critical | 重复申诉竞态条件 | `appeal_service.go` |
| High | AI judge 分数解析鲁棒性 | `ai_service.go` |
| High | Dashboard 除零保护 | `dashboard_service.go` |
| Medium | WebSocket 并发安全 | `websocket_hub.go` |
| Medium | 租户隔离边界条件 | `middleware/tenant.go` |
| Medium | 端到端冒烟测试 | 全链路 |
| Low | Repository 层单元测试 | 所有 repo |

---

## 📊 总体问题汇总

| 严重度 | 数量 | 典型代表 |
|--------|------|----------|
| **P0（阻塞）** | 5 | notifier 前向引用、uuid.MustParse panic、零测试覆盖、AI 引擎未接入、auth persist 冲突 |
| **P1（高）** | 12 | AIService 空转、租户隔离缺失、大量 as any、前端 filter 不调后端、状态机不完整 |
| **P2（中）** | 15+ | 响应格式不统一、缺少事务、Mock 数据过多、缺少索引、交互缺陷 |
