@echo off
echo === 关闭旧进程 ===
taskkill /f /im OvOproxy.exe >nul 2>&1
echo === 清理 ===
if exist "release\OvOproxy.exe" del /f /q "release\OvOproxy.exe"
if exist "build" rmdir /s /q "build"
echo === 准备图标 ===
if not exist "build" mkdir build
copy /y "appicon.png" "build\appicon.png" >nul
if exist "build\windows\icon.ico" del /f /q "build\windows\icon.ico"
echo === 编译 ===
wails build
echo === 复制到 release ===
if not exist "release" mkdir release
copy /y "build\bin\OvOproxy.exe" "release\OvOproxy.exe"
echo === 复制配置文件 ===
xcopy /y /e /i "configs" "release\configs" >nul
echo === 完成: release\OvOproxy.exe ===
pause
