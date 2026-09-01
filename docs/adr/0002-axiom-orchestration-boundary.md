# ADR-0002: Использовать Axiom для значимых durable-переходов, но не для эфемерного UI

- Статус: Accepted
- Дата: 2026-09-01

## Контекст

HWS имеет два принципиально разных класса состояния.

Первый класс — быстрое эфемерное UI-состояние:

- focus;
- hover;
- animation;
- временный selection;
- раскрытие/прокрутка;
- cached projection.

Второй класс — значимые операции с реальными side effects:

- Activate/Recover/Close workspace;
- запуск процессов;
- управление user services;
- подключение к удалённым системам;
- сетевые действия;
- применение layout;
- восстановление после частичных отказов.

Второму классу нужны инварианты, история, retry/idempotency и explainability.

## Решение

Axiom становится обязательным orchestration runtime для второго класса.

Основной frontend HWS — declarative Go package `github.com/Homiakus/axiom/model`.

Прямое использование Axiom локализуется в:

```text
internal/orchestration/axiomruntime/
internal/orchestration/models/
internal/orchestration/activities/
```

UI и базовый domain не экспортируют и не зависят от внутренних типов Axiom.

## Production semantics

- production orchestration использует transactional durable store;
- external activities проектируются идемпотентными;
- retry не интерпретируется как exactly-once;
- process-local serialization не интерпретируется как distributed lock;
- после restart observed state перечитывается;
- `Explain`/history доступны через application-level representation.

## Почему не Flow по умолчанию

HWS требуется статически анализируемая модель с rules, claims и activities. Поэтому default — declarative `model` frontend.

Typed Flow может применяться локально только для небольших reducer-сценариев, где произвольный Go важнее статического анализа и durable activity orchestration не требуется.

## Почему не AXM/TOML по умолчанию

Основная lifecycle-модель относится к коду HWS и должна эволюционировать вместе с типизированными domain contracts. Внешние DSL/decision tables допускаются позднее для plugin/provider сценариев, но не являются стартовым API core.

## Compatibility

До стабильного Axiom v1:

1. HWS закрепляет конкретную совместимую версию/pseudo-version;
2. обновление выполняется отдельной задачей;
3. ключевые runtime semantics покрываются contract tests;
4. Axiom types не протекают в IPC;
5. migration/replay compatibility проверяется до обновления production state.

## Следствия

Плюсы:

- явная lifecycle-модель;
- объяснимые ошибки;
- durable history;
- fault/retry semantics;
- тестируемые claims;
- возможность recovery после daemon restart.

Цена:

- дополнительные модели и activity contracts;
- необходимость дисциплины idempotency;
- миграция durable state при эволюции схем;
- зависимость от pre-v1 библиотеки, которую нужно локализовать.

## Нельзя

Нельзя создавать Axiom execution для каждого UI click/hover только ради единообразия. Это ухудшит latency, усложнит history и размоет смысл durable runtime.