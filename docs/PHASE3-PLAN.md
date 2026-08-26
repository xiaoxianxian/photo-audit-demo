# Phase 3 规划与决策记录

> 创建时间：2026-08-26
> 状态：规划中，未启动实施
> 触发条件：数据量 / 并发量 / 产品 PRD 成熟

---

## 一、Phase 2 完成情况（前置上下文）

| 子系统 | Commit | 状态 |
|--------|--------|------|
| Kafka 审核任务队列 | `ddaf70e1` | ✅ 已完成 |
| Elasticsearch 全文检索 | `836b599d` | ✅ 已完成（comment 字段 edge_ngram） |
| WebRTC WHIP/WHEP 信令 | `be5b375f` | ✅ 已完成（信令面，媒体 P2P） |
| 租户 RBAC 门禁 | `447b6763` | ✅ 已完成 |

Phase 1 MVP + Phase 2 全部完成，进入稳定期。

---

## 二、Phase 3 规划任务

### T3-1：ES 运营报表聚合

**触发条件（满足任一即启动）**：
- [ ] 审核记录 > 100 万条（PG `COUNT(*)` 慢查询阈值）
- [ ] 报表 PRD 定稿（确认多维下钻：按 reviewer / reason / tenant / review_type 组合过滤）
- [ ] 前端报表界面设计稿交付

**当前方案占位**：
- 复用现有 `GET /review/logs/search` 搜索通道（ES filter + PG 降级）
- 新增 `GET /dashboard/audit-stats`：返回近 30 天按天/按审核员/按原因分类的聚合
- 前端 `Dashboard.tsx` 新增「运营报表」Tab，ECharts 折线图 + 饼图

**不做当前做的依据**：

**性能基准实测（2026-08-26）**：
```
dashboard/stats 单次：14ms
dashboard/stats 10次平均：2.7ms
PG 计算无瓶颈，10 万条以内完全够用
```

**核心原因**：
1. **数据量未达阈值**：当前 PG `COUNT(*)` + `GROUP BY date_trunc('day', created_at)` 在 10 万条以内 <50ms，ES 优势还没体现
2. **报表 PRD 未定**：产品形态（图表 vs 表格）、维度组合、时间粒度均无文档，写代码是空对空
3. **索引 schema 一旦确定难改**：edge_ngram 分词器、字段类型定下来后加新维度需重建索引
4. **已有兜底路径**：SearchAuditLogs 已实现 ES 优先 + PG 降级，未来报表层可直接复用

**预估工作量**：后端 200-300 行 + 前端 150 行 + Kibana 仪表板（可选）

---

### T3-2：K8s 部署配置

**触发条件**：
- [ ] 稳定版决定（当前为 demo/开发环境，生产需正式 release）
- [ ] CI/CD 流水线搭建（GitHub Actions / GitLab CI）

**当前方案占位**：
- `deployment/k8s/` 目录已存在（空占位）
- 将 docker-compose 中的 5 个服务拆分为 k8s Deployment + Service + PVC
- 涉及：postgres / redis / minio / kafka / elasticsearch + backend + frontend

**预估工作量**：yaml 模板 ~400 行 + Helm chart（可选）+ CI/CD pipeline ~150 行

---

### T3-3：媒体面 SRS 接入

**触发条件**：
- [ ] 真实用户并发观看到 50+（当前 P2P 上限约 20-30 路）
- [ ] 需要录制 / 录像回放 / 多路混流能力

**当前方案占位**：
- 信令面已解耦（WHIP/WHEP 端点在 `internal/api/signaling_handlers.go`）
- 后续可接入 SRS 或 mediamtx 做媒体转发（SFU 模式）
- 媒体流从 P2P 改走 SFU 需要重新设计 NAT 穿透 + 带宽监控

**不做当前做的依据**：
1. **独立部署复杂度**：SRS/mediamtx 需单独进程 + UDP 端口范围 + TLS 证书 + STUN/TURN
2. **架构大改**：媒体流从 P2P 改走 SFU → NAT 穿透 + 录像存储是新子系统
3. **场景未收敛**：审核场景真实并发观看人数不确定（目前是演示/内网）
4. **升级路径已预留**：CLAUDE.md 明确写了"媒体面 P2P，后续规模化可加 SRS/mediamtx 做媒体转发"

