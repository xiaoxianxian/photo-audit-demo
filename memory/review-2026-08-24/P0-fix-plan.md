# P0 修复计划（2026-08-24）

> 来源：九角色评审共识（见 `00-SUMMARY.md`）。每项含：根因、修复方案、验证方式。
> 原则：最小改动、不动无关代码、每项修完跑 `go build/vet/test` + `tsc --noEmit`。

---

## P0-1 跨租户查询泄漏

**问题**：审核队列核心查询没有 tenantID 参数，中间件校验了却没人消费。文档宣称的 RLS 不存在。

**涉及**：`element_repo.go FindByStatus/CountByFilters`、`log_repo.go ListAllFiltered/CountPendingAppeals(已有)`、`user_repo.go List/FindByUsername`、`appeal_repo.go ListByStatus`

**方案**（应用层过滤，不引入 RLS——RLS 留 Phase 2）：
1. 给上述方法签名加 `tenantID uuid.UUID` 参数
2. 查询 JOIN contents（element/log 类）或直接 WHERE tenant_id=$n（user/appeal 表自带列）
3. Handler 层从 `c.Locals("tenant_id")` 取值传入；platform_admin bypass 时传 nil → 方法内 nil 跳过过滤
4. `FindByUsername` 登录场景保留全局查找，但 Login 后校验 user.TenantID 归属

**验证**：
- 新增单测：两个 repo 内存测试不可行（需 PG），改为 handler 层集成测试 + 代码走查清单
- 手工验证：注册两个租户用户，A 租户 token 查待审列表应看不到 B 的数据
- `grep -rn "FindByStatus(ctx" internal/api/` 确认全部调用点都传了 tenantID

**工作量**：半天

---

## P0-2 注册接口可自授 platform_admin

**问题**：公开 `/auth/register` 接受任意合法 role；前端 role 写死 reviewer 且 tenant_id 未传后端。

**方案**：
1. `auth_service.go Register`：加白名单 `allowedRoles = {reviewer, quality_inspector}`（以 init.sql CHECK 枚举为准），role 为 admin/platform_admin 时返回 400 "特权角色请联系管理员创建"
2. 前端 `api.ts postRegister`：透传用户选择的 role 和 tenantId（Register.tsx 已有选择 UI）
3. `getTenants` 公开枚举问题：新建公开端点 `GET /api/v1/tenants/public` 只返回 id+name（供注册页选择），原列表接口挂鉴权

**顺带修复冒烟发现的 invalid role 问题**（同根因）。

**验证**：
- curl 测试：注册 role=platform_admin → 400；role=reviewer → 200
- 前端 tsc 通过 + 手工注册流程走通

**工作量**：2 小时

---

## P0-3 分页 LIMIT 参数错位

**问题**：`element_repo.go:172` Query 绑定 `(page, offset)` 但 SQL 占位符顺序是 `(LIMIT, OFFSET)` → 第 1 页实际 LIMIT 1。

**方案**：
```go
rows, err := r.db.Query(ctx, listQ, append(args, pageSize, (page-1)*pageSize)...)
```
同时全仓 grep 同类模式：`grep -rn "page, pageSize\*(page-1)\|page, offset" internal/repository/` 排查其他 repo 是否复制了同样的错。

**验证**：
- 单测不可行（需 PG），用真实 DB 冒烟：插入 5 个元素后第 1 页应返回 pageSize 条
- 检查 Review 页面实际显示数量

**工作量**：半小时

---

## P0-4 ResolveAppeal 写错 element_id

**问题**：`review_service.go:238` 把 `appeal.ContentID` 写进 `audit_records.element_id`（FK 指向 content_elements）→ 改判事务必回滚。

**方案**：
改判时 appeal 关联的是 content，但 audit_record 需要 element 维度。两种路径：
- **推荐**：取该 content 下被 human_rejected 的第一个元素 ID 作为 ElementID（改判针对的就是这些元素）；若找不到 rejected 元素则记 content 级记录——检查 schema 是否允许 element_id NULL，若 NOT NULL 则必须选一个元素
- AuditRecord 增加 `ContentID` 字段（表若无此列则不加，保持最小改动）

**验证**：
- 冒烟链路第 9 步：申诉改判应成功（不再报 FK 违约）
- 数据库验证：`SELECT element_id FROM audit_records WHERE review_type='appeal'` 应为真实元素 UUID

**工作量**：1 小时

---

## P0-5 fiber c.Context() 进后台 goroutine（3 处）

**问题**：fasthttp RequestCtx 在 handler 返回后被池化复用，goroutine 稍后使用会读脏数据/panic。

**位置**：`content_handlers.go:164`、`content_handlers.go:513`、`review_service.go:135`（HumanReview 内）

**方案**：
handler 侧统一封装：
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel() // 注意：不能 defer，goroutine 要用；改为在 goroutine 内 cancel
go func() {
    defer cancel()
    h.ingestionSvc.TriggerAIReview(ctx, ...)
}()
```
三处逐一替换。`review_service.go` 内部同理（由 service 自己创建独立 ctx）。

**验证**：
- `go test -race ./...`
- 冒烟上传内容后等 AI 审核完成（fallback 引擎秒级完成），确认元素风险分正常落库无 panic 日志

**工作量**：1 小时

---

## P0-6 judge 解析失败返 0 分误标分歧

**问题**：`ai_service.go parseJudgeResponse` 找不到 score 返回 `(0, nil)` → diff>20 必误标 is_conflict。

**方案**：
```go
return 0, errors.New("no score found in judge response")
```
调用方（TriggerAIReview 中 judge 调用处）处理：judge 出错时**跳过裁判环节**，日志 Warn，不标记分歧（is_conflict 保持 false），主审结果照常生效。

**验证**：
- 新增单测：parseJudgeResponse 输入不含分数的 JSON → 应返回 error
- 修改 TriggerAIReview 对 judge error 的处理逻辑 + decision_test 补一个"judge 失败不标分歧"用例

**工作量**：1.5 小时

---

## 执行顺序与总量

| 顺序 | 项 | 工作量 | 风险 |
|---|---|---|---|
| 1 | P0-3 分页错位 | 0.5h | 低（一行） |
| 2 | P0-6 judge 0分 | 1.5h | 低 |
| 3 | P0-5 ctx goroutine | 1h | 中（并发语义） |
| 4 | P0-4 element_id | 1h | 低 |
| 5 | P0-2 注册提权 | 2h | 低（前后端各一半） |
| 6 | P0-1 租户过滤 | 0.5天 | 中（动面最广） |

**总计约 1 天**。每项独立 commit，全部完成后跑全量回归 + 重跑冒烟链路（重点验证第 5 步列表行数和第 9 步改判成功）。
