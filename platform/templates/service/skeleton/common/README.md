# ${{ values.name }}

${{ values.description }}

Створено з golden path шаблону платформи RoomFlow (мова: ${{ values.language }}, база даних: ${{ values.database }}).
Власник: ${{ values.owner }}.

- `GET /healthz` — перевірка стану, `GET /metrics` — метрики Prometheus.
- Розгортання: `kubectl apply -f k8s/`.
