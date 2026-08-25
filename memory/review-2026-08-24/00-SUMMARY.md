# 2026-08-24 九角色全面代码评审 — 汇总索引

> 评审方式：9 个子代理并行字段级只读评审。完整报告在本目录 01~09 号文件（按角色命名）。
> 同日已完成并提交的修复：cd68412e（代码入库）、1f79a8d0（repo Create 未 Scan 严重 bug）、d52ba4f2（CORS/bcrypt/BodyLimit/ResolveAppeal 乐观锁）。

## 跨角色 P0 共识（多角色独立确认）

| # | 问题 | 确认角色 | 关键位置 |
|---|------|---------|---------|
| P0-1 | 跨租户泄漏：FindByStatus/CountByFilters/ListAllFiltered 等无 tenantID 参数；文档宣称的 RLS 不存在 | 后端+数据库+架构师 | element_repo / log_repo / user_repo |
| P0-2 | 公开注册可自授 platform_admin（ValidRoles 含超管）；前端注册 role 写死 reviewer 且 tenant_id 未传后端；getTenants 未鉴权可枚举全部租户 | 后端+普通用户 | auth_service.Register / api.ts L81 |
| P0-3 | 分页参数错位：Query 绑定 (page, offset) 但 SQL 是 (LIMIT, OFFSET) → 第1页实际 LIMIT 1，审核队列只出 1 行 | 数据库 | element_repo.go:172 |
| P0-4 | ResolveAppeal 把 contentID 写进 audit_records.element_id（FK 指向 elements）→ 改判必违约回滚，改判闭环从未跑通 | 产品+架构师+后端 | review_service.go:238 |
| P0-5 | fiber c.Context() 传入后台 goroutine（3处）→ fasthttp ctx 复用后读脏数据/panic | 架构师+后端 | content_handlers.go:164/513、review_service.go:135 |
| P0-6 | parseJudgeResponse 解析失败静默返 0 分 → diff>20 必误标分歧，污染决策引擎 | 架构师+后端 | ai_service.go:377 |

## 其他高危（P1）

- getDB sync.Once 闭包 `:=` 遮蔽 err → DB 连不上返回 (nil,nil)，首请求 panic
- HumanReview 非事务 + check-then-act 竞态（双重判罚/记录状态漂移）
- 前端 Dashboard「添加成员」表单未绑定 Form 实例 → 提交空对象静默失败（功能整体失效）
- Esc 键 = 无确认秒打回 + 穿透 Popconfirm 取消意图；批量选择跨页泄漏
- tenant_audit_levels.level_code 全局 UNIQUE → 租户间冲突（应 UNIQUE(tenant_id,level_code)）
- StreamScheduler.isStreamHealthy(nil,"") → 定时把所有直播误判离线
- GET /ai-config 明文回传 API Key；PUT /api/v1/appeals/:id 是无乐观锁的旁路后门
- DATABASE_URL 密码占位符 `***` 贯穿 deployment 5 个文件；deploy-native.sh 无条件删库；零备份体系（运维结论：修完 P0 前仅限 demo）
- 测试三大系统性问题：decision_test 测的是手工复制的副本非生产代码；CI 不跑测试；api 层 mock handler 自测。实测覆盖率 api 0.4%/service 10.9%。go test -race 本机可通过但无门禁
- 冒烟实测发现：注册接口 role 校验与前端发送值不一致（400 invalid role）

## 各角色亮点共识

分层清晰、SSRF 防护完整（含 CGNAT）、裁判分歧全链路闭环、防疲劳二次确认、reviewer_id 强制取 JWT、三段式健康检查、优雅停机、设计令牌体系、空态分流。

## 完整报告文件

01-产品经理.md（PRD漂移/申诉入口断裂/判罚等级空转等商业视角）
02-架构师.md（含 Phase 2 微服务拆分路线 + 技术债优先级表）
03-前端工程师.md（12 bug + 类型安全/性能/a11y，修复 Top5）
04-后端工程师.md（12 确认级 bug + API 一致性问题清单）
05-数据库工程师.md（schema S1-S10 + 双轨迁移漂移分析）
06-运维工程师.md（部署物 P0 清单 + 生产化差距）
07-测试工程师.md（覆盖率实测 + 7 处现有测试无法捕获的疑似 bug）
08-UI设计师.md（8 功能性样式 bug + 对比度/响应式问题）
09-普通用户.md（三类用户全旅程体验断点）

## 待办（新会话接续）

> **2026-08-25 P0 全部修复完成**（冒烟报告见 `SMOKE-REPORT-2026-08-25.md`）：
>
> | # | 项 | Commit | 验证 |
> |---|---|---|---|
> | P0-1 | 跨租户查询泄漏 | `a052af3e` | 双租户实测：B租户0条/A租户2条，冒充403 |
> | P0-2 | 注册自授超管 | `f5808f74` | curl四连测 + languages NOT NULL bug 顺带修 |
> | P0-3 | 分页 LIMIT 错位 | `5bb5b0a2` | 真实PG：page=1 返回 pageSize 条 |
> | P0-4 | 改判 FK 违约 | `f4ae99c9` | 冒烟链路实证修复 |
> | P0-5 | ctx 进 goroutine | `11e64c60` | go test -race 全绿 |
> | P0-6 | 裁判 0 分误标 | `2b73937d` | 单测更新通过 |
>
> 回归：go test -race 全绿、tsc --noEmit 通过。
>
> **2026-08-25 晚间追加**：HTTP 404 谜题已破案并修复（`5da39a02`）——tenant 中间件嵌套直调 auth，
> auth 内的 c.Next() 多推路由栈指针跳过真实路由。修复后 8 项 curl 终验全通，前端联调阻塞已解除。
>
> **遗留**：
> 1. P0-2 计划中的 `GET /tenants/public` 公开端点未做（注册页当前用鉴权 getTenants）
> 2. P1 高危项见上表"其他高危"节

1. 补 /tenants/public 公开端点（可选）
2. P1 项按需排期
