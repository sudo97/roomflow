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
  # GitHub відхиляє файли > 100 МБ (сирі результати JMeter бувають > 1 ГБ): великі файли стискаються,
  # а якщо й після стиснення > 45 МБ — замінюються приміткою (зведення зберігаються окремими файлами)
  find "$WORK/repo/evidence/$LABEL" -type f -size +45M | while read -r f; do
    case "$f" in *.gz) z=$f ;; *) gzip -9 "$f"; z=$f.gz ;; esac
    if [ "$(stat -c %s "$z")" -gt 47185920 ]; then
      rm -f "$z"; echo "файл $(basename "$z") пропущено: завеликий для git навіть стиснутим" > "${z%.gz}.skipped.txt"
    fi
  done
  git -C "$WORK/repo" add evidence
  git -C "$WORK/repo" commit -qm "evidence: $LABEL" || { echo "nothing to publish"; exit 0; }
  if git -C "$WORK/repo" -c http.extraheader="$AUTH" push -q origin evidence; then echo "published evidence/$LABEL"; exit 0; fi
  echo "push rejected, retry $attempt"; sleep $((attempt * 3))
done
exit 1
