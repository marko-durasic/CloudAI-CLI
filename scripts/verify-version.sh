#!/usr/bin/env bash
#
# Verify that the version ldflags in .goreleaser.yaml actually reach the built
# binary, before anything is published.
#
# Why the assertions are shaped this way. The first version of this script
# asserted by absence: it grepped for the literals "dev", "commit: none" and
# "built at: unknown" and passed if it found none of them. That has two ways to
# report success while proving nothing, both measured:
#
#   1. Renaming the placeholder defaults in internal/version/version.go — say
#      "dev" to "0.0.0-local", which is what you would do to make `go install`
#      builds sort — disarms it completely. With all three -X ldflags deleted
#      and all three placeholders renamed, it printed SUCCESS over
#      "cloudai version 0.0.0-local (commit: unset, built at: n/a)".
#   2. A binary that prints nothing matches none of the three patterns, so it
#      passed too, having tested nothing at all.
#
# So the assertions below are positive, and the one that matters compares
# against a value this script computes itself rather than against a pattern:
# the binary must report the commit `git rev-parse HEAD` returns. No rename of
# a placeholder can survive that, because there is nothing to spell correctly —
# there is a fact to match.
set -euo pipefail

fail() { echo "FAIL: $*" >&2; exit 1; }

# ---------------------------------------------------------------- the binary --
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

BINARY=""
for candidate in dist/*_${OS}_${ARCH}*/cloudai dist/*_${OS}_*/cloudai dist/*/cloudai; do
  if [ -x "$candidate" ] && "$candidate" version >/dev/null 2>&1; then
    BINARY="$candidate"
    break
  fi
done

if [ -z "$BINARY" ]; then
  fail "no executable cloudai binary found in dist/. Run 'goreleaser build --snapshot --clean' first."
fi

echo "Testing binary: $BINARY"
OUTPUT="$("$BINARY" version)"
echo "Binary output: $OUTPUT"

# ------------------------------------------------------- the expected commit --
# Computed here, not read from the binary. This is what turns the check from
# "does not look unset" into "reports the truth".
command -v git >/dev/null 2>&1 || fail "git is required: this check compares the binary's commit against the checkout's."
git rev-parse --git-dir >/dev/null 2>&1 || fail "not a git checkout: cannot determine the commit the binary should report."
EXPECTED_COMMIT="$(git rev-parse HEAD)"

# --------------------------------------------------------------- assertions --
# Structural first. Everything below reads a captured group, so if the output
# does not carry all three fields there is nothing to examine and the later
# assertions would pass vacuously on an empty string.
# Held in a variable: an inline pattern with parentheses is parsed by bash
# before the regex engine ever sees it.
# Deliberately permissive about what the fields contain: this assertion is only
# about the output having the three of them. Each value is judged by the
# assertion named for it below, so a failure points at the field that is wrong
# rather than at the shape.
FIELDS_RE='version[[:space:]]+([^[:space:]]+)[[:space:]]+\(commit:[[:space:]]*([^,]+),[[:space:]]*built[[:space:]]+at:[[:space:]]*([^)]+)\)'
if ! [[ "$OUTPUT" =~ $FIELDS_RE ]]; then
  fail "version output did not carry all three fields (version, commit, built at).
      got: ${OUTPUT}
      A binary that prints nothing, or prints a different shape, must not pass this check."
fi
GOT_VERSION="${BASH_REMATCH[1]}"
GOT_COMMIT="${BASH_REMATCH[2]}"
GOT_DATE="${BASH_REMATCH[3]}"

# Commit: compared against the value computed above. This is the assertion a
# placeholder rename cannot survive.
if [ "$GOT_COMMIT" != "$EXPECTED_COMMIT" ]; then
  fail "binary reports a commit that is not this checkout's HEAD.
      binary reports: ${GOT_COMMIT}
      git rev-parse HEAD: ${EXPECTED_COMMIT}
      The -X ldflag for Commit in .goreleaser.yaml did not reach the binary."
fi

# Version: shape, since the exact value depends on the tag or snapshot template.
SEMVER_RE='^v?[0-9]+\.[0-9]+\.[0-9]+([-+.][0-9A-Za-z.-]+)?$'
if ! [[ "$GOT_VERSION" =~ $SEMVER_RE ]]; then
  fail "binary reports a version that is not semver-shaped: ${GOT_VERSION}
      The -X ldflag for Version in .goreleaser.yaml did not reach the binary,
      or it carried a placeholder rather than a real version."
fi

# Date: must actually parse as RFC3339, not merely differ from a known string.
if ! date -u -d "$GOT_DATE" >/dev/null 2>&1 && ! date -u -j -f '%Y-%m-%dT%H:%M:%SZ' "$GOT_DATE" >/dev/null 2>&1; then
  fail "binary reports a build date that does not parse as a timestamp: ${GOT_DATE}
      The -X ldflag for Date in .goreleaser.yaml did not reach the binary."
fi

echo "SUCCESS: version ldflags reach the binary."
echo "  version: ${GOT_VERSION} (semver-shaped)"
echo "  commit:  ${GOT_COMMIT} (matches git rev-parse HEAD)"
echo "  date:    ${GOT_DATE} (parses as a timestamp)"
