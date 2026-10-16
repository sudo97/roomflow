#!/usr/bin/env python3
"""Зведення результатів k6 і JMeter (ЛР9) у таблиці Markdown."""
import csv, glob, json, os, statistics, sys

E = sys.argv[1]


def k6(path):
    m = json.load(open(path))["metrics"]
    d = m.get("http_req_duration", {})
    return {
        "rps": m.get("http_reqs", {}).get("rate", 0), "reqs": m.get("http_reqs", {}).get("count", 0),
        "avg": d.get("avg", 0), "p50": d.get("med", 0), "p95": d.get("p(95)", 0), "p99": d.get("p(99)", 0), "max": d.get("max", 0),
        "err": m.get("http_req_failed", {}).get("value", 0),
    }


def pct(values, p):
    values = sorted(values)
    return values[min(len(values) - 1, int(len(values) * p))] if values else 0


def jtl(path):
    rows = list(csv.DictReader(open(path)))
    if not rows:
        return None
    el = [int(r["elapsed"]) for r in rows]
    ts = [int(r["timeStamp"]) for r in rows]
    dur = (max(ts) - min(ts)) / 1000 or 1
    return {"rps": len(rows) / dur, "reqs": len(rows), "avg": statistics.mean(el), "p50": pct(el, .5), "p95": pct(el, .95),
            "p99": pct(el, .99), "max": max(el), "err": sum(r["success"] != "true" for r in rows) / len(rows)}


def row(name, r):
    return f"| {name} | {r['reqs']:.0f} | {r['rps']:.0f} | {r['avg']:.1f} | {r['p50']:.1f} | {r['p95']:.1f} | {r['p99']:.1f} | {r['max']:.0f} | {r['err'] * 100:.2f}% |"


print("#### k6: базові показники за кінцевими точками (мс)\n")
print("| Варіант / точка | Запитів | RPS | avg | p50 | p95 | p99 | max | Помилки |")
print("|---|---|---|---|---|---|---|---|---|")
for label in ["before", "after", "cache"]:
    for ep in ["health", "availability", "booking", "stress"]:
        p = os.path.join(E, "k6", f"{label}-{ep}.json")
        if os.path.exists(p):
            print(row(f"{label} / {ep}", k6(p)))
print("\n#### JMeter (до оптимізації)\n")
print("| Точка | Запитів | RPS | avg | p50 | p95 | p99 | max | Помилки |")
print("|---|---|---|---|---|---|---|---|---|")
for p in sorted(glob.glob(os.path.join(E, "jmeter", "*.jtl"))):
    r = jtl(p)
    if r:
        print(row(os.path.basename(p)[:-4], r))
print("\n#### Ресурси контейнера (CPU %, макс.)\n")
for p in sorted(glob.glob(os.path.join(E, "stats", "*.csv"))):
    cpu = [float(l.split(",")[1].rstrip("%")) for l in open(p) if l.count(",") >= 2 and l.split(",")[1].rstrip("%").replace(".", "").isdigit()]
    mem = [l.split(",")[2].split("/")[0].strip() for l in open(p) if l.count(",") >= 2]
    if cpu:
        print(f"- {os.path.basename(p)[:-4]}: CPU max {max(cpu):.0f}%, avg {statistics.mean(cpu):.0f}%, пам'ять (остання) {mem[-1]}")
