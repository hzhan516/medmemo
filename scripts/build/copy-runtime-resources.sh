#!/bin/bash
set -euo pipefail

TARGET_DIR="${1:?target directory required}"
PLATFORM="${2:?platform required}"
RESOURCE_ROOT="${TARGET_DIR}/resources"

rm -rf "$RESOURCE_ROOT"
mkdir -p "$RESOURCE_ROOT"

if [ -d resources/rules ]; then
  cp -R resources/rules "$RESOURCE_ROOT/"
fi

if [ -d resources/models ]; then
  cp -R resources/models "$RESOURCE_ROOT/"
fi

if [ -d "resources/lib/${PLATFORM}" ]; then
  mkdir -p "$RESOURCE_ROOT/lib/${PLATFORM}"
  libs=()
  case "$PLATFORM" in
    windows)
      libs=(resources/lib/windows/*.dll)
      ;;
    darwin)
      libs=(resources/lib/darwin/*.dylib)
      ;;
    linux)
      libs=(resources/lib/linux/*.so*)
      ;;
  esac
  if [ -e "${libs[0]}" ]; then
    cp "${libs[@]}" "$RESOURCE_ROOT/lib/${PLATFORM}/"
  fi
fi
