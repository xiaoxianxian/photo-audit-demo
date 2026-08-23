# P0 安全修复执行计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。

**目标：** 修复9角色代码评审中发现的5项P0级安全问题（JWT验签、密码哈希、CORS、硬编码凭证、JWT Secret持久化）

**架构：** 每项修复独立执行，完成后立即回归测试+代码复审，确保不引入新问题

**技术栈：** Go 1.23+, Fiber 2.x, JWT, bcrypt, PostgreSQL

---

## 任务 1：JWT签名验证修复

**文件：**
- 修改：`backend/internal/service/jwt.go` — ValidateToken 方法
- 修改：`backend/internal/middleware/auth.go` — JWT中间件
- 修改：`backend/internal/config/config.go` — JWT_SECRET环境变量

- [ ] **步骤 1：检查当前 JWT 验证逻辑**

读取 `backend/internal/service/jwt.go` 的 `ValidateToken` 方法，确认是否存在签名验证缺失。

- [ ] **步骤 2：修复 JWT 签名验证**

确保 `ValidateToken` 方法正确验证 JWT 签名（HS256算法）：

```go
func (s *JWTService) ValidateToken(tokenString string) (*model.User, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(s.secret), nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }
    
    // 提取 user_id 并查询用户
    userID := claims["user_id"].(string)
    // ... 查询用户逻辑
}
```

- [ ] **步骤 3：运行后端编译验证**

运行：`cd backend && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 4：代码复审**

检查项：
- [ ] 签名验证使用 HMAC（HS256）
- [ ] 密钥从环境变量读取，非硬编码
- [ ] 无效Token返回401错误
- [ ] 过期Token返回401错误

- [ ] **步骤 5：Commit**

```bash
git add backend/internal/service/jwt.go backend/internal/middleware/auth.go
git commit -m "fix(security): add JWT signature verification to prevent token forgery"
```

---

## 任务 2：密码哈希升级（MD5→bcrypt）

**文件：**
- 修改：`backend/internal/service/auth_service.go` — Register 和 Login 方法
- 修改：`backend/internal/model/user.go` — 确认 Password 字段类型

- [ ] **步骤 1：检查当前密码哈希方式**

读取 `backend/internal/service/auth_service.go`，确认是否存在 MD5 哈希。

- [ ] **步骤 2：替换 MD5 为 bcrypt**

修改 Register 方法：

```go
import "golang.org/x/crypto/bcrypt"

func (s *AuthService) Register(input RegisterInput) (*model.User, error) {
    // ... 现有逻辑
    
    // 替换：hashedPassword, err := md5.Sum([]byte(input.Password))
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    
    user.Password = string(hashedPassword)
    // ...
}
```

修改 Login 方法：

```go
func (s *AuthService) Login(username, password string) (*model.User, error) {
    // ... 查询用户逻辑
    
    // 替换：md5验证
    err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        return nil, fmt.Errorf("invalid credentials")
    }
    
    // ...
}
```

- [ ] **步骤 3：运行后端编译验证**

运行：`cd backend && go build ./...`
预期：编译成功

- [ ] **步骤 4：代码复审**

检查项：
- [ ] Register 使用 `bcrypt.GenerateFromPassword`
- [ ] Login 使用 `bcrypt.CompareHashAndPassword`
- [ ] 无残留 MD5 哈希调用
- [ ] 错误处理完整

- [ ] **步骤 5：Commit**

```bash
git add backend/internal/service/auth_service.go
git commit -m "fix(security): upgrade password hashing from MD5 to bcrypt"
```

---

## 任务 3：CORS白名单配置

**文件：**
- 修改：`backend/cmd/server/main.go` — CORS 配置
- 修改：`.env.example` — 添加 CORS_ALLOWED_ORIGINS

- [ ] **步骤 1：检查当前 CORS 配置**

读取 `backend/cmd/server/main.go` 的 CORS 配置部分。

- [ ] **步骤 2：修改 CORS 配置为白名单模式**

```go
allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowedOrigins == "" {
    allowedOrigins = "http://localhost:3000" // 开发环境默认值
}

origins := strings.Split(allowedOrigins, ",")

