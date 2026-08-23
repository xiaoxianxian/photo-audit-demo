---
name: code_review_2026_07_03
description: "2026-07-03 9角色全面代码评审结果 + 修复优先级规划 + 设计师任务清单"
metadata:
  node_type: memory
  type: review
  reviewDate: "2026-07-03"
  status: "completed"
---

# 9角色全面代码评审报告（2026-07-03）

## 评审总览

| 角色 | 评分 | 必须修复 | 建议修改 | 仅供参考 |
|------|------|----------|----------|----------|
| 产品经理 | 8.7/10 | - | 2 | - |
| 架构师 | - | 3 | 4 | 2 |
| 前端工程师 | - | 3 (P0) | 9 (P1) | 5 (P2) | - |
| 后端工程师 | - | 6 (P0) | 9 (P1) | 8 (P2) | - |
| 数据库工程师 | - | 6 | 8 | 4 |
| 运维工程师 | 4.5/10 | 7 | 8 | 4 |
| 测试工程师 | 4.5/10 | 12 | 10 | 3 |
| UI设计师 | 6.5/10 | 8 | 12 | 2 |
| 普通用户 | 6.5/10 | 5 | 7 | 8 |

---

## 🔴 P0 级修复任务（立即执行）

### 安全与数据完整性（5项）

| ID | 任务 | 涉及角色 | 文件 | 工作量 |
|----|------|----------|------|--------|
| 1 | JWT签名验证修复 | 后端+架构师 | `backend/internal/service/jwt.go` + `backend/internal/middleware/auth.go` | 2h |
| 2 | 密码哈希升级（MD5→bcrypt） | 后端 | `backend/internal/service/auth_service.go` | 1h |
| 3 | CORS白名单配置 | 运维+后端 | `backend/cmd/server/main.go` | 1h |
| 4 | 硬编码凭证移除 | 运维 | `.env.example` + `deployment/docker-compose.yml` + `backend/internal/config/config.go` | 1h |
| 5 | JWT Secret持久化 | 运维+后端 | `backend/internal/config/config.go` | 1h |

### 测试体系建设（5项）

| ID | 任务 | 涉及角色 | 文件 | 工作量 |
|----|------|----------|------|--------|
| 6 | 核心service单元测试 | 测试+后端 | `ingestion_service_test.go` + `review_service_test.go` + `appeal_service_test.go` | 16h |
| 7 | 集成测试框架搭建 | 测试+后端 | `backend/internal/api/integration_test.go` | 8h |
| 8 | 前端测试框架（Vitest+RTL） | 测试+前端 | `frontend/vitest.config.ts` + 首批组件测试 | 4h |
| 9 | CI/CD流水线（GitHub Actions） | 测试+运维 | `.github/workflows/ci.yml` | 4h |
| 10 | 并发安全测试（race detector） | 测试+后端 | `go test -race` + WebSocket并发测试 | 4h |

### 运维能力补齐（6项）

| ID | 任务 | 涉及角色 | 文件 | 工作量 |
|----|------|----------|------|--------|
| 11 | Prometheus指标暴露 | 运维+后端 | 新增 `/metrics` 端点 + `prometheus/client_golang` | 4h |
| 12 | 健康检查深度化 | 运维+后端 | 新增 `/health` + `/ready` + `/live` 端点 | 2h |
| 13 | 日志轮转（lumberjack） | 运维 | `backend/internal/logger/logger.go` | 2h |
| 14 | 数据库备份策略 | 运维 | `deployment/docker-compose.yml` 添加 backup 服务 | 4h |
| 15 | Redis密码认证 | 运维 | `deployment/docker-compose.yml` | 1h |
| 16 | Docker资源限制 | 运维 | `deployment/docker-compose.full.yml` | 1h |

---

## 🎨 设计师执行任务清单（35项）

### 视觉一致性（8项必须修复）

| ID | 任务 | 文件参考 |
|----|------|----------|
| D1 | 统一颜色体系（CSS变量+TS常量） | `styles/global.css` + `utils/constants.ts` |
| D2 | 字体系统规范化 | `utils/constants.ts` 新增 FONT 常量组 |
| D3 | 间距系统规范化 | `utils/constants.ts` 新增 SPACING 常量组 |
| D4 | 圆角系统 | `utils/constants.ts` 新增 RADIUS 常量组 |
| D5 | 图标尺寸体系 | `utils/constants.ts` 新增 ICON 常量组 |
| D6 | 动画时长体系 | `utils/constants.ts` 新增 ANIMATION 常量组 |
| D7 | 密码强度可视化组件 | `components/PasswordStrength.tsx` |
| D8 | 上传进度提示UI | `pages/Review.tsx` 上传区域 |

