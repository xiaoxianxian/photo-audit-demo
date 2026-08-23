@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0"

echo.
echo ========================================
echo   Photo Audit Platform — 本地部署
echo ========================================
echo.

REM Check Docker
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] 未找到 Docker，请先安装 Docker Desktop
    pause
    exit /b 1
)

docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Docker 守护进程未运行，请启动 Docker Desktop
    pause
    exit /b 1
)

echo [OK] Docker 已安装并运行
echo.

echo [INFO] 正在清理旧容器...
docker compose -f deployment\docker-compose.full.yml down --remove-orphans 2>nul

echo [INFO] 正在启动所有服务...
docker compose -f deployment\docker-compose.full.yml up -d --build

echo [INFO] 等待后端就绪 (最多 60 秒)...
set /a waited=0
:wait_loop
curl -sf http://localhost:8080/health >nul 2>&1
if !errorlevel! equ 0 (
    echo [OK] 后端已就绪 (等待 !waited! 秒)
    goto :open_browser
)
set /a waited+=1
if !waited! geq 60 (
    echo [ERROR] 后端启动超时
    docker compose -f deployment\docker-compose.full.yml logs backend
    pause
    exit /b 1
)
timeout /t 1 /nobreak >nul
goto :wait_loop

:open_browser
echo.
echo ========================================
echo   全部就绪！
echo ========================================
echo   前端: http://localhost:3000
echo   后端: http://localhost:8080
echo   MinIO: http://localhost:9001  (minioadmin/minioadmin)
echo.

start http://localhost:3000
echo [INFO] 已自动打开浏览器
echo.
pause
