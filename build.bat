@echo off
echo === 关闭旧进程 ===
taskkill /f /im MultiHostProxy.exe >nul 2>&1
echo === 清理 ===
if exist "release\MultiHostProxy.exe" del /f /q "release\MultiHostProxy.exe"
if exist "build" rmdir /s /q "build"
echo === 编译 (输出到 release\) ===
wails build -clean
echo === 完成: release\MultiHostProxy.exe ===
pause
