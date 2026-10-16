#!/usr/bin/env bash
# ЛР12, інд. завдання 10: поведінковий аналіз коду ("Your Code as a Crime Scene") інструментом Charlie
# (github.com/sudo97/charlie): hotspots (складність x частота змін), sum of coupling, пари з часовою
# зв'язаністю та архітектурні групи-кандидати в bounded contexts. Потребує повної історії (fetch-depth: 0).
source "$(dirname "$0")/../lib/common.sh"
E=$EVIDENCE/charlie; mkdir -p "$E"
CHARLIE_REF=${CHARLIE_REF:-334b16fcf837f4c08f018fcdcb4b4805658f01c3}
git clone -q https://github.com/sudo97/charlie.git /tmp/charlie && git -C /tmp/charlie checkout -q "$CHARLIE_REF"
(cd /tmp/charlie && npm ci --silent && npm run build --silent) > /dev/null
CHARLIE="node /tmp/charlie/dist/cli/cli/index.js"
{
  echo "commits: $(git rev-list --count HEAD), first: $(git log --reverse --format=%cI | head -1), last: $(git log -1 --format=%cI)"
  echo "authors: $(git log --format=%aN | sort -u | tr '\n' ',' )"
} | tee "$E/history.txt"
$CHARLIE --json . > "$E/charlie.json" 2>/dev/null
$CHARLIE --json --all . > "$E/charlie-all.json" 2>/dev/null
$CHARLIE . > "$E/charlie.log" 2>&1 && mv charlie-report.html "$E/"
python3 - "$E/charlie.json" "$E/charlie-all.json" <<'PY' | tee "$E/summary.md"
import json, re, sys
d = json.load(open(sys.argv[1]))
groups = json.load(open(".charlie.config.json"))["architecturalGroups"]
def group(f):
    return next((g for p, g in groups.items() if re.search(p, f)), "інше")
print("| Файл | Складність | Ревізій | Оцінка | Група |\n|---|---|---|---|---|")
for h in d["hotspots"][:15]:
    print(f'| {h["file"]} | {h["complexity"]} | {h["revisions"]} | {h["complexity"] * h["revisions"]} | {group(h["file"])} |')
print("\n| Файл | SOC | Найсильніше пов'язані (частка спільних змін) |\n|---|---|---|")
for c in d["coupling"][:12]:
    pairs = ", ".join(f'{p["file"]} ({p["percentage"]:.0%})' for p in c["coupledFiles"][:3])
    print(f'| {c["file"]} | {c["soc"]} | {pairs} |')
full = json.load(open(sys.argv[2]))
cross = {}
for c in full["coupling"]:
    for p in c["coupledFiles"]:
        a, b = sorted((group(c["file"]), group(p["file"])))
        if a != b and p["percentage"] >= 0.5:
            cross[(a, b)] = cross.get((a, b), 0) + 1
print("\nЧасова зв'язаність між групами (пар файлів із ≥50% спільних змін):")
for (a, b), n in sorted(cross.items(), key=lambda x: -x[1])[:12]:
    print(f"- {a} ↔ {b}: {n}")
PY
playwright_setup
NODE_PATH=/tmp/pw/node_modules node - "$E" <<'JS'
const { createRequire } = require('module');
const r = createRequire('/tmp/pw/package.json');
const { chromium } = r('playwright');
(async () => {
  const out = process.argv[2];
  const b = await chromium.launch();
  const p = await b.newPage({ viewport: { width: 1500, height: 1000 } });
  await p.goto('file://' + out + '/charlie-report.html');
  await p.waitForTimeout(3000);
  await p.screenshot({ path: out + '/charlie-hotspots.png' });
  for (const name of ['Coupling', 'Grouped', 'Architectural']) {
    const link = p.getByText(name, { exact: false }).first();
    if (await link.count()) { await link.click(); await p.waitForTimeout(2500); await p.screenshot({ path: `${out}/charlie-${name.toLowerCase()}.png` }); }
  }
  await b.close();
})();
JS
section "Charlie: hotspots і зв'язаність" "$E/summary.md"
