#!/usr/bin/env bash
# Застосовує підготовлений патч із ci/patches/ до main і публікує коміт (запуск через workflow run-script).
# Використання: ci/apply-patch.sh ci/patches/NAME.patch "повідомлення коміту"
set -euo pipefail
P=$1; MSG=${2:-"chore: apply $P"}
git config user.name "github-actions[bot]"; git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git apply --index --whitespace=nowarn "$P"
git rm -q "$P"
git commit -q -m "$MSG" -m "Applied from $P by the run-script workflow."
git push origin HEAD:main
git log --oneline -1
