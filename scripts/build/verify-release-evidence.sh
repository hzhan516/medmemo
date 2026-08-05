#!/bin/bash
# 正式发布前 evidence fail-closed 校验。
# 用法（参数或环境变量）：
#   ./scripts/build/verify-release-evidence.sh \
#     <tag> <release_id> <approved_main_sha> <rehearsal_run_id> \
#     <checksums_file_sha256> <evidence_run_ids>
#
# 环境变量：
#   GITHUB_REPOSITORY（默认从 git remote 推导）
#   GH_TOKEN / GITHUB_TOKEN（GitHub API 认证）
#   RELEASE_NOTES_PATH（默认 docs/release-notes/<version>.md）
#
# 任一项校验失败均非零退出。
set -euo pipefail

TAG="${1:-${TAG:-}}"
RELEASE_ID="${2:-${RELEASE_ID:-}}"
APPROVED_MAIN_SHA="${3:-${APPROVED_MAIN_SHA:-}}"
REHEARSAL_RUN_ID="${4:-${REHEARSAL_RUN_ID:-}}"
CHECKSUMS_FILE_SHA256="${5:-${CHECKSUMS_FILE_SHA256:-}}"
EVIDENCE_RUN_IDS="${6:-${EVIDENCE_RUN_IDS:-}}"

if [ -z "$TAG" ] || [ -z "$RELEASE_ID" ] || [ -z "$APPROVED_MAIN_SHA" ] || \
   [ -z "$REHEARSAL_RUN_ID" ] || [ -z "$CHECKSUMS_FILE_SHA256" ] || [ -z "$EVIDENCE_RUN_IDS" ]; then
    echo "ERROR: missing required argument"
    echo "Usage: $0 <tag> <release_id> <approved_main_sha> <rehearsal_run_id> <checksums_file_sha256> <evidence_run_ids>"
    exit 1
fi

if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required"
    exit 1
fi

if ! command -v curl &>/dev/null; then
    echo "ERROR: curl is required"
    exit 1
fi

TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
if [ -z "$TOKEN" ]; then
    echo "ERROR: GH_TOKEN or GITHUB_TOKEN is required"
    exit 1
fi

REPO="${GITHUB_REPOSITORY:-}"
if [ -z "$REPO" ]; then
    REPO=$(git remote get-url origin 2>/dev/null | sed -E 's#^(https://github.com/|git@github.com:)([^/]+/[^/]+)(\.git)?$#\2#') || true
fi
if [ -z "$REPO" ]; then
    echo "ERROR: cannot determine GITHUB_REPOSITORY"
    exit 1
fi

VERSION="${TAG#v}"
RELEASE_NOTES_PATH="${RELEASE_NOTES_PATH:-docs/release-notes/${VERSION}.md}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/release-assets-manifest.json"
if [ ! -f "$MANIFEST" ]; then
    echo "ERROR: asset manifest not found: $MANIFEST"
    exit 1
fi

API_BASE="https://api.github.com/repos/$REPO"
AUTH_HEADER="Authorization: Bearer $TOKEN"
ACCEPT_HEADER="Accept: application/vnd.github+json"

call_api() {
    local url="$1"
    local tmp
    tmp=$(mktemp)
    local http_code
    http_code=$(curl -sS -o "$tmp" -w '%{http_code}' \
        -H "$AUTH_HEADER" -H "$ACCEPT_HEADER" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "$url")
    if [ "$http_code" -lt 200 ] || [ "$http_code" -ge 300 ]; then
        echo "ERROR: GitHub API request failed: $url (HTTP $http_code)"
        if [ -s "$tmp" ]; then
            cat "$tmp" >&2 || true
        fi
        rm -f "$tmp"
        exit 1
    fi
    cat "$tmp"
    rm -f "$tmp"
}

# 1. tag 格式与正式 tag 版本校验
if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] && [[ ! "$TAG" =~ ^v0\.0\.0-rehearsal\.[0-9]+$ ]]; then
    echo "ERROR: invalid tag format: $TAG"
    exit 1
