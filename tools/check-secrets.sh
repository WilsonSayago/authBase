#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GITLEAKS="${ROOT}/.tmp/bin/gitleaks"
SCAN_DIR="$(mktemp -d "${TMPDIR:-/tmp}/authbase-gitleaks.XXXXXX")"
trap 'rm -rf "${SCAN_DIR}"' EXIT

if [[ ! -x "${GITLEAKS}" ]]; then
  echo "gitleaks is missing; run 'make tools' first" >&2
  exit 127
fi

"${GITLEAKS}" detect \
  --source "${ROOT}" \
  --config "${ROOT}/.gitleaks.toml" \
  --no-banner \
  --redact

while IFS= read -r -d '' path; do
  [[ -f "${ROOT}/${path}" ]] || continue
  mkdir -p "${SCAN_DIR}/$(dirname "${path}")"
  cp "${ROOT}/${path}" "${SCAN_DIR}/${path}"
done < <(git -C "${ROOT}" ls-files -z --cached --others --exclude-standard)

exec "${GITLEAKS}" detect \
  --source "${SCAN_DIR}" \
  --config "${ROOT}/.gitleaks.toml" \
  --no-git \
  --no-banner \
  --redact
