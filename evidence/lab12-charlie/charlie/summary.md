| Файл | Складність | Ревізій | Оцінка | Група |
|---|---|---|---|---|
| cmd/roomflow/main.go | 178 | 7 | 1246 | Entry points |
| cmd/roomflow/main_test.go | 203 | 6 | 1218 | Entry points |
| internal/httpapi/handlers.go | 183 | 6 | 1098 | HTTP API |
| internal/booking/service_test.go | 171 | 5 | 855 | Booking domain |
| ci/lab12/mesh.sh | 176 | 4 | 704 | CI/CD |
| internal/obs/metrics.go | 185 | 3 | 555 | Cross-cutting platform libs |
| ci/lab11/backstage.sh | 101 | 5 | 505 | CI/CD |
| .github/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| ci/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| internal/booking/service.go | 113 | 4 | 452 | Booking domain |
| tools/sqlcheck/main.go | 199 | 2 | 398 | Developer tools |
| test/load/seed/main.go | 125 | 3 | 375 | Tests (integration, e2e, load) |
| ci/lab6/deploy.sh | 124 | 3 | 372 | CI/CD |
| tools/rfctl/commands.go | 351 | 1 | 351 | Developer tools |
| ci/lab9/run.sh | 112 | 3 | 336 | CI/CD |

| Файл | SOC | Найсильніше пов'язані (частка спільних змін) |
|---|---|---|
| cmd/roomflow/main.go | 280 | cmd/roomflow/main_test.go (62%), internal/httpapi/handlers.go (62%), internal/obs/metrics.go (43%) |
| ci/lab11/backstage.sh | 260 | ci/lab12/mesh.sh (80%), ci/lab12/monitoring.sh (80%), ci/lab11/argocd.sh (60%) |
| cmd/roomflow/main_test.go | 252 | cmd/roomflow/main.go (62%), internal/arch/arch_test.go (50%), ci/lab10/loop.sh (33%) |
| internal/httpapi/handlers.go | 250 | cmd/roomflow/main.go (62%), internal/booking/model.go (50%), internal/store/sql.go (50%) |
| ci/lab12/monitoring.sh | 249 | ci/lab11/backstage.sh (80%), infra/terraform/modules/lb/main.tf (75%), ci/lab12/rollouts.sh (75%) |
| ci/lab12/rollouts.sh | 242 | ci/lab9/gate.sh (100%), ci/lab9/run.sh (100%), test/load/seed/main.go (100%) |
| ci/lab9/gate.sh | 242 | ci/lab12/rollouts.sh (100%), ci/lab9/run.sh (100%), test/load/seed/main.go (100%) |
| ci/lab9/run.sh | 242 | ci/lab12/rollouts.sh (100%), ci/lab9/gate.sh (100%), test/load/seed/main.go (100%) |
| test/load/seed/main.go | 242 | ci/lab12/rollouts.sh (100%), ci/lab9/gate.sh (100%), ci/lab9/run.sh (100%) |
| ci/lab12/mesh.sh | 238 | ci/lab11/backstage.sh (80%), infra/terraform/modules/lb/main.tf (75%), ci/lab12/monitoring.sh (60%) |
| internal/obs/metrics.go | 236 | ci/lab10/loop.sh (67%), ci/lab10/strategies.sh (67%), ci/lab11/crossplane.sh (67%) |
| ci/lab11/argocd.sh | 235 | ci/lab12/lib.sh (100%), ci/lab8/secrets.sh (67%), ci/lib/shoot.mjs (67%) |

Часова зв'язаність між групами (пар файлів із ≥50% спільних змін):
- CI/CD ↔ Infrastructure as code: 8142
- Cross-cutting platform libs ↔ Infrastructure as code: 2290
- Developer tools ↔ Infrastructure as code: 2080
- Infrastructure as code ↔ інше: 1884
- Infrastructure as code ↔ Tests (integration, e2e, load): 1072
- CI/CD ↔ Cross-cutting platform libs: 858
- CI/CD ↔ Developer tools: 798
- CI/CD ↔ інше: 752
- Entry points ↔ Infrastructure as code: 574
- CI/CD ↔ Tests (integration, e2e, load): 426
- Booking domain ↔ Infrastructure as code: 420
- HTTP API ↔ Infrastructure as code: 420
