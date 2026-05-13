#!/usr/bin/env bash
#
# This script is called by the GitHub actions runner,
# see .github/workflows/deploy.yml.

set -e

COMMIT="${SSH_ORIGINAL_COMMAND:-HEAD}"
if [[ ! "$COMMIT" =~ ^[a-f0-9]{40}$ ]]; then
    echo "Invalid commit SHA"
    exit 1
fi

nix-shell -p figlet --run "figlet -f smslant npipe"
echo "deploying commit $COMMIT"

set -x

cd /home/ekzyis/npipe
git fetch
git switch --detach "$COMMIT"
go build -o npipe
tmux kill-session -t npipe || true
tmux new-session -d -s npipe './npipe 3333'
