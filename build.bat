@echo off
echo === 关闭旧进程 ===
taskkill /f /im HostOVO.exe >nul 2>&1
echo === 清理 ===
if exist "release\HostOVO.exe" del /f /q "release\HostOVO.exe"
if exist "build" rmdir /s /q "build"
echo === 准备图标 ===
mkdir build 2>nul
copy /y "appicon.png" "build\appicon.png" >nul
echo === 编译 ===
wails build -clean
echo === 复制到 release ===
copy /y "build\bin\HostOVO.exe" "release\HostOVO.exe"
echo === 完成: release\HostOVO.exe ===
pause
