#!/bin/sh
set -e

TAG=$1

usage() {
	{
		echo "Usage of $0: <tag>"
	} >&2

	exit 2
}

[ "x${TAG}" = "x" ] && usage

exec 6>&1
exec 7>&2
exec > build.log 2>&1

trap "[ \$? -ne 0 ] && cat build.log >&7; rm build.log || true" EXIT

set -x

image_name="tehmoon/cryptocli:${TAG}"

docker rmi "${image_name}" || true

VERSION=$(git log @ -1 --format='%H %d')
VERSION=${VERSION} docker build --build-arg VERSION -t "${image_name}" .

docker push "${image_name}"

docker images "${image_name}" >&6
