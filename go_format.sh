#!/usr/bin/env bash

unset \
  DIRS \
  FILES

DIRS=()
FILES=()

while read -r dir; do
  DIRS+=( "${dir}" )
done < <(go list -f "{{.Dir}}" ./... | grep -v '/vendor/')

for dir in "${DIRS[@]}"; do
  while read -r file; do
    #shellcheck disable=SC1001
    if [[ ! "${file}" =~ ^.+\/helpers\/assets\.go$ ]]; then
      FILES+=( "${file}" )
    fi
  done < <(find "${dir}" -type f -name '*.go')
done


diff <(gofmt -d "${FILES[@]}") <(echo -n)
