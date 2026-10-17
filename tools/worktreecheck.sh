#!/usr/bin/env bash
# Use Git's worktree and ancestry records to list or remove only clean checkouts fully integrated into local main.
set -euo pipefail

usage() {
	cat <<'EOF'
Usage: worktreecheck.sh {list-finished-worktrees|clean-finished-worktrees}

list-finished-worktrees lists linked worktrees that are clean and whose HEAD is
an ancestor of local main. clean-finished-worktrees cleans their local Go
module caches and removes them. Shared Go module caches are never removed.
EOF
}

remove_done=false
case "${1:-}" in
	list-finished-worktrees) ;;
	clean-finished-worktrees) remove_done=true ;;
	-h|--help) usage; exit 0 ;;
	*) usage >&2; exit 2 ;;
esac
if (($# != 1)); then
	usage >&2
	exit 2
fi

repo_root=$(git -C "$(dirname -- "$0")/.." rev-parse --show-toplevel)
main_commit=$(git -C "$repo_root" rev-parse --verify 'refs/heads/main^{commit}') || {
	printf 'Cannot resolve local main in %s\n' "$repo_root" >&2
	exit 1
}
shared_cache_root=$(bash "$repo_root/tools/go-cache-root.sh" "$repo_root")
shared_modcache=${GOMODCACHE:-"$shared_cache_root/mod"}

main_worktree=$(git -C "$repo_root" rev-parse --show-toplevel)
found=false
printf 'Finished worktrees (clean and integrated into local main %s):\n' "${main_commit:0:12}"

clean_local_modcache() {
	local worktree=$1 modcache physical_modcache physical_shared
	modcache="$worktree/.scratch/go/mod"
	if [[ ! -d "$modcache" ]]; then
		return 0
	fi
	physical_modcache=$(cd -- "$modcache" && pwd -P)
	if [[ -d "$shared_modcache" ]]; then
		physical_shared=$(cd -- "$shared_modcache" && pwd -P)
	else
		physical_shared=$shared_modcache
	fi
	if [[ "$physical_modcache" == "$physical_shared" ]]; then
		printf 'Skipping shared Go module cache %s for worktree %s\n' "$physical_modcache" "$worktree"
		return 0
	fi
	if [[ -L "$modcache" ]]; then
		printf 'Refusing to clean non-local module cache symlink %s in %s\n' "$modcache" "$worktree" >&2
		return 1
	fi
	printf 'Cleaning local Go module cache %s\n' "$physical_modcache"
	(
		cd -- "$worktree"
		GOENV=off GOTOOLCHAIN=local GOWORK=off GOPATH="$worktree/.scratch/go" GOMODCACHE="$physical_modcache" go clean -modcache
	)
}

process_worktree() {
	local status
	if [[ -z "${worktree:-}" || "$worktree" == "$main_worktree" || "$on_main" == true || "$prunable" == true ]]; then
		return 0
	fi
	status=$(git -C "$worktree" status --porcelain --untracked-files=normal) || {
		printf 'Cannot inspect worktree status for %s\n' "$worktree" >&2
		return 1
	}
	if [[ -n "$status" ]] || ! git -C "$repo_root" merge-base --is-ancestor "$head" "$main_commit"; then
		return 0
	fi
	found=true
	if [[ "$remove_done" == true ]]; then
		clean_local_modcache "$worktree"
		git -C "$repo_root" worktree remove -- "$worktree"
		printf 'Removed %s\n' "$worktree"
	else
		printf '%s\n' "$worktree"
	fi
}

while IFS= read -r -d '' field; do
	if [[ -z "$field" ]]; then
		process_worktree
		worktree=
		head=
		prunable=false
		on_main=false
		continue
	fi
	case "$field" in
		'worktree '*) worktree=${field#worktree } ;;
		'HEAD '*) head=${field#HEAD } ;;
		'branch refs/heads/main') on_main=true ;;
		'branch '*|detached|bare) ;;
		prunable|prunable\ *) prunable=true ;;
		*) printf 'Unexpected git worktree record: %q\n' "$field" >&2; exit 1 ;;
	esac
done < <(git -C "$repo_root" worktree list --porcelain -z)

# git emits a blank record terminator, but process the final record defensively.
process_worktree

if [[ "$found" == false ]]; then
	printf 'None.\n'
fi
