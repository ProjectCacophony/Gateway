#!/usr/bin/env bash

unset \
  DIRS \
  FILES

DIRS=()
FILES=()

while read -r dir; do
  DIRS+=( "${dir}" )
done < <(go list -f {{.Dir}} ./... | grep -v '/vendor/')

for dir in "${DIRS[@]}"; do
  find "${dir}" -type f -name '*.go' | while read -r file; do
    if [[ ! "${file}" =~ ^.+\/helpers\/assets\.go$ ]]; then
      FILES+=( "${file}" )
    fi
  done
done


diff <(gofmt -d "${FILES[@]}") <(echo -n)