#!/bin/bash
set -euo pipefail

# 清理 MedMemo 历史 tags。
# 默认 dry-run，加 --execute 才真正执行删除/推送。
# 设计原则：
#   1. 保留每个 X.Y.Z 版本的最新一个预发布 tag（含 -rc 或 -Pre-release-build）
#   2. 保留所有正式版 tag（vX.Y.Z）
#   3. 将不规范正式版 tag（如 1.1.7.77）重命名为 vX.Y.Z
#   4. 删除旧 -build.N tag（不含 Pre-release）
#   5. 保留资源标签（embedding-model-v1 等）

REMOTE="origin"
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

# 1. 删除同一版本的旧预发布 build tag，只保留最新的一个。
# 同时兼容带 v 前缀和不带 v 前缀的历史 tag。
delete_older_prereleases() {
  echo "==> 清理旧预发布 build tag..."
  # 匹配历史两种格式：v1.1.9-Pre-release-build.83 或 1.1.7-Pre-release-build.76
  local tags
  tags=$(git tag -l '*-Pre-release-build.*' | sort -V)

  # 按版本前缀分组，保留每组最新一个
  local current_prefix="" keep=""
  for tag in $tags; do
    # 提取版本前缀：v1.1.9 或 1.1.7
    local prefix
    prefix=$(echo "$tag" | sed -E 's/(-Pre-release-build\..*)//')
    if [[ "$prefix" != "$current_prefix" ]]; then
      # 新组开始，先处理上一组
      if [[ -n "$current_prefix" && -n "$keep" ]]; then
        for old in $(echo "$tags" | grep "^${current_prefix}-Pre-release-build\." | sort -V); do
          if [[ "$old" != "$keep" ]]; then
            run git tag -d "$old" 2>/dev/null || true
            run git push "$REMOTE" :refs/tags/"$old" 2>/dev/null || true
          fi
        done
      fi
      current_prefix="$prefix"
      keep="$tag"
    else
      keep="$tag"
    fi
  done
  # 处理最后一组
  if [[ -n "$current_prefix" && -n "$keep" ]]; then
    for old in $(echo "$tags" | grep "^${current_prefix}-Pre-release-build\." | sort -V); do
      if [[ "$old" != "$keep" ]]; then
        run git tag -d "$old" 2>/dev/null || true
        run git push "$REMOTE" :refs/tags/"$old" 2>/dev/null || true
      fi
    done
  fi
}

# 2. 将不规范正式版 tag 重命名为 vX.Y.Z。
rename_to_semver() {
  echo "==> 重命名不规范正式版 tag 为 vX.Y.Z..."
  # 旧 tag 到 commit 的映射
  declare -A rename_map=(
    ["1.1.7.77"]="v1.1.7"
    ["1.1.6.73"]="v1.1.6"
    ["1.1.5.69"]="v1.1.5"
    ["1.1.4.65"]="v1.1.4"
    ["1.1.3.57"]="v1.1.3"
    ["1.1.2.54"]="v1.1.2"
  )

  for old in "${!rename_map[@]}"; do
    local new="${rename_map[$old]}"
    if git tag -l "$old" | grep -q "$old"; then
      local commit
      commit=$(git rev-list -n1 "$old")
      if ! git tag -l "$new" | grep -q "$new"; then
        run git tag "$new" "$commit"
        run git push "$REMOTE" "$new"
      else
        echo "Tag $new already exists, skipping creation."
      fi
      run git tag -d "$old" 2>/dev/null || true
      run git push "$REMOTE" :refs/tags/"$old" 2>/dev/null || true
    fi
  done
}

# 3. 删除旧的 -build.N tag（不含 Pre-release）。
delete_old_builds() {
  echo "==> 清理旧 -build.N tag..."
  for tag in $(git tag -l '*-build.*' | grep -v 'Pre-release-build'); do
    run git tag -d "$tag" 2>/dev/null || true
    run git push "$REMOTE" :refs/tags/"$tag" 2>/dev/null || true
  done
}

# 4. 打印保留的 tag 列表用于核对。
print_kept_tags() {
  echo "==> 保留的 tags："
  git tag --sort=-v:refname
}

main() {
  echo "Tag cleanup script (DRY_RUN=$DRY_RUN)"
  echo "Run with --execute to actually perform changes."
  echo ""

  delete_older_prereleases
  rename_to_semver
  delete_old_builds

  echo ""
  print_kept_tags

  if $DRY_RUN; then
    echo ""
    echo "Dry-run complete. No changes were made."
    echo "Run '$0 --execute' to apply changes."
  fi
}

main
