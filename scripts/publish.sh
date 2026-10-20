#!/bin/sh
# Publish every package of this repository to a bnbloc deployment as
# platform content (the default content), signed with your publisher key:
#
#   scripts/publish.sh --gateway https://bnbloc.example.com --rollout auto
#
# Every package is packed and validated first; nothing is uploaded when one
# fails. A version already installed with the same bytes is kept, so a
# rerun is safe. Extra flags go to `bnbloc content publish` (--seckey f,
# --unsigned for a dev gateway, --publisher p, --token t). BNBLOC names the
# CLI to run (default: the platform version pinned in go.mod).
set -eu
cd "$(dirname "$0")/.."
${BNBLOC:-go tool bnbloc} content publish --all packages "$@"
