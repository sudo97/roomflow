# ADR-0003: Service mesh (Istio) для mTLS, стійкості і трасування

Статус: прийнято (ЛР12)

## Контекст
Після виділення frontend і Redis з'явилися міжсервісні виклики: потрібні взаємна автентифікація,
тайм-аути, повтори, circuit breaker і наскрізне трасування без дублювання логіки в кожному сервісі.

## Рішення
Istio: PeerAuthentication STRICT, AuthorizationPolicy за ServiceAccount (frontend → backend → redis),
DestinationRule (ліміти з'єднань, outlier detection), VirtualService (тайм-аут 3 с, 3 повтори).
Застосунки передають заголовок traceparent і створюють власні спани (OTLP → Jaeger).

## Наслідки
+ політики безпеки і стійкості декларативні й однакові для всіх сервісів;
− додаткова затримка sidecar і складність експлуатації; для двох сервісів це "на виріст".
