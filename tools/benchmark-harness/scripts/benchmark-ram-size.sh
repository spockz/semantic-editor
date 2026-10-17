#!/usr/bin/env bash
# Estimate RAM for one benchmark workspace using complete historical samples and explicit uncertainty.
set -euo pipefail
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_dir/benchmark-ram-memory.sh"

usage() { echo "usage: benchmark-ram-size.sh estimate [--repo-root DIR] [--concurrency N] [--safety-factor FLOAT] [--json] [workspace ...] | budget REQUESTED_BYTES" >&2; }
human() { awk -v n="$1" 'BEGIN {if (n>=1073741824) printf "%.2f GiB",n/1073741824; else if(n>=1048576) printf "%.0f MiB",n/1048576; else printf "%d bytes",n}'; }
available_bytes() {
	case "$(uname -s)" in
		Darwin)
			pagesize=$(sysctl -n hw.pagesize) || return 1
			vm_stat | awk -F: -v p="$pagesize" '/Pages free|Pages inactive|Pages speculative/ {gsub(/[^0-9]/,"",$2); n+=$2} END {printf "%.0f\n",n*p}' ;;
		Linux)
			avail=$(awk '/^MemAvailable:/ {printf "%.0f",$2*1024; found=1} END {if(!found) exit 1}' /proc/meminfo) || return 1
			cg_path=$(awk -F: '$1=="0" && $2=="" {print $3; exit}' /proc/self/cgroup)
			avail=$(ram_cgroup_v2_available "$avail" /sys/fs/cgroup "$cg_path")
			cg_path=$(awk -F: '$2 ~ /(^|,)memory(,|$)/ {print $3; exit}' /proc/self/cgroup)
			avail=$(ram_cgroup_v1_available "$avail" /sys/fs/cgroup/memory "$cg_path")
			echo "$avail" ;;
		*) echo 'unsupported operating system' >&2; return 2 ;;
	esac
}
physical_bytes() {
	case "$(uname -s)" in
		Darwin) sysctl -n hw.memsize ;;
		Linux) awk '/^MemTotal:/ {printf "%.0f\n",$2*1024; found=1} END {if(!found) exit 1}' /proc/meminfo ;;
		*) return 2 ;;
	esac
}
safe_bytes() {
	concurrency=${1:-1}; per_job=${BENCHMARK_RAM_PER_JOB_RESERVE_BYTES:-536870912}
	case "$concurrency" in ''|*[!0-9]*) echo 'concurrency must be an integer' >&2; return 2 ;; esac
	[ "$concurrency" -gt 0 ] && [ "$concurrency" -le 1024 ] || { echo 'concurrency must be between 1 and 1024' >&2; return 2; }
	concurrency=$((10#$concurrency))
	available=$(available_bytes) || return $?
	reserve=$((available/4)); minimum=${BENCHMARK_RAM_MIN_FREE_BYTES:-2147483648}
	minimum=$(awk -v n="$minimum" 'BEGIN {if(n !~ /^[0-9]+$/ || n>9007199254740991) exit 1; printf "%.0f",n}') || { echo 'minimum free memory must be an integer number of bytes within the supported range' >&2; return 2; }
	per_job=$(awk -v n="$per_job" 'BEGIN {if(n !~ /^[0-9]+$/ || n>9007199254740991) exit 1; printf "%.0f",n}') || { echo 'per-job reserve must be an integer number of bytes within the supported range' >&2; return 2; }
	[ "$reserve" -ge "$minimum" ] || reserve=$minimum
	[ "$per_job" -le $(((9007199254740991-reserve)/concurrency)) ] || { echo 'combined memory reserve exceeds supported range' >&2; return 2; }
	reserve=$((reserve + per_job * concurrency))
	[ "$available" -gt "$reserve" ] && echo $((available-reserve)) || echo 0
}
measure_directory() {
	directory=$1
	[ -d "$directory" ] || { echo '0 0'; return 0; }
	measurement=$(find "$directory" -type f -exec bash -c 'set -o pipefail; files=$#; total=$(du -kc "$@" | tail -n 1) || exit $?; set -- $total; printf "%s %s\\n" "$1" "$files"' _ {} + | awk '{s+=$1;c+=$2} END {printf "%.0f %.0f",s,c}') || { echo "could not measure $directory; it may be changing or unreadable (stop active benchmarks or select a stable sample)" >&2; return 1; }
	echo "$measurement"
}
budget() {
	requested=${1:-}; case "$requested" in ''|*[!0-9]*) echo 'requested bytes must be a positive integer' >&2; return 2 ;; esac
	[ "$requested" -gt 0 ] || return 2
	concurrency=${2:-1}
	physical=$(physical_bytes) || return $?
	available=$(available_bytes) || return $?
	safe=$(safe_bytes "$concurrency") || return $?
	if [ "$safe" -lt "$requested" ]; then echo "insufficient safe memory: request $(human "$requested"), safe budget $(human "$safe"), available $(human "$available")" >&2; return 4; fi
	reserved=$((available-safe))
	printf 'requested_bytes=%s\nphysical_bytes=%s\navailable_bytes=%s\nreserved_bytes=%s\nsafe_bytes=%s\n' "$requested" "$physical" "$available" "$reserved" "$safe"
}
estimate() {
	shift
	repo_root=${BENCHMARK_RAM_REPO_ROOT:-.}; concurrency=${BENCHMARK_RAM_CONCURRENCY:-1}; safety=${BENCHMARK_RAM_SAFETY_FACTOR:-1.75}; json=false
	while [ "$#" -gt 0 ]; do
		case "$1" in
			--repo-root) [ "$#" -ge 2 ] || { usage; return 2; }; repo_root=$2; shift 2 ;;
			--concurrency) [ "$#" -ge 2 ] || { usage; return 2; }; concurrency=$2; shift 2 ;;
			--safety-factor) [ "$#" -ge 2 ] || { usage; return 2; }; safety=$2; shift 2 ;;
			--json) json=true; shift ;;
			--*) echo "unknown option: $1" >&2; return 2 ;;
			*) break ;;
		esac
	done
	case "$concurrency" in ''|*[!0-9]*) echo 'concurrency must be a positive integer' >&2; return 2 ;; esac
	[ "$concurrency" -gt 0 ] && [ "$concurrency" -le 1024 ] || { echo 'concurrency must be between 1 and 1024' >&2; return 2; }
	concurrency=$((10#$concurrency))
	factor_pct=$(awk -v x="$safety" 'BEGIN {if(x !~ /^[0-9]+([.][0-9]+)?$/ || x<1 || x>10) exit 1; printf "%.0f",x*100}') || { echo 'safety factor must be between 1 and 10' >&2; return 2; }
	safety_print=$(awk -v p="$factor_pct" 'BEGIN {printf "%.2f",p/100}')
	root=${BENCHMARK_RAM_SAMPLE_ROOT:-$repo_root/.scratch/benchmarks}
	if [ "$#" -eq 0 ]; then
		for candidate in "$root"/run_*; do
			[ -d "$candidate" ] || continue
			complete=false
			for marker in go.mod package.json Cargo.toml pyproject.toml pom.xml Makefile; do [ ! -f "$candidate/$marker" ] || { complete=true; break; }; done
			if [ "$complete" = true ]; then set -- "$@" "$candidate"
			elif find "$candidate" \( -type d \( -name evidence -o -name results -o -name transcripts \) -prune \) -o -type f -print -quit | grep -q .; then incomplete=1
			fi
		done
	fi
	incomplete=${incomplete:-0}; partial=0
	if [ "$#" -eq 0 ]; then
		cache_root=${BENCHMARK_RAM_REPO_CACHE:-$repo_root/.scratch/benchmarks/cache/repos}
		if [ -d "$cache_root" ]; then
			for repo in "$cache_root"/*; do
				[ -d "$repo" ] || continue
				root_manifest=false
				for marker in go.mod package.json Cargo.toml pyproject.toml pom.xml Makefile; do [ ! -f "$repo/$marker" ] || { root_manifest=true; break; }; done
				if [ "$root_manifest" = true ]; then set -- "$@" "$repo"
				else
					for candidate in "$repo"/*; do
						[ -d "$candidate" ] || continue
						for marker in go.mod package.json Cargo.toml pyproject.toml pom.xml Makefile; do [ ! -f "$candidate/$marker" ] || { set -- "$@" "$candidate"; break; }; done
					done
				fi
			done
			[ "$#" -eq 0 ] || { partial=1; incomplete=1; }
		fi
	fi
	[ "$#" -gt 0 ] || { echo "no measured workspace samples or cached repositories found under $root; auto sizing is uncertain" >&2; return 3; }
	max_bytes=0; count=0; max_files=0; total_files=0; samples=
	max_build=0; max_tmp=0; max_gopls=0; max_opencode=0; build_files=0; tmp_files=0; gopls_files=0; opencode_files=0
	for workspace do
		[ -d "$workspace" ] && [ ! -L "$workspace" ] || { echo "invalid sample directory: $workspace" >&2; return 2; }
		complete=false
		for marker in go.mod package.json Cargo.toml pyproject.toml pom.xml Makefile; do [ ! -f "$workspace/$marker" ] || { complete=true; break; }; done
		[ "$complete" = true ] || incomplete=1
		if [ -d "$workspace/.scratch/go" ]; then
			[ -d "$workspace/.scratch/go/build" ] && [ -d "$workspace/.scratch/go/tmp" ] || incomplete=1
		else incomplete=1; fi
		measurement=$(find "$workspace" \
			\( -type d \( -path "$workspace/.scratch/go/mod" -o -path "$workspace/.scratch/go/pkg/mod" -o -path "$workspace/.scratch/cache" -o -path "$workspace/.scratch/go/cache" -o -path "$workspace/.scratch/benchmarks/cache" -o -path "$workspace/.scratch/benchmarks/results" -o -path "$workspace/.scratch/benchmarks/transcripts" -o -path "$workspace/.scratch/benchmarks/evidence" \) -prune \) \
			-o -type f -exec bash -c 'set -o pipefail; files=$#; total=$(du -kc "$@" | tail -n 1) || exit $?; set -- $total; printf "%s %s\\n" "$1" "$files"' _ {} + | awk '{total += $1; files += $2} END {printf "%.0f %.0f\n", total, files}') || { echo "could not measure $workspace; it may be changing or unreadable (stop active benchmarks or select a stable sample)" >&2; return 1; }
		kb=${measurement%% *}; sample_files=${measurement#* }
		case "$kb" in ''|*[!0-9]*) echo "could not measure sample: $workspace" >&2; return 1 ;; esac
		bytes=$((kb*1024)); count=$((count+1)); total_files=$((total_files+sample_files)); [ "$sample_files" -le "$max_files" ] || max_files=$sample_files; samples="$samples $bytes"
		component=$(measure_directory "$workspace/.scratch/go/build") || return $?
		component_bytes=$(( ${component%% *} * 1024 )); [ "$component_bytes" -le "$max_build" ] || { max_build=$component_bytes; build_files=${component#* }; }
		component=$(measure_directory "$workspace/.scratch/go/tmp") || return $?
		component_bytes=$(( ${component%% *} * 1024 )); [ "$component_bytes" -le "$max_tmp" ] || { max_tmp=$component_bytes; tmp_files=${component#* }; }
		component=$(measure_directory "$workspace/.scratch/go/gopls") || return $?
		component_bytes=$(( ${component%% *} * 1024 )); [ "$component_bytes" -le "$max_gopls" ] || { max_gopls=$component_bytes; gopls_files=${component#* }; }
		component=$(measure_directory "$workspace/.scratch/bin/opencode") || return $?
		component_bytes=$(( ${component%% *} * 1024 )); [ "$component_bytes" -le "$max_opencode" ] || { max_opencode=$component_bytes; opencode_files=${component#* }; }
		if [ "$bytes" -gt "$max_bytes" ]; then
			max_bytes=$bytes
		fi
	done
	[ "$max_bytes" -gt 0 ] || { echo 'samples contain no measurable workspace files; auto sizing is uncertain' >&2; return 3; }
	overhead=${BENCHMARK_RAM_FIXED_OVERHEAD_BYTES:-1073741824}
	overhead=$(awk -v n="$overhead" 'BEGIN {if(n !~ /^[0-9]+$/ || n>9007199254740991) exit 1; printf "%.0f",n}') || { echo 'fixed overhead must be an integer number of bytes within the supported range' >&2; return 2; }
	requested=$(awk -v m="$max_bytes" -v c="$concurrency" -v p="$factor_pct" -v o="$overhead" 'BEGIN {n=m*c*p/100+o; if(n<=0 || n>9007199254740991) exit 1; printf "%.0f",int((n+1073741823)/1073741824)*1073741824}') || { echo 'requested size overflows supported capacity' >&2; return 2; }
	estimated_inodes=$(awk -v n="$max_files" -v c="$concurrency" -v p="$factor_pct" 'BEGIN {v=n*c*p/100+64; if(v>9007199254740991) exit 1; printf "%.0f",v}') || { echo 'estimated inode count overflows supported capacity' >&2; return 2; }
	typical=$(printf '%s\n' $samples | awk '{n++; a[n]=$1; sum+=$1} END {for(i=1;i<=n;i++) for(j=i+1;j<=n;j++) if(a[j]<a[i]) {t=a[i];a[i]=a[j];a[j]=t} if(n%2) med=a[(n+1)/2]; else med=(a[n/2]+a[n/2+1])/2; printf "%.0f %.0f\n",med,sum/n}')
	median_bytes=${typical%% *}; mean_bytes=${typical#* }
	safe=$(safe_bytes "$concurrency") || return $?
	available=$(available_bytes) || return $?; physical=$(physical_bytes) || return $?; reserved=$((available-safe)); [ "$reserved" -ge 0 ] || reserved=$available
	if [ "$requested" -le "$safe" ]; then budget_status=within; else budget_status=insufficient; fi
	if [ "$incomplete" -eq 1 ] || [ "$partial" -eq 1 ]; then confidence=uncertain; else confidence=measured; fi
	if [ "$json" = true ]; then
		printf '{"sample_count":%s,"max_sample_files":%s,"total_sample_files":%s,"estimated_inodes":%s,"sample_max_bytes":%s,"typical_median_bytes":%s,"typical_mean_bytes":%s,"build_bytes":%s,"build_files":%s,"tmp_bytes":%s,"tmp_files":%s,"gopls_bytes":%s,"gopls_files":%s,"opencode_bytes":%s,"opencode_files":%s,"physical_bytes":%s,"available_bytes":%s,"reserved_bytes":%s,"safe_budget_bytes":%s,"budget_status":"%s","concurrency":%s,"safety_factor":%s,"fixed_overhead_bytes":%s,"requested_bytes":%s,"confidence":"%s","partial_fallback":%s}\n' "$count" "$max_files" "$total_files" "$estimated_inodes" "$max_bytes" "$median_bytes" "$mean_bytes" "$max_build" "$build_files" "$max_tmp" "$tmp_files" "$max_gopls" "$gopls_files" "$max_opencode" "$opencode_files" "$physical" "$available" "$reserved" "$safe" "$budget_status" "$concurrency" "$safety_print" "$overhead" "$requested" "$confidence" "$partial"
	else
		printf 'sample_count=%s\nmax_sample_files=%s\ntotal_sample_files=%s\nestimated_inodes=%s\nsample_max_bytes=%s\nsample_max=%s\ntypical_median=%s\ntypical_mean=%s\nbuild_bytes=%s\nbuild_files=%s\ntmp_bytes=%s\ntmp_files=%s\ngopls_bytes=%s\ngopls_files=%s\nopencode_bytes=%s\nopencode_files=%s\nphysical_bytes=%s\navailable_bytes=%s\nreserved_bytes=%s\nsafe_budget_bytes=%s\nbudget_status=%s\nconcurrency=%s\nsafety_factor=%s\nfixed_overhead_bytes=%s\nrequested_bytes=%s\nrequested=%s\nconfidence=%s\nexclusions=.scratch/go/mod,.scratch/go/pkg/mod,.scratch/cache,.scratch/go/cache,.scratch/benchmarks/cache,.scratch/benchmarks/results,.scratch/benchmarks/transcripts,.scratch/benchmarks/evidence\nswap_warning=tmpfs may use kernel swap under memory pressure\n' "$count" "$max_files" "$total_files" "$estimated_inodes" "$max_bytes" "$(human "$max_bytes")" "$(human "$median_bytes")" "$(human "$mean_bytes")" "$max_build" "$build_files" "$max_tmp" "$tmp_files" "$max_gopls" "$gopls_files" "$max_opencode" "$opencode_files" "$physical" "$available" "$reserved" "$safe" "$budget_status" "$concurrency" "$safety_print" "$overhead" "$requested" "$(human "$requested")" "$confidence"
		[ "$partial" -eq 0 ] || echo 'fallback used cached repository materialization; estimate is partial'
		[ "$incomplete" -eq 0 ] || echo 'some workspace samples were incomplete; automatic mounting requires complete samples'
	fi
}
case "${1:-}" in estimate) estimate "$@" ;; budget) shift; budget "$@" ;; *) usage; exit 2 ;; esac
