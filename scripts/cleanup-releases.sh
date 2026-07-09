#!/bin/bash
set -euo pipefail

# 清理 MedMemo 历史 GitHub Releases。
# 默认 dry-run，加 --execute 才真正执行删除。
#
# 保留：
#   1. 所有正式版 release（vX.Y.Z）
#   2. 每个 X.Y.Z 版本的最新一个预发布 release
#   3. 资源 release（embedding-model-v1 等）
# 删除：
#   - 重复 draft release（同一 tag 仅保留一个）
#   - 旧中间 build release
#   - 已被删除/重命名 tag 对应的 release
#
# 使用 GitHub API 按 release ID 删除，避免同一 tag 多个 draft 时 gh CLI 无法精确删除。

REPO="hzhan516/medmemo"
DRY_RUN=true

if [[ "${1:-}" == "--execute" ]]; then
  DRY_RUN=false
fi

run() {
  if $DRY_RUN; then
    echo "[DRY-RUN] $*"
  else
    echo "[EXECUTE] $*"
    "$@"
  fi
}

# 必须保留的 tag 列表
KEEP_TAGS=(
  "v1.0.1"
  "v1.1.1"
  "v1.1.2"
  "v1.1.3"
  "v1.1.4"
  "v1.1.5"
  "v1.1.6"
  "v1.1.7"
  "v1.1.8-Pre-release-build.80"
  "v1.1.9-Pre-release-build.83"
  "1.0.1-Pre-release-build.6"
  "1.0.2-Pre-release-build.7"
  "0.1.0-Pre-release-build.5"
  "embedding-model-v1"
  "onnx-runtime-v1.26.0"
  "tokenizers-v1.27.0"
)

is_keep_tag() {
  local tag="$1"
  for keep in "${KEEP_TAGS[@]}"; do
    if [[ "$tag" == "$keep" ]]; then
      return 0
    fi
  done
  return 1
}

main() {
  echo "Release cleanup script (DRY_RUN=$DRY_RUN)"
  echo "Run with --execute to actually perform changes."
  echo ""

  echo "==> 获取所有 GitHub releases..."
  local releases_json
  # 使用 --paginate 获取所有 pages，避免默认 30 条限制
  releases_json=$(gh api --paginate "repos/${REPO}/releases" --jq '[.[] | {id, tag_name, draft, prerelease, created_at}]')

  # 按 tag_name 分组，每组保留最新一个 release（按 id 降序，id 越大创建越晚）
  for tag in $(echo "$releases_json" | jq -r '.[].tag_name' | sort -u); do
    local count
    count=$(echo "$releases_json" | jq --arg t "$tag" '[.[] | select(.tag_name == $t)] | length')

    if is_keep_tag "$tag"; then
      if [[ "$count" -gt 1 ]]; then
        echo "==> $tag 有 $count 个重复 release，保留 id 最大的一个"
        # 取 id 最大的保留，删除其余
        local keep_id
        keep_id=$(echo "$releases_json" | jq --arg t "$tag" '[.[] | select(.tag_name == $t)] | max_by(.id) | .id')
        for rid in $(echo "$releases_json" | jq --arg t "$tag" --argjson keep "$keep_id" '[.[] | select(.tag_name == $t and .id != $keep)] | .[].id'); do
          run gh api --method DELETE "repos/${REPO}/releases/${rid}"
        done
      else
        echo "KEEP: $tag (id=$(echo "$releases_json" | jq --arg t "$tag" '[.[] | select(.tag_name == $t)] | .[0].id'))"
      fi
    else
      echo "==> DELETE all releases for tag: $tag ($count 个)"
      for rid in $(echo "$releases_json" | jq --arg t "$tag" '[.[] | select(.tag_name == $t)] | .[].id'); do
        run gh api --method DELETE "repos/${REPO}/releases/${rid}"
      done
    fi
  done

  if $DRY_RUN; then
    echo ""
    echo "Dry-run complete. No changes were made."
    echo "Run '$0 --execute' to apply changes."
  fi
}

main
