#!/usr/bin/env bash
for f in $(find . -name go.mod)
do (cd "$(dirname "$f")" || exit; go mod tidy)
done