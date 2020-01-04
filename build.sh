#!/bin/sh
set -e

cd "$( cd "$(dirname "$0")"; pwd -P)"

exec 6>&1
exec 7>&2
exec > build.log 2>&1

trap "[ \$? -ne 0 ] && cat ../../build.log >&7; rm ../../build.log || true" EXIT

set -x

cd src/cryptocli

rm -rf cryptocli-*
rm build.log || true

GOPATH=$(pwd)/.go
GOPATH=${GOPATH} go get -v ./... || true
GOPATH=${GOPATH} go get -u -v ./... || true

(cd "$GOPATH/src/github.com/olivere/elastic/"; git fetch -t -f; git reset --hard origin/release-branch.v7)

GOPATH=${GOPATH} go build -o cryptocli-new .
VERSION=$(git log @ -1 --format='%H %d')

compile() {
	local GOOS=$1
	local GOARCH=$2
	local DEST="cryptocli-${GOOS}-${GOARCH}"
	local BIN="cryptocli"

	GOPATH=${GOPATH} \
	GOOS=${GOOS} \
	GOARCH=${GOARCH} \
	go build \
		-tags netgo \
		-ldflags "-X 'main.VERSION=${VERSION}' -extldflags '-static' -s -w"

	echo "Done compiling for ${GOOS} ${GOARCH}"

	[ "${GOOS}" = "windows" ] && BIN="${BIN}.exe"

	./cryptocli-new \
		-- fork zip - "${BIN}" \
		-- tee --pipe "-- dgst --algo sha256 -- hex --encode -- write-file --path \"${DEST}.zip.sha256\"" \
		-- write-file --path "${DEST}.zip"
}

compile darwin amd64
compile linux amd64
compile linux 386
compile linux arm64
compile windows amd64
compile windows 386
compile openbsd amd64
compile netbsd amd64

mv cryptocli-new cryptocli

for a in $(ls cryptocli*.zip.sha256);
do
	{ printf '%s - ' "${a}"; cat "${a}"; echo; } >&6
done
