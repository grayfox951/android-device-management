#!/usr/bin/env bash
#
# install.sh — builds Android Device Management (TUI) and installs it system
# wide.
#
# The default target is /usr/bin. If /usr/bin turns out not to be on PATH, the
# first writable directory that *is* on PATH is used instead, so the command is
# actually reachable after installing. With nothing writable on PATH the script
# falls back to /usr/local/bin and says so.
#
# Usage:
#   ./install.sh                 build, then install
#   ./install.sh --prefix DIR    install into DIR instead of the auto-detected one
#   ./install.sh --dry-run       show what would happen, change nothing
#   ./install.sh --lang ru       language for the environment check
#
# Author: grayfox951

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

BINARY="adm"
PREFERRED_DIR="/usr/bin"
FALLBACK_DIR="/usr/local/bin"
DRY_RUN=0
PREFIX=""
LANG_OPT=""

die() {
	echo "error: $*" >&2
	exit 1
}

note() {
	printf '    %s\n' "$*"
}

usage() {
	sed -n '3,16p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
	case "$1" in
		--prefix)
			[ $# -ge 2 ] || die "--prefix needs a directory"
			PREFIX="$2"
			shift 2
			;;
		--prefix=*)
			PREFIX="${1#*=}"
			shift
			;;
		--lang)
			[ $# -ge 2 ] || die "--lang needs a language code"
			LANG_OPT="$2"
			shift 2
			;;
		--lang=*)
			LANG_OPT="${1#*=}"
			shift
			;;
		-dry-run | --dry-run)
			DRY_RUN=1
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1 (try --help)"
			;;
	esac
done

# --------------------------------------------------------------- PATH logic

# Prints every PATH entry, one per line.
path_dirs() {
	local IFS=':'
	local d
	# shellcheck disable=SC2086 # IFS=: on purpose, so $PATH splits into fields.
	for d in $PATH; do
		[ -n "$d" ] || continue
		printf '%s\n' "$d"
	done
}

# Canonical path of a directory, or nothing when it is not a usable directory.
resolve_dir() {
	[ -d "$1" ] || return 1
	(cd -- "$1" 2>/dev/null && pwd -P)
}

# True when the given directory appears on PATH, symlinks resolved.
on_path() {
	local want="$1" d resolved
	while IFS= read -r d; do
		resolved="$(resolve_dir "$d")" || continue
		if [ "$resolved" = "$want" ]; then
			return 0
		fi
	done < <(path_dirs)
	return 1
}

# Picks where the binary should go.
choose_target() {
	if [ -n "$PREFIX" ]; then
		printf '%s\n' "$PREFIX"
		return
	fi

	local preferred
	preferred="$(resolve_dir "$PREFERRED_DIR")" || preferred="$PREFERRED_DIR"
	if on_path "$preferred"; then
		printf '%s\n' "$preferred"
		return
	fi

	# /usr/bin is not on PATH, so an install there would leave the command
	# unreachable. Prefer the first PATH entry the user can write to.
	local d resolved
	while IFS= read -r d; do
		resolved="$(resolve_dir "$d")" || continue
		if [ -w "$resolved" ]; then
			printf '%s\n' "$resolved"
			return
		fi
	done < <(path_dirs)

	printf '%s\n' "$FALLBACK_DIR"
}

# ------------------------------------------------------------------- build

echo "==> building"
if ! "$SCRIPT_DIR/build.sh"; then
	die "build failed, nothing was installed"
fi

SOURCE="$SCRIPT_DIR/$BINARY"
[ -f "$SOURCE" ] || die "build.sh did not produce $BINARY"

# ----------------------------------------------------------------- setup step

# The program owns the platform-tools prompt, so the installer runs it in
# --setup mode instead of duplicating the check here. That way the question
# about adb and fastboot looks exactly like it does on a normal run, and there
# is one implementation of the install logic rather than two.
# The check needs a terminal: the program renders an interactive screen. When
# there is none the step is skipped rather than failing the whole install, so
# the script still works from CI or a cron job.
have_terminal() {
	[ -t 0 ] && [ -t 1 ]
}

