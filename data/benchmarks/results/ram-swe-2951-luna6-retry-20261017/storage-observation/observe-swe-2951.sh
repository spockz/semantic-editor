#!/usr/bin/env bash
# Record RAM allocation and selected process cache paths without collecting credentials.
set -euo pipefail
ram_root=/Volumes/semedit-bench-ram
output=${BENCH_STORAGE_OBSERVATION:-$PWD/.scratch/swe-2951-storage}
mkdir -p "$output"
measure() {
  local path=$1 data
  if [ ! -d "$path" ]; then printf '0'; return; fi
  if data=$(du -sk "$path" 2>> "$output/measurement-errors.log"); then
    printf '%s\n' "$data" | awk 'NR==1 {print $1}'
  else
    printf 'null'
  fi
}
while [ ! -e "$output/stop" ]; do
  timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  volume=$(df -k "$ram_root" | awk 'NR==2 {print $2, $3, $4}')
  read -r capacity used available <<< "$volume"
  inodes=$(df -i "$ram_root" | awk 'NR==2 {print $6}')
  jq -nc --arg timestamp "$timestamp" --argjson capacity "$capacity" --argjson used "$used" --argjson available "$available" --argjson inodes "$inodes" '{timestamp:$timestamp,kind:"volume",capacity_kib:$capacity,used_kib:$used,available_kib:$available,inodes_used:$inodes}' >> "$output/usage.jsonl"
  for fixture in "$ram_root"/run_codex_*grpc__grpc-go-2951_*; do
    [ -d "$fixture" ] || continue
    total=$(measure "$fixture")
    build=$(measure "$fixture/.scratch/go/build")
    temporary=$(measure "$fixture/.scratch/go/tmp")
    gopls=$(measure "$fixture/.scratch/go/gopls")
    private_modcache=false
    if [ -e "$fixture/.scratch/go/mod" ]; then private_modcache=true; fi
    jq -nc --arg timestamp "$timestamp" --arg workspace "$fixture" --argjson total "$total" --argjson build "$build" --argjson temporary "$temporary" --argjson gopls "$gopls" --argjson private_modcache "$private_modcache" '{timestamp:$timestamp,kind:"workspace",workspace:$workspace,allocated_kib:$total,build_kib:$build,temp_kib:$temporary,gopls_kib:$gopls,private_module_cache_present:$private_modcache}' >> "$output/usage.jsonl"
  done
  ps -ax -o pid=,comm= | while read -r pid command; do
    case "$command" in *codex*|*semedit*|*gopls*|*benchmark-harness*|*/compile|*/link) ;; *) continue ;; esac
    if paths=$(ps eww -p "$pid" -o command= | tr ' ' '\n' | awk '/^(GOCACHE|GOMODCACHE|GOTMPDIR|TMPDIR|GOPLSCACHE|GOBIN|GOPROXY|GOSUMDB|GOTOOLCHAIN)=/'); then
      case "$paths" in *"$ram_root"*) printf '%s process=%s executable=%s\n%s\n' "$timestamp" "$pid" "$command" "$paths" >> "$output/process-paths.log" ;; esac
    fi
  done
  if [ "$available" -lt 65536 ]; then printf '%s LOW_SPACE available_kib=%s\n' "$timestamp" "$available" >> "$output/warnings.log"; fi
  sleep 1
done
