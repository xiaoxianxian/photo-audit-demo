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

## 未解问题：HTTP 层条件性 404（不阻塞业务，需专项排查）

**现象**：对运行中服务，携带 `Authorization + X-Tenant-ID(本人租户)` 的请求返回路由器级 404（"Cannot GET …"）；缺 X-Tenant-ID、或租户头非法/非成员时反而正常走中间件链。

**已排除**：路由注册缺失（diag 程序 + 运行时 ROUTE dump 双证齐全）、代理劫持（no_proxy 生效）、旧二进制（go version -m 为 devel）、tenant 中间件逻辑（三态输出符合源码）、limiter/cors 源码审查无嫌疑。

**关键矛盾**：进程内打点显示 handler 进入且 `Next returned, status: 200`，但同一请求最终响应是 404 且错误在 logger 中间件浮出 —— 单请求内不可能先 200 后 404，疑似 fasthttp Ctx 池交叉污染或本机环境干扰。服务层直驱完全不受影响，生产影响面待评估。

## 遗留现场

- 后端调试二进制可能仍在运行：`ps aux | grep audit-server` 后 kill
- 测试数据：内容 A/B 各元素状态已被冒烟脚本改变（A cover 已通过；B title 经历 reject→改判回滚→pending_human）
- P0-1/2/3/5/6 待修（见 00-SUMMARY.md）

## 提交记录

- `35e85689` fix: 冒烟测试发现的4个阻断性bug + 交接文档（前会话）
- `f4ae99c9` fix: P0-4 改判外键违约 + JWT issuer 统一（本次）
