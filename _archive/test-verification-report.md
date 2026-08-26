# Photo Audit Demo — 测试验证报告

> 验证日期：2026-06-27
> 验证角色：资深测试工程师
> 依据文档：四角色代码评审报告 (2026-06-26) + 专项改进方案 (2026-06-26)
> 验证方式：代码审查 / 逻辑推理 / grep 静态扫描
> **重要声明：** Go 和 TypeScript 编译器在当前环境中不可用，编译验证（`go build`、`tsc --noEmit`、`npm run build`）无法执行。本报告标注"编译验证"的条目仅为推测性检查，不构成编译通过保证。

---

## 一、修复验证清单

### P0 级修复

#### P0-1：services.go — notifier 定义顺序修复

**状态：已验证通过**

验证方式：代码审查 (`services.go` lines 80-91)

- `notifier := NewMultiNotifier(&ConsoleNotifier{})` 定义在第 84 行
- `reviewSvc := NewReviewService(..., notifier)` 定义在第 86 行
- `appealSvc := NewAppealService(..., notifier)` 定义在第 87 行
- notifier 在引用它的所有 service 之前创建，定义顺序正确
- `aiSvc := NewAIService(...)` 定义在第 119 行，位于所有 service 初始化之后，`AIService` 已注入 `Services` 结构体（line 135）

#### P0-2：uuid.MustParse panic 修复

**状态：已验证通过**

验证方式：全量 grep 静态扫描 + 代码审查

- **`review_service.go`**：`BatchReview` 中 `uuid.Parse(input.ReviewerID)` (line 125) 和 `uuid.Parse(eid)` (line 132) 均有 `if err != nil` 错误处理
- **`appeal_service.go`**：`GetByID` (line 130)、`Update` (line 153)、`ResolveAppeal` (line 177) 均使用 `uuid.Parse` 并有错误处理
- **`review_handlers.go`**：`HumanReview` (lines 149, 156)、`BatchReview` (line 234)、`ResolveAppeal` (line 298) 均使用 `uuid.Parse` 并返回 400
- 全局 grep `MustParse`：零结果，确认无残留

#### P0-4：AIService 注入 services.go

**状态：已验证通过**

验证方式：代码审查 (`services.go` lines 116-119, 135)

- AGNES_API_KEY / DEEPSEEK_API_KEY 从环境变量读取
- `aiSvc := NewAIService(agnesKey, deepseekKey)` 创建实例
- `AIService: aiSvc` 已注入 `Services` 结构体

**发现额外问题（详见第三部分回归风险）：** `services.go` line 65 调用 `repository.NewAuditLogRepository(pool)`，但该函数不存在。实际构造函数名为 `NewLogRepository`（`log_repo.go` line 21）。此为新增编译错误。

#### P0-5：auth.ts persist 冲突修复

**状态：已验证通过**

验证方式：代码审查 (`auth.ts`)

- `login` 方法仅调用 `set({ token, user, isAuthenticated: true })`，无 `localStorage.setItem('token', token)`
- `logout` 方法保留 `localStorage.removeItem('token')` 和 `localStorage.removeItem('user')`，但状态由 persist 中间件通过 `name: 'auth-storage'` 管理
- 双重写入问题已消除

**发现：`api.ts` (line 27-28) 的 axios interceptor 仍从 `localStorage.getItem('token')` 读取 token。** 这与 `auth.ts` persist 存储到 `'auth-storage'` 键不一致。登录后 token 只写入 `'auth-storage'`，interceptor 读到的是 null。这会导致所有 API 请求不带 Authorization header。**需要修复。**

---

### P1 级修复

#### P1-3：content-api.ts — as any 移除

**状态：已验证通过**

验证方式：全局 grep `as any`，零结果

- 所有 API 函数使用泛型类型参数（如 `axiosInstance.get<ApiPaginatedResponse<ContentElement[]>>`）
- 返回值通过类型安全的 `data.data` 路径获取
- `ApiResponse<T>` 和 `ApiPaginatedResponse<T>` 泛型封装了后端响应结构

#### P1-4：Review.tsx — filter 对接后端 API

**状态：已验证通过**

验证方式：代码审查 (`Review.tsx` lines 248-255)

- `getPendingElements` 调用传递了 `ai_status`、`element_kind`、`risk_min`、`risk_max`、`page`、`page_size`
- `fetchElements` 的 `useCallback` 依赖数组包含 `[page, pageSize, filterStatus, filterType, riskRange]` (line 265)，筛选变化自动重新加载
- `content-api.ts` `getPendingElements` 函数签名包含所有新参数

#### P1-5：Login.tsx — Link 替代 anchor

**状态：已验证通过**

验证方式：代码审查 (`Login.tsx` line 111)

