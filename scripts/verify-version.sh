#!/usr/bin/env bash
set -euo pipefail

# Locate built binary in dist
BINARY=""
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

# Try host-matched binary first
for candidate in dist/*_${OS}_${ARCH}*/cloudai dist/*_${OS}_*/cloudai dist/*/cloudai; do
  if [ -x "$candidate" ]; then
    if "$candidate" version >/dev/null 2>&1; then
      BINARY="$candidate"
      break
    fi
  fi
done

if [ -z "$BINARY" ]; then
  echo "ERROR: No executable cloudai binary found in dist/ directory."
  echo "Ensure 'goreleaser build --snapshot' has been run."
  exit 1
fi

echo "Testing binary: $BINARY"
OUTPUT="$("$BINARY" version)"
echo "Binary output: $OUTPUT"

# Assert version is not 'dev'
if echo "$OUTPUT" | grep -E -q "version dev|\(dev\)|\sdev\s"; then
  echo "FAIL: Binary version reports 'dev'. ldflags -X for Version failed or missing."
  exit 1
fi

# Assert commit is not 'none'
if echo "$OUTPUT" | grep -E -q "commit: none"; then
  echo "FAIL: Binary commit reports 'none'. ldflags -X for Commit failed or missing."
  exit 1
fi

# Assert date is not 'unknown'
if echo "$OUTPUT" | grep -E -q "built at: unknown"; then
  echo "FAIL: Binary build date reports 'unknown'. ldflags -X for Date failed or missing."
  exit 1
fi

echo "SUCCESS: Version ldflags verified successfully in binary output."
