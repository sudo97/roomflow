# RoomFlow

Навчальний мультитенантний SaaS-сервіс бронювання для малих готелів (Go, лише стандартна бібліотека).
Репозиторій супроводжує лабораторні роботи 1–12 з дисципліни «Удосконалення процесів інженерії програмного забезпечення» (ХНТУ).

```bash
go run ./cmd/roomflow          # API на :8080
go run ./cmd/frontend          # веб-інтерфейс на :8081 (BACKEND_URL=http://localhost:8080)
go test -cover ./...
```

Кожен запит до `/api` виконується в контексті орендаря (готелю), заданого заголовком `X-Tenant-ID`.
Специфікація — [docs/api/openapi.yaml](docs/api/openapi.yaml).

| Метод | Шлях | Опис |
|---|---|---|
| GET | /healthz | перевірка стану |
| GET | /api/rooms | номери орендаря |
| POST | /api/rooms | додати номер |
| GET | /api/rooms/{id}/availability | календар доступності і ціни ночей (cache-aside, Redis) |
| POST | /api/bookings | створити бронювання |
| POST | /api/bookings/{id}/cancel | скасувати бронювання |
| GET | /api/flags | стан feature flags для орендаря |
| GET | /metrics | метрики Prometheus |

Необов'язкові змінні середовища: `CACHE_REDIS_ADDR`, `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`, `FLAGS_FILE`,
`OTEL_EXPORTER_OTLP_ENDPOINT`, `PPROF_ADDR`, `VAULT_ADDR` (секрети через AppRole), `FAULT_RATE` (ін'єкція збоїв для експериментів).

## Структура

| Каталог | Зміст |
|---|---|
| `cmd/roomflow`, `cmd/frontend` | сервіси API і веб-інтерфейсу |
| `internal/` | домен (`booking`), сховище (`store`), HTTP (`httpapi`), кеш, rate limiting, feature flags, трасування, метрики, архітектурні тести (`arch`) |
| `tools/` | `dora-exporter`, `platform-exporter`, CLI платформи `rfctl`, `sqlcheck` |
| `infra/terraform` | модулі network, app, lb, database, monitoring; стеки platform і app (workspaces blue/green) |
| `policies/` | політики OPA для Terraform і golden path |
| `deploy/` | спостережуваність (Prometheus, Grafana, Loki), Kubernetes, Helm-чарт, GitOps, Istio |
| `platform/` | Crossplane (XRD, Composition), Software Template для нових сервісів |
| `test/load` | k6, JMeter, наповнення тестовими даними |
| `docs/` | ADR, OpenAPI, runbooks |
| `ci/` | сценарії лабораторних робіт (`ci/labN`), спільні функції, workflow-файли |

## Лабораторні роботи і конвеєри

| ЛР | Тема | Workflow | Сценарії |
|---|---|---|---|
| 1–5 | процес, CI, тестування, якість коду | `ci.yml`, `tests.yml`, `quality.yml`, `mutation.yml`, `sonar.yml` | `scripts/`, `ci/sonar` |
| 6 | Infrastructure as Code (Terraform) | `lab6-iac.yml` | `ci/lab6` |
| 7 | моніторинг, логування, DORA | `lab7-observability.yml` | `ci/lab7` |
| 8 | DevSecOps | `lab8-devsecops.yml` | `ci/lab8` |
| 9 | навантажувальне тестування і продуктивність | `lab9-performance.yml`, ворота продуктивності в `ci.yml` | `ci/lab9` |
| 10 | стратегії розгортання, feature flags | `lab10-deployments.yml` | `ci/lab10` |
| 11 | platform engineering (Backstage, Crossplane, ArgoCD) | `lab11-platform.yml` | `ci/lab11` |
| 12 | service mesh, GitOps, спостережуваність, аналіз коду | `lab12-cloud-native.yml` | `ci/lab12` |

Усі експерименти виконуються в GitHub Actions (кластери kind, контейнери Docker). Результати кожного запуску
(журнали, таблиці, знімки екрана) публікуються в гілку `evidence` у каталог з назвою сценарію.

Workflow-файли зберігаються також у `ci/workflows`; `scripts/sync-workflows.sh` копіює їх у `.github/workflows`.
`run.yml` (run-script) запускає будь-який сценарій з `ci/` вручну; через нього, зокрема, застосовувались
виправлення з `ci/patches/` (`ci/apply-patch.sh`).

## Гілки

| Гілка | Призначення |
|---|---|
| `main` | поточна версія |
| `evidence` | результати запусків сценаріїв ЛР6–12 |
| `baseline` | рання версія проєкту (коміт 32a122c) — еталон для порівнянь якості коду і безпеки (ЛР3, ЛР8) |
| `lab3` | стан репозиторію після виправлення зауважень SonarQube (ЛР3) |
| `lab4-weak-tests` | навмисно слабкі тести для мутаційного тестування (ЛР4) |
| `demo/unsafe-report-export` | навмисно вразлива зміна (SQL injection) для перевірки SAST і воріт PR (ЛР8) |
