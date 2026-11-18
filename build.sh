#!/usr/bin/env sh
set -eu

for target in linux/amd64 linux/arm64 darwin/arm64 windows/amd64; do
	os=${target%/*}
	arch=${target#*/}
	out="privatesync-${os}-${arch}"
	[ "$os" = windows ] && out="${out}.exe"
	echo "building ${out}"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$out" .
done
