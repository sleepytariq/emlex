#!/usr/bin/env bash

if [[ -z "${GITHUB_TOKEN}" ]]; then
    echo "GITHUB_TOKEN is not set"
    exit 1
fi

VERSION=$(grep '^const version' main.go | cut -d '"' -f2)

if git tag --list | grep -q "${VERSION}"; then
    echo "version found in tags"
    exit 1
fi

git tag -a "v${VERSION}" -m "${VERSION}"
git push origin "v${VERSION}"

goreleaser release --clean