### 组件化设计（5项必须修复）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D9 | EmptyState 通用组件 | `components/EmptyState.tsx` |
| D10 | RiskScoreBar 风险分进度条 | `components/RiskScoreBar.tsx` |
| D11 | StatusBadge 状态标签 | `components/StatusBadge.tsx` |
| D12 | ConfirmModal 确认弹窗 | `components/ConfirmModal.tsx` |
| D13 | PageHeader 页面标题 | `components/PageHeader.tsx` |

### 品牌与登录页（3项必须修复）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D14 | 设计品牌 Logo（SVG） | 输出 SVG 文件（32x32 + 128x128） |
| D15 | 登录页品牌色统一 | `pages/Login.tsx` 渐变改为品牌色 |
| D16 | 添加 Favicon | `frontend/public/favicon.*` |

### 响应式设计（4项必须修复）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D17 | 设计断点系统 | `utils/constants.ts` 新增 BREAKPOINTS |
| D18 | 审核工作台响应式布局 | `pages/Review.tsx` |
| D19 | Dashboard统计卡片响应式 | `pages/Dashboard.tsx` |
| D20 | 直播电视墙响应式 | `pages/LiveWall.tsx` |

### 无障碍访问（4项必须修复）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D21 | 颜色对比度修正 | `utils/constants.ts` textSecondary/textMuted |
| D22 | prefers-reduced-motion | `styles/global.css` |
| D23 | Skip Navigation 链接样式 | `components/Layout.tsx` |
| D24 | 图标按钮无障碍规范 | 全局图标按钮 |

### 交互设计（6项建议修改）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D25 | 批量审核模式Banner | `pages/Review.tsx` |
| D26 | Dashboard核心指标突出 | `pages/Dashboard.tsx` |
| D27 | 直播高风险流视觉强化 | `pages/LiveWall.tsx` |
| D28 | 批量打回确认弹窗 | `pages/Review.tsx` |
| D29 | Toast通知持久化设计 | 全局 message/notification |
| D30 | 短视频播放器控制UI | `pages/ShortVideoReview.tsx` |

### 动画与过渡（2项建议修改）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D31 | 统一卡片hover效果 | `styles/global.css` |
| D32 | Skeleton加载组件语义化 | `components/Skeleton.tsx` |

### 表单与输入优化（3项建议修改）

| ID | 任务 | 涉及文件 |
|----|------|----------|
| D33 | 密码强度实时反馈 | `pages/Register.tsx` |
| D34 | 上传进度提示 | `pages/Review.tsx` |
| D35 | 申诉提交后引导UI | `pages/Appeal.tsx` |

---

## 📋 执行顺序规划

### 第一轮：P0 安全修复（预计 5h）
- 任务 1→2→3→4→5
- 每完成一项：回归测试 + 代码复审
- 完成后更新记忆文件

### 第二轮：设计师任务 D1-D6（预计 4h）
- 基础设计系统统一
- 完成后更新 constants.ts

### 第三轮：P1 测试体系建设（预计 36h）
- 任务 6→7→8→9→10
- 每完成一项：回归测试 + 代码复审

### 第四轮：设计师任务 D9-D13（预计 6h）
- 通用组件抽取

### 第五轮：P2 运维能力补齐（预计 17h）
- 任务 11→12→13→14→15→16
- 每完成一项：回归测试 + 代码复审

### 第六轮：设计师任务 D14-D35（预计 12h）
- 品牌/响应式/无障碍/交互优化

---

## ⚠️ 上下文长度管理

- 当前评审报告总字数约 20,000+ 字
- 每轮修复任务单独会话执行，避免上下文溢出
- 每轮完成后更新本记忆文件，记录已完成任务和待办事项

---

## 🔄 每轮执行流程

1. **执行修复任务**
2. **回归测试：**
   - 前端：`tsc --noEmit` + `vite build`
   - 后端：`go build ./...` + `go test ./...`
3. **代码复审：** 检查是否符合评审建议
4. **更新记忆文件：** 记录完成情况
5. **准备下一轮**
