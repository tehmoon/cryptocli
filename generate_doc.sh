#!/bin/sh

{ printf '### Cryptocli\n\n```\n'; ./src/cryptocli/cryptocli -h 2>&1; printf '```\n\n### Modules\n\n'; mods=$(./src/cryptocli/cryptocli -h 2>&1 | grep -E '^\t([a-z0-9\-]+):' | cut -d ':' -f 1); for mod in $(echo $mods); do printf '```\n' ; ./src/cryptocli/cryptocli --  "${mod}" -h 2>&1 ; printf '```\n';done ; }