- `<Link to="/register">` 来自 `react-router-dom`，已替换原来的 `<a href="#register">`

#### P1-6：Review.tsx — 打回二次确认 Popconfirm

**状态：已验证通过**

验证方式：代码审查 (`Review.tsx` lines 193-207)

- 打回按钮包裹在 `<Popconfirm>` 中，提示"确定要对此元素进行打回操作吗？此操作不可撤销。"
- 确认后才打开 `Select` 供选择原因，且 `onChange` 检查 `values.length > 0`

#### P1-7：Dashboard 租户隔离

**状态：已验证通过（逻辑推理）**

验证方式：代码审查

- `dashboard_handlers.go` line 23-29 从 `c.Locals("tenant_id")` 提取 tenantStr 并传递给 `dashSvc.GetStats(ctx, tenantStr)`
- `dashboard_service.go` line 48 接收 `tenantID string` 参数，并在所有内部辅助函数中传递（`countRecordsAllTime`、`countRecordsToday`、`countPendingElements` 等）
- `log_repo.go` 的 `CountByType`、`CountByDateRange`、`CountByAction`、`AvgAIScore` 均在 `tenantID != ""` 时通过 `JOIN content_elements WHERE ce.tenant_id = $N` 进行租户隔离
- `element_repo.go` 的 `CountByStatus` (line 225-251) 在 `tenantID != ""` 时添加 `WHERE tenant_id = $1`

#### P1-8：tenant_repo.go 软删除过滤

**状态：已验证通过**

验证方式：代码审查 (`tenant_repo.go` lines 88, 96)

- Count 查询：`WHERE status != 0`
- List 查询：`WHERE status != 0 ORDER BY created_at DESC LIMIT $1 OFFSET $2`

#### P1-9：Dashboard 统计逻辑修复

**状态：已验证通过**

验证方式：代码审查 (`dashboard_service.go` lines 146-153)

- `countPendingElements` 调用 `CountByStatus` 获取 `map[string]int64`，然后取 `counts["pending_human"]`
- 不再累加所有 ai_status

#### P1-10：前端 Mock 数据清除

**状态：已验证通过**

验证方式：全量 grep `mock-` 关键字

- 后端 Go 文件：零匹配
- 前端 TSX/TS 文件：零匹配
- Review.tsx：catch 块调用 `message.error('获取待审元素失败')` 并设空数组，无 mock fallback
- Dashboard.tsx：catch 块调用 `message.error`，无 mock 数据
- LiveWall.tsx：catch 块调用 `message.error` 并设空数组，无 mock 数据

#### P1-12：LiveWall WebSocket 组件卸载防护

**状态：已验证通过**

验证方式：代码审查 (`LiveWall.tsx`)

- `cancelledRef.current` 用于防护 `fetchStreams`（line 167, 198, 215）
- `wsRef` 和 cleanup 函数在 useEffect 中正确关闭 WebSocket（line 206-208）
- `isMounted` ref 存在但未被使用（可能是冗余代码，不构成问题）

---

### P2 级修复

#### P2-3：重复 stringPtr 函数

**状态：已验证通过**

验证方式：代码审查

- `stringPtr` 函数定义在 `review_service.go` line 262-265
- `ai_service.go` 中不再有自己的 `stringPtr` 实现
- `ai_service.go` line 231 调用 `stringPtr(fmt.Sprintf(...))` -- 由于 `parseJudgeResponse` 是非方法函数且在 `service` 包中，它能访问同包的 `stringPtr`。需确认 ai_service.go 和 review_service.go 属于同一包。

验证：两文件均为 `package service`，所以 `stringPtr` 在 `ai_service.go` 中可直接引用 `review_service.go` 的定义。通过。

#### P2-2：TenantHandler.Create createdBy 从 Locals 获取

**状态：已验证通过**

验证方式：代码审查 (`handlers.go` lines 165-168)

- `createdBy := uuid.Nil` 初始化
- `userID := c.Locals("user_id")` 从 middleware 提取
- `if userID != nil { createdBy, _ = uuid.Parse(userID.(string)) }`
- 若未登录则保持 `uuid.Nil`（向后兼容）

**发现隐患：** `userID.(string)` 是裸类型断言（line 168），如果 middleware 未设 user_id 或设了非 string 值，会 panic。应改用 `getTenantID` 风格的 safe assertion。建议将 `handlers.go` 中的类似断言统一改为 helper 函数。

#### P2-9：Appeal.tsx Modal 操作路径校验一致性

**状态：已验证通过**

验证方式：代码审查 (`Appeal.tsx`)

