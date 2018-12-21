#!/bin/sh
cd src/cryptocli
go build .

rm cryptocli-*

GOPATH=$(pwd)/.go
GOPATH=${GOPATH} go get -v ./...
GOPATH=${GOPATH} go get -u -v ./...

compile() {
	GOOS=$1
	GOARCH=$2
	NAME="cryptocli-${GOOS}-${GOARCH}"

	if [ "${GOOS}" = "windows" ]
	then
		NAME="${NAME}.exe"
	fi

	GOPATH=${GOPATH} GOOS=${GOOS} GOARCH=${GOARCH} go build -o "${NAME}" -tags netgo -ldflags "-w -extldflags \"-static\""
	echo "Done compiling for ${GOOS} ${GOARCH}"
	./cryptocli -- file --path "${NAME}" --read -- gzip -- file --path "${NAME}.gz" --write
}

compile darwin amd64
compile linux amd64
compile linux 386
compile windows amd64
compile windows 386
compile openbsd amd64
compile netbsd amd64
