#!/usr/bin/env bash
# Exercise RAM sizing and safety behavior without mounting filesystems or requiring privileges.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
tmp_root=${TMPDIR:-/tmp}
if [ "$(uname -s)" = Darwin ] && [ -d /private/tmp ]; then tmp_root=/private/tmp; fi
tmp=$(mktemp -d "$tmp_root/benchmark-ram-test.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
fail() { echo "FAIL: $*" >&2; exit 1; }
mkdir -p "$tmp/tools"
cp "$script_dir/../../go-environment.mk" "$tmp/tools/go-environment.mk"
cp "$script_dir/../../go-cache-root.sh" "$tmp/tools/go-cache-root.sh"
makefile="$script_dir/../Makefile"
make_mount="$tmp/make-mount"
mkdir -p "$make_mount"
make_plan=$(make -n -f "$makefile" REPO_ROOT="$tmp" GOMODCACHE="$tmp/.scratch/go/mod" BENCH_RAM_DEFAULT="$make_mount" bench TASK=task-09-composite-refactor)
case "$make_plan" in *--ram-fixture-root*) fail 'Make selected RAM without a managed marker' ;; esac
touch "$make_mount/.semedit-benchmark-ram"
make_plan=$(make -n -f "$makefile" REPO_ROOT="$tmp" GOMODCACHE="$tmp/.scratch/go/mod" BENCH_RAM_DEFAULT="$make_mount" bench TASK=task-09-composite-refactor)
case "$make_plan" in *"--ram-fixture-root \"$make_mount\""*"--module-cache \"$tmp/.scratch/go/mod\""*) ;; *) fail "Make did not select managed RAM and existing module cache: $make_plan" ;; esac
make_plan=$(make -n -f "$makefile" REPO_ROOT="$tmp" GOMODCACHE="$tmp/.scratch/go/mod" BENCH_RAM_DEFAULT="$make_mount" BENCH_RAM_ROOT= bench TASK=task-09-composite-refactor)
case "$make_plan" in *--ram-fixture-root*|*--module-cache*) fail 'Make ignored explicit SSD selection' ;; esac
make_plan=$(make -n -f "$makefile" REPO_ROOT="$tmp" GOMODCACHE="$tmp/.scratch/go/mod" BENCH_RAM_DEFAULT="$make_mount" BENCH_RAM_ROOT="$tmp/custom-mount" BENCH_MODCACHE="$tmp/existing-modules" bench TASK=task-09-composite-refactor)
case "$make_plan" in *"--ram-fixture-root \"$tmp/custom-mount\""*"--module-cache \"$tmp/existing-modules\""*) ;; *) fail 'Make ignored explicit storage overrides' ;; esac
. "$script_dir/benchmark-ram-memory.sh"
mkdir -p "$tmp/cgroup/slice/child" "$tmp/cgroup-v1/group/child"
printf '5000\n' > "$tmp/cgroup/slice/memory.max"
printf '1000\n' > "$tmp/cgroup/slice/memory.current"
printf '3000\n' > "$tmp/cgroup/slice/child/memory.max"
printf '2000\n' > "$tmp/cgroup/slice/child/memory.current"
[ "$(ram_cgroup_v2_available 8000 "$tmp/cgroup" /slice/child)" -eq 1000 ] || fail 'ignored nested cgroup v2 memory limit'
printf 'max\n' > "$tmp/cgroup/slice/child/memory.max"
[ "$(ram_cgroup_v2_available 8000 "$tmp/cgroup" /slice/child)" -eq 4000 ] || fail 'ignored ancestor cgroup v2 memory limit'
printf '5000\n' > "$tmp/cgroup-v1/group/memory.limit_in_bytes"
printf '1000\n' > "$tmp/cgroup-v1/group/memory.usage_in_bytes"
printf '3000\n' > "$tmp/cgroup-v1/group/child/memory.limit_in_bytes"
printf '2000\n' > "$tmp/cgroup-v1/group/child/memory.usage_in_bytes"
[ "$(ram_cgroup_v1_available 8000 "$tmp/cgroup-v1" /group/child)" -eq 1000 ] || fail 'ignored nested cgroup v1 memory limit'
mkdir -p "$tmp/samples/run_small/src" "$tmp/samples/run_small/.scratch/go/build" "$tmp/samples/run_small/.scratch/go/tmp" "$tmp/samples/run_large/src" "$tmp/samples/run_large/.scratch/go/build" "$tmp/samples/run_large/.scratch/go/tmp" "$tmp/samples/run_large/.scratch/cache/repos" "$tmp/bin"
cat > "$tmp/bin/sysctl" <<'MOCK'
#!/bin/sh
echo 4096
MOCK
cat > "$tmp/bin/vm_stat" <<'MOCK'
#!/bin/sh
printf 'Pages free: 3000000.\nPages inactive: 3000000.\nPages speculative: 1000000.\n'
MOCK
chmod +x "$tmp/bin/sysctl" "$tmp/bin/vm_stat"
touch "$tmp/samples/run_small/go.mod" "$tmp/samples/run_large/go.mod"
dd if=/dev/zero of="$tmp/samples/run_small/src/data" bs=1024 count=1 2>/dev/null
dd if=/dev/zero of="$tmp/samples/run_large/src/data" bs=1024 count=4 2>/dev/null
dd if=/dev/zero of="$tmp/samples/run_large/.scratch/cache/repos/data" bs=1024 count=50 2>/dev/null
out=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/samples" "$script_dir/benchmark-ram-size.sh" estimate --concurrency 2 --safety-factor 1.5 --json)
case "$out" in *'"sample_max_bytes":'*'"concurrency":2'*'"safety_factor":1.5'*'"confidence":"measured"'*) ;; *) fail "unexpected estimate: $out" ;; esac
bytes=$(printf '%s' "$out" | sed -n 's/.*"requested_bytes":\([0-9][0-9]*\).*/\1/p')
[ "$bytes" -ge 1048576 ] && [ $((bytes % 1073741824)) -eq 0 ] || fail "estimate was not rounded up: $bytes"
normalized=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/samples" "$script_dir/benchmark-ram-size.sh" estimate --safety-factor 01.75 --json)
case "$normalized" in *'"safety_factor":1.75'*) ;; *) fail "safety factor was not normalized: $normalized" ;; esac
normalized=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/samples" BENCHMARK_RAM_FIXED_OVERHEAD_BYTES=0100 "$script_dir/benchmark-ram-size.sh" estimate --concurrency 010 --json)
case "$normalized" in *'"concurrency":10'*'"fixed_overhead_bytes":100'*) ;; *) fail "numeric inputs were not normalized: $normalized" ;; esac
two_jobs=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/samples" BENCHMARK_RAM_PER_JOB_RESERVE_BYTES=1000000000 BENCHMARK_RAM_MIN_FREE_BYTES=0 "$script_dir/benchmark-ram-size.sh" estimate --concurrency 02 --json)
ten_jobs=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/samples" BENCHMARK_RAM_PER_JOB_RESERVE_BYTES=1000000000 BENCHMARK_RAM_MIN_FREE_BYTES=0 "$script_dir/benchmark-ram-size.sh" estimate --concurrency 010 --json)
two_reserved=$(printf '%s' "$two_jobs" | sed -n 's/.*"reserved_bytes":\([0-9][0-9]*\).*/\1/p')
ten_reserved=$(printf '%s' "$ten_jobs" | sed -n 's/.*"reserved_bytes":\([0-9][0-9]*\).*/\1/p')
[ "$ten_reserved" -eq $((two_reserved + 8000000000)) ] || fail "leading-zero concurrency changed reserve arithmetic: $two_reserved -> $ten_reserved"
mkdir -p "$tmp/uncertain/run_partial/src" "$tmp/cache/repos/repo_partial/src"
touch "$tmp/uncertain/run_partial/go.mod" "$tmp/cache/repos/repo_partial/go.mod"
dd if=/dev/zero of="$tmp/uncertain/run_partial/src/source.go" bs=1024 count=1 2>/dev/null
dd if=/dev/zero of="$tmp/cache/repos/repo_partial/src/source.go" bs=1024 count=1 2>/dev/null
partial=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/uncertain" "$script_dir/benchmark-ram-size.sh" estimate --repo-root "$tmp/none" --json)
case "$partial" in *'"confidence":"uncertain"'*) ;; *) fail "incomplete sample was not uncertain: $partial" ;; esac
fallback=$(PATH="$tmp/bin:$PATH" BENCHMARK_RAM_SAMPLE_ROOT="$tmp/no-samples" BENCHMARK_RAM_REPO_CACHE="$tmp/cache/repos" "$script_dir/benchmark-ram-size.sh" estimate --repo-root "$tmp/none" --json)
case "$fallback" in *'"confidence":"uncertain"'*'"partial_fallback":1'*) ;; *) fail "cached repo fallback was not partial: $fallback" ;; esac
if "$script_dir/benchmark-ram-size.sh" estimate --concurrency 0 >/dev/null 2>&1; then fail 'accepted zero concurrency'; fi
if "$script_dir/benchmark-ram-size.sh" estimate --safety-factor 0.5 >/dev/null 2>&1; then fail 'accepted safety factor below one'; fi
if BENCHMARK_RAM_MIN_FREE_BYTES=invalid "$script_dir/benchmark-ram-size.sh" budget 1 >/dev/null 2>&1; then fail 'accepted nonnumeric minimum-free-memory override'; fi
if "$script_dir/benchmark-ram-size.sh" estimate --repo-root "$tmp/empty" >/dev/null 2>&1; then fail 'accepted missing measurements'; fi
newline_sample="$tmp/newline-sample"
mkdir -p "$newline_sample/src" "$newline_sample/.scratch/go/build" "$newline_sample/.scratch/go/tmp"
touch "$newline_sample/go.mod"
dd if=/dev/zero of="$newline_sample/src/payload" bs=1024 count=1 2>/dev/null
plain_bytes=$(PATH="$tmp/bin:$PATH" "$script_dir/benchmark-ram-size.sh" estimate --json "$newline_sample" | sed -n 's/.*"sample_max_bytes":\([0-9][0-9]*\).*/\1/p')
newline_name=$(printf 'payload\n-999999')
mv "$newline_sample/src/payload" "$newline_sample/src/$newline_name"
newline_bytes=$(PATH="$tmp/bin:$PATH" "$script_dir/benchmark-ram-size.sh" estimate --json "$newline_sample" | sed -n 's/.*"sample_max_bytes":\([0-9][0-9]*\).*/\1/p')
[ -n "$plain_bytes" ] && [ "$plain_bytes" = "$newline_bytes" ] || fail "newline filename changed measured bytes: $plain_bytes != $newline_bytes"
repo=$tmp/repo
mkdir -p "$repo"
mkdir -p "$tmp/nonempty"
touch "$tmp/nonempty/keep"
ln -s "$tmp/nonempty" "$tmp/link"
for platform in macos linux; do
	ram_script="$script_dir/benchmark-ram-$platform.sh"
	if [ "$platform" = macos ]; then
		out=$(PATH="$tmp/bin:$PATH" "$ram_script" mount --repo-root "$repo" --mountpoint "$tmp/new-mount-$platform" --size 1G --dry-run)
		case "$out" in *'would run: sudo mount_tmpfs -e -s 1073741824 '*) ;; *) fail "unexpected macOS dry-run: $out" ;; esac
	else
		out=$(PATH="$tmp/bin:$PATH" "$ram_script" mount --repo-root "$repo" --mountpoint "$tmp/new-mount-$platform" --size 1G --dry-run)
		case "$out" in *'would run: sudo mount -t tmpfs'*'mode=0700,exec'*) ;; *) fail "unexpected Linux dry-run: $out" ;; esac
		out=$(PATH="$tmp/bin:$PATH" "$ram_script" mount --repo-root "$repo" --mountpoint "$tmp/new-noswap" --size 1G --no-swap --dry-run)
		case "$out" in *'would run: sudo mount -t tmpfs'*'mode=0700,exec,noswap'*) ;; *) fail "noswap option missing: $out" ;; esac
	fi
	[ ! -e "$tmp/new-mount-$platform" ] || fail 'dry-run created mountpoint'
	if PATH="$tmp/bin:$PATH" "$ram_script" mount --mountpoint "$tmp/nonempty" --size 1M --dry-run >/dev/null 2>&1; then fail "accepted nonempty $platform mountpoint"; fi
	if PATH="$tmp/bin:$PATH" "$ram_script" mount --mountpoint "$tmp/link" --size 1M --dry-run >/dev/null 2>&1; then fail "accepted symlink $platform mountpoint"; fi
