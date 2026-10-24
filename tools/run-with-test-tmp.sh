#!/usr/bin/env bash
# Runs tests with disposable fixtures on managed tmpfs and refreshes Darwin linker outputs.
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
ram_root=
toolexec=
while [ "$#" -gt 0 ]; do
	case "$1" in
		--ram-root) [ "$#" -ge 2 ] || { echo '--ram-root requires a path' >&2; exit 2; }; ram_root=$2; shift 2 ;;
		--toolexec) [ "$#" -ge 2 ] || { echo '--toolexec requires a path' >&2; exit 2; }; toolexec=$2; shift 2 ;;
		--) shift; break ;;
		*) echo "unknown option: $1" >&2; exit 2 ;;
	esac
done
[ "$#" -gt 0 ] || { echo 'a test command is required after --' >&2; exit 2; }

if [ -z "$ram_root" ]; then
	if [ "${ALLOW_SSD_TEST_TEMP:-0}" = 1 ]; then
		exec "$@"
	fi
	echo 'managed benchmark tmpfs is unavailable; mount it with make bench-ram-mount or explicitly set ALLOW_SSD_TEST_TEMP=1' >&2
	exit 2
fi

platform=$(uname -s)
case "$platform" in
	Darwin) ram_script="$script_dir/benchmark-harness/scripts/benchmark-ram-macos.sh" ;;
	Linux) ram_script="$script_dir/benchmark-harness/scripts/benchmark-ram-linux.sh" ;;
	*) echo "managed test tmpfs is unsupported on $platform" >&2; exit 2 ;;
esac
status=$("$ram_script" status --mountpoint "$ram_root") || { echo 'cannot verify the selected test tmpfs' >&2; exit 2; }
managed_marker="managed_by=semedit-benchmark-ram uid=$(id -u)"
if ! printf '%s\n' "$status" | grep -Fq "$managed_marker"; then
	echo "test temporary root is not a mounted benchmark-managed tmpfs: $ram_root" >&2
	exit 2
fi

temp_root=$(mktemp -d "${ram_root%/}/semedit-check.XXXXXX") || { echo "cannot create test temporary root under $ram_root" >&2; exit 2; }
cleanup() { rm -rf -- "$temp_root"; }
trap cleanup EXIT
mkdir -p "$temp_root/tmp" "$temp_root/go-tmp"
if [ "$platform" = Darwin ]; then
	if [ -n "$toolexec" ]; then
		[ -x "$toolexec" ] || { echo "Go linker refresh tool is not executable: $toolexec" >&2; exit 2; }
		case "$toolexec" in
			*" "*) toolexec_flag="'-toolexec=\"$toolexec\"'" ;;
			*) toolexec_flag="-toolexec=$toolexec" ;;
		esac
		GOFLAGS="${GOFLAGS:+$GOFLAGS }$toolexec_flag" TMPDIR="$temp_root/tmp" GOTMPDIR="$temp_root/go-tmp" "$@"
	else
		TMPDIR="$temp_root/tmp" GOTMPDIR="$temp_root/go-tmp" "$@"
	fi
else
	TMPDIR="$temp_root/tmp" GOTMPDIR="$temp_root/go-tmp" "$@"
fi
