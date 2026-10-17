#!/bin/bash
# Exercise real Make entry points against primary, linked, and exported checkouts so cache sharing cannot accidentally share mutable worktree state.
set -euo pipefail

repo=$(cd -- "$(dirname -- "$0")/.." && pwd -P)
mkdir -p -- "$repo/.scratch"
fixture=$(mktemp -d "$repo/.scratch/go-environment.XXXXXX")
trap 'rm -rf -- "$fixture"' EXIT
cat > "$fixture/probe.mk" <<'MAKE'
cache-probe:
	@printf '%s\n' '$(GOMODCACHE)' '$(GOCACHE)' '$(GOTMPDIR)' '$(GOBIN)' '$(GOENV)' '$(GOPLSCACHE)' '$(GOPATH)'
MAKE

copy_configuration() {
    local target=$1
    mkdir -p -- "$target/tools/benchmark-harness"
    cp -- "$repo/Makefile" "$target/Makefile"
    cp -- "$repo/tools/go-environment.mk" "$repo/tools/go-cache-root.sh" "$target/tools/"
    cp -- "$repo/tools/benchmark-harness/Makefile" "$target/tools/benchmark-harness/Makefile"
}

assert_configuration() {
    local checkout=$1 entry=$2 cache=$3 module=$4 build=$5 actual expected
    shift 5
    actual=$(env -u GOMODCACHE -u GOCACHE -u GO_CACHE_ROOT -u GO_SCRATCH -u MAKEFLAGS -u MAKELEVEL -u MFLAGS GOTOOLCHAIN=local \
        make --no-print-directory -s -C "$checkout" -f "$entry" -f "$fixture/probe.mk" cache-probe "$@")
    expected=$(printf '%s\n' "$module" "$build" "$checkout/.scratch/go/tmp" "$checkout/.scratch/go/bin" \
        "$checkout/.scratch/go/env" "$checkout/.scratch/go/gopls" "$checkout/.scratch/go")
    if [[ "$actual" != "$expected" ]]; then
        printf 'Unexpected Go environment for %s (%s, base %s):\n%s\nExpected:\n%s\n' "$checkout" "$entry" "$cache" "$actual" "$expected" >&2
        exit 1
    fi
}

primary="$fixture/primary"
linked="$fixture/linked"
archive="$fixture/exported"
mkdir -p -- "$primary"
git -C "$primary" init -q
git -C "$primary" -c user.name=Fixture -c user.email=fixture@example.invalid -c core.hooksPath=/dev/null commit --allow-empty -qm 'Cache fixture'
git -C "$primary" -c core.hooksPath=/dev/null worktree add --detach -q "$linked" HEAD
for checkout in "$primary" "$linked" "$archive"; do
    copy_configuration "$checkout"
done

for entry in Makefile tools/benchmark-harness/Makefile; do
    for checkout in "$primary" "$linked"; do
        assert_configuration "$checkout" "$entry" "$primary/.scratch/go" "$primary/.scratch/go/mod" "$primary/.scratch/go/build"
    done
    assert_configuration "$archive" "$entry" "$archive/.scratch/go" "$archive/.scratch/go/mod" "$archive/.scratch/go/build"
    assert_configuration "$linked" "$entry" "$fixture/isolated" "$fixture/isolated/mod" "$fixture/isolated/build" "GO_CACHE_ROOT=$fixture/isolated"
    assert_configuration "$linked" "$entry" "$primary/.scratch/go" "$fixture/custom modules" "$fixture/custom build" \
        "GOMODCACHE=$fixture/custom modules" "GOCACHE=$fixture/custom build"
done

printf 'Go cache configuration: primary, linked, exported, isolated, and overridden paths passed for both Make entry points.\n'