app.Use(cors.New(cors.Config{
    AllowOrigins: origins,
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID"},
    AllowCredentials: true,
    MaxAge: 86400,
}))
```

- [ ] **步骤 3：更新 .env.example**

在 `.env.example` 中添加：

```
# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://audit.example.com
```

- [ ] **步骤 4：运行后端编译验证**

运行：`cd backend && go build ./...`
预期：编译成功

- [ ] **步骤 5：代码复审**

检查项：
- [ ] AllowOrigins 不为 "*"
- [ ] 支持多域名逗号分隔
- [ ] AllowCredentials: true
- [ ] .env.example 有占位符

- [ ] **步骤 6：Commit**

```bash
git add backend/cmd/server/main.go .env.example
git commit -m "fix(security): restrict CORS to whitelist origins instead of allowing all"
```

---

## 任务 4：硬编码凭证移除

**文件：**
- 修改：`.env.example` — 数据库/MinIO默认凭证
- 修改：`deployment/docker-compose.yml` — 使用环境变量引用
- 修改：`deployment/photo-audit.service` — 使用 EnvironmentFile

- [ ] **步骤 1：检查硬编码凭证位置**

扫描以下文件中的硬编码密码：
- `.env.example`
- `deployment/docker-compose.yml`
- `deployment/photo-audit.service`

- [ ] **步骤 2：修改 .env.example**

```bash
# 数据库
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<CHANGE_ME_IN_PRODUCTION>  # 替换 postgres:postgres
DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/photo_audit

# MinIO
MINIO_ROOT_USER=<CHANGE_ME_IN_PRODUCTION>
MINIO_ROOT_PASSWORD=<CHANGE_ME_IN_PRODUCTION>
```

- [ ] **步骤 3：修改 docker-compose.yml**

```yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}  # 使用环境变量
    # ...

  minio:
    image: minio/minio:RELEASE.2024-06-28
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
    # ...
```

- [ ] **步骤 4：修改 systemd 单元文件**

```ini
[Service]
EnvironmentFile=/etc/photo-audit/.env  # 改为加载环境变量文件
# 移除：Environment=DATABASE_URL=postgresql://postgres:postgres@...
```

- [ ] **步骤 5：代码复审**

检查项：
- [ ] .env.example 无明文密码
- [ ] docker-compose.yml 使用 ${VAR:-default} 语法
- [ ] systemd 单元使用 EnvironmentFile
- [ ] 无残留硬编码密码

- [ ] **步骤 6：Commit**

```bash
git add .env.example deployment/docker-compose.yml deployment/photo-audit.service
git commit -m "fix(security): remove hardcoded credentials, use environment variables"
```

---

## 任务 5：JWT Secret持久化

**文件：**
- 修改：`backend/internal/config/config.go` — JWT_SECRET 加载逻辑
- 修改：`.env.example` — 添加 JWT_SECRET 占位符

- [ ] **步骤 1：检查当前 JWT Secret 生成逻辑**

读取 `backend/internal/config/config.go` 的 `generateJWTSecret()` 方法。

- [ ] **步骤 2：修改为从环境变量读取**

```go
// 替换：func generateJWTSecret() string { return uuid.New().String() }
func getJWTSecret() string {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }
    return secret
}
```

- [ ] **步骤 3：更新 .env.example**

```
# JWT
JWT_SECRET=<GENERATE_WITH_jwt_secret_gen_cli_tool>  # 至少32字符
JWT_EXPIRATION=24h
```

- [ ] **步骤 4：运行后端编译验证**

运行：`cd backend && go build ./...`
预期：编译成功

- [ ] **步骤 5：代码复审**

检查项：
- [ ] JWT_SECRET 从环境变量读取
- [ ] 无自动生成逻辑
- [ ] 缺失时启动失败（而非使用随机值）
- [ ] .env.example 有提示

- [ ] **步骤 6：Commit**

```bash
git add backend/internal/config/config.go .env.example
git commit -m "fix(security): persist JWT_SECRET in environment variable instead of regenerating on startup"
```

---

## 回归测试清单

每项任务完成后，必须执行以下回归测试：

### 前端验证
```bash
cd frontend && npx tsc --noEmit
cd frontend && npm run build
```
预期：0 errors, 构建成功

### 后端验证
```bash
cd backend && go build ./...
cd backend && go vet ./...
```
预期：编译成功，无 vet 错误

### 安全扫描
```bash
# 检查残留硬编码密码
grep -r "postgres:postgres" backend/ deployment/
grep -r "minioadmin:minioadmin" backend/ deployment/
grep -r "md5.Sum" backend/
```
预期：无结果

### 功能测试
1. 注册新用户 → 密码使用 bcrypt 哈希
2. 登录 → JWT 签名验证通过
3. 伪造 JWT Token → 401 Unauthorized
4. CORS 跨域请求 → 仅允许白名单域名

---

## 执行顺序

1. **任务1** (JWT验签) → 回归测试
2. **任务2** (bcrypt) → 回归测试
3. **任务3** (CORS) → 回归测试
4. **任务4** (硬编码凭证) → 回归测试
5. **任务5** (JWT Secret) → 回归测试

**总预计工作量：** 约 9 小时

**注意：** 每项任务完成后更新本计划，标记已完成项，并更新 `memory/code_review_2026_07_03.md` 记忆文件。
