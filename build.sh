#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v go >/dev/null 2>&1; then
	echo "error: Go is not installed or is not available in PATH" >&2
	exit 1
fi

target_os="${GOOS:-$(go env GOOS)}"
target_arch="${GOARCH:-$(go env GOARCH)}"
cgo_enabled="${CGO_ENABLED:-0}"
output_root="${OUTPUT_DIR:-"${project_dir}/dist"}"
version="${VERSION:-}"

if [[ -z "${version}" ]]; then
	version="$(git -C "${project_dir}" describe --tags --always --dirty 2>/dev/null || true)"
	version="${version:-devel}"
fi

binary_name="sshw"
if [[ "${target_os}" == "windows" ]]; then
	binary_name+=".exe"
fi

output_dir="${output_root}/${target_os}-${target_arch}"
output_path="${output_dir}/${binary_name}"
mkdir -p "${output_dir}"

printf 'Building sshw %s for %s/%s\n' "${version}" "${target_os}" "${target_arch}"

(
	cd "${project_dir}"
	CGO_ENABLED="${cgo_enabled}" GOOS="${target_os}" GOARCH="${target_arch}" \
		go build \
		-trimpath \
		-ldflags="-s -w -X main.Build=${version}" \
		-o "${output_path}" \
		./cmd/sshw
)

printf 'Built %s\n' "${output_path}"
