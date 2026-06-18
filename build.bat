@echo off
echo === 关闭旧进程 ===
taskkill /f /im HostOVO.exe >nul 2>&1
echo === 清理 ===
if exist "release\HostOVO.exe" del /f /q "release\HostOVO.exe"
if exist "build" rmdir /s /q "build"
echo === 准备图标 ===
if not exist "build" mkdir build
copy /y "appicon.png" "build\appicon.png" >nul
if exist "build\windows\icon.ico" del /f /q "build\windows\icon.ico"
echo === 编译 ===
wails build
echo === 复制到 release ===
copy /y "build\bin\HostOVO.exe" "release\HostOVO.exe"
echo === 完成: release\HostOVO.exe ===
pause
