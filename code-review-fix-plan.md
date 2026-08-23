# Photo Audit Platform — 代码评审修复清单

> 汇总自 10 个角色的独立评审，按执行人和优先级分类，附个人判断。

---

## 一、按严重程度分类（个人判断）

### 🔴 必须改（不改会出线上事故）

| # | 问题 | 严重程度 | 涉及角色 | 执行人 | 预估工时 |
|---|------|---------|---------|--------|---------|
| 1 | 审核通过无二次确认，误操作风险极高 | P0-致命 | 用户、前端、UI | 前端 | 2h |
| 2 | reviewer_id 可从前端 body 伪造，任何人可冒充他人审核 | P0-安全 | 架构师、后端 | 后端 | 2h |
| 3 | JWT secret 使用 os.Getpid() 伪随机，非密码学安全 | P0-安全 | 架构师、后端 | 后端 | 1h |
| 4 | WebSocket Hub Broadcast 持 RLock 但内部 close/delete 需写锁，map panic 可崩溃整个服务 | P0-稳定性 | 后端 | 后端 | 2h |
| 5 | CountPendingAppeals 参数占位符错误（IN $1,$2 传3个参数），运行时 SQL 报错 | P0-Bug | 后端、数据库 | 后端 | 1h |
| 6 | 测试覆盖率 <5%，任何改动都可能引入回归 | P0-质量 | 测试 | 测试 | 持续 |
| 7 | 健康检查路径错误（检查 /api/v1/health 但实际在 /admin/health），部署脚本永远失败 | P0-部署 | 运维 | 运维 | 1h |
| 8 | ContentRepository.Create 不在事务中，孤儿记录风险 | P0-数据 | 数据库 | 后端 | 2h |

### 🟡 改了更好（影响体验或长期维护）

| # | 问题 | 严重程度 | 涉及角色 | 执行人 | 预估工时 |
|---|------|---------|---------|--------|---------|
| 9 | 前端零代码分割，首屏 bundle 1.3MB | P1-性能 | 前端、UI | 前端 | 4h |
| 10 | api.ts 和 content-api.ts 滥用 `any` 类型 | P1-质量 | 前端 | 前端 | 4h |
| 11 | 缺少 Rate Limiting，登录接口可被暴力破解 | P1-安全 | 架构师、测试 | 后端 | 4h |
| 12 | WebSocket Token 通过 URL query 传递，泄露到日志 | P1-安全 | 架构师、后端 | 后端 | 2h |
| 13 | 响应格式不统一，部分 handler 返回无 code 字段 | P1-规范 | 后端、测试 | 后端 | 4h |
| 14 | 前端 formatDate 在 6 个页面各自定义一遍 | P1-维护 | 前端 | 前端 | 2h |
| 15 | 缺少 Skeleton loading，全用 Spin 覆盖 | P1-体验 | UI、前端 | 前端 | 4h |
| 16 | 缺少帮助系统（Tooltip、快捷键列表入口） | P1-体验 | 用户、UI | 前端 | 8h |
| 17 | 缺少 CI/CD，无自动化构建验证 | P1-流程 | 运维 | 运维 | 4h |
| 18 | audit_records 未按月分区，百万级数据性能灾难 | P1-性能 | 数据库 | 后端+DBA | 8h |
| 19 | 缺少 schema migration 工具，无法版本化升级 | P1-流程 | 数据库 | 后端 | 4h |
| 20 | 前端 Space 键触发页面滚动冲突 | P1-体验 | 用户、前端 | 前端 | 1h |
| 21 | 打回操作流程太长（Popconfirm → Select → 选原因，4-5步） | P1-体验 | 用户、UI | 前端 | 4h |
| 22 | 缺少操作撤销功能，误审核无法补救 | P1-体验 | 用户 | 后端+前端 | 8h |
| 23 | Dashboard 5 个 fetch 无 Promise.allSettled 聚合处理 | P1-性能 | 前端 | 前端 | 2h |
| 24 | Review.tsx 用空 Table 做分页控件，应改用 Pagination | P1-规范 | UI、前端 | 前端 | 1h |
| 25 | 缺少 CORS 配置，跨域部署会阻断请求 | P1-部署 | 架构师、运维 | 后端 | 2h |
| 26 | main.go 优雅关闭不完整，重启丢失 WebSocket 和正在处理的请求 | P1-稳定性 | 架构师 | 后端 | 4h |
| 27 | 前端无角色守卫，审核员可访问租户管理页面 | P1-安全 | 前端 | 前端 | 4h |
| 28 | LiveWall WebSocket 未做 JWT 验证 | P1-安全 | 后端 | 后端 | 2h |
| 29 | 缺少密码强度提示（注册页） | P1-体验 | UI、用户 | 前端 | 2h |
| 30 | 缺少移动端适配，审核员无法手机应急审核 | P2-需求 | 用户、UI | 前端 | 40h |