done
if (cd "$repo" && PATH="$tmp/bin:$PATH" "$script_dir/benchmark-ram-linux.sh" mount --repo-root . --mountpoint "$repo/inside" --size 1M --dry-run >/dev/null 2>&1); then fail 'relative repository root bypassed mountpoint containment'; fi
mkdir -p "$tmp/mockbin" "$tmp/simulated-mount"
cat > "$tmp/mockbin/sudo" <<'MOCK'
#!/bin/sh
printf '%s\n' "$*" >> "$MOCK_LOG"
case "$1" in chown|chmod) exit 0 ;; esac
exec "$@"
MOCK
cat > "$tmp/mockbin/stat" <<'MOCK'
#!/bin/sh
case "$MOCK_FORGED" in
	true) printf '%s 644\n' "$(id -u)" ;;
	*) printf '0 444\n' ;;
esac
MOCK
cat > "$tmp/mockbin/mountpoint" <<'MOCK'
#!/bin/sh
[ -f "$MOCK_MOUNTED" ]
MOCK
cat > "$tmp/mockbin/mount" <<'MOCK'
#!/bin/sh
if [ "$#" -eq 0 ]; then
	if [ -f "$MOCK_MOUNTED" ]; then printf 'tmpfs on %s type %s (rw,size=2147483648)\n' "$MOCK_PATH" "${MOCK_FSTYPE:-tmpfs}"; fi
	 exit 0
