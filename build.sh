#!/bin/bash

APP_NAME="bex"
VERSION="3.0.0"
OUT_DIR="build"

echo "=================================================="
echo "   BeaconEX 跨平台编译脚本"
echo "   版本: $VERSION"
echo "=================================================="

GO_VER=$(go version 2>&1)
echo "[信息] $GO_VER"
echo "--------------------------------------------------"

mkdir -p "$OUT_DIR"

PASS=0
FAIL=0
FAIL_LIST=""

build() {
    B_GOOS=$1
    B_GOARCH=$2
    B_SUFFIX=$3

    if [ "$B_GOOS" = "windows" ]; then
        B_OUT="$OUT_DIR/${APP_NAME}_${B_SUFFIX}.exe"
    else
        B_OUT="$OUT_DIR/${APP_NAME}_${B_SUFFIX}"
    fi

    echo "[编译] $B_GOOS / $B_GOARCH  ->  $B_OUT"

    export GOOS=$B_GOOS
    export GOARCH=$B_GOARCH
    export CGO_ENABLED=0

    go build -trimpath -ldflags="-s -w -X main.CurrentVersion=$VERSION" -o "$B_OUT" . 2>build_err.tmp

    if [ $? -eq 0 ]; then
        echo "[  OK ] $B_OUT"
        PASS=$((PASS + 1))
    else
        echo "[ FAIL] $B_GOOS/$B_GOARCH"
        cat build_err.tmp
        FAIL=$((FAIL + 1))
        FAIL_LIST="$FAIL_LIST $B_GOOS/$B_GOARCH"
    fi

    rm -f build_err.tmp
    echo ""
}

echo "开始编译 Windows..."
echo "--------------------------------------------------"
build windows amd64 windows_amd64
build windows arm64 windows_arm64

echo "开始编译 Linux..."
echo "--------------------------------------------------"
build linux amd64 linux_amd64
build linux arm64 linux_arm64

echo "开始编译 macOS..."
echo "--------------------------------------------------"
build darwin amd64 darwin_amd64
build darwin arm64 darwin_arm64

echo "=================================================="
echo "   编译完成"
echo "   成功: $PASS  失败: $FAIL"
if [ -n "$FAIL_LIST" ]; then
    echo "   失败平台: $FAIL_LIST"
fi
echo "=================================================="

echo ""
echo "产物列表:"
ls -1 "$OUT_DIR"

read -n 1 -s -r -p "按任意键继续..."
echo