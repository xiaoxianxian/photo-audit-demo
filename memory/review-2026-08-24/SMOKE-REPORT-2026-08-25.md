# 冒烟测试验收报告（2026-08-25）

> 接续 `SMOKE-HANDOVER.md` Step 7-10。Step 1-5 已于前一会话完成。

## 执行结果总览

| # | 步骤 | 结果 | 证据 |
|---|------|------|------|
| 7a | smoker2 绑定租户 + 登录 | ✅ | SQL UPDATE RETURNING 确认；token 签发成功 |
| 7b | 人审通过（cover_image `6f60f2db`） | ✅* | 服务层直驱，audit_record 落库 |
| 7c | 打回（title `bdb05694`，带 reason） | ✅ | record `cea72963`：action=reject, reason=标题含违规词汇 |
| 8a | 提交申诉（content B） | ✅ | appeal `ab3a12e3` status=submitted + [NOTIFY] appeal_submitted |
| 8b | 改判 resolve approved | ✅（修复后） | **修复前必失败**（实证 P0-4），修复后全链路通过 + [NOTIFY] appeal_approved |
| 9 | 看板 /dashboard/stats | ✅ | total_reviewed/today_reviewed/approval_rate/appeal_count 均真实计算 |
| 10a | git 提交修复 | ✅ | `f4ae99c9` |
| 10b | 更新 00-SUMMARY + 本报告 | ✅ | — |

\* approve 首跑已消耗（前次 HTTP 尝试实际写入了 DB），回归时返回"已审核"409 属预期行为。

## P0-4 实证与修复（本次核心产出）

**实证**（修复前）：`ResolveAppeal → create audit record (tx): ERROR: insert or update on table "audit_records" violates foreign key constraint "audit_records_element_id_fkey" (SQLSTATE 23503)` —— 与评审报告预测完全一致，改判闭环此前从未跑通。

**根因**：`review_service.go:238` 把 `appeal.ContentID` 写进 `audit_records.element_id`（FK 指向 content_elements.id）。

**修复**：改判时选取该内容下首个 `human_rejected` 元素（无则首个元素）作为代表元素写入 FK 列。约 15 行，含注释。

**回归验证**（服务层直驱真实 PG）：
1. audit_records 新增 review_type=appeal, action=reverse 记录 ✅
2. 被打回元素 human_status 回滚为 pending_human ✅
3. appeal status=resolved_approved + resolution 落库 ✅
4. 申诉人通知发出（appeal_submitted → appeal_approved）✅
5. `go test -count=1 ./...` 全绿 ✅

## 顺手修复：JWT issuer 不一致（原"未解之谜"）

`service/jwt.go` GenerateToken 签发 `iss=audit-platform`，而 `model/jwt_claims.go` NewJWTClaims 用 `iss=photo-audit-platform`。已统一为后者并加注释互指。历史 token 需重新登录获取。

## ✅ 已解问题：HTTP 层条件性 404（2026-08-25 晚间破案，commit `5da39a02`）

**现象**：携带 `Authorization + X-Tenant-ID(本人租户)` 的请求返回路由器级 404（"Cannot GET …"）；缺头/非法头/非成员租户时反而正常。

**根因**：`middleware.Tenant()` 内部**直接函数调用**了 `Auth(cfg)` 返回的 handler，而 Auth 以 `c.Next()` 结尾。fiber 的 `c.Next()` 会推进 `Ctx.indexRoute` 路由栈指针，嵌套直调导致指针被多推一格、跳过真实路由：
1. authMW(use) → `c.Next()` → indexRoute 指向 tenantMW
2. tenantMW → 直调 `auth(c)` → 内层 `c.Next()` → indexRoute **跳过 ListPending**
3. tenantMW 自己的 `c.Next()` 从栈尾继续扫 → `Cannot GET` 错误冒泡 → 全局错误处理器把已写好的 200 响应覆盖成 404

**定位过程**：三处打点（ENTRY/TENANT/HANDLER）发现日志顺序完全颠倒（最内层 handler 先执行完、中间件后到）；routeprobe 发现路由链只有 1 个 handler；最小复现工程逐项二分中间件组合，最终 NESTED=true（tenantMW 直调 authMW）一键复现、false 一切正常，实锤根因。

**修复**：Tenant() 改为从 `c.Locals` 读取 role/user_id（Auth 中间件已在 protected 组先行执行并填充），不再嵌套调用。签名 `Tenant(db, cfg)` → `Tenant(db)`。8 项 curl 终验全通（pending 200 / 无头400 / 冒充403 / dashboard / contents / appeals / audit-rules 全 200）。

## 遗留现场（已清理）

- 调试二进制与临时 driver 已删，调试补丁已还原
- 测试数据：内容 A/B 各元素状态已被冒烟脚本改变（A cover 已通过；B title 经历 reject→改判回滚→pending_human）
- P0 全部修复完成，见 00-SUMMARY.md 台账

## 提交记录

- `35e85689` fix: 冒烟测试发现的4个阻断性bug + 交接文档（前会话）
- `f4ae99c9` fix: P0-4 改判外键违约 + JWT issuer 统一
- `5bb5b0a2` fix(P0-3) / `2b73937d` fix(P0-6) / `11e64c60` fix(P0-5)
- `f5808f74` fix(P0-2) / `a052af3e` fix(P0-1)
- `5da39a02` fix: HTTP 404 谜题破案（tenant 嵌套调用 auth 致路由栈错位）