### 🟢 改了影响不大（锦上添花，可延后）

| # | 问题 | 严重程度 | 涉及角色 | 执行人 | 预估工时 |
|---|------|---------|---------|--------|---------|
| 31 | constants.ts 缺少 FONT.display(56px) 定义 | P2-完善 | UI | 前端 | 30min |
| 32 | TenantConfig.tsx 未导入 SPACING/FONT/RADIUS 常量 | P2-完善 | UI | 前端 | 30min |
| 33 | AuditLog 操作类型无图标 | P2-体验 | UI | 前端 | 2h |
| 34 | 快捷键按下无视觉反馈 | P2-体验 | 用户、UI | 前端 | 4h |
| 35 | 全局 CSS hover 色硬编码暗色系值，亮色主题切换时对比度不足 | P2-未来 | UI | 前端 | 4h |
| 36 | Layout.tsx Logo 区域双重字体设置（level=3 + fontSize） | P2-规范 | UI | 前端 | 30min |
| 37 | Login.tsx 同样双重字体设置 | P2-规范 | UI | 前端 | 30min |
| 38 | 申诉管理空状态无自定义文案 | P2-体验 | UI | 前端 | 1h |
| 39 | 审核工作台卡片元素类型标签无图标辅助 | P2-体验 | UI | 前端 | 2h |
| 40 | 短视频审核菜单项用 FileTextOutlined 图标，与审核工作台区分度不够 | P2-体验 | UI | 前端 | 30min |
| 41 | 审核规则 priority 无 min/max 约束 | P2-体验 | 用户 | 前端 | 30min |
| 42 | 全局 CSS 滚动条硬编码 #333 不跟随主题 | P2-未来 | UI | 前端 | 30min |
| 43 | LiveWall WebSocket 事件监听器为空（onopen/onclose/onerror） | P2-体验 | 用户 | 前端 | 2h |
| 44 | QualityAudit 样本列表切换无 loading 状态 | P2-体验 | 用户 | 前端 | 2h |
| 45 | Review.tsx 错误处理过于笼统（只 message.error('操作失败')） | P2-体验 | 用户 | 前端 | 2h |
| 46 | 缺少审计记录按 review_type 复合索引 | P2-性能 | 数据库 | DBA | 1h |
| 47 | 缺少 appeals 表 (tenant_id, status) 复合索引 | P2-性能 | 数据库 | DBA | 1h |
| 48 | CountByFilters 执行 3-4 次独立 COUNT，可合并为 FILTER 聚合 | P2-性能 | 数据库 | 后端 | 4h |
| 49 | Responder 与每个 Handler 内部响应方法重复 | P2-维护 | 架构师 | 后端 | 4h |
| 50 | StreamScheduler 使用 rand.Intn 模拟数据应移除 | P2-代码整洁 | 架构师 | 后端 | 1h |
| 51 | user_repo.go 硬编码 PostgreSQL 错误码 "23505" | P2-维护 | 架构师 | 后端 | 2h |
| 52 | LiveWallService 中 AIConfidence*1000 估算观众数是占位代码 | P2-代码整洁 | 架构师 | 后端 | 1h |
| 53 | processVideoAsync goroutine 无 recover 机制 | P2-健壮性 | 架构师 | 后端 | 2h |
| 54 | Docker 容器以 root 运行 | P2-安全 | 运维 | 运维 | 2h |
| 55 | 缺少 .dockerignore 文件 | P2-安全 | 运维 | 运维 | 30min |
| 56 | Redis 无认证（requirepass 未配置） | P2-安全 | 运维 | 运维 | 1h |
| 57 | docker-compose 无 CPU/内存限制 | P2-安全 | 运维 | 运维 | 1h |
| 58 | MinIO 凭证硬编码 minioadmin | P2-安全 | 运维 | 运维 | 1h |
| 59 | 缺少数据库备份策略 | P2-灾备 | 运维 | 运维 | 8h |
| 60 | 缺少分布式追踪（OpenTelemetry/Jaeger） | P2-可观测性 | 运维 | 后端 | 16h |

### 🔵 不建议改（当前阶段投入产出比低）

