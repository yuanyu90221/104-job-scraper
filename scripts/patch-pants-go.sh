#!/usr/bin/env bash
# Workaround: Pants Go backend sets GOEXPERIMENT=coverageredesign for Go>=1.20,
# but Go>=1.24 removed that experiment, causing:
#   unknown GOEXPERIMENT coverageredesign
#
# This script patches Pants's vendored sdk.py to only set the experiment for
# Go 1.20-1.23, allowing Pants to work with Go 1.24/1.25+.
#
# Idempotent; re-run after Pants version upgrades (which re-download the venv).
# Remove once the upstream fix is released.
set -euo pipefail

found=0
while IFS= read -r f; do
  found=1
  if grep -q 'is_compatible_version("1.20") and not goroot.is_compatible_version("1.24")' "$f"; then
    echo "Already patched, skipping: $f"
    continue
  fi
  sed -i \
    's/if goroot.is_compatible_version("1.20"):/if goroot.is_compatible_version("1.20") and not goroot.is_compatible_version("1.24"):/' \
    "$f"
  echo "Patched: $f"
done < <(find "${HOME}/.cache/nce" -type f -path "*backend/go/util_rules/sdk.py" 2>/dev/null)

if [ "$found" -eq 0 ]; then
  echo "Pants sdk.py not found. Run 'pants version' first to download the venv." >&2
  exit 1
fi
