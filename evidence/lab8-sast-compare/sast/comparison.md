| Версія | Інструмент | Знахідок | Виявлено еталонних | Інших знахідок | Правила |
|---|---|---|---|---|---|
| legacy | gosec | 20 | 2/3 | 18 | G104, G114, G201 |
| legacy | semgrep | 1 | 0/3 | 1 | use-tls |
| legacy | codeql | 0 | 0/3 | 0 |  |
| demo | gosec | 6 | 1/1 | 5 | G104, G201, G202 |
| demo | semgrep | 0 | 0/1 | 0 |  |
| demo | codeql | 1 | 1/1 | 0 | go/sql-injection |
| current | gosec | 8 | 0/0 | 8 | G104, G108, G201, G202, G304, G404 |
| current | semgrep | 1 | 0/0 | 1 | math-random-used |
| current | codeql | 2 | 0/0 | 2 | go/log-injection |

| Інструмент | Виявлено (TP) | Пропущено (FN) | Інші знахідки |
|---|---|---|---|
| gosec | 3 | 1 | 31 |
| semgrep | 0 | 4 | 2 |
| codeql | 1 | 3 | 2 |
