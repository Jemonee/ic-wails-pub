#!/usr/bin/env bash

set -uo pipefail

# 修改这里的变量即可调整应用名称、版本号和输出目录。
APP_NAME="ic-wails"
VERSION="0.1.0"
OUTPUT_DIR="./bin"

# 如果需要为 CGO 目标指定交叉编译器，可以在这里配置。
WINDOWS_AMD64_CC=""
LINUX_AMD64_CC=""
LINUX_ARM64_CC=""
DARWIN_AMD64_CC=""
DARWIN_ARM64_CC=""

# amd64 会在输出文件名中映射为 x86，便于和 arm64 区分。
# Linux 和 Darwin 的 Wails 构建依赖 CGO，跨平台时通常需要交叉编译器。
TARGETS=(
  "windows|amd64|.exe|WINDOWS_AMD64_CC|0"
  "linux|amd64||LINUX_AMD64_CC|1"
  "linux|arm64||LINUX_ARM64_CC|1"
  "darwin|amd64||DARWIN_AMD64_CC|1"
  "darwin|arm64||DARWIN_ARM64_CC|1"
)

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
HOST_OS="$(go env GOHOSTOS)"

SUCCESS_OUTPUTS=()
SKIPPED_TARGETS=()
FAILED_TARGETS=()

if [[ "$OUTPUT_DIR" = /* ]]; then
  OUTPUT_PATH="$OUTPUT_DIR"
else
  OUTPUT_PATH="$SCRIPT_DIR/${OUTPUT_DIR#./}"
fi

arch_name() {
  case "$1" in
    amd64) echo "x86" ;;
    arm64) echo "arm64" ;;
    *) echo "$1" ;;
  esac
}

prepare_common_assets() {
  echo "==> preparing frontend assets"
  wails3 task -f common:build:frontend PRODUCTION=true
  wails3 task common:generate:bindings
  wails3 task common:generate:icons
}

build_target() {
  local goos="$1"
  local goarch="$2"
  local extension="${3:-}"
  local cc_var_name="$4"
  local requires_cgo="$5"
  local arch_label
  local cc_value
  local target_name
  local output_base
  local output_file

  arch_label="$(arch_name "$goarch")"
  cc_value="${!cc_var_name:-}"
  target_name="${goos}/${goarch}"
  output_base="${APP_NAME}-${VERSION}-${goos}-${arch_label}"
  output_file="$OUTPUT_PATH/${output_base}${extension}"

  if [[ "$requires_cgo" == "1" && "$goos" != "$HOST_OS" && -z "$cc_value" ]]; then
    echo "==> skipping ${target_name}"
    echo "    Wails ${goos} build requires CGO cross toolchain; set ${cc_var_name} in build.sh and retry"
    SKIPPED_TARGETS+=("$target_name")
    return 0
  fi

  echo "==> building ${target_name}"

  if [[ -n "$cc_value" ]]; then
    if env CC="$cc_value" \
      wails3 task "${goos}:build" PRODUCTION=true ARCH="$goarch" APP_NAME="$output_base" BIN_DIR="$OUTPUT_PATH"; then
      SUCCESS_OUTPUTS+=("$output_file")
      return 0
    fi
  else
    if wails3 task "${goos}:build" PRODUCTION=true ARCH="$goarch" APP_NAME="$output_base" BIN_DIR="$OUTPUT_PATH"; then
      SUCCESS_OUTPUTS+=("$output_file")
      return 0
    fi
  fi

  FAILED_TARGETS+=("$target_name")
  return 1
}

print_summary() {
  if [[ ${#SUCCESS_OUTPUTS[@]} -gt 0 ]]; then
    echo "==> outputs:"
    for output_file in "${SUCCESS_OUTPUTS[@]}"; do
      echo "    $output_file"
    done
  fi

  if [[ ${#SKIPPED_TARGETS[@]} -gt 0 ]]; then
    echo "==> skipped: ${SKIPPED_TARGETS[*]}"
  fi

  if [[ ${#FAILED_TARGETS[@]} -gt 0 ]]; then
    echo "==> failed: ${FAILED_TARGETS[*]}"
    return 1
  fi

  return 0
}

main() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go command not found"
    exit 1
  fi

  if ! command -v wails3 >/dev/null 2>&1; then
    echo "wails3 command not found"
    exit 1
  fi

  mkdir -p "$OUTPUT_PATH"

  prepare_common_assets

  for target in "${TARGETS[@]}"; do
    IFS='|' read -r goos goarch extension cc_var_name requires_cgo <<< "$target"
    build_target "$goos" "$goarch" "$extension" "$cc_var_name" "$requires_cgo"
  done

  if ! print_summary; then
    echo "==> build finished with errors: $OUTPUT_PATH"
    exit 1
  fi

  echo "==> build finished: $OUTPUT_PATH"
}

main "$@"