#!/usr/bin/env bash
# Shared safety checks and lifecycle for the native benchmark RAM disk wrappers.
set -euo pipefail
platform=$1
shift
action=${1:-}; [ "$#" -gt 0 ] && shift
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(git -C "$PWD" rev-parse --show-toplevel 2>/dev/null || pwd)
mountpoint=
size=auto
concurrency=1
safety=1.75
json=false
dry_run=false
no_swap=false
usage() { echo "usage: benchmark-ram-$platform.sh estimate|mount|status|unmount [--repo-root DIR] [--mountpoint PATH] [--size auto|SIZE] [--concurrency N] [--safety-factor FLOAT] [--json] [--dry-run] [--no-swap]" >&2; }
while [ "$#" -gt 0 ]; do
	case "$1" in
		--repo-root) [ "$#" -ge 2 ] || { usage; exit 2; }; repo_root=$2; shift 2 ;;
		--mountpoint) [ "$#" -ge 2 ] || { usage; exit 2; }; mountpoint=$2; shift 2 ;;
		--size) [ "$#" -ge 2 ] || { usage; exit 2; }; size=$2; shift 2 ;;
		--concurrency) [ "$#" -ge 2 ] || { usage; exit 2; }; concurrency=$2; shift 2 ;;
		--safety-factor) [ "$#" -ge 2 ] || { usage; exit 2; }; safety=$2; shift 2 ;;
		--json) json=true; shift ;;
		--dry-run) dry_run=true; shift ;;
		--no-swap) no_swap=true; shift ;;
		--help|-h) usage; exit 0 ;;
		*) echo "unknown option: $1" >&2; usage; exit 2 ;;
	esac
