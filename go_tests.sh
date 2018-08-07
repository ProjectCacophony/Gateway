#!/usr/bin/env bash
PKGS=()
while read -r pkg; do
  PKGS+=( "${pkg}" )
done < <(go list ./... | grep -v '/vendor/')

go test -v -race "${PKGS}"
