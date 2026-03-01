@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

set APP_NAME=bex
set VERSION=3.0.0

set OUT_DIR=build

echo ==================================================
echo   BeaconEX 跨平台编译脚本
echo   版本: %VERSION%
echo ==================================================

for /f "tokens=*" %%i in ('go version 2^>^&1') do set GO_VER=%%i
echo [信息] %GO_VER%
echo --------------------------------------------------

if not exist "%OUT_DIR%" mkdir "%OUT_DIR%"

set PASS=0
set FAIL=0
set FAIL_LIST=

goto :start_build

:build
    set B_GOOS=%1
    set B_GOARCH=%2
    set B_SUFFIX=%3

    if "%B_GOOS%"=="windows" (
        set B_OUT=%OUT_DIR%\%APP_NAME%_%B_SUFFIX%.exe
    ) else (
        set B_OUT=%OUT_DIR%\%APP_NAME%_%B_SUFFIX%
    )

    echo [编译] %B_GOOS% / %B_GOARCH%  -^>  %B_OUT%

    set GOOS=%B_GOOS%
    set GOARCH=%B_GOARCH%
    go build -trimpath -ldflags="-s -w -X main.CurrentVersion=%VERSION%" -o "%B_OUT%" . 2>build_err.tmp

    if !errorlevel! == 0 (
        echo [  OK ] %B_OUT%
        set /a PASS+=1
    ) else (
        echo [ FAIL] %B_GOOS%/%B_GOARCH%
        type build_err.tmp
        set /a FAIL+=1
        set FAIL_LIST=!FAIL_LIST! %B_GOOS%/%B_GOARCH%
    )

    del /f /q build_err.tmp 2>nul
    echo.
    goto :eof

:start_build

echo 开始编译 Windows...
echo --------------------------------------------------
call :build windows amd64 windows_amd64
call :build windows arm64 windows_arm64

echo 开始编译 Linux...
echo --------------------------------------------------
call :build linux amd64 linux_amd64
call :build linux arm64 linux_arm64

echo 开始编译 macOS...
echo --------------------------------------------------
call :build darwin amd64 darwin_amd64
call :build darwin arm64 darwin_arm64

echo ==================================================
echo   编译完成
echo   成功: %PASS%  失败: %FAIL%
if not "%FAIL_LIST%"=="" (
    echo   失败平台: %FAIL_LIST%
)
echo ==================================================

echo.
echo 产物列表:
dir /b "%OUT_DIR%"

endlocal
pause