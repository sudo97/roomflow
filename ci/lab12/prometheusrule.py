#!/usr/bin/env python3
"""Обгортає файли правил Prometheus (ЛР7) у ресурс PrometheusRule для Prometheus Operator."""
import json, sys
import yaml

groups = []
for path in sys.argv[1:]:
    groups += yaml.safe_load(open(path))["groups"]
groups = [g for g in groups if g["name"] != "dora"]
print(json.dumps({
    "apiVersion": "monitoring.coreos.com/v1", "kind": "PrometheusRule",
    "metadata": {"name": "roomflow-slo", "namespace": "rf-mon", "labels": {"release": "kps"}},
    "spec": {"groups": groups},
}))
