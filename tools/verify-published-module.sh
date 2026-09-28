#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
if [[ -z "${VERSION}" ]]; then
  echo "usage: $0 <published-version>  # example: v4.3.1" >&2
  echo "v4.0.0 is not a resolvable published module version" >&2
  exit 2
fi
if [[ "${VERSION}" == "v4.0.0" ]]; then
  echo "v4.0.0 is not a resolvable published module version; use v4.0.1 or later" >&2
  exit 2
fi
MODULE="github.com/WilsonSayago/authBase/v4"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/authbase-consumer.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

export GOWORK=off
export GOPROXY=https://proxy.golang.org
export GOSUMDB=sum.golang.org
export GOPRIVATE=
export GONOSUMDB=

cp "${ROOT}"/examples/quickstart/*.go "${TMP_DIR}/"
cd "${TMP_DIR}"

go mod init example.com/authbase-consumer-smoke
go get "${MODULE}@${VERSION}"
go mod tidy
go test -count=1 ./...

MODULE_DIR="$(go list -m -f '{{.Dir}}' "${MODULE}")"
MODULE_CACHE="$(go env GOMODCACHE)"

case "${MODULE_DIR}" in
  "${MODULE_CACHE}"/*) ;;
  *)
    echo "${MODULE}@${VERSION} resolved outside the module cache: ${MODULE_DIR}" >&2
    exit 1
    ;;
esac

case "${MODULE_DIR}" in
  "${ROOT}"|"${ROOT}"/*)
    echo "${MODULE}@${VERSION} resolved to the authBase workspace" >&2
    exit 1
    ;;
esac

go list -m -json "${MODULE}"
echo "published consumer OK: ${MODULE}@${VERSION}"