run_setup_step() {
	if [ "$DRY_RUN" -eq 1 ]; then
		note "would run the environment check in the interface"
		return 0
	fi
	if ! have_terminal; then
		echo
		echo "==> environment check"
		note "no terminal available, skipping the interactive check"
		note "the program will ask about platform-tools on its first run"
		return 0
	fi

	local -a args=(--setup)
	if [ -n "$LANG_OPT" ]; then
		args+=(--lang "$LANG_OPT")
	fi

	echo
	echo "==> environment check"

	local rc=0
	"$SOURCE" "${args[@]}" || rc=$?

	case "$rc" in
		0)
			echo "==> adb and fastboot are ready"
			;;
		2)
			note "platform-tools installation was declined"
			note "installing the interface anyway; it will ask again on the first run"
			;;
		3)
			note "platform-tools could not be installed, see the output above"
			note "installing the interface anyway"
			;;
		*)
			note "the environment check failed unexpectedly (exit $rc)"
			note "installing the interface anyway"
			;;
	esac
}

run_setup_step

# ------------------------------------------------------------------ target

TARGET="$(choose_target)"
PREFERRED_CANON="$(resolve_dir "$PREFERRED_DIR")" || PREFERRED_CANON="$PREFERRED_DIR"

echo
echo "==> install target: $TARGET"
if [ "$TARGET" != "$PREFERRED_CANON" ]; then
	if [ -n "$PREFIX" ]; then
		note "chosen with --prefix"
	elif [ "$TARGET" = "$FALLBACK_DIR" ]; then
		note "$PREFERRED_DIR is not on PATH and no PATH entry is writable"
		note "falling back to $FALLBACK_DIR — add it to your PATH to use '$BINARY'"
	else
		note "$PREFERRED_DIR is not on PATH, using a PATH directory instead"
	fi
fi

if ! on_path "$(resolve_dir "$TARGET" || printf '%s' "$TARGET")"; then
	note "warning: $TARGET is not on PATH; you will need to add it yourself"
fi

if [ "$DRY_RUN" -eq 1 ]; then
	echo
	echo "==> dry run, stopping here"
	note "would install $SOURCE -> $TARGET/$BINARY"
	exit 0
fi

# Nearest existing ancestor of the target: creating the target only needs this
# directory to be writable, not root.
creation_parent() {
	local d="$TARGET" p
	while [ ! -d "$d" ]; do
		p="$(dirname -- "$d")"
		[ "$p" = "$d" ] && break
		d="$p"
	done
	[ -d "$d" ] && printf '%s\n' "$d"
}

# ---------------------------------------------------------------- privilege

# Building never needs root, and neither does copying into a directory the
# user already owns, so escalate only when the destination is really not
# writable. That keeps "--prefix ~/bin" usable without any password prompt.
SUDO=""
needs_escalation=0
if [ -d "$TARGET" ]; then
	[ -w "$TARGET" ] || needs_escalation=1
else
	parent="$(creation_parent || true)"
	if [ -z "$parent" ] || [ ! -w "$parent" ]; then
		needs_escalation=1
	fi
fi

if [ "$needs_escalation" -eq 1 ] && [ "$(id -u)" -ne 0 ]; then
	if ! command -v sudo >/dev/null 2>&1; then
		die "root is required to write to $TARGET and sudo is not installed"
	fi
	SUDO="sudo"
	note "not running as root, using sudo for the copy"
fi

if [ ! -d "$TARGET" ]; then
	note "$TARGET does not exist, creating it"
	# shellcheck disable=SC2086 # SUDO is empty or a single command.
	$SUDO mkdir -p -- "$TARGET" || die "could not create $TARGET"
fi

if [ ! -w "$TARGET" ] && [ -z "$SUDO" ]; then
	die "$TARGET is not writable"
fi

# ------------------------------------------------------------------ install

echo "==> installing"
# install(1) copies with the right mode in one step and creates parents, so a
# partially written binary is never left behind.
# shellcheck disable=SC2086 # SUDO is empty or a single command.
$SUDO install -m 0755 -- "$SOURCE" "$TARGET/$BINARY" ||
	die "could not install into $TARGET"

INSTALLED="$TARGET/$BINARY"
[ -x "$INSTALLED" ] || die "$INSTALLED is missing or not executable"

echo "==> verifying"
"$INSTALLED" --version

echo
echo "==> done: $INSTALLED"
if command -v "$BINARY" >/dev/null 2>&1; then
	echo
	note "run it with: $BINARY"
else
	echo
	note "run it with: $INSTALLED"
	note "$TARGET is not on PATH — add it, e.g. export PATH=\"$TARGET:\$PATH\""
fi

cat <<'EOF'

Next: connect the phone over USB and run the program. It checks for adb and
fastboot on start-up and offers to install them if they are missing.
EOF

# ------------------------------------------------------------------ removal
# Kept as a comment rather than a flag so the script stays focused:
#   sudo rm -f <target>/adm