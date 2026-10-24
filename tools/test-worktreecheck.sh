#!/usr/bin/env bash
# Exercise finished-worktree detection and cache cleanup against isolated Git repositories before exposing the destructive Make target.
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "$0")/.." && pwd -P)
fixture=$(mktemp -d "${TMPDIR:-/tmp}/worktreecheck.XXXXXX")
trap 'rm -rf -- "$fixture"' EXIT

main="$fixture/main checkout"
ready="$fixture/ready checkout"
dirty="$fixture/dirty checkout"
diverged="$fixture/diverged checkout"
mkdir -p -- "$main/tools"
git -C "$main" init -q -b main
git -C "$main" config user.name Test
git -C "$main" config user.email test@example.invalid
printf '.scratch/\n' > "$main/.gitignore"
printf 'base\n' > "$main/tracked.txt"
cp -- "$repo_root/tools/worktreecheck.sh" "$repo_root/tools/go-cache-root.sh" "$main/tools/"
git -C "$main" add .
git -C "$main" commit -qm 'Fixture base'

git -C "$main" worktree add --detach -q "$ready" main
cache_file="$ready/.scratch/go/mod/cache/download/example.com/module/@v/v1.0.0.zip"
mkdir -p -- "$(dirname -- "$cache_file")"
printf 'cached module\n' > "$cache_file"
chmod 0444 "$cache_file"

git -C "$main" worktree add --detach -q "$dirty" main
printf 'uncommitted\n' > "$dirty/untracked.txt"
git -C "$main" worktree add -b not-integrated -q "$diverged" main
printf 'diverged\n' >> "$diverged/tracked.txt"
git -C "$diverged" add tracked.txt
git -C "$diverged" commit -qm 'Not integrated'

listing=$(bash "$main/tools/worktreecheck.sh" list-finished-worktrees)
if [[ "$listing" != *"$ready"* || "$listing" == *"$dirty"* || "$listing" == *"$diverged"* ]]; then
	printf 'Unexpected finished worktree list:\n%s\n' "$listing" >&2
	exit 1
fi

removed=$(bash "$main/tools/worktreecheck.sh" clean-finished-worktrees)
if [[ -d "$ready" || -e "$cache_file" || "$removed" != *"Removed $ready"* ]]; then
	printf 'Finished worktree or read-only module cache was not cleaned:\n%s\n' "$removed" >&2
	exit 1
fi
if [[ ! -d "$dirty" || ! -d "$diverged" ]]; then
	printf 'A dirty or unintegrated worktree was removed.\n' >&2
	exit 1
fi

printf 'Worktree check: only clean, integrated worktrees are listed and removed; local module cache cleanup passed.\n'