fi
if [ "$1" = -t ]; then touch "$MOCK_MOUNTED"; fi
MOCK
cat > "$tmp/mockbin/umount" <<'MOCK'
#!/bin/sh
rm -f "$MOCK_MOUNTED"
MOCK
cat > "$tmp/mockbin/df" <<'MOCK'
#!/bin/sh
if [ "$1" = -i ]; then printf 'Filesystem Inodes IUsed IFree IUse%% Mounted on\ntmpfs 262144 1 262143 1%% %s\n' "$MOCK_PATH"
else printf 'Filesystem 1K-blocks Used Available Use%% Mounted on\ntmpfs 2097152 0 2097152 0%% %s\n' "$MOCK_PATH"; fi
MOCK
chmod +x "$tmp/mockbin/"*
mock_path="$tmp/mockbin:$tmp/bin:$PATH"
export MOCK_PATH="$tmp/simulated-mount" MOCK_MOUNTED="$tmp/mounted-state" MOCK_LOG="$tmp/sudo.log" MOCK_FORGED=false
PATH="$mock_path" "$script_dir/benchmark-ram-linux.sh" mount --repo-root "$repo" --mountpoint "$MOCK_PATH" --size 1G >/dev/null
grep -q 'mode=0700,exec' "$MOCK_LOG" || fail 'Linux mount did not use private executable tmpfs options'
if grep -q 'noswap' "$MOCK_LOG"; then fail 'enabled noswap without explicit request'; fi
PATH="$mock_path" "$script_dir/benchmark-ram-linux.sh" mount --repo-root "$repo" --mountpoint "$MOCK_PATH" --size 1G >/dev/null
[ "$(grep -c 'mount -t tmpfs' "$MOCK_LOG")" -eq 1 ] || fail 'repeat mount was not idempotent'
status=$(PATH="$mock_path" "$script_dir/benchmark-ram-linux.sh" status --mountpoint "$MOCK_PATH")
case "$status" in *'managed_by=semedit-benchmark-ram'*) ;; *) fail "managed status missing: $status" ;; esac
PATH="$mock_path" "$script_dir/benchmark-ram-linux.sh" unmount --mountpoint "$MOCK_PATH" >/dev/null
[ ! -f "$MOCK_MOUNTED" ] || fail 'unmount did not call normal unmount'
touch "$MOCK_MOUNTED"
MOCK_FORGED=true
export MOCK_FORGED
printf 'semedit-benchmark-ram-v1\nuid=%s\n' "$(id -u)" > "$MOCK_PATH/.semedit-benchmark-ram"
if PATH="$mock_path" "$script_dir/benchmark-ram-linux.sh" unmount --mountpoint "$MOCK_PATH" >/dev/null 2>&1; then fail 'unmounted an unmanaged tmpfs'; fi
[ -f "$MOCK_MOUNTED" ] || fail 'unmanaged refusal changed mount state'
echo 'benchmark RAM script tests passed'
