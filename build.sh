#!/usr/bin/env bash
#
# build.sh — builds Android Device Management (TUI).
#
# The binary is written next to the sources, in this script's own directory,
# so the source tree stays self contained and nothing is installed on the way.
# Use install.sh to put the result on the system PATH.
#
# Author: grayfox951

set -euo pipefail

# Always resolve paths from this script's location, not the caller's cwd, so
# the script can be called from anywhere.
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BINARY="adm"
AUTHOR="grayfox951"

# Strips the "v" prefix and any pre-release suffix from a git tag.
version_from_git() {
	local tag
	tag="$(git describe --tags --always --dirty 2>/dev/null || true)"
	[ -n "$tag" ] || return 1
	printf '%s' "${tag#v}"
}

# The version can be forced with VERSION=1.2.3, otherwise it comes from git,
# otherwise it falls back to whatever the source already declares.
if [ -z "${VERSION:-}" ]; then
	VERSION="$(version_from_git || true)"
fi
if [ -z "${VERSION:-}" ]; then
	VERSION="$(grep -oE 'version = "[^"]+"' internal/tui/app.go | head -1 | cut -d'"' -f2)"
fi
VERSION="${VERSION:-0.0.0}"

# ---------------------------------------------------------------- toolchain

if ! command -v go >/dev/null 2>&1; then
	echo "error: the Go toolchain is not installed" >&2
	echo "       install it from https://go.dev/dl/ and try again" >&2
	exit 1
fi

# Refuse to build with a toolchain older than the one the module declares.
REQUIRED="$(awk '/^go /{print $2; exit}' go.mod)"
if ! go version >/dev/null 2>&1; then
	echo "error: 'go' is present but not usable: $(command -v go)" >&2
	exit 1
fi
echo "==> toolchain: $(go version)"
echo "==> version:   $VERSION"
echo "==> author:    $AUTHOR"

# ------------------------------------------------------------------ targets

# GOFLAGS from the environment is honoured; the defaults below only apply when
# it is unset, so CI can override them without editing this script.
GOFLAGS="${GOFLAGS:--trimpath}"
CGO_ENABLED="${CGO_ENABLED:-0}"

echo "==> checking"
go vet ./...

echo "==> testing"
if ! go test ./...; then
	echo "error: tests failed, refusing to build a binary" >&2
	exit 1
fi

echo "==> building $SCRIPT_DIR/$BINARY"
# shellcheck disable=SC2086 # GOFLAGS is intentionally word-split.
CGO_ENABLED="$CGO_ENABLED" go build \
	${GOFLAGS} \
	-ldflags "-s -w \
		-X adm/internal/tui.version=$VERSION \
		-X adm/internal/tui.author=$AUTHOR" \
	-o "$BINARY" \
	.

chmod +x "$BINARY"

echo "==> done: $SCRIPT_DIR/$BINARY"
# The ./ prefix is required: a bare name would be looked up in PATH, and the
# freshly built binary is not on PATH yet.
"./$BINARY" --version
echo
echo "Run it from here with:  ./$BINARY"
echo "Or install it with:    ./install.sh"