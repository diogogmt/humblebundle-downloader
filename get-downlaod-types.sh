#!/bin/bash

# Get download types for humblebundle download for key (param 1)

[[ $# -lt 1 ]] && echo exit

KEYX="${1}"
URLX="https://www.humblebundle.com/api/v1/order/${KEYX}"

wget -c "${URLX}"

[[ -f "${KEYX}" ]] && {
	python -m json.tool "${KEYX}"| \
		grep \"name\"| \
		sed -e 's/^.*name\": //' -e 's/,$//'| \
		tr -d \"| \
		awk '{print $0}'|sort -u
}
