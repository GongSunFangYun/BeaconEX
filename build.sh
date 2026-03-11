#!/bin/bash
set -euo pipefail

APP_NAME="beaconex"
OUT_DIR="build"

log() {
    local LEVEL="$1"
    shift
    printf "[%-5s] %s\n" "$LEVEL" "$*"
}

VERSION_DISPLAY="3.0.1"
VERSION="${VERSION_DISPLAY//.}"

GO_VER=$(go version 2>&1)
log INFO "BeaconEX Build Script v${VERSION_DISPLAY}"
log INFO "Go toolchain: $GO_VER"

mkdir -p "$OUT_DIR"
log INFO "Output directory: $OUT_DIR"
log INFO "Build version: ${VERSION_DISPLAY} (filename tag: v${VERSION})"

PASS=0
FAIL=0
FAIL_LIST=""

build() {
    local B_GOOS="$1"
    local B_GOARCH="$2"
    local B_PLATFORM="$3"
    local B_ARCH="$4"

    local B_OUT="${OUT_DIR}/${APP_NAME}-${B_PLATFORM}-${B_ARCH}-v${VERSION}"
    if [ "$B_GOOS" = "windows" ]; then
        B_OUT="${B_OUT}.exe"
    fi

    log INFO "Compiling ${B_GOOS}/${B_GOARCH} -> ${B_OUT}"

    # Set env locally to avoid polluting subsequent builds
    local BUILD_ENV
    BUILD_ENV="GOOS=${B_GOOS} GOARCH=${B_GOARCH} CGO_ENABLED=0"
    if [ "$B_GOARCH" = "arm" ]; then
        BUILD_ENV="${BUILD_ENV} GOARM=7"
    fi

    if env $BUILD_ENV go build \
        -trimpath \
        -ldflags="-s -w -X main.CurrentVersion=${VERSION_DISPLAY}" \
        -o "$B_OUT" . \
        2>build_err.tmp
    then
        local SIZE
        SIZE=$(wc -c < "$B_OUT")
        log INFO  "OK: ${B_OUT} (${SIZE} bytes)"
        PASS=$((PASS + 1))
    else
        log ERROR "FAILED: ${B_GOOS}/${B_GOARCH}"
        while IFS= read -r line; do
            log ERROR "  ${line}"
        done < build_err.tmp
        FAIL=$((FAIL + 1))
        FAIL_LIST="${FAIL_LIST} ${B_GOOS}/${B_GOARCH}"
    fi

    rm -f build_err.tmp
    echo ""
}

log INFO "Building Windows targets..."
build windows amd64   windows x86_64
build windows arm64   windows arm64
build windows 386     windows x86

log INFO "Building Linux targets..."
build linux amd64     linux x86_64
build linux arm64     linux arm64
build linux arm       linux armv7
build linux 386       linux x86
build linux riscv64   linux riscv64

log INFO "Building macOS targets..."
build darwin amd64    darwin x86_64
build darwin arm64    darwin arm64

log INFO "Building FreeBSD targets..."
build freebsd amd64   freebsd x86_64
build freebsd arm64   freebsd arm64

log INFO "Build finished. Succeeded: ${PASS}  Failed: ${FAIL}"
if [ -n "$FAIL_LIST" ]; then
    log WARN "Failed targets:${FAIL_LIST}"
fi

log INFO "Artifacts in \"${OUT_DIR}\":"
for f in "${OUT_DIR}"/*; do
    [ -f "$f" ] || continue
    SIZE=$(wc -c < "$f")
    printf "        %s  (%s bytes)\n" "$(basename "$f")" "$SIZE"
done
echo ""
echo "Press any key to exit..."
read -n 1 -s -r
echo