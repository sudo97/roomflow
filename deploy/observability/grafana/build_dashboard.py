#!/usr/bin/env python3
"""Генерує дашборд Grafana RoomFlow (dashboard as code): python3 build_dashboard.py > dashboards/roomflow.json"""
import json

PROM = {"type": "prometheus", "uid": "prometheus"}
LOKI = {"type": "loki", "uid": "loki"}
panels, pid = [], 0


def panel(kind, title, exprs, x, y, w, h, unit=None, ds=PROM, **opts):
    global pid
    pid += 1
    p = {
        "id": pid, "type": kind, "title": title, "datasource": ds,
        "gridPos": {"x": x, "y": y, "w": w, "h": h},
        "targets": [{"refId": chr(65 + i), "expr": e, "legendFormat": lf, "datasource": ds} for i, (e, lf) in enumerate(exprs)],
        "fieldConfig": {"defaults": {}, "overrides": []},
        "options": {},
    }
    if unit:
        p["fieldConfig"]["defaults"]["unit"] = unit
    if "thresholds" in opts:
        p["fieldConfig"]["defaults"]["thresholds"] = {"mode": "absolute", "steps": opts.pop("thresholds")}
    if kind in ("stat", "gauge"):
        p["options"] = {"reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False}, "colorMode": "background"}
    p["options"].update(opts)
    panels.append(p)


def row(title, y):
    global pid
    pid += 1
    panels.append({"id": pid, "type": "row", "title": title, "gridPos": {"x": 0, "y": y, "w": 24, "h": 1}, "collapsed": False, "panels": []})


green_red = lambda limit: [{"color": "green", "value": None}, {"color": "red", "value": limit}]

row("DORA: пропускна здатність і стабільність (6 год)", 0)
panel("stat", "Розгортання", [("dora:deployments:increase6h", "")], 0, 1, 4, 4, "none")
panel("stat", "Частота розгортань / добу (історія 7 днів)", [("dora_deployment_frequency_per_day", "")], 4, 1, 4, 4, "none")
panel("stat", "Lead time for changes (сер.)", [("dora:lead_time_seconds:avg_6h", "")], 8, 1, 4, 4, "s")
panel("gauge", "Change failure rate", [("dora:change_failure_ratio:6h", "")], 12, 1, 4, 4, "percentunit",
      thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 0.15}, {"color": "red", "value": 0.3}])
panel("stat", "Time to restore (сер.)", [("dora:restore_seconds:avg_6h", "")], 16, 1, 4, 4, "s")
panel("stat", "Відкриті інциденти", [("dora_open_incidents", "")], 20, 1, 4, 4, "none", thresholds=green_red(1))

row("Застосунок: RED і SLO", 5)
panel("timeseries", "Запити за секунду за маршрутами", [("route:roomflow_http_requests:rate1m", "{{route}}")], 0, 6, 12, 8, "reqps")
panel("timeseries", "p95 затримки за маршрутами", [("route:roomflow_http_latency_seconds:p95_1m", "{{route}}")], 12, 6, 12, 8, "s")
panel("timeseries", "Частка помилок 5xx", [("roomflow:http_errors:ratio_rate1m", "5xx ratio")], 0, 14, 8, 7, "percentunit")
panel("gauge", "Доступність (SLO 99.5%)", [("slo:availability:ratio_rate5m", "")], 8, 14, 4, 7, "percentunit",
      thresholds=[{"color": "red", "value": None}, {"color": "green", "value": 0.995}], min=0.9, max=1)
panel("timeseries", "Швидкість вигорання бюджету помилок", [("slo:error_budget_burn_rate:1m", "1m"), ("slo:error_budget_burn_rate:5m", "5m")], 12, 14, 8, 7, "none")
panel("stat", "Запити в обробці", [("sum(roomflow_http_in_flight_requests)", "")], 20, 14, 4, 7, "none")
panel("timeseries", "Бізнес-метрики: бронювання і конфлікти / хв",
      [("sum(rate(roomflow_bookings_created_total[1m])) * 60", "бронювання"), ("sum(rate(roomflow_booking_conflicts_total[1m])) * 60", "конфлікти")], 0, 21, 12, 7, "none")
panel("timeseries", "Записи журналу за рівнем (Loki)", [('sum by (level) (count_over_time({container="roomflow"} | json [1m]))', "{{level}}")], 12, 21, 12, 7, "none", ds=LOKI)

row("Журнали", 28)
panel("logs", "Помилки і відмови (status >= 400)", [('{container="roomflow"} | json | status >= 400', "")], 0, 29, 24, 10, ds=LOKI,
      showTime=True, wrapLogMessage=True, sortOrder="Descending")

dashboard = {
    "uid": "roomflow", "title": "RoomFlow — DORA, RED, SLO, журнали", "schemaVersion": 39, "version": 1,
    "time": {"from": "now-15m", "to": "now"}, "refresh": "10s", "timezone": "browser",
    "tags": ["roomflow", "dora", "slo"], "panels": panels, "templating": {"list": []}, "annotations": {"list": []},
}
print(json.dumps(dashboard, ensure_ascii=False, indent=1))