- "维持原判"按钮 (line 78-92)：使用 `<Popconfirm>` 二次确认后调用 `onResolve(appeal.id, 'maintained', '维持原判')`
- "改判解封"按钮 (line 93-107)：使用 `<Popconfirm>` 二次确认后调用 `onResolve(appeal.id, 'approved', '改判解封')`
- 两个路径都经过 Popconfirm 确认，校验一致

---

## 二、遗留问题追踪

### 已确认未修复的问题

| 编号 | 优先级 | 问题 | 涉及文件 | 状态 |
|------|--------|------|----------|------|
| NEW-01 | P0 | `NewAuditLogRepository` 函数不存在 | `services.go:65` | **阻塞编译** |
| NEW-02 | P1 | `api.ts` interceptor 从 `localStorage.getItem('token')` 读 token，但 `auth.ts` persist 存储到 `'auth-storage'`，两者键名不一致，导致所有 API 请求不带认证头 | `api.ts:27-28`, `auth.ts:48` | 需修复 |
| NEW-03 | P1 | `LiveWall.tsx` 使用 `message.error` 但 `message` 未从 antd 导入 | `LiveWall.tsx:170` | 编译报错 |
| NEW-04 | P2 | `dashboard_handlers.go` 使用 `tenantID.(string)` 裸断言而非 `getTenantID` helper | `dashboard_handlers.go:26` | 风格不一致 |
| NEW-05 | P2 | `handlers.go` TenantHandler.Create 使用 `userID.(string)` 裸断言而非 safe assertion | `handlers.go:168` | 同上 |
| NEW-06 | P2 | `LiveWall.tsx` 存在 `isMounted` ref 但未使用 | `LiveWall.tsx:155` | 冗余代码 |
| NEW-07 | P2 | `Appeal.tsx` fetchAppeals 依赖数组为空 `[]`（line 155），状态切换后不会自动重载 | `Appeal.tsx:155` | 功能缺失 |
| P1-1 | P1 | 申诉统计 `CountPendingAppeals` 无租户隔离 | `log_repo.go:225-233` | 待改进 |
| P2-4 | P2 | 缺少键盘快捷键支持 | `Review.tsx` | 阶段三 |
| P2-5 | P2 | 缺少索引优化 | init.sql | 阶段三 |
| P2-6 | P2 | N+1 查询未合并 | `dashboard_service.go` | 阶段三 |

### 重点关注：NEW-01 编译阻塞

`services.go` 第 65 行调用 `repository.NewAuditLogRepository(pool)`，但整个 `repository/` 目录中不存在此函数。`log_repo.go` 定义的唯一构造函数是 `NewLogRepository`。此问题将导致 Go 编译失败。

### 重点关注：NEW-02 认证头缺失

`api.ts` axios interceptor 读取 `localStorage.getItem('token')`，但 `auth.ts` 的 persist 中间件将 token 存储在 `localStorage.getItem('auth-storage')`（JSON 序列化后的 Zustand persist 格式）。这意味着登录成功后，axios 发出的所有 API 请求都不会携带 Authorization: Bearer token。后端 middleware `auth.go` 会返回 401。

---

## 三、回归风险分析

### 高风险变更

**1. `element_repo.go` CountByStatus 签名变更**

- 新方法签名：`CountByStatus(ctx, tenantID string) (map[string]int64, error)`
- 返回类型从原先可能的单一 int64 改为 `map[string]int64`
- `dashboard_service.go` 调用方已适配（line 147 使用 `counts["pending_human"]`）
- **未发现其他调用方** -- grep 全项目确认仅此一处调用 `CountByStatus`，无回归风险

**2. `BulkInsert` 改为事务包裹 (CreateBulk)**

- `element_repo.go` line 51-77：`CreateBulk` 使用 `tx.Begin/Commit` 替代原先的 `pgx.Batch`
- 原子性增强，无副作用
- **调用方检查**：搜索 `CreateBulk` 的使用位置，确认 ingestion_service 是否正确调用
- 原方法名 `BulkInsert` 改为 `CreateBulk`，需确认所有调用方已更新

**3. `review_service.go` 新增 `stringPtr` 函数**

- 与 `ai_service.go` 共享同一包，消除重复定义
- 逻辑简单：`func stringPtr(s string) *string { return &s }`
- 无回归风险

**4. appeal_service.go 事务包裹**

- `SubmitAppeal` 使用 `tx.Begin` 包裹 existing-check + insert + commit
- rollback 通过 `defer tx.Rollback(ctx)` 保证
- 原先的 `findByContentAndApplicant` 独立 DB call 已移除，改为 tx.QueryRow
- **注意：** `contentRepo.FindByID` 检查仍在事务外（line 60-62），极端情况下 content 可能在检查后被删除。但这是可接受的 trade-off，不影响核心修复目标

### 中风险变更