**预估工作量**：独立 SRS 进程部署 + 信令适配（~150 行）+ 前端播放器改造（~100 行）

---

### T3-4：Redis 缓存层

**触发条件**：
- [ ] 高 QPS 场景（> 100 QPS）
- [ ] 电视墙状态 / 审核任务列表实时性要求极高

**当前方案占位**：
- Redis 已用于会话管理（auth middleware）和电视墙 WebSocket 广播
- 待加：审核任务列表缓存、Dashboard 统计缓存（TTL 30s）

**预估工作量**：~200 行（缓存 key 设计 + 失效策略 + 一致性保证）

---

## 三、技术债务 / 注意事项

### 3.1 文档一致性（待修复）

| 位置 | 问题 | 状态 |
|------|------|------|
| `CLAUDE.md:489` | 仍写"Elasticsearch 全文检索：审核记录搜索 + 运营报表"，应标记为 `[x]` 完成 | ⚠️ 待修复 |
| `CLAUDE.md:12` | 技术栈描述"ES（Phase 3 引入）"过时 | ⚠️ 待修复 |
| `CLAUDE.md:47` | `.env.example` 中 ES 环境变量注释"Phase 3"过时 | ⚠️ 待修复 |

### 3.2 架构备注

| 项目 | 说明 |
|------|------|
| **WebRTC 媒体面 P2P** | 信令过服务器，媒体直连浏览器；小规模够用，规模化需 SRS |
| **Phase 3 目录占位** | `deployment/k8s/` 存在但为空；临时占位，不用删除 |
| **docker-compose 依赖** | 5 个服务（PG/Redis/MinIO/Kafka/ES）本地开发约 4GB 内存占用 |

### 3.3 容器资源实测（2026-08-26）

| 服务 | 内存 |
|------|------|
| PostgreSQL | 39MB |
| Kafka | 984MB |
| Elasticsearch | 1005MB |
| **Photo-Audit 合计** | **~2GB** |
| **全宿主（含 hujing-ai 等）** | **3.4GB** |

**精简建议**：
- Kafka 和 ES 占内存大头（各 ~1GB），若暂时不用可注释掉 docker-compose 中的服务定义
- MinIO 实际测试用 mock 替代可省 ~100MB

### 3.4 已知坑（本轮新增）

1. **handlers.go 大括号计数**：patch 时吞掉 struct 字面量收尾 `}` 导致函数不闭合，后续签名全解析错 → 用 `awk '{print}' | grep -c '{'` 对比数检查平衡
2. **edge_ngram 索引重建**：新加分析器必须删旧索引重建，否则字段 mapping 不生效
3. **Kafka DialLeader 无限阻塞**：已改为 DialContext + CreateTopics（5s 超时）
4. **公开注册防提权**：P0-2 校验拦截 tenant_admin/quality_checker 自封，需在注册页提示用户找管理员创建账号

---

## 四、下一步执行建议

### 推荐顺序

1. **CLAUDE.md 同步**（3 处修改，立即做） ← 待老板批准
2. **容器资源精简**（按需启用 Kafka/ES，~30 分钟）
3. **Phase 3 PRD 等待**（等产品侧确定报表形态 + 数据量增长后再启动 T3-1/T3-3）

### 不推荐立即做

- **T3-1 ES 报表**：无 PRD、数据量未达阈值，写了也是空对空
- **T3-3 SRS 接入**：场景未收敛，过早规模化是过度工程

---

## 五、验收标准（未来 Phase 3 启动时对照）

- [ ] Phase 3 规划文档与代码现状一致
- [ ] CLAUDE.md Phase 2 状态段全部标记 `[x]`
- [ ] 性能基准测试报告记录 PG 慢查询阈值
- [ ] 容器资源评估结论（是否精简）
- [ ] Phase 3 任一任务启动前，触发条件 checklist 逐项确认

---

_本文档由 Agnes 于 2026-08-26 创建，随迭代持续更新。_
