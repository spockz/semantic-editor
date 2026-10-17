#!/bin/bash
# Resolve the surviving primary checkout so linked worktrees reuse Go caches without sharing mutable tools or temporary files.
set -euo pipefail

checkout=$(cd -- "$1" && pwd -P)
if [[ ! -e "$checkout/.git" ]]; then
    printf '%s\n' "$checkout/.scratch/go"
    exit 0
fi

git -C "$checkout" worktree list --porcelain -z | {
    primary=
    while IFS= read -r -d '' field; do
        if [[ -z "$primary" && "$field" == 'worktree '* ]]; then
            primary=${field#worktree }
        fi
    done
    if [[ -z "$primary" ]]; then
        printf 'Cannot resolve primary checkout for %s\n' "$checkout" >&2
        exit 1
    fi
    printf '%s\n' "$primary/.scratch/go"
}
