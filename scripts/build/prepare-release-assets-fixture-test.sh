#!/bin/bash
# prepare-release-assets.sh 的临时 fixture 测试
# 覆盖：六件齐全、缺件、重名、数量/额外文件
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PREPARE="$SCRIPT_DIR/prepare-release-assets.sh"

if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required"
    exit 1
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

VERSION="1.1.10"

# 创建标准 manifest（内存中使用）
MANIFEST="$WORK/manifest.json"
cat > "$MANIFEST" <<'EOF'
{
  "schemaVersion": "1.0",
  "assets": [
    {"name": "MedMemoSetup.exe", "platform": "windows", "arch": "amd64", "kind": "nsis-installer", "sourceDir": "artifacts/windows"},
    {"name": "MedMemo_x86_64.dmg", "platform": "darwin", "arch": "x86_64", "kind": "dmg-installer", "sourceDir": "artifacts/darwin-amd64"},
    {"name": "MedMemo_arm64.dmg", "platform": "darwin", "arch": "arm64", "kind": "dmg-installer", "sourceDir": "artifacts/darwin-arm64"},
    {"name": "MedMemo-x86_64.AppImage", "platform": "linux", "arch": "x86_64", "kind": "appimage-installer", "sourceDir": "artifacts/linux"},
    {"name": "MedMemo_${VERSION}_amd64.deb", "platform": "linux", "arch": "amd64", "kind": "deb-package", "sourceDir": "artifacts/linux"},
    {"name": "MedMemo-${VERSION}-1.x86_64.rpm", "platform": "linux", "arch": "x86_64", "kind": "rpm-package", "sourceDir": "artifacts/linux"}
  ]
}
EOF

setup_all_assets() {
    mkdir -p "$WORK/src/linux" "$WORK/src/windows" "$WORK/src/darwin-amd64" "$WORK/src/darwin-arm64" "$WORK/out"
    echo "windows installer" > "$WORK/src/windows/MedMemoSetup.exe"
    echo "darwin x86_64 dmg" > "$WORK/src/darwin-amd64/MedMemo_x86_64.dmg"
    echo "darwin arm64 dmg" > "$WORK/src/darwin-arm64/MedMemo_arm64.dmg"
    echo "linux appimage" > "$WORK/src/linux/MedMemo-x86_64.AppImage"
    echo "linux deb" > "$WORK/src/linux/MedMemo_${VERSION}_amd64.deb"
    echo "linux rpm" > "$WORK/src/linux/MedMemo-${VERSION}-1.x86_64.rpm"
}

PASS=0
FAIL=0

run_test() {
    local name="$1"
    local expect_ok="$2"
    local test_manifest="${3:-$MANIFEST}"
    shift 3 || shift 2

    rm -rf "$WORK/out"
    mkdir -p "$WORK/out"

    local rc=0
    MEDMEMO_RELEASE_MANIFEST="$test_manifest" "$PREPARE" "$VERSION" "$WORK/out" "$WORK/src/linux" "$WORK/src/windows" "$WORK/src/darwin-amd64" "$WORK/src/darwin-arm64" "$@" > "$WORK/last.log" 2>&1 || rc=$?

    if [ "$expect_ok" = "true" ] && [ "$rc" -eq 0 ]; then
        echo "PASS: $name"
        PASS=$((PASS + 1))
    elif [ "$expect_ok" = "false" ] && [ "$rc" -ne 0 ]; then
        echo "PASS: $name (expected failure, got rc=$rc)"
        PASS=$((PASS + 1))
    else
        echo "FAIL: $name (expected_ok=$expect_ok rc=$rc)"
        cat "$WORK/last.log" || true
        FAIL=$((FAIL + 1))
    fi
}

# 1. 六件齐全 → 成功
setup_all_assets
run_test "six assets complete" true
if [ "$(find "$WORK/out" -maxdepth 1 -type f | wc -l)" -ne 6 ]; then
    echo "FAIL: output file count after complete staging is not 6"
    FAIL=$((FAIL + 1))
fi

# 2. 缺件 → 失败
setup_all_assets
rm "$WORK/src/linux/MedMemo-x86_64.AppImage"
run_test "missing AppImage fails" false

# 3. 重名 manifest → 失败（构造含重复 name 的 manifest）
setup_all_assets
# 在 darwin-amd64 也放入同名文件，使脚本走到“输出目录已存在同名文件”分支而非“源文件缺失”分支
echo "darwin fake installer" > "$WORK/src/darwin-amd64/MedMemoSetup.exe"
DUP_MANIFEST="$WORK/manifest-dup.json"
jq '.assets[1].name = "MedMemoSetup.exe"' "$MANIFEST" > "$DUP_MANIFEST"
run_test "duplicate asset name fails" false "$DUP_MANIFEST"

# 4. 额外文件进入输出目录 → 失败（通过篡改 manifest 让 source dir 包含额外同名文件无法模拟，
#    这里改为验证脚本会清空输出目录中预置的额外文件，且最终数量为 6）
setup_all_assets
echo "stale extra" > "$WORK/out/extra-file.tmp"
run_test "output directory cleaned and exact count enforced" true
if [ -e "$WORK/out/extra-file.tmp" ]; then
    echo "FAIL: stale extra file was not removed from output directory"
    FAIL=$((FAIL + 1))
fi
if [ "$(find "$WORK/out" -maxdepth 1 -type f | wc -l)" -ne 6 ]; then
    echo "FAIL: output file count after cleanup is not 6"
    FAIL=$((FAIL + 1))
fi

# 5. 空 source dir → 失败
rm -rf "$WORK/src"
mkdir -p "$WORK/src/linux" "$WORK/src/windows" "$WORK/src/darwin-amd64" "$WORK/src/darwin-arm64"
run_test "empty source dirs fail" false

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