| # | 问题 | 原因 | 建议 |
|---|------|------|------|
| 61 | 移动端适配 | 审核工作台是桌面端重度工具，移动端适配工作量极大（40h+），且审核员核心场景在桌面。可作为 P2 远期规划，但不是当前阶段重点 | 延后，产品定位明确为桌面端 |
| 62 | 操作撤销功能 | 审核日志是 append-only 审计trail，撤销会破坏审计完整性。如需补救应通过"二次审核"流程而非撤销 | 重新设计为二次审核机制而非撤销 |
| 63 | 全局 CSS 条件化 hover 色（为亮色主题准备） | 项目当前仅支持暗色主题，亮色切换是未来假设。过早抽象增加复杂度 | 等到真正需要亮色切换时再加 |
| 64 | 前端引入 React Query | 当前 17 个页面用 useState+useEffect 模式运行正常，引入 React Query 需要改造所有页面，工作量巨大且收益有限 | 维持现状，除非后续页面数量翻倍 |
| 65 | 审核工作台和短视频审核统一交互模式 | 两种内容形态差异大（图片卡片 vs 视频播放器+ASR），统一交互反而会降低效率 | 维持两套模式，在文档中说明差异 |
| 66 | 补充 seed 数据（reviewer/tenant_admin 等） | 仅用于开发便利，不影响生产运行 | 延后 |
| 67 | 前端引入 ESLint/Prettier | 项目 TypeScript 严格模式已提供足够的类型安全，lint 配置可延后 | 延后到团队规模扩大时 |

---

## 二、按执行人分类的执行计划

### 后端开发（Go）

#### 第一周（P0，约 12 小时）
1. ~~reviewer_id 从 JWT claims 提取~~ → 1h
2. ~~JWT secret 改用 crypto/rand~~ → 1h
3. ~~WebSocket Hub Broadcast 写锁保护~~ → 2h
4. ~~CountPendingAppeals 参数占位符修复~~ → 1h
5. ~~ContentRepository.Create 包裹事务~~ → 2h
6. ~~删除 AppealService.ResolveAppeal 重复方法~~ → 1h
7. ~~Tenant 中间件 nil role 断言加 ok 守卫~~ → 1h
8. ~~统一错误响应格式~~ → 3h

#### 第二周（P1，约 20 小时）
9. ~~添加 Rate Limiting 中间件~~ → 4h
10. ~~WebSocket Token 迁移到 Header~~ → 2h
11. ~~LiveWall WebSocket 添加 JWT 验证~~ → 2h
12. ~~main.go 优雅关闭~~ → 4h
13. ~~添加 CORS 配置~~ → 2h
14. ~~移除 StreamScheduler 模拟数据~~ → 1h
15. ~~user_repo.go 硬编码错误码修复~~ → 2h
16. ~~processVideoAsync goroutine recover~~ → 2h
17. ~~LiveWallService 占位代码清理~~ → 1h

#### 第三周（P1-P2，约 20 小时）
18. ~~audit_records 按月分区~~ → 8h
19. ~~引入 golang-migrate~~ → 4h
20. ~~CountByFilters 合并为 FILTER 聚合~~ → 4h
21. ~~Responder 统一抽取~~ → 4h
22. ~~添加 Prometheus 指标暴露~~ → 4h

### 前端开发（React + TypeScript）

#### 第一周（P0，约 4 小时）
1. ~~审核通过增加确认机制（连续 5+ 次弹出确认框）~~ → 2h
2. ~~Space 键 preventDefault 阻止页面滚动~~ → 1h
3. ~~Review.tsx 空 Table 分页改为 Pagination 组件~~ → 1h

#### 第二周（P1，约 20 小时）
4. ~~前端代码分割（React.lazy + Suspense）~~ → 4h
5. ~~消除 any 类型（api.ts + content-api.ts）~~ → 4h
6. ~~提取重复 formatDate 函数到 utils/~~ → 2h
7. ~~Dashboard Promise.allSettled 聚合~~ → 2h
8. ~~添加角色守卫（ProtectedRoute 扩展）~~ → 4h
9. ~~添加 Skeleton loading~~ → 4h

#### 第三周（P1-P2，约 24 小时）
10. ~~帮助系统（Tooltip、快捷键列表入口）~~ → 8h
11. ~~打回操作优化（Modal 一步选原因）~~ → 4h
12. ~~操作撤销（审核日志增加撤销按钮）~~ → 8h
13. ~~密码强度提示~~ → 2h
14. ~~AuditLog 操作类型图标~~ → 2h
15. ~~全局字体设置清理（Layout + Login 双重设置）~~ → 1h
16. ~~审核规则 priority min/max 约束~~ → 0.5h
17. ~~申诉管理空状态文案~~ → 1h
18. ~~WebSocket 事件监听器填充~~ → 2h

### 测试（QA）

