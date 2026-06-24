@echo off
echo === 关闭旧进程 ===
taskkill /f /im OvOproxy.exe >nul 2>&1
echo === 清理 ===
if exist "release\OvOproxy.exe" del /f /q "release\OvOproxy.exe"
if exist "build" rmdir /s /q "build"
echo === 清理前端缓存 ===
if exist "frontend\dist" rmdir /s /q "frontend\dist"
if exist "frontend\.vite" rmdir /s /q "frontend\.vite"
if exist "frontend\node_modules\.vite" rmdir /s /q "frontend\node_modules\.vite"
echo === 重新构建前端 ===
cd frontend
call npm install
call npm run build
cd ..
echo === 准备图标 ===
if not exist "build" mkdir build
copy /y "appicon.png" "build\appicon.png" >nul
if exist "build\windows\icon.ico" del /f /q "build\windows\icon.ico"
echo === 编译 (版本号请在此修改) ===
set APP_VERSION=v1.0.1
wails build -ldflags "-X main.Version=%APP_VERSION%"
echo === 复制到 release ===
if not exist "release" mkdir release
copy /y "build\bin\OvOproxy.exe" "release\OvOproxy.exe"
echo === 复制配置文件 ===
if not exist "release\configs" xcopy /y /e /i "configs.example" "release\configs" >nul
echo === 复制配置模板（分发用） ===
xcopy /y /e /i "configs.example" "release\configs.example" >nul
echo === 完成: release\OvOproxy.exe ===
pause
