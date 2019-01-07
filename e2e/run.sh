#!/usr/bin/env sh
set -eu -o pipefail

# Run the e2e tests
cd ./e2e
./e2e.test
