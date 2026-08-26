# Photo Audit Platform — 部署验收操作手册

> 本文档用于在本地环境中启动并验证 Phase 1 MVP 的全部功能。

## 前置条件

确保本地已安装：
- Docker Desktop（含 docker compose）
- Node.js 20+（可选，用于本地开发模式）
- Go 1.23+（可选，用于本地开发模式）

## 方式一：Docker 全栈部署（推荐）

### 1. 启动基础设施

```bash
cd /Users/xiaota/Documents/Photo-Audit-Demo/deployment
docker compose -f docker-compose.full.yml up -d
```

这会启动 5 个服务：
- **PostgreSQL** (端口 5432) — 自动执行 init.sql 建表
- **Redis** (端口 6379)
- **MinIO** (端口 9000/9001) — 对象存储 + 控制台
- **Backend** (端口 8080) — Go Fiber API
- **Frontend** (端口 3000) — Vite 开发服务器

### 2. 等待服务就绪

```bash
# 检查所有容器状态
docker compose -f docker-compose.full.yml ps

# 查看后端日志
docker compose -f docker-compose.full.yml logs -f backend
```

后端就绪后，health 端点会返回 200：

```bash
curl http://localhost:8080/health
# 期望响应: {"status":"ok","version":"0.1.0"}
```

### 3. 访问应用

- **前端:** http://localhost:3000
- **后端 API:** http://localhost:8080
- **MinIO 控制台:** http://localhost:9001 (minioadmin/minioadmin)

### 4. 停止服务

```bash
docker compose -f docker-compose.full.yml down
```

---

## 方式二：本地开发模式（前后端分离）

### 1. 启动基础设施

```bash
cd deployment
docker compose up -d postgres redis minio minio-mc
```

### 2. 初始化数据库

```bash
docker exec -i $(docker compose -f docker-compose.yml ps -q postgres) psql -U postgres -d photo_audit < ../backend/sql/init.sql
```

### 3. 启动后端

```bash
cd backend
export DATABASE_URL=postgresql://postgres:postgres@localhost:5432/photo_audit
export REDIS_URL=redis://localhost:6379
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
export MINIO_BUCKET=audit-platform
export FALLBACK_ENABLED=true

go mod tidy && go build -o audit-server ./cmd/server/ && ./audit-server
```

### 4. 启动前端（新终端）

```bash
cd frontend
npm install
npm run dev
```

---

## 验收操作步骤

### 第一步：注册 + 登录

1. 访问 http://localhost:3000/register
2. 选择"创建新租户"，填写：
   - 租户名称：TestTenant
   - 国家代码：CN
   - 用户名：admin
   - 密码：admin123
3. 注册成功后自动登录，跳转仪表盘

### 第二步：租户管理

1. 在仪表盘"租户管理"Tab：
   - 查看租户列表（应显示刚创建的租户）
   - 编辑租户名称
   - 删除租户（软删除）

### 第三步：团队管理

1. 切换到"团队管理"Tab：
   - 创建团队，指定负责人
   - 添加团队成员
   - 删除成员

### 第四步：内容上传

1. 进入审核工作台 `/review`
2. 拖拽一张图片到上传区域
3. 等待 AI 审核完成（fallback 本地规则引擎）
4. 查看元素卡片网格

### 第五步：人工审核

1. 在审核工作台查看待审元素卡片
2. 测试键盘快捷键：
   - `Enter`/`Space` → 通过
   - `Esc` → 打回
   - `←`/`→` → 切换元素
3. 勾选多个元素，批量通过

### 第六步：申诉管理

1. 访问 `/appeals`
2. 查看申诉列表（Tabs 切换）
3. 点击申诉详情，验证显示原始 AI 审核结果
4. 点击"改判通过"或"维持原判"，验证状态变更

### 第七步：租户配置

1. 访问 `/tenant-config`
2. 切换到"审核规则"Tab，创建一条规则
3. 切换到"判罚等级"Tab，创建等级
4. 切换到"敏感词库"Tab，添加敏感词

### 第八步：质量抽检

1. 访问 `/quality-audit`
2. 点击"创建批次"，填写筛选条件
3. 打开批次详情，验证样本列表和统计面板
4. 提交抽检评分

### 第九步：审核日志

1. 访问 `/audit-log`
2. 验证日志表格渲染
3. 使用筛选器测试 action 和 review_type 过滤

### 第十步：直播电视墙

1. 访问 `/live-wall`
2. 验证流网格显示，查看在线/离线状态区分
3. 点击"启动直播"，填写内容 ID 和流密钥
4. 验证离线流显示灰显 + OFFLINE 标签

### 第十一步：AI 模型配置

1. 访问 `/ai-config`
2. 填写 Agnes API Key 和 DeepSeek API Key（可选，留空会使用 fallback）
3. 保存后验证配置持久化

---

## 验收检查清单

| # | 检查项 | 预期结果 | 实际结果 |
|---|--------|----------|----------|
| 1 | 前端页面加载 | http://localhost:3000 正常打开 | |
| 2 | 注册功能 | 创建租户 + 用户成功，自动登录 | |
| 3 | 登录功能 | 用户名密码正确可登录 | |
| 4 | 租户 CRUD | 创建/编辑/删除正常 | |
| 5 | 团队管理 | 创建团队 + 添加成员正常 | |
| 6 | 图片上传 | 拖拽上传成功，AI 审核触发 | |
| 7 | 审核工作台 | 元素卡片网格显示，快捷键可用 | |
| 8 | 批量审核 | 多选元素批量通过正常 | |
| 9 | 申诉列表 | Tabs 切换正常，详情含原始 AI 结果 | |
| 10 | 申诉改判 | 改判/维持操作正常 | |
| 11 | 租户配置 | 规则/等级/敏感词 CRUD 正常 | |
| 12 | 质量抽检 | 创建批次 + 提交评分正常 | |
| 13 | 审核日志 | 列表显示 + 筛选器正常 | |
| 14 | 直播电视墙 | 启停流 + 网格显示正常 | |
| 15 | AI 模型配置 | 保存 API Key 正常 | |
| 16 | 租户隔离 | 不同租户数据不互通 | |
| 17 | WebSocket | 审核工作台连接状态指示正常 | |
| 18 | 设计系统 | 颜色/字号/圆角/阴影统一 | |
| 19 | 健康检查 | curl http://localhost:8080/health 返回 200 | |
| 20 | 审核通过确认 | 连续 5+ 次通过弹出警告 | |
| 21 | 角色守卫 | 非管理员无法访问租户配置 | |
