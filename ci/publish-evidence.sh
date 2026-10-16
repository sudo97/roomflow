#!/usr/bin/env bash
# Публікує каталог /tmp/evidence у гілку evidence під міткою $1 (з повторами при конкурентних push).
set -euo pipefail
LABEL=$1; SRC=${2:-/tmp/evidence}
WORK=$(mktemp -d)
git config --global user.name "github-actions"; git config --global user.email "actions@github.com"
URL=$(git config --get remote.origin.url)
AUTH=$(git config --get http.https://github.com/.extraheader || true)
for attempt in 1 2 3 4 5; do
  rm -rf "$WORK/repo"
  if git -c http.extraheader="$AUTH" clone -q --depth 1 --branch evidence "$URL" "$WORK/repo" 2>/dev/null; then :; else
    git init -q "$WORK/repo"; git -C "$WORK/repo" checkout -q --orphan evidence; git -C "$WORK/repo" remote add origin "$URL"
  fi
  mkdir -p "$WORK/repo/evidence/$LABEL"; cp -r "$SRC"/* "$WORK/repo/evidence/$LABEL/"
  git -C "$WORK/repo" add evidence
  git -C "$WORK/repo" commit -qm "evidence: $LABEL" || { echo "nothing to publish"; exit 0; }
  if git -C "$WORK/repo" -c http.extraheader="$AUTH" push -q origin evidence; then echo "published evidence/$LABEL"; exit 0; fi
  echo "push rejected, retry $attempt"; sleep $((attempt * 3))
done
exit 1
