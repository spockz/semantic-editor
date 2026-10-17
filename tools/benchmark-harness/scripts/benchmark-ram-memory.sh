#!/usr/bin/env bash
# Apply nested cgroup limits to host-available memory and preserve the lowest remaining budget.
ram_cgroup_v2_available() {
	local available=$1 root=$2 path=$3 current_dir
	current_dir="$root$path"
	while :; do
		if [ -r "$current_dir/memory.max" ] && [ -r "$current_dir/memory.current" ]; then
			limit=$(cat "$current_dir/memory.max"); used=$(cat "$current_dir/memory.current")
			if [ "$limit" != max ]; then left=$((limit-used)); [ "$left" -ge 0 ] || left=0; [ "$left" -lt "$available" ] && available=$left; fi
		fi
		[ "$current_dir" != "$root" ] || break
		current_dir=${current_dir%/*}; [ -n "$current_dir" ] || current_dir=$root
	done
	echo "$available"
}
ram_cgroup_v1_available() {
	local available=$1 root=$2 path=$3 current_dir
	current_dir="$root$path"
	while :; do
		if [ -r "$current_dir/memory.limit_in_bytes" ] && [ -r "$current_dir/memory.usage_in_bytes" ]; then
			limit=$(cat "$current_dir/memory.limit_in_bytes"); used=$(cat "$current_dir/memory.usage_in_bytes")
			if [ "$limit" -lt 9223372036854771712 ]; then left=$((limit-used)); [ "$left" -ge 0 ] || left=0; [ "$left" -lt "$available" ] && available=$left; fi
		fi
		[ "$current_dir" != "$root" ] || break
		current_dir=${current_dir%/*}; [ -n "$current_dir" ] || current_dir=$root
	done
	echo "$available"
}