done
case "$action" in estimate|mount|status|unmount) ;; *) usage; exit 2 ;; esac
[ "$#" -eq 0 ] || { usage; exit 2; }
if [ "$platform" = macos ] && [ "$no_swap" = true ]; then echo '--no-swap is supported only on Linux' >&2; exit 2; fi
repo_root=$(CDPATH= cd -- "$repo_root" 2>/dev/null && pwd -P) || { echo 'repository root must be an existing directory' >&2; exit 2; }
case "$concurrency" in ''|*[!0-9]*) echo 'concurrency must be a positive integer' >&2; exit 2 ;; esac
[ "$concurrency" -gt 0 ] || { echo 'concurrency must be a positive integer' >&2; exit 2; }
[ "$concurrency" -le 1024 ] || { echo 'concurrency must be between 1 and 1024' >&2; exit 2; }
concurrency=$((10#$concurrency))
factor_percent=$(awk -v x="$safety" 'BEGIN {if(x !~ /^[0-9]+([.][0-9]+)?$/ || x<1 || x>10) exit 1; printf "%.0f",x*100}') || { echo 'safety factor must be between 1 and 10' >&2; exit 2; }
if [ -z "$mountpoint" ]; then
	if [ "$platform" = macos ]; then mountpoint=/Volumes/semedit-bench-ram
	else uid=$(id -u); if [ -d "/run/user/$uid" ]; then mountpoint="/run/user/$uid/semedit-bench-ram"; else mountpoint=/mnt/semedit-bench-ram; fi; fi
fi
case "$mountpoint" in /*) ;; *) echo 'mountpoint must be an absolute path' >&2; exit 2 ;; esac
case "$mountpoint" in *[[:space:]]*) echo 'mountpoint cannot contain whitespace' >&2; exit 2 ;; esac
case "$mountpoint" in *'/../'*|*'/./'*) echo 'mountpoint cannot contain dot path components' >&2; exit 2 ;; esac
parent_path=$mountpoint
while [ "$parent_path" != / ]; do
	[ ! -L "$parent_path" ] || { echo "refusing symlink path component: $parent_path" >&2; exit 2; }
	parent_path=$(dirname "$parent_path")
done
case "$mountpoint" in /) echo 'refusing to use filesystem root as mountpoint' >&2; exit 2 ;; esac
case "$mountpoint" in "$repo_root"|"$repo_root"/*) echo 'refusing a mountpoint inside the repository' >&2; exit 2 ;; esac
if [ "$action" = estimate ] && [ "$size" = auto ]; then
	if [ "$json" = true ]; then exec "$script_dir/benchmark-ram-size.sh" estimate --repo-root "$repo_root" --concurrency "$concurrency" --safety-factor "$safety" --json; fi
	exec "$script_dir/benchmark-ram-size.sh" estimate --repo-root "$repo_root" --concurrency "$concurrency" --safety-factor "$safety"
fi
if [ "$action" = status ] || [ "$action" = unmount ]; then
	requested=0
elif [ "$size" = auto ]; then
	estimate=$("$script_dir/benchmark-ram-size.sh" estimate --repo-root "$repo_root" --concurrency "$concurrency" --safety-factor "$safety" --json) || { echo 'automatic size unavailable; provide a manual --size after reviewing the incomplete sample warning' >&2; exit 3; }
	requested=$(printf '%s\n' "$estimate" | sed -n 's/.*"requested_bytes":\([0-9][0-9]*\).*/\1/p')
	confidence=$(printf '%s\n' "$estimate" | sed -n 's/.*"confidence":"\([^"]*\)".*/\1/p')
	inodes=$(printf '%s\n' "$estimate" | sed -n 's/.*"estimated_inodes":\([0-9][0-9]*\).*/\1/p')
	[ -n "$requested" ] || { echo 'could not parse automatic size estimate' >&2; exit 1; }
	[ "$confidence" = measured ] || { echo 'automatic size sample is uncertain; refusing mount' >&2; exit 3; }
else
	numeric=$size; suffix=
	case "$size" in *[KkMmGgTt]) suffix=${size#${size%?}}; numeric=${size%?} ;; *[KkMmGgTt][Bb]) suffix=${size%[Bb]}; suffix=${suffix#${suffix%?}}; numeric=${size%[KkMmGgTt][Bb]} ;; esac
	requested=$(awk -v n="$numeric" -v u="$suffix" 'BEGIN {if (n !~ /^[0-9]+([.][0-9]+)?$/ || n <= 0) exit 1; u=toupper(u); m=(u=="K"?1024:u=="M"?1048576:u=="G"?1073741824:u=="T"?1099511627776:1); v=n*m; if(v>9007199254740991) exit 1; printf "%.0f", v}') || { echo 'invalid or unsupported size' >&2; exit 2; }
fi
case "$requested" in ''|*[!0-9]*) echo 'invalid calculated size' >&2; exit 2 ;; esac
if [ "$action" = estimate ]; then
	budget=$("$script_dir/benchmark-ram-size.sh" budget "$requested" "$concurrency") || exit $?
	physical=$(printf '%s\n' "$budget" | sed -n 's/^physical_bytes=//p')
	available=$(printf '%s\n' "$budget" | sed -n 's/^available_bytes=//p')
	reserved=$(printf '%s\n' "$budget" | sed -n 's/^reserved_bytes=//p')
	safe=$(printf '%s\n' "$budget" | sed -n 's/^safe_bytes=//p')
	if [ "$json" = true ]; then printf '{"requested_bytes":%s,"requested_size":"%s","budget_status":"within","physical_bytes":%s,"available_bytes":%s,"reserved_bytes":%s,"safe_budget_bytes":%s}\n' "$requested" "$(awk -v n="$requested" 'BEGIN {printf "%.2f GiB",n/1073741824}')" "$physical" "$available" "$reserved" "$safe"
	else printf '%s\nrequested_size=%s\n' "$budget" "$(awk -v n="$requested" 'BEGIN {printf "%.2f GiB",n/1073741824}')"; fi
	exit 0
fi
if [ -L "$mountpoint" ]; then echo "refusing symlink mountpoint: $mountpoint" >&2; exit 2; fi
mounted=false
if [ -d "$mountpoint" ]; then
		if [ "$platform" = linux ]; then
			if mountpoint -q "$mountpoint"; then mounted=true; fi
		else
			if mount | awk -v p="$mountpoint" '$2=="on" && $3==p {found=1} END {exit !found}'; then mounted=true; fi
		fi
fi
marker="$mountpoint/.semedit-benchmark-ram"
marker_uid=
if [ "$mounted" = true ] && [ -f "$marker" ] && [ ! -L "$marker" ]; then
	marker_uid=$(sed -n 's/^uid=//p' "$marker")
	marker_header=$(sed -n '1p' "$marker")
	if [ "$platform" = linux ]; then marker_stat=$(stat -c '%u %a' "$marker" 2>/dev/null) || marker_stat=
	else marker_stat=$(stat -f '%u %Lp' "$marker" 2>/dev/null) || marker_stat=; fi
	[ "$marker_header" = semedit-benchmark-ram-v1 ] && [ "$marker_stat" = '0 444' ] || marker_uid=
fi
managed=false
if [ -n "$marker_uid" ] && [ "$marker_uid" = "$(id -u)" ]; then managed=true; fi
if [ "$action" = status ]; then
	if [ "$mounted" = true ]; then
		if [ "$platform" = linux ]; then mount | awk -v p="$mountpoint" '$2=="on" && $3==p {print; found=1} END {exit !found}' || exit 1
		else mount | awk -v p="$mountpoint" 'index($0," on " p " (tmpfs,") {print; found=1} END {exit !found}' || exit 1; fi
		df -k "$mountpoint" || exit 1
		df -i "$mountpoint" || exit 1
		if [ "$managed" = true ]; then echo "managed_by=semedit-benchmark-ram uid=$marker_uid"; else echo 'managed_by=unknown'; fi
	else echo "not mounted: $mountpoint"; fi
	exit 0
fi
if [ "$action" = unmount ]; then
	[ "$mounted" = true ] || { echo "not mounted: $mountpoint"; exit 0; }
	[ "$managed" = true ] || { echo 'refusing to unmount an unmanaged or differently owned filesystem' >&2; exit 2; }
	if [ "$platform" = linux ]; then mount | awk -v p="$mountpoint" '$2=="on" && $3==p && $4=="type" && $5=="tmpfs" {found=1} END {exit !found}' || { echo 'refusing to unmount a non-tmpfs filesystem' >&2; exit 2; }
	else mount | awk -v p="$mountpoint" 'index($0," on " p " (tmpfs,") {found=1} END {exit !found}' || { echo 'refusing to unmount a non-tmpfs filesystem' >&2; exit 2; }; fi
	if [ "$dry_run" = true ]; then echo "would unmount $mountpoint"; exit 0; fi
	sudo umount "$mountpoint"
	echo "unmounted $mountpoint (swap may still be used by the kernel for memory pressure)"
	exit 0
fi
if [ "$mounted" = true ]; then
	[ "$action" = mount ] || exit 0
	[ "$managed" = true ] || { echo 'refusing to reuse an unmanaged or differently owned filesystem' >&2; exit 2; }
	if [ "$platform" = linux ]; then mount | awk -v p="$mountpoint" '$2=="on" && $3==p && $4=="type" && $5=="tmpfs" {found=1} END {exit !found}' || { echo 'refusing incompatible filesystem already mounted here' >&2; exit 2; }
	else mount | awk -v p="$mountpoint" 'index($0," on " p " (tmpfs,") {found=1} END {exit !found}' || { echo 'refusing incompatible filesystem already mounted here' >&2; exit 2; }; fi
	existing=$(df -k "$mountpoint" | awk 'NR==2 {printf "%.0f",$2*1024}')
	[ "$existing" -ge "$requested" ] || { echo "existing tmpfs is smaller than requested size" >&2; exit 2; }
	if [ -n "${inodes:-}" ]; then existing_inodes=$(df -i "$mountpoint" | awk 'NR==2 {print $2}'); [ "$existing_inodes" -ge "$inodes" ] || { echo 'existing tmpfs has fewer inodes than requested' >&2; exit 2; }; fi
budget=$("$script_dir/benchmark-ram-size.sh" budget "$existing" "$concurrency") || exit $?
	echo "already mounted with sufficient capacity: $mountpoint"; exit 0
fi
if [ -e "$mountpoint" ]; then
	[ -d "$mountpoint" ] || { echo "mountpoint is not a directory: $mountpoint" >&2; exit 2; }
	[ -z "$(ls -A "$mountpoint")" ] || { echo "refusing nonempty mountpoint: $mountpoint" >&2; exit 2; }
fi
budget=$("$script_dir/benchmark-ram-size.sh" budget "$requested" "$concurrency") || exit $?
if [ ! -e "$mountpoint" ] && [ "$dry_run" = true ]; then echo "would run: sudo mkdir -p $mountpoint"; fi
if [ ! -e "$mountpoint" ] && [ "$dry_run" = false ]; then sudo mkdir -p "$mountpoint" || exit 1; fi
if [ "$dry_run" = true ]; then
	if [ "$platform" = macos ]; then echo "would run: sudo mount_tmpfs -e -s $requested $mountpoint"
	else
		uid=$(id -u); gid=$(id -g); options="size=$requested,uid=$uid,gid=$gid,mode=0700,exec"; [ "$no_swap" = false ] || options="$options,noswap"
		echo "would run: sudo mount -t tmpfs -o $options tmpfs $mountpoint"
	fi
	exit 0
fi
if [ "$platform" = macos ]; then sudo mount_tmpfs -e -s "$requested" "$mountpoint"
else
	uid=$(id -u); gid=$(id -g)
	options="size=$requested,uid=$uid,gid=$gid,mode=0700,exec"
		[ -z "${inodes:-}" ] || options="$options,nr_inodes=$inodes"
	[ "$no_swap" = false ] || options="$options,noswap"
	sudo mount -t tmpfs -o "$options" tmpfs "$mountpoint"
fi
if [ "$platform" = macos ]; then sudo chown "$(id -u):$(id -g)" "$mountpoint" && sudo chmod 0700 "$mountpoint"; fi
if [ "$platform" = linux ]; then mountpoint -q "$mountpoint" || { echo 'mount command succeeded but mountpoint verification failed' >&2; exit 1; }; mount | awk -v p="$mountpoint" '$2=="on" && $3==p && $4=="type" && $5=="tmpfs" {found=1} END {exit !found}' || { echo 'mounted filesystem is not tmpfs' >&2; exit 1; }
else mount | awk -v p="$mountpoint" 'index($0," on " p " (tmpfs,") {found=1} END {exit !found}' || { echo 'mounted filesystem is not tmpfs' >&2; exit 1; }; fi
capacity=$(df -k "$mountpoint" | awk 'NR==2 {printf "%.0f",$2*1024}')
[ "$capacity" -ge "$requested" ] || { echo "mounted tmpfs capacity $capacity is below requested $requested" >&2; exit 1; }
if [ -n "${inodes:-}" ]; then capacity_inodes=$(df -i "$mountpoint" | awk 'NR==2 {print $2}'); [ "$capacity_inodes" -ge "$inodes" ] || { echo "mounted tmpfs inode capacity $capacity_inodes is below requested $inodes" >&2; exit 1; }; fi
marker_tmp=$(mktemp "$mountpoint/.semedit-benchmark-ram.XXXXXX") || { echo 'could not create managed mount marker' >&2; exit 1; }
trap 'if [ -n "${marker_tmp:-}" ]; then rm -f "$marker_tmp"; fi' EXIT
printf 'semedit-benchmark-ram-v1\nuid=%s\n' "$(id -u)" > "$marker_tmp"
sudo chown 0:0 "$marker_tmp"
sudo chmod 0444 "$marker_tmp"
mv "$marker_tmp" "$marker"
if [ "$platform" = macos ] || [ "$no_swap" = false ]; then echo "mounted $requested bytes at $mountpoint; kernel swap behavior may still apply under memory pressure"
else echo "mounted $requested bytes at $mountpoint with noswap"; fi
