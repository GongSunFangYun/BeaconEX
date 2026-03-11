@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

set APP_NAME=beaconex
set OUT_DIR=build
set VERSION_DISPLAY=3.0.1
set VERSION=301

echo [INFO]  BeaconEX Build Script v%VERSION_DISPLAY%

for /f "tokens=*" %%i in ('go version 2^>^&1') do set GO_VER=%%i
echo [INFO]  Go toolchain: %GO_VER%

if not exist "%OUT_DIR%" (
    mkdir "%OUT_DIR%"
    echo [INFO]  Created output directory: %OUT_DIR%
)

echo [INFO]  Build version: %VERSION_DISPLAY% ^(filename tag: v%VERSION%^)

set PASS=0
set FAIL=0
set FAIL_LIST=

goto :start_build

:build
    set B_GOOS=%1
    set B_GOARCH=%2
    set B_PLATFORM=%3
    set B_ARCH=%4

    set B_OUT=%OUT_DIR%\%APP_NAME%-%B_PLATFORM%-%B_ARCH%-v%VERSION%
    if "%B_GOOS%"=="windows" set B_OUT=%B_OUT%.exe

    echo [INFO]  Compiling %B_GOOS%/%B_GOARCH% -^> %B_OUT%

    set GOOS=%B_GOOS%
    set GOARCH=%B_GOARCH%
    set CGO_ENABLED=0
    set GOARM=
    if "%B_GOARCH%"=="arm" set GOARM=7

    go build -trimpath -ldflags="-s -w -X main.CurrentVersion=%VERSION_DISPLAY%" -o "%B_OUT%" . 2>build_err.tmp

    if errorlevel 1 goto :build_fail

    for %%F in ("%B_OUT%") do set B_SIZE=%%~zF
    echo [INFO]  OK: %B_OUT% (!B_SIZE! bytes)
    set /a PASS+=1
    goto :build_done

    :build_fail
    echo [ERROR] FAILED: %B_GOOS%/%B_GOARCH%
    for /f "tokens=*" %%L in (build_err.tmp) do echo [ERROR]   %%L
    set /a FAIL+=1
    set FAIL_LIST=!FAIL_LIST! %B_GOOS%/%B_GOARCH%

    :build_done
    del /f /q build_err.tmp 2>nul
    set GOOS=
    set GOARCH=
    set CGO_ENABLED=
    set GOARM=
    goto :eof

:start_build

echo [INFO]  Building Windows targets...
call :build windows amd64   windows x86_64
call :build windows arm64   windows arm64
call :build windows 386     windows x86

echo [INFO]  Building Linux targets...
call :build linux amd64     linux x86_64
call :build linux arm64     linux arm64
call :build linux arm       linux armv7
call :build linux 386       linux x86
call :build linux riscv64   linux riscv64

echo [INFO]  Building macOS targets...
call :build darwin amd64    darwin x86_64
call :build darwin arm64    darwin arm64

echo [INFO]  Building FreeBSD targets...
call :build freebsd amd64   freebsd x86_64
call :build freebsd arm64   freebsd arm64

echo [INFO]  Build finished. Succeeded: %PASS%  Failed: %FAIL%
if not "%FAIL_LIST%"=="" echo [WARN]  Failed targets:%FAIL_LIST%

echo [INFO]  Artifacts in "%OUT_DIR%":
for %%F in ("%OUT_DIR%\*") do echo         %%~nxF  (%%~zF bytes)

echo.
echo Press any key to exit...
pause >nul
endlocal