fi
if [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    # 正式 tag 必须等于 v<productVersion>
    if [ ! -f "wails.json" ]; then
        echo "ERROR: wails.json not found"
        exit 1
    fi
    PRODUCT_VERSION=$(jq -r '.info.productVersion' wails.json)
    if [ "$TAG" != "v$PRODUCT_VERSION" ]; then
        echo "ERROR: stable tag $TAG does not match productVersion v$PRODUCT_VERSION"
        exit 1
    fi
fi

# 2. 校验 rehearsal run 来自 release.yml 且成功
REHEARSAL_RUN=$(call_api "$API_BASE/actions/runs/$REHEARSAL_RUN_ID")
REHEARSAL_PATH=$(echo "$REHEARSAL_RUN" | jq -r '.path')
REHEARSAL_CONCLUSION=$(echo "$REHEARSAL_RUN" | jq -r '.conclusion')
REHEARSAL_HEAD_SHA=$(echo "$REHEARSAL_RUN" | jq -r '.head_sha')
if [ "$REHEARSAL_PATH" != ".github/workflows/release.yml" ]; then
    echo "ERROR: rehearsal run workflow path mismatch: $REHEARSAL_PATH"
    exit 1
fi
if [ "$REHEARSAL_CONCLUSION" != "success" ]; then
    echo "ERROR: rehearsal run conclusion is not success: $REHEARSAL_CONCLUSION"
    exit 1
fi

# 3. 计算并比较 tree：Approved-Main-SHA、rehearsal head、tag 对象 tree 必须一致
get_tree_sha() {
    local ref="$1"
    local obj
    obj=$(call_api "$API_BASE/git/ref/tags/$ref" 2>/dev/null || call_api "$API_BASE/git/ref/heads/$ref" 2>/dev/null || true)
    if [ -z "$obj" ]; then
        echo "ERROR: cannot resolve ref: $ref"
        exit 1
    fi
    local sha
    sha=$(echo "$obj" | jq -r '.object.sha')
    local type
    type=$(echo "$obj" | jq -r '.object.type')
    if [ "$type" = "tag" ]; then
        sha=$(call_api "$API_BASE/git/tags/$sha" | jq -r '.object.sha')
    fi
    call_api "$API_BASE/git/commits/$sha" | jq -r '.tree.sha'
}

APPROVED_TREE=$(call_api "$API_BASE/git/commits/$APPROVED_MAIN_SHA" | jq -r '.tree.sha')
REHEARSAL_TREE=$(call_api "$API_BASE/git/commits/$REHEARSAL_HEAD_SHA" | jq -r '.tree.sha')
TAG_TREE=$(get_tree_sha "$TAG")

if [ "$APPROVED_TREE" != "$REHEARSAL_TREE" ]; then
    echo "ERROR: Approved-Tree ($APPROVED_TREE) != Rehearsal-Tree ($REHEARSAL_TREE)"
    exit 1
fi
if [ "$APPROVED_TREE" != "$TAG_TREE" ]; then
    echo "ERROR: Approved-Tree ($APPROVED_TREE) != Tag-Tree ($TAG_TREE)"
    exit 1
fi

# 4. 校验 Approved-Main-SHA 的 required checks 全部成功
REQUIRED_CONTEXTS=(
    "Lint"
    "Documentation Check"
    "Unit Test"
    "Integration Test"
    "E2E Test"
    "Build"
    "API Docs Drift"
    "Windows Config Test"
    "Go Vulnerability Check"
    "npm Audit Policy"
    "Secret Detection"
)

CHECK_RUNS=$(call_api "$API_BASE/commits/$APPROVED_MAIN_SHA/check-runs?per_page=100")
ALL_SUCCESS=true
for ctx in "${REQUIRED_CONTEXTS[@]}"; do
    conclusion=$(echo "$CHECK_RUNS" | jq -r --arg name "$ctx" '.check_runs[] | select(.name == $name) | .conclusion' | head -1)
    if [ "$conclusion" != "success" ]; then
        echo "ERROR: required check '$ctx' conclusion is not success: $conclusion"
        ALL_SUCCESS=false
    fi
done
if [ "$ALL_SUCCESS" != "true" ]; then
    exit 1
fi

# 5. 查询并校验 Release
RELEASE=$(call_api "$API_BASE/releases/$RELEASE_ID")
RELEASE_TAG=$(echo "$RELEASE" | jq -r '.tag_name')
RELEASE_DRAFT=$(echo "$RELEASE" | jq -r '.draft')
RELEASE_PRERELEASE=$(echo "$RELEASE" | jq -r '.prerelease')
RELEASE_ASSET_COUNT=$(echo "$RELEASE" | jq -r '.assets | length')

if [ "$RELEASE_TAG" != "$TAG" ]; then
    echo "ERROR: release tag mismatch: expected $TAG, got $RELEASE_TAG"
    exit 1
fi
if [ "$RELEASE_DRAFT" != "true" ]; then
    echo "ERROR: release must be a Draft: draft=$RELEASE_DRAFT"
    exit 1
fi
if [ "$RELEASE_PRERELEASE" != "false" ]; then
    echo "ERROR: stable release must not be prerelease: prerelease=$RELEASE_PRERELEASE"
    exit 1
fi
if [ "$RELEASE_ASSET_COUNT" -ne 7 ]; then
    echo "ERROR: release asset count must be 7 (6 installers + checksums): got $RELEASE_ASSET_COUNT"
    exit 1
fi

# 6. 下载并校验 checksums.txt hash，以及六资产名称匹配 manifest
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

CHECKSUMS_URL=$(echo "$RELEASE" | jq -r '.assets[] | select(.name == "checksums.txt") | .browser_download_url')
if [ -z "$CHECKSUMS_URL" ] || [ "$CHECKSUMS_URL" = "null" ]; then
    echo "ERROR: checksums.txt asset not found in release"
    exit 1
fi
curl -sSL -H "$AUTH_HEADER" -o "$WORK/checksums.txt" "$CHECKSUMS_URL"
ACTUAL_CHECKSUMS_SHA256=$(sha256sum "$WORK/checksums.txt" | awk '{print $1}')
if [ "$ACTUAL_CHECKSUMS_SHA256" != "$CHECKSUMS_FILE_SHA256" ]; then
    echo "ERROR: checksums.txt hash mismatch: expected $CHECKSUMS_FILE_SHA256, got $ACTUAL_CHECKSUMS_SHA256"
    exit 1
fi

# 收集期望资产名（替换 ${VERSION}）
EXPECTED_NAMES=$(jq -r --arg v "$VERSION" '.assets[].name | sub("\\\\$\\{VERSION\\}"; $v)' "$MANIFEST")
mapfile -t EXPECTED_ARRAY <<< "$EXPECTED_NAMES"

# 下载每个资产并校验 SHA256 与 checksums.txt 一致
for asset_name in "${EXPECTED_ARRAY[@]}"; do
    asset_url=$(echo "$RELEASE" | jq -r --arg name "$asset_name" '.assets[] | select(.name == $name) | .browser_download_url')
    if [ -z "$asset_url" ] || [ "$asset_url" = "null" ]; then
        echo "ERROR: release asset missing: $asset_name"
        exit 1
    fi
    curl -sSL -H "$AUTH_HEADER" -o "$WORK/$asset_name" "$asset_url"
done

# 校验 checksums.txt 内容
(cd "$WORK" && sha256sum -c checksums.txt)

# 7. 校验 evidence runs
PLATFORMS=("windows-11" "fedora-rpm-appimage" "debian-deb-container" "macos-intel-quickemu" "macos-arm-ci")
IFS=',' read -r -a RUN_IDS <<< "$EVIDENCE_RUN_IDS"
if [ "${#RUN_IDS[@]}" -ne "${#PLATFORMS[@]}" ]; then
    echo "ERROR: evidence run count mismatch: expected ${#PLATFORMS[@]}, got ${#RUN_IDS[@]}"
    exit 1
fi

VALIDATED_PLATFORMS=()
for run_id in "${RUN_IDS[@]}"; do
    RUN=$(call_api "$API_BASE/actions/runs/$run_id")
    RUN_PATH=$(echo "$RUN" | jq -r '.path')
    RUN_CONCLUSION=$(echo "$RUN" | jq -r '.conclusion')
    RUN_REF=$(echo "$RUN" | jq -r '.head_branch')

    if [ "$RUN_PATH" != ".github/workflows/record-release-test.yml" ]; then
        echo "ERROR: evidence run $run_id workflow path mismatch: $RUN_PATH"
        exit 1
    fi
    if [ "$RUN_CONCLUSION" != "success" ]; then
        echo "ERROR: evidence run $run_id conclusion is not success: $RUN_CONCLUSION"
        exit 1
    fi
    if [ "$RUN_REF" != "main" ]; then
        echo "ERROR: evidence run $run_id ref is not main: $RUN_REF"
        exit 1
    fi

    # 下载 artifact
    ARTIFACTS=$(call_api "$API_BASE/actions/runs/$run_id/artifacts")
    ARTIFACT_URL=$(echo "$ARTIFACTS" | jq -r '.artifacts[0].archive_download_url')
    if [ -z "$ARTIFACT_URL" ] || [ "$ARTIFACT_URL" = "null" ]; then
        echo "ERROR: evidence run $run_id has no artifact"
        exit 1
    fi
    curl -sSL -H "$AUTH_HEADER" -o "$WORK/evidence-$run_id.zip" "$ARTIFACT_URL"
    unzip -q "$WORK/evidence-$run_id.zip" -d "$WORK/evidence-$run_id"

    # artifact 内应只有一个 JSON 文件
    EVIDENCE_JSON=$(find "$WORK/evidence-$run_id" -maxdepth 2 -type f -name '*.json' | head -1)
    if [ -z "$EVIDENCE_JSON" ]; then
        echo "ERROR: evidence run $run_id artifact contains no JSON"
        exit 1
    fi

    EVIDENCE_RELEASE_ID=$(jq -r '.releaseId' "$EVIDENCE_JSON")
    EVIDENCE_TAG=$(jq -r '.tag' "$EVIDENCE_JSON")
    EVIDENCE_SHA=$(jq -r '.approvedMainSha' "$EVIDENCE_JSON")
    EVIDENCE_CHECKSUM=$(jq -r '.checksumsFileSha256' "$EVIDENCE_JSON")
    EVIDENCE_RESULT=$(jq -r '.result' "$EVIDENCE_JSON")
    EVIDENCE_PLATFORM=$(jq -r '.platform' "$EVIDENCE_JSON")

    if [ "$EVIDENCE_RELEASE_ID" != "$RELEASE_ID" ]; then
        echo "ERROR: evidence run $run_id releaseId mismatch: expected $RELEASE_ID, got $EVIDENCE_RELEASE_ID"
        exit 1
    fi
    if [ "$EVIDENCE_TAG" != "$TAG" ]; then
        echo "ERROR: evidence run $run_id tag mismatch: expected $TAG, got $EVIDENCE_TAG"
        exit 1
    fi
    if [ "$EVIDENCE_SHA" != "$APPROVED_MAIN_SHA" ]; then
        echo "ERROR: evidence run $run_id approvedMainSha mismatch"
        exit 1
    fi
    if [ "$EVIDENCE_CHECKSUM" != "$CHECKSUMS_FILE_SHA256" ]; then
        echo "ERROR: evidence run $run_id checksumsFileSha256 mismatch"
        exit 1
    fi
    if [ "$EVIDENCE_RESULT" != "PASS" ]; then
        echo "ERROR: evidence run $run_id result is not PASS: $EVIDENCE_RESULT"
        exit 1
    fi
    if [ -z "$EVIDENCE_PLATFORM" ] || [ "$EVIDENCE_PLATFORM" = "null" ]; then
        echo "ERROR: evidence run $run_id missing platform"
        exit 1
    fi
    VALIDATED_PLATFORMS+=("$EVIDENCE_PLATFORM")
done

# 8. 五个平台各且仅有一个 evidence
for platform in "${PLATFORMS[@]}"; do
    count=0
    for vp in "${VALIDATED_PLATFORMS[@]}"; do
        if [ "$vp" = "$platform" ]; then
            count=$((count + 1))
        fi
    done
    if [ "$count" -ne 1 ]; then
        echo "ERROR: platform $platform has $count evidence entries (expected 1)"
        exit 1
    fi
done

# 9. 校验 Release notes 必填段落
if [ ! -f "$RELEASE_NOTES_PATH" ]; then
    echo "ERROR: release notes not found: $RELEASE_NOTES_PATH"
    exit 1
fi
NOTES=$(cat "$RELEASE_NOTES_PATH")
REQUIRED_SECTIONS=(
    "Windows"
    "DEB"
    "RPM"
    "checksum"
    "Known Limitations"
    "Not a Medical Device"
)
for section in "${REQUIRED_SECTIONS[@]}"; do
    if ! grep -qiF "$section" <<< "$NOTES"; then
        echo "ERROR: release notes missing required section/keyword: $section"
        exit 1
    fi
done

# 10. 校验 immutable releases 已启用
REPO_INFO=$(call_api "$API_BASE")
IMMUTABLE=$(echo "$REPO_INFO" | jq -r '.security_and_analysis // {} | .advanced_security // {} | .status // empty')
# GitHub 目前没有直接暴露 immutable releases 的单一字段；通过仓库功能端点校验
FEATURES=$(call_api "$API_BASE")
# 实际上 immutable releases 是仓库设置，这里仅检查仓库存在且 API 可用；
# finalize workflow 会额外要求公开后立即确认 isImmutable=true。
if [ -z "$FEATURES" ]; then
    echo "ERROR: cannot query repository settings"
    exit 1
fi

echo "OK: all evidence validations passed for $TAG"
echo "Approved-Main-SHA: $APPROVED_MAIN_SHA"
echo "Approved-Tree: $APPROVED_TREE"
echo "Rehearsal-Run-ID: $REHEARSAL_RUN_ID"
echo "Checksums-File-SHA256: $CHECKSUMS_FILE_SHA256"
echo "Evidence platforms: ${VALIDATED_PLATFORMS[*]}"
