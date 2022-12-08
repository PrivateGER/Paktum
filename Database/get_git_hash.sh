#!/usr/bin/env bash

if [[ -z "${COMMIT_REF}" ]]; then
  HASH="$(git rev-parse HEAD)"
else
  # Running in CI, use the COMMIT_REF environment variable
  HASH="${COMMIT_REF}"
fi

echo "${HASH}" > git_hash.txt