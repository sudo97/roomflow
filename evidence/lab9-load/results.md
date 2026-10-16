#### k6: базові показники за кінцевими точками (мс)

| Варіант / точка | Запитів | RPS | avg | p50 | p95 | p99 | max | Помилки |
|---|---|---|---|---|---|---|---|---|
| before / health | 756091 | 8401 | 8.1 | 4.3 | 37.0 | 50.3 | 89 | 0.00% |
| before / availability | 36666 | 406 | 171.3 | 90.0 | 686.9 | 1202.9 | 3906 | 0.00% |
| before / booking | 36620 | 407 | 171.1 | 131.3 | 423.1 | 488.7 | 547 | 0.00% |
| before / stress | 55027 | 408 | 678.0 | 211.2 | 2523.8 | 5204.1 | 10002 | 0.05% |
| after / health | 769272 | 8547 | 8.0 | 4.1 | 37.4 | 50.4 | 109 | 0.00% |
| after / availability | 205336 | 2280 | 30.4 | 10.1 | 97.8 | 165.4 | 461 | 0.00% |
| after / booking | 179407 | 1991 | 34.8 | 12.6 | 106.8 | 174.6 | 273 | 0.00% |
| after / stress | 313691 | 2323 | 117.5 | 73.0 | 413.5 | 792.2 | 2237 | 0.00% |
| cache / availability | 424580 | 4717 | 14.5 | 8.1 | 53.4 | 66.1 | 153 | 0.00% |
| cache / stress | 431053 | 3193 | 85.4 | 27.7 | 377.9 | 689.3 | 1564 | 0.00% |

#### JMeter (до оптимізації)

| Точка | Запитів | RPS | avg | p50 | p95 | p99 | max | Помилки |
|---|---|---|---|---|---|---|---|---|

#### Ресурси контейнера (CPU %, макс.)

- after-availability: CPU max 104%, avg 101%, пам'ять (остання) 63.66MiB
- after-booking: CPU max 105%, avg 102%, пам'ять (остання) 228.8MiB
- after-health: CPU max 103%, avg 101%, пам'ять (остання) 62.09MiB
- after-profiled-availability: CPU max 104%, avg 101%, пам'ять (остання) 66.93MiB
- after-stress: CPU max 103%, avg 100%, пам'ять (остання) 98.61MiB
- before-availability: CPU max 104%, avg 100%, пам'ять (остання) 55.81MiB
- before-booking: CPU max 110%, avg 105%, пам'ять (остання) 75.98MiB
- before-health: CPU max 104%, avg 101%, пам'ять (остання) 57.9MiB
- before-profiled-availability: CPU max 103%, avg 100%, пам'ять (остання) 57.54MiB
- before-stress: CPU max 103%, avg 100%, пам'ять (остання) 71.46MiB
- cache-availability: CPU max 104%, avg 101%, пам'ять (остання) 60.21MiB
- cache-stress: CPU max 105%, avg 101%, пам'ять (остання) 95.77MiB