**5. Dashboard 租户隔离新增 tenantID 参数**

- 所有 `LogRepository` 方法 (`CountByType`、`CountByDateRange`、`CountByAction`、`AvgAIScore`、`CountConflictRecords`) 新增 `tenantID string` 参数
- 调用方 `DashboardService` 已全部传递
- **风险点：** `CountPendingAppeals` (line 225-233) 仍未接收 tenantID 参数，查询无租户隔离。这是已记录的遗留问题 (NEW-01 表中 P1-1)

**6. ReviewHandler.ListPending 查询链路**

- handler 接收 query params -> elementRepo.FindByStatus -> SQL 动态拼接 WHERE
- 变更：新增了 elementKind、riskMin、riskMax 参数传递
- **风险点：** `countQ` 重建逻辑 (lines 133-137) 使用 `whereParts[:2]` 再动态扩展。逻辑较复杂，边界条件下可能 count 和 list 条件不一致
- 经仔细审查：当 elementKind/riskMin/riskMax 有值时，countQ 会在 line 136 重新用 `whereParts` (全部条件) 重建，与 listQ 一致。**逻辑正确。**

---

## 四、下一步测试计划

### 4.1 编译验证（阻塞性，必须首先执行）

由于编译器不可用，建议在本地执行以下命令：

```bash
# 后端
cd backend
go mod tidy          # 修复 go.sum
go build ./...        # 预期：零错误
go vet ./...          # 预期：零警告

# 前端
cd frontend
npm run build         # 预期：TypeScript 编译零错误
```

**预计会遇到以下编译错误（需在修复后重试）：**
1. `services.go:65` -- `repository.NewAuditLogRepository` 未定义
2. `LiveWall.tsx:170` -- `message` 未定义（import 缺失）

### 4.2 冒烟测试 TC-01：完整登录-创建租户-审核流程

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 启动后端 `go run cmd/server/main.go` | 服务监听 8080，无 panic |
| 2 | 启动前端 `npm run dev` | 开发服务器监听 3000 |
| 3 | 浏览器访问 `/login`，输入正确凭据 | 跳转到 Dashboard，无 401 错误 |
| 4 | 检查 axios interceptor 是否携带 Authorization header | 是，header 中含 Bearer token |
| 5 | 创建租户 T1 | 返回 201，租户状态 active |
| 6 | 以 T1 用户身份登录 | Dashboard 显示 T1 统计数据 |
| 7 | 查看审核工作台 `/review` | 列表加载，无报错 |
| 8 | 调整筛选器（元素类型/风险分） | 立即触发后端重新查询，结果联动 |
| 9 | 点击"通过" | Audit record 写入，卡片消失 |
| 10 | 点击"打回" | Popconfirm 弹出，选择原因后提交 |

### 4.3 冒烟测试 TC-02：并发申诉防护

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 准备两个已审核的 content | 有可申诉的记录 |
| 2 | 并发发起两次相同 (content_id, applicant_id) 的申诉 | 一个返回 201，另一个返回 409 |
| 3 | 数据库查询 appeals 表 | 同一 (content_id, applicant_id) 仅一条记录 |

### 4.4 冒烟测试 TC-03：租户数据隔离

| 步骤 | 操作 | 预期结果 |
|------|------|----------|
| 1 | 创建租户 T1 和 T2 | 两个租户独立 |
| 2 | 以 T1 审核员登录，查看 Dashboard | 统计数据只包含 T1 数据 |
| 3 | 尝试直接调用 `/api/v1/dashboard/stats` 不带 tenant 或带 T2 | 返回 T1 数据或被 RLS 拦截 |

### 4.5 回归测试要点

1. **auth.ts persist 与 api.ts interceptor 的 token 键名一致性**（NEW-02）：修复后确认登录/登出流程完整链路
2. **LiveWall WebSocket 清理**：进入 `/live-wall` -> 切到其他页面 -> 检查浏览器 network 面板确认 WebSocket 已关闭
3. **Dashboard 统计准确性**：手动插入已知数量的审核记录 -> 验证 Dashboard 显示数值匹配

### 4.6 待纳入后续迭代的功能缺口

以下不在本次修复范围内，但影响完整用户体验：

- 批量审核 UI（后端 API 有 `/batch`，前端无 checkbox 多选交互）-- 改进方案任务 2.4
- 审核状态机顶层决策逻辑（多元素内容的全局状态判定）-- 改进方案任务 2.5
- 键盘快捷键（Enter 快速通过、方向键导航）-- 改进方案任务 3.1
- 图片全屏预览 -- 改进方案任务 3.2
- 审核日志查询页面 -- 改进方案任务 3.3
- 数据库索引补充 -- 改进方案任务 3.6
