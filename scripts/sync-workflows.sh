#!/usr/bin/env bash
# Копіює підготовлені workflow з ci/workflows у .github/workflows.
# Потрібен, бо інтеграція, через яку генерується код, не має права змінювати .github/workflows.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git pull --ff-only
cp ci/workflows/*.yml .github/workflows/
git add .github/workflows
git commit -m "ci: sync workflows from ci/workflows" && git push
