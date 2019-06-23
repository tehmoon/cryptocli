#!/bin/sh
set -e

printf '### Cryptocli\n\n```\n'
usage=$(./src/cryptocli/cryptocli -h 2>&1 || true)

echo "${usage}"
printf '```\n\n### Modules\n\n'
mods=$(echo "${usage}" | jq -nrR 'inputs | select(test("^\t[a-z0-9-]+:")) | split(":")[0][1:]')

for mod in $mods; do
	printf '```\n'
	./src/cryptocli/cryptocli -- "${mod}" -h 2>&1 || true
	printf '```\n'
done
