# Runbook: RoomFlowDown

**Alert:** `RoomFlowDown` — `up{job="roomflow"} == 0` for 15 s (severity: critical).
**Meaning:** Prometheus cannot scrape any RoomFlow instance; guests and hotel staff most likely cannot book.

## 1. Confirm (≤ 2 min)

```bash
curl -fsS http://<lb>/healthz            # through the load balancer
docker ps --filter name=roomflow          # Docker / Terraform environments
kubectl -n roomflow get pods -o wide      # Kubernetes environments
```

- Prometheus → Status → Targets: is the target `down` with `connection refused` (process gone) or `context deadline exceeded` (process hung / overloaded)?
- Grafana dashboard "RoomFlow": did RPS drop to zero for all tenants or only for one replica?

## 2. Mitigate first, investigate later

| Symptom | Action |
|---|---|
| Started right after a deployment | Roll back: switch the LB to the previous color (`terraform workspace select <prev> && terraform apply`) (Docker) or `helm rollback roomflow` / `kubectl argo rollouts undo frontend` (Kubernetes). |
| Container restarting / OOMKilled | `docker inspect <c> --format '{{.State.OOMKilled}}'`; raise memory limit or roll back. |
| One replica down, others healthy | Remove it from the LB pool (blue-green: switch color), restart it. |
| Dependency failure (Postgres, Redis, Vault) | Check the dependency; Redis outage must not break availability (cache-aside falls back to the store). |

## 3. Diagnose

```bash
docker logs --since 10m <container> | grep -v http_request
kubectl -n roomflow logs deploy/roomflow --previous
```

Loki: `{service="roomflow"} | json | level="ERROR"`.

## 4. Close

- Verify `up == 1` and the error-rate SLI back under 1 %.
- Record the incident end — the DORA exporter computes MTTR from it.
- Blameless postmortem within 2 working days if the error budget burn exceeded 2 %.
