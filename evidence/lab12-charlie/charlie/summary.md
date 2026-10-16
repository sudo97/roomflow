| Файл | Складність | Ревізій | Оцінка | Група |
|---|---|---|---|---|
| cmd/roomflow/main.go | 178 | 7 | 1246 | Entry points |
| cmd/roomflow/main_test.go | 203 | 6 | 1218 | Entry points |
| internal/httpapi/handlers.go | 183 | 6 | 1098 | HTTP API |
| internal/booking/service_test.go | 171 | 5 | 855 | Booking domain |
| internal/obs/metrics.go | 185 | 3 | 555 | Cross-cutting platform libs |
| .github/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| ci/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| internal/booking/service.go | 113 | 4 | 452 | Booking domain |
| tools/sqlcheck/main.go | 199 | 2 | 398 | Developer tools |
| tools/rfctl/commands.go | 351 | 1 | 351 | Developer tools |
| ci/lab10/strategies.sh | 154 | 2 | 308 | CI/CD |
| .github/workflows/sonar.yml | 132 | 2 | 264 | CI/CD |
| .github/workflows/lab8-devsecops.yml | 255 | 1 | 255 | CI/CD |
| ci/workflows/lab8-devsecops.yml | 255 | 1 | 255 | CI/CD |
| internal/booking/properties_test.go | 85 | 3 | 255 | Booking domain |

| Файл | SOC | Найсильніше пов'язані (частка спільних змін) |
|---|---|---|
| cmd/roomflow/main.go | 280 | cmd/roomflow/main_test.go (62%), internal/httpapi/handlers.go (62%), internal/obs/metrics.go (43%) |
| cmd/roomflow/main_test.go | 252 | cmd/roomflow/main.go (62%), internal/arch/arch_test.go (50%), ci/lab10/loop.sh (33%) |
| internal/httpapi/handlers.go | 250 | cmd/roomflow/main.go (62%), internal/booking/model.go (50%), internal/store/sql.go (50%) |
| internal/obs/metrics.go | 236 | ci/lab10/loop.sh (67%), ci/lab10/strategies.sh (67%), ci/lab11/argocd.sh (67%) |
| ci/lab10/loop.sh | 224 | ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%), ci/lab11/backstage.sh (100%) |
| ci/lab10/strategies.sh | 224 | ci/lab10/loop.sh (100%), ci/lab11/argocd.sh (100%), ci/lab11/backstage.sh (100%) |
| ci/lab11/argocd.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/backstage.sh (100%) |
| ci/lab11/backstage.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%) |
| ci/lab11/crossplane.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%) |
| ci/lab12/lib.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%) |
| ci/lab12/monitoring.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%) |
| ci/lab12/rollouts.sh | 224 | ci/lab10/loop.sh (100%), ci/lab10/strategies.sh (100%), ci/lab11/argocd.sh (100%) |

Часова зв'язаність між групами (пар файлів із ≥50% спільних змін):
- CI/CD ↔ Infrastructure as code: 10166
- Cross-cutting platform libs ↔ Infrastructure as code: 2332
- Developer tools ↔ Infrastructure as code: 2120
- Infrastructure as code ↔ інше: 2100
- Infrastructure as code ↔ Tests (integration, e2e, load): 1272
- CI/CD ↔ Cross-cutting platform libs: 1054
- CI/CD ↔ Developer tools: 974
- CI/CD ↔ інше: 956
- Entry points ↔ Infrastructure as code: 636
- CI/CD ↔ Tests (integration, e2e, load): 582
- Booking domain ↔ Infrastructure as code: 424
- HTTP API ↔ Infrastructure as code: 424
