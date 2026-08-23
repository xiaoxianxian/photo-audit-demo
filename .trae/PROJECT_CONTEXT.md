# Photo Audit Demo — 项目对话上下文

## 项目概述

这是一个**摄影平台供稿审核后台 MVP**，使用 **HTML + Tailwind CSS + 原生 JavaScript** 构建的单文件前端应用（`index.html`）。

## 启动方式

```bash
cd /path/to/Photo-Audit-Demo
python3 -m http.server 8088
# 访问 http://localhost:8088
```

## 核心功能

1. **AI 图片审核**：上传图片后由 AI 模型自动审核（安全评分 + 结论）
2. **裁判模型（Judge）**：DeepSeek 对主审结论进行逻辑一致性校验（0-100 分）
3. **批量上传**：支持多张图片同时上传（串行处理，间隔 800ms 避免频率限制）
4. **审核日志**：完整的审核历史记录
5. **申诉入口**：摄影师可申诉被拒绝的作品
6. **Mock 数据**：预设的待审卡片铺满屏幕
7. **Lightbox**：点击图片悬浮查看原图
8. **自适应卡片**：卡片尺寸跟随图片比例
9. **Supabase 持久化**：审核任务存储到 Supabase（可选）

## 主审模型支持

| 模型 | API | 视觉格式 |
|------|-----|---------|
| **Gemini Flash** (默认) | `generativelanguage.googleapis.com` | `inline_data` (Base64) |
| **GPT-4o** | `api.openai.com/v1/chat/completions` | `image_url` |
| **Kimi** | `api.moonshot.cn/v1/chat/completions` | `image_url` |

⚠️ **DeepSeek Chat 不支持图片输入**，只能作文本裁判模型。已从主审选项中移除。

## API Key 管理

所有 Key 存储在 `localStorage`，必须手动填写：

| localStorage Key | 说明 |
|-----------------|------|
| `photoAudit_primaryKey` | 主审模型 API Key |
| `photoAudit_primaryModel` | 模型选择：`gemini` / `gpt` / `kimi` |
| `photoAudit_judgeKey` | 裁判模型 API Key（DeepSeek） |

- 顶部黄色横幅：输入 Key + 选择模型 + 裁判 Key
- 绿色横幅：显示已连接状态 + 修改配置 + 重置全部
- 未填写主审 Key 时，上传区域会提示并聚焦到输入框

## 关键技术细节

### AI 审核流程
1. `processSingleFile(file)`：Base64 编码图片（Canvas 缩放到 max 1024px）
2. `callPrimaryModel(base64, mimeType)`：分发到对应模型
3. `callJudgeForReview(primaryResult)`：DeepSeek 纯文本逻辑校验
4. `createReviewCard(result, fileName, ...)`：生成审核卡片

### 错误处理
- `QuotaExhaustedError`：区分 `'rate_limit'`（429 频率限制）和 `'quota'`（402 额度耗尽）
- `detectQuotaError(provider, status, errorText)` 现在精确匹配，避免误判
- 上传是**串行**的（每张图间隔 800ms），避免触发 API 频率限制

### 裁判逻辑
- 主审成功 → 裁判评分 → 卡片边框颜色：绿(>80) / 橙(50-80) / 红(<50)
- 主审失败 → 卡片边框灰色，显示异常信息
- 裁判无需 Key → `consistency_score` 和 `judge_comment` 不存在

## 重要文件

| 文件 | 说明 |
|------|------|
| `index.html` | 主文件，包含所有 HTML/CSS/JS |
| `README.md` | 项目说明 |
| `prd.md` | 产品需求文档 |
| `tech-arch.md` | 技术架构设计 |
| `.trae/PROJECT_CONTEXT.md` | 本文件：对话上下文 |

## 历史修复记录

1. **重置按钮无反应**：`window.resetDemo` 存在但无按钮调用 → 添加"重置全部"按钮
2. **上传区域无反应**：`triggerUpload` 用硬编码 Key → 改为检查 localStorage Key + 提示
3. **Kimi 额度误报**：`detectQuotaError` 所有 429 都返回 rate_limit → 改为精确匹配关键词
4. **DeepSeek 400 错误**：用了 `image_url` 但 API 不支持 → 移除主审 DeepSeek 选项
5. **并行请求触发频率限制**：`Promise.allSettled` 同时发 N 个请求 → 改为串行 for 循环
6. **裁判优先级**：主审失败优先处理，其次才看裁判分数

## 本期对话核心改动（最近一次）

- **主审不再支持 DeepSeek Vision**（API 不支持图片输入）
- **删除顶部横幅**的"获取 Key →"链接和 x 关闭按钮
- **detectQuotaError 逻辑收紧**：429 只在确认为 rate limit 时才返回 `'rate_limit'`
- **串行上传**替代并行，间隔 800ms
