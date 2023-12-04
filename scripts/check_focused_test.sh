#!/usr/bin/env bash

# When an integration test is focused, no other test is run.
# This can be useful in development, but allowing a test to be
# checked in to main while focused means other tests are not
# actually run in CI.

set -o pipefail

FOCUSED=$(grep -rn "\.Focus" integration/)
if [ -n "${FOCUSED}" ]; then
    echo -e "Focused tests should not be checked in:\n\n${FOCUSED}"
    exit 1
fi