#### 第一周（P0，约 16 小时）
1. ~~引入 testcontainers-go + testify~~ → 4h
2. ~~Handler 层 HTTP 测试（Login/Register/AuthMiddleware）~~ → 8h
3. ~~租户隔离安全测试~~ → 4h

#### 第二周（P1，约 24 小时）
4. ~~Service 层单测（AIService mock + 裁判分歧 + 402/429）~~ → 8h
5. ~~Repository 层单测（核心查询）~~ → 8h
6. ~~端到端冒烟测试（上传→AI→人审→看板）~~ → 8h

#### 第三周（持续）
7. ~~GitHub Actions CI 配置~~ → 4h
8. ~~golangci-lint + pre-commit hook~~ → 2h
9. ~~每周回归测试套件维护~~ → 持续

### 运维/DevOps

#### 第一周（P0，约 4 小时）
1. ~~修复 deploy-local.sh 健康检查路径~~ → 1h
2. ~~添加 .dockerignore 文件~~ → 0.5h
3. ~~添加 GitHub Actions CI（go build + tsc + vite build）~~ → 2.5h

#### 第二周（P1，约 12 小时）
4. ~~生产前端构建流程（Dockerfile.frontend.prod）~~ → 4h
5. ~~docker-compose 添加 CPU/内存限制~~ → 1h
6. ~~Redis requirepass 配置~~ → 1h
7. ~~MinIO 非默认凭证~~ → 1h
8. ~~数据库备份策略（每日 pg_dump）~~ → 4h
9. ~~Docker 容器非 root 运行~~ → 1h

#### 第三周（P2，约 16 小时）
10. ~~TLS 配置（Caddy/Nginx 反向代理）~~ → 8h
11. ~~PostgreSQL RLS 启用~~ → 4h
12. ~~日志聚合（ELK/Loki）~~ → 4h

### 数据库（DBA）

#### 第一周（P0，约 8 小时）
1. ~~audit_records 按月分区~~ → 4h
2. ~~引入 schema migration 工具~~ → 4h

#### 第二周（P1-P2，约 6 小时）
3. ~~audit_records (element_id, review_type) 复合索引~~ → 1h
4. ~~appeals (tenant_id, status) 复合索引~~ → 1h
5. ~~content_elements (content_id, ai_status) 复合索引~~ → 1h
6. ~~VACUUM 策略配置~~ → 1h
7. ~~ai_configs seed 数据补充~~ → 1h
8. ~~task_id 死列处理（移除或赋予用途）~~ → 1h

---

## 三、个人判断总结

### 必须改（不改不能上线）
**8 项，预估 16 小时**

这些问题直接导致线上事故风险：安全漏洞（reviewer_id 伪造、JWT 密钥不安全）、服务崩溃（WebSocket map panic）、数据不一致（孤儿记录、事务不完整）、运行时错误（SQL 参数占位符）、部署失败（健康检查路径错误）、回归风险（零测试）。每一项都应在第一周内完成。

### 改了更好（影响体验或长期维护）
**22 项，预估 168 小时**

这些问题不会导致立即崩溃，但会显著影响产品质量、可维护性或安全性。包括代码分割、类型安全、Rate Limiting、帮助系统、操作撤销、Schema 迁移等。建议第二、三周分批处理。

### 改了影响不大（锦上添花）
**30 项，预估 120 小时**

这些问题主要是 UI 细节、代码整洁、未来假设（亮色主题切换）、开发便利（seed 数据）等。可以在产品稳定后逐步处理，不阻塞上线。

### 不应该改（当前阶段）
**7 项**

- 移动端适配：桌面端重度工具，40h+ 投入产出比低
- 操作撤销：破坏审计完整性，应重新设计为二次审核机制
- 亮色主题提前抽象：当前仅需暗色，过早增加复杂度
- 引入 React Query：当前模式运行正常，改造成本远大于收益
- 统一两种审核交互模式：内容形态差异大，统一反而降低效率
- ESLint/Prettier：TypeScript 严格模式已足够，延后到团队扩大
- 补充 seed 数据：仅开发便利，不影响生产

---

## 四、总工时预估

| 阶段 | 后端 | 前端 | 测试 | 运维 | 数据库 | 合计 |
|------|------|------|------|------|--------|------|
| 第一周（P0） | 12h | 4h | 16h | 4h | 8h | **44h** |
| 第二周（P1） | 20h | 20h | 24h | 12h | 6h | **82h** |
| 第三周（P1-P2） | 20h | 24h | 持续 | 16h | 持续 | **60h+持续** |
| **总计** | **52h** | **48h** | **40h+** | **32h** | **14h** | **~190h** |

按每人 8h/天计算，约 **24 个工作日（约 5 周）** 可完成所有 P0+P1 修复。
