@echo off
REM Ant Browser Personal - prefer_ipv4 fix runtime verification (run in TEST env)
REM Start the problem node in ant-chrome.exe FIRST, then double-click this file.
REM Optional: pass the socks port explicitly, e.g.  verify-proxy.bat 41080
setlocal
chcp 65001 >nul
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify-proxy.ps1" %*
echo.
pause
