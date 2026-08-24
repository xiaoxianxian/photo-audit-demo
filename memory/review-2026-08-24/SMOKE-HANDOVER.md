# 冒烟测试交接文档（2026-08-25）

> 新会话接续指南：读完本文件即可无缝继续。详细背景见 `00-SUMMARY.md` 和 `P0-fix-plan.md`。

## 当前状态

- **代码修复**：4 个冒烟新 bug 已修，`go build/vet/test` 全绿，**尚未 git 提交**
- **git 未提交变更**：
  - `backend/internal/api/review_handlers.go` — HumanReview/BatchReview 裸断言 panic → 安全断言 + 400
  - `backend/internal/repository/element_repo.go` — nil 数组绕过 DEFAULT → 传空数组
  - `backend/internal/service/ai_service.go` — 网络传输错误也走本地 fallback
  - `deployment/docker-compose.yml` — PG 宿主机端口改 `${POSTGRES_PORT:-5433}`（5432 被 hujing-postgres 占用）
- **环境**：PG 容器 `photo-audit-postgres-1` 运行中（宿主机端口 **5433**，库 photo_audit 20表，postgres/postgres）；后端可能仍在 8080 运行（`/tmp/audit-server`，先 `curl localhost:8080/health` 检查）

## 启动后端命令

```bash
cd /Users/xiaota/Documents/Photo-Audit-Demo/backend
go build -o /tmp/audit-server ./cmd/server
DATABASE_URL="postgresql://postgres:postgres@localhost:5433/photo_audit?sslmode=disable" \
JWT_SECRET="smoke-test-secret-key-32-chars-long!!" \
ALLOWED_ORIGINS="*" SERVER_PORT=8080 /tmp/audit-server > /tmp/audit-server.log 2>&1 &
```

## 剩余冒烟步骤（Step 7-10）

1. **登录 smoker2** 拿新 token：`POST /api/v1/auth/login {"username":"smoker2","password":"Smoke12345!"}`
2. **绑定租户**（smoker2 目前 tenant_id=NULL）：
   ```sql
   UPDATE users SET tenant_id='482693c2-5ed1-47f3-b2fb-a0c3ea616ef0' WHERE username='smoker2';
   ```
   注意：DB 更新不影响已发 token 的 claims；tenantMW 用 DB 校验所以放行，但 handler 里 `Locals("tenant_id")` 来自 JWT claim（authMW 设置）——**若仍 400 tenant context missing，需重新登录让新 token 带上 tenant_id？不会**——注册/登录时 user.tenant_id=NULL 则签发的 token 无 tenant_id claim。**正确做法**：先 SQL 绑定 tenant_id，再重新登录拿新 token。
3. **人审通过**：`POST /api/v1/review/human {"element_id":"6f60f2db-cb35-4138-a89a-54f8c4875b75","action":"approve"}` + header `X-Tenant-ID: 482693c2-...`
4. **打回另一个元素**（title 元素需先把 ai_status 改为 ai_passed 才会出现在待审列表）：action=reject 必须带 reason
5. **提交申诉**：`POST /api/v1/appeals`（看 appeal_handlers.go 的 body 结构）
6. **改判**：`PUT /api/v1/review/appeal/<appeal_id> {"decision":"approved"}` —— **预期失败**（评审 P0-4：element_id 写成 contentID 外键违约）。若失败，记录完整错误 = 实证完成
7. **看板**：`GET /api/v1/dashboard/stats`
8. **git 提交全部修复**（一个 commit 或按 bug 分开均可）
9. **更新 memory/review-2026-08-24/00-SUMMARY.md** 的待办区，向用户出验收报告

## 已知未解之谜（不阻塞验收）

旧 token（iss=`audit-platform`）稳定 404，新 token（iss=`photo-audit-platform`）正常。历史遗留，生产用户只会拿到新格式 token。可顺手统一 issuer 字符串常量。

## P0 修复计划

见同目录 `P0-fix-plan.md`（6 项，约 1 天工作量）。冒烟完成后按计划执行。
