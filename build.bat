@echo off
echo === 关闭旧进程 ===
taskkill /f /im MultiHostProxy.exe >nul 2>&1
echo === 清理旧文件 ===
if exist "release\MultiHostProxy.exe" del /f /q "release\MultiHostProxy.exe"
if exist "build\bin" rmdir /s /q "build\bin"
echo === 编译中 ===
wails build -clean
echo === 复制到 release ===
if not exist "release" mkdir "release"
copy /y "build\bin\MultiHostProxy.exe" "release\MultiHostProxy.exe"
echo === 完成 ===
echo 输出: release\MultiHostProxy.exe
pause
