#!/bin/bash
set -euo pipefail

TARGET_DIR="${1:?target directory required}"
PLATFORM="${2:?platform required}"
REQUIRE_UNIVERSAL="${3:-false}"
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

if [ "$PLATFORM" = "darwin" ]; then
  darwin_lib="$RESOURCE_ROOT/lib/darwin/libonnxruntime.dylib"
  if [ ! -f "$darwin_lib" ]; then
    echo "Error: missing macOS ONNX Runtime dylib in app bundle: $darwin_lib" >&2
    exit 1
  fi

  if [ "$REQUIRE_UNIVERSAL" = "true" ] && ! command -v lipo >/dev/null 2>&1; then
    echo "Error: lipo is required to validate universal macOS runtime library" >&2
    exit 1
  fi

  if command -v lipo >/dev/null 2>&1; then
    lipo_info="$(lipo -info "$darwin_lib")"
    if [ "$REQUIRE_UNIVERSAL" = "true" ] && [[ "$lipo_info" != *"arm64"* || "$lipo_info" != *"x86_64"* ]]; then
      echo "Error: macOS ONNX Runtime dylib must be universal arm64/x86_64: $lipo_info" >&2
      exit 1
    fi
  fi
fi
