@echo off
REM ============================================================
REM  prefer_ipv4 fix - developer self-check (needs Go toolchain + source)
REM  Reproduces: go test green + generate both kernel configs + real kernel validation
REM  Double-click on the source machine.
REM ============================================================
setlocal enabledelayedexpansion
chcp 65001 >nul
pushd "%~dp0.."
set REPO=%CD%
set GENDIR=%REPO%\_verify_gen

echo.
echo ============================================================
echo [1/4] go test ./backend/...
echo ============================================================
go test ./backend/... 2>&1
if errorlevel 1 ( echo [X] go test FAILED & goto :end )
echo [OK] all backend tests passed

echo.
echo ============================================================
echo [2/4] Generate sample configs (anytls / vless reality) into _verify_gen
echo ============================================================
if exist "%GENDIR%" rmdir /s /q "%GENDIR%"
set GEN_CONFIG_DIR=%GENDIR%
go test ./backend/internal/proxy/ -run TestGenerateSampleConfigs -v 2>&1
if errorlevel 1 ( echo [X] generation FAILED & goto :end )

echo.
echo ------ singbox-config.json (expect server=IPv4, tls.server_name=domain, ipv4_only) ------
type "%GENDIR%\singbox-config.json"
echo.
echo ------ xray-config.json (expect vnext.address=IPv4, serverName=domain, queryStrategy=UseIPv4) ------
type "%GENDIR%\xray-config.json"

echo.
echo ============================================================
echo [3/4] Validate generated config with sing-box 1.12
echo ============================================================
".\bin\sing-box.exe" check -c "%GENDIR%\singbox-config.json"
if errorlevel 1 ( echo [X] sing-box check FAILED & goto :end )
echo [OK] sing-box config valid (1.12.x emits deprecation warnings for legacy fields - expected)

echo.
echo ============================================================
echo [4/4] Validate generated config with xray 26
echo ============================================================
".\bin\xray.exe" run -test -config "%GENDIR%\xray-config.json"
if errorlevel 1 ( echo [X] xray test FAILED & goto :end )
echo [OK] xray config valid (Configuration OK)

echo.
echo ============================================================
echo All checks passed. Cleaning up temp dir...
echo ============================================================
rmdir /s /q "%GENDIR%"

:end
popd
echo.
pause
