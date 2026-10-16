#!/usr/bin/env python3
"""Зводить результати SAST-інструментів у порівняльну таблицю (Markdown).

Еталон (ground truth) — навмисно внесені вразливості:
  legacy  (32a122c): жорстко закодований пароль адміністратора, дві SQL-ін'єкції у sql.go
  demo    (PR #1):   SQL-ін'єкція в export.go
  current (main):    відомих вразливостей немає
"""
import glob, json, os, sys

D = sys.argv[1] if len(sys.argv) > 1 else "/tmp/evidence/sast"
TRUTH = {
    "legacy": [("cmd/roomflow/main.go", 14, "hardcoded credential"), ("internal/store/sql.go", 17, "SQL injection"), ("internal/store/sql.go", 35, "SQL injection")],
    "demo": [("internal/store/export.go", 11, "SQL injection")],
    "current": [],
}


def rel(path):
    for marker in ("/src/", "src-legacy/", "src-demo/", "src-current/"):
        if marker in path:
            path = path.split(marker, 1)[1]
    return path.lstrip("./")


def load(name, tool):
    p = os.path.join(D, f"{name}-{tool}")
    if tool == "gosec":
        data = json.load(open(p + ".json")) if os.path.exists(p + ".json") else {}
        return [(rel(i["file"]), int(i["line"]), i["rule_id"]) for i in data.get("Issues") or []]
    if tool == "semgrep":
        data = json.load(open(p + ".json")) if os.path.exists(p + ".json") else {}
        return [(rel(r["path"]), r["start"]["line"], r["check_id"].split(".")[-1]) for r in data.get("results", [])]
    out = []
    for f in glob.glob(os.path.join(D, f"{name}-codeql", "*.sarif")):
        for run in json.load(open(f)).get("runs", []):
            for r in run.get("results", []):
                loc = r["locations"][0]["physicalLocation"]
                out.append((rel(loc["artifactLocation"]["uri"]), loc["region"]["startLine"], r["ruleId"]))
    return out


def detected(findings, file, line):
    return any(f == file and abs(l - line) <= 3 for f, l, _ in findings)


tools = ["gosec", "semgrep", "codeql"]
rows, totals = [], {t: {"tp": 0, "fn": 0, "other": 0} for t in tools}
for name, truth in TRUTH.items():
    for t in tools:
        fs = load(name, t)
        tp = sum(detected(fs, f, l) for f, l, _ in truth)
        matched = sum(1 for f, l, _ in fs if any(f == tf and abs(l - tl) <= 3 for tf, tl, _ in truth))
        totals[t]["tp"] += tp
        totals[t]["fn"] += len(truth) - tp
        totals[t]["other"] += len(fs) - matched
        rows.append((name, t, len(fs), f"{tp}/{len(truth)}", len(fs) - matched, sorted({r for _, _, r in fs})[:8]))

print("| Версія | Інструмент | Знахідок | Виявлено еталонних | Інших знахідок | Правила |")
print("|---|---|---|---|---|---|")
for r in rows:
    print(f"| {r[0]} | {r[1]} | {r[2]} | {r[3]} | {r[4]} | {', '.join(r[5])} |")
print()
print("| Інструмент | Виявлено (TP) | Пропущено (FN) | Інші знахідки |")
print("|---|---|---|---|")
for t, v in totals.items():
    print(f"| {t} | {v['tp']} | {v['fn']} | {v['other']} |")
