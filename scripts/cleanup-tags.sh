#!/bin/bash
set -euo pipefail

# 清理 MedMemo 历史 tags（批量删除版，减少网络往返）。
# 默认 dry-run，加 --execute 才真正执行删除/推送。
# 设计原则：
#   1. 保留每个 X.Y.Z 版本的最新一个预发布 tag（含 -Pre-release-build）
#   2. 保留所有正式版 tag（vX.Y.Z）
#   3. 将不规范正式版 tag（如 1.1.7.77）重命名为 vX.Y.Z
#   4. 删除旧 -build.N tag（不含 Pre-release）
#   5. 保留资源标签（embedding-model-v1 等）

REMOTE="origin"
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

# 批量删除本地 + 远程 tags
batch_delete_tags() {
  local tags=("$@")
  if [[ ${#tags[@]} -eq 0 ]]; then
    return 0
  fi

  # 本地删除
  if ! $DRY_RUN; then
    git tag -d "${tags[@]}" 2>/dev/null || true
  else
    for t in "${tags[@]}"; do
      echo "[DRY-RUN] git tag -d $t"
    done
  fi

  # 远程删除：使用 GitHub API 逐个删除（gh api 在部分规则下会返回 422）
  local token
  token=$(gh auth token)
  for t in "${tags[@]}"; do
    run curl -s -X DELETE \
      -H "Authorization: token ${token}" \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/repos/${REPO}/git/refs/tags/${t}" \
      -w "\nHTTP %{http_code}\n"
  done
}

# 批量创建并推送 tags
batch_create_tags() {
  local tags=("$@")
  if [[ ${#tags[@]} -eq 0 ]]; then
    return 0
  fi

  if ! $DRY_RUN; then
    git tag "${tags[@]}"
    git push "$REMOTE" "${tags[@]}"
  else
    for t in "${tags[@]}"; do
      echo "[DRY-RUN] git tag $t"
      echo "[DRY-RUN] git push $REMOTE $t"
    done
  fi
}

# 1. 删除同一版本的旧预发布 build tag，只保留最新的一个。
delete_older_prereleases() {
  echo "==> 清理旧预发布 build tag..."
  local tags
  tags=$(git tag -l '*-Pre-release-build.*' | sort -V)

  local to_delete=()
  local current_prefix="" keep=""
  for tag in $tags; do
    local prefix
    prefix=$(echo "$tag" | sed -E 's/(-Pre-release-build\..*)//')
    if [[ "$prefix" != "$current_prefix" ]]; then
      if [[ -n "$current_prefix" && -n "$keep" ]]; then
        for old in $(echo "$tags" | grep "^${current_prefix}-Pre-release-build\." | sort -V); do
          if [[ "$old" != "$keep" ]]; then
            to_delete+=("$old")
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
        to_delete+=("$old")
      fi
    done
  fi

  if [[ ${#to_delete[@]} -gt 0 ]]; then
    batch_delete_tags "${to_delete[@]}"
  fi
}

# 2. 将不规范正式版 tag 重命名为 vX.Y.Z。
rename_to_semver() {
  echo "==> 重命名不规范正式版 tag 为 vX.Y.Z..."
  declare -A rename_map=(
    ["1.1.7.77"]="v1.1.7"
    ["1.1.6.73"]="v1.1.6"
    ["1.1.5.69"]="v1.1.5"
    ["1.1.4.65"]="v1.1.4"
    ["1.1.3.57"]="v1.1.3"
    ["1.1.2.54"]="v1.1.2"
  )

  local to_create=()
  local to_delete=()
  for old in "${!rename_map[@]}"; do
    local new="${rename_map[$old]}"
    if git tag -l "$old" | grep -q "$old"; then
      if ! git tag -l "$new" | grep -q "$new"; then
        local commit
        commit=$(git rev-list -n1 "$old")
        to_create+=("$new" "$commit")
      else
        echo "Tag $new already exists, skipping creation."
      fi
      to_delete+=("$old")
    fi
  done

  # 批量创建新 tag（tag 名和 commit 成对出现）
  if [[ ${#to_create[@]} -gt 0 ]]; then
    if ! $DRY_RUN; then
      local i=0
      while [[ $i -lt ${#to_create[@]} ]]; do
        git tag "${to_create[$i]}" "${to_create[$((i+1))]}"
        ((i+=2))
      done
      # 批量推送所有新建 tag
      local create_names=()
      i=0
      while [[ $i -lt ${#to_create[@]} ]]; do
        create_names+=("${to_create[$i]}")
        ((i+=2))
      done
      git push "$REMOTE" "${create_names[@]}"
    else
      local i=0
      while [[ $i -lt ${#to_create[@]} ]]; do
        echo "[DRY-RUN] git tag ${to_create[$i]} ${to_create[$((i+1))]}"
        echo "[DRY-RUN] git push $REMOTE ${to_create[$i]}"
        ((i+=2))
      done
    fi
  fi

  # 批量删除旧 tag
  if [[ ${#to_delete[@]} -gt 0 ]]; then
    batch_delete_tags "${to_delete[@]}"
  fi
}

# 3. 删除旧的 -build.N tag（不含 Pre-release）。
delete_old_builds() {
  echo "==> 清理旧 -build.N tag..."
  local to_delete=()
  for tag in $(git tag -l '*-build.*' | grep -v 'Pre-release-build'); do
    to_delete+=("$tag")
  done

  if [[ ${#to_delete[@]} -gt 0 ]]; then
    batch_delete_tags "${to_delete[@]}"
  fi
}

print_kept_tags() {
  echo ""
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

  print_kept_tags

  if $DRY_RUN; then
    echo ""
    echo "Dry-run complete. No changes were made."
    echo "Run '$0 --execute' to apply changes."
  fi
}

main
