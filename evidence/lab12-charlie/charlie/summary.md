| Файл | Складність | Ревізій | Оцінка | Група |
|---|---|---|---|---|
| internal/httpapi/handlers.go | 183 | 6 | 1098 | HTTP API |
| cmd/roomflow/main.go | 154 | 6 | 924 | Entry points |
| internal/booking/service_test.go | 171 | 5 | 855 | Booking domain |
| cmd/roomflow/main_test.go | 157 | 5 | 785 | Entry points |
| .github/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| ci/workflows/ci.yml | 233 | 2 | 466 | CI/CD |
| internal/booking/service.go | 113 | 4 | 452 | Booking domain |
| tools/sqlcheck/main.go | 199 | 2 | 398 | Developer tools |
| internal/obs/metrics.go | 185 | 2 | 370 | Cross-cutting platform libs |
| tools/rfctl/commands.go | 351 | 1 | 351 | Developer tools |
| .github/workflows/sonar.yml | 132 | 2 | 264 | CI/CD |
| .github/workflows/lab8-devsecops.yml | 255 | 1 | 255 | CI/CD |
| ci/workflows/lab8-devsecops.yml | 255 | 1 | 255 | CI/CD |
| internal/booking/properties_test.go | 85 | 3 | 255 | Booking domain |
| ci/sonar/setup.sh | 61 | 4 | 244 | CI/CD |

| Файл | SOC | Найсильніше пов'язані (частка спільних змін) |
|---|---|---|
| cmd/roomflow/main.go | 258 | internal/httpapi/handlers.go (71%), cmd/roomflow/main_test.go (57%), internal/booking/model.go (50%) |
| internal/httpapi/handlers.go | 250 | cmd/roomflow/main.go (71%), internal/booking/model.go (50%), internal/store/sql.go (50%) |
| cmd/roomflow/main_test.go | 230 | internal/arch/arch_test.go (60%), cmd/roomflow/main.go (57%), Dockerfile (40%) |
| .gitignore | 219 | .charlie.config.json (50%), .pre-commit-config.yaml (50%), catalog-info.yaml (50%) |
| internal/obs/metrics.go | 214 | internal/obs/middleware.go (100%), .charlie.config.json (50%), .pre-commit-config.yaml (50%) |
| internal/obs/middleware.go | 214 | internal/obs/metrics.go (100%), .charlie.config.json (50%), .pre-commit-config.yaml (50%) |
| internal/arch/arch_test.go | 211 | Dockerfile (67%), cmd/roomflow/main_test.go (60%), .charlie.config.json (33%) |
| Dockerfile | 210 | internal/arch/arch_test.go (67%), .charlie.config.json (50%), .pre-commit-config.yaml (50%) |
| ci/workflows/ci.yml | 206 | .charlie.config.json (50%), .pre-commit-config.yaml (50%), catalog-info.yaml (50%) |
| .charlie.config.json | 202 | .pre-commit-config.yaml (100%), catalog-info.yaml (100%), ci/lab10/bluegreen.sh (100%) |
| .pre-commit-config.yaml | 202 | .charlie.config.json (100%), catalog-info.yaml (100%), ci/lab10/bluegreen.sh (100%) |
| catalog-info.yaml | 202 | .charlie.config.json (100%), .pre-commit-config.yaml (100%), ci/lab10/bluegreen.sh (100%) |

Часова зв'язаність між групами (пар файлів із ≥50% спільних змін):
- CI/CD ↔ Infrastructure as code: 10176
- Cross-cutting platform libs ↔ Infrastructure as code: 2544
- Infrastructure as code ↔ інше: 2120
- Developer tools ↔ Infrastructure as code: 2120
- Infrastructure as code ↔ Tests (integration, e2e, load): 1272
- CI/CD ↔ Cross-cutting platform libs: 1148
- CI/CD ↔ інше: 1000
- CI/CD ↔ Developer tools: 976
- Entry points ↔ Infrastructure as code: 636
- CI/CD ↔ Tests (integration, e2e, load): 584
- Booking domain ↔ Infrastructure as code: 424
- HTTP API ↔ Infrastructure as code: 424
