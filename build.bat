@echo off
echo === 关闭旧进程 ===
taskkill /f /im ZephyHosts.exe >nul 2>&1
echo === 清理 ===
if exist "release\ZephyHosts.exe" del /f /q "release\ZephyHosts.exe"
if exist "build" rmdir /s /q "build"
echo === 编译 ===
wails build -clean
echo === 复制到 release ===
copy /y "build\bin\ZephyHosts.exe" "release\ZephyHosts.exe"
echo === 完成: release\ZephyHosts.exe ===
pause
