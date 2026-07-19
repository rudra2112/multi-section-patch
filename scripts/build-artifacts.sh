#!/bin/sh
set -eu

export LC_ALL=C
umask 022

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root=$(CDPATH= cd -- "$script_dir/.." && pwd)
output_root=${OUTPUT_ROOT:-"$root/skills/multi-section-patch"}
targets=${TARGETS:-"darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64"}
go_toolchain=go1.26.1
maximum_binary_bytes=3145728
maximum_total_bytes=18874368

case "$output_root" in
	/*) ;;
	*) output_root="$root/$output_root" ;;
esac

if [ -z "$targets" ]; then
	echo "TARGETS must contain at least one supported OS/architecture pair" >&2
	exit 2
fi

if ! cmp -s "$root/LICENSE" "$root/skills/multi-section-patch/LICENSE.txt"; then
	echo "skills/multi-section-patch/LICENSE.txt must match the repository LICENSE" >&2
	exit 1
fi

mkdir -p "$output_root/scripts"
checksums="$output_root/SHA256SUMS"
: >"$checksums"
total_bytes=0

for target in $targets; do
	case "$target" in
		linux/amd64 | linux/arm64 | darwin/amd64 | darwin/arm64 | windows/amd64 | windows/arm64) ;;
		*)
			echo "unsupported target: $target" >&2
			exit 2
			;;
	esac

	goos=${target%/*}
	goarch=${target#*/}
	name="multi-section-patch-$goos-$goarch"
	if [ "$goos" = windows ]; then
		name="$name.exe"
	fi

	relative="scripts/$name"
	output="$output_root/$relative"

	(
		cd "$root"
		CGO_ENABLED=0 GOENV=off GOOS="$goos" GOARCH="$goarch" GOTOOLCHAIN="$go_toolchain" \
			go build -mod=readonly -trimpath -buildvcs=false \
			-ldflags="-s -w -buildid=" \
			-o "$output" ./cmd/multi-section-patch
	)

	size=$(wc -c <"$output")
	size=${size##* }
	if [ "$size" -gt "$maximum_binary_bytes" ]; then
		echo "$relative is $size bytes; the release limit is $maximum_binary_bytes" >&2
		exit 1
	fi
	total_bytes=$((total_bytes + size))

	if command -v sha256sum >/dev/null 2>&1; then
		hash=$(sha256sum "$output")
	elif command -v shasum >/dev/null 2>&1; then
		hash=$(shasum -a 256 "$output")
	else
		echo "sha256sum or shasum is required" >&2
		exit 1
	fi
	hash=${hash%% *}
	printf '%s  %s\n' "$hash" "$relative" >>"$checksums"
done

if [ "$total_bytes" -gt "$maximum_total_bytes" ]; then
	echo "release binaries total $total_bytes bytes; the limit is $maximum_total_bytes" >&2
	exit 1
fi

printf 'Built %s\n' "$output_root"
