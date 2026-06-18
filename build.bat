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
echo === 生成托盘图标 ===
if not exist "assets" mkdir assets
powershell -Command "$bmp=[System.Drawing.Bitmap]::FromFile('appicon.png');$h=$bmp.GetHicon();$ico=[System.Drawing.Icon]::FromHandle($h);$f=[System.IO.File]::Create('assets\tray.ico');$ico.Save($f);$f.Close();$ico.Dispose();$bmp.Dispose()"
echo === 编译 ===
wails build
echo === 复制到 release ===
copy /y "build\bin\HostOVO.exe" "release\HostOVO.exe"
echo === 完成: release\HostOVO.exe ===
pause
