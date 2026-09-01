# HWS Test Strategy

Статус: Proposed baseline.

## 1. Цель

HWS управляет реальными desktop/process/network side effects, поэтому обычных unit tests недостаточно. Тестовая стратегия строится слоями: pure domain → Axiom lifecycle → adapters → nested desktop session → fault/restart tests.

## 2. Test pyramid

```text
                    E2E desktop
                 restart/fault E2E
               adapter integration
             Axiom lifecycle contract
          property/fuzz/mutation tests
        pure unit tests / deterministic core
```

Чем ниже уровень, тем больше тестов и быстрее обратная связь.

## 3. Pure domain tests

Пакеты дерева, path, desired/observed diff и reconcile planning должны тестироваться без Axiom, D-Bus и GNOME.

Обязательные свойства:

- дерево не принимает cycles;
- stable ordering детерминирован;
- path валиден только внутри snapshot;
- rename не меняет identity;
- canonical snapshot hash стабилен;
- одинаковые desired/observed дают пустой diff;
- required resource никогда не теряется при diff normalization;
- adopted/external не попадают в destructive cleanup plan без policy.

## 4. Property testing

Кандидаты:

### Tree

Генерировать случайные trees и проверять:

```text
validate(serialize(parse(tree))) == valid
```

а также отсутствие cycles/duplicate IDs и стабильность ordering.

### Reconcile

Свойства:

```text
Diff(D, D) = empty
Apply(Plan(D,O), O) при идеальном fake adapter → D
Plan(D,O) deterministic
```

### Ownership

Ни одна последовательность допустимых operations не создаёт intent `destroy` для `external` без explicit override policy.

## 5. Fuzzing

Fuzz targets:

- config parser;
- tree snapshot decoder;
- IPC request decoder;
- workspace definition migration;
- canonical hashing input;
- provider payload normalization;
- error mapping.

Fuzz test не должен иметь доступ к реальной системе пользователя.

## 6. Mutation testing

Наиболее полезно для pure critical packages:

- tree validation;
- requirement evaluation;
- desired/observed diff;
- ownership rules;
- reconcile planner;
- error classification.

Цель mutation testing — выявлять tests, которые проходят, даже если удалить/инвертировать критичную проверку.

Не требуется высокий mutation score для glue/integration code любой ценой.

## 7. Axiom model tests

Каждая lifecycle model получает table-driven transition tests.

Проверять:

- допустимые события;
- запрещённые переходы;
- claims;
- history;
- explain;
- retry semantics;
- cancellation;
- replay;
- restart с durable store.

## 8. Axiom compatibility contract

Поскольку Axiom до стабильного v1 может эволюционировать, HWS должен иметь отдельный contract suite, который запускается при каждом обновлении зависимости.

Минимальные контракты:

1. typed model компилируется;
2. claim violation блокирует переход;
3. typed activity input/output сохраняют форму;
4. retry переживает reopen durable store;
5. history доступна после restart;
6. replay не повторяет уже записанный внешний activity result;
7. transactional production mode отклоняет неподходящий store;
8. idempotency/dedup behavior соответствует ожиданию HWS;
9. cancel не интерпретируется как rollback;
10. Explain возвращает достаточные данные для application mapping.

Если contract suite падает после обновления Axiom, зависимость не принимается до явной адаптации.

## 9. Fake integration environment

До desktop adapter создаётся fake session model:

```text
FakeProcessAdapter
FakeWindowAdapter
FakeSystemdAdapter
FakeNetworkAdapter
FakeClock
FakeCapabilityProvider
```

Fake должен уметь внедрять:

- transient error;
- permanent error;
- delayed appearance;
- duplicate observation;
- lost resource;
- crash after side effect;
- timeout;
- capability loss.

## 10. Fault injection

Ключевой класс тестов HWS.

Точки отказа:

```text
before activity
inside activity before side effect
after side effect before return
after return before checkpoint
store commit failure
process restart
IPC disconnect
capability changes between observations
```

Для каждой critical activity должен существовать минимум один тест неизвестного исхода (`side effect may have happened`).

## 11. Restart tests

Production correctness HWS нельзя считать подтверждённой без restart tests.

Сценарий:

1. durable store во временной директории;
2. начать Activate;
3. остановить engine/daemon в заданной точке;
4. создать новый process/runtime instance;
5. повторно observe fake environment;
6. выполнить Recover;
7. проверить отсутствие destructive duplicate и корректный final state.

## 12. Adapter contract tests

Каждый реальный adapter обязан пройти общую suite.

Например `ProcessAdapter`:

```text
Ensure(existing matching process) → reuse/adopt
Ensure(absent) → create
Ensure(repeated) → stable identity/no duplicate
Observe(created) → present
Release(external) → denied/no-op by policy
```

## 13. Desktop E2E

E2E не должен запускаться в основной пользовательской сессии CI runner.

Предпочтительно использовать изолированную nested/virtual desktop session.

Проверять:

- запуск shell UI;
- IPC connection;
- hierarchy rendering;
- Activate test workspace;
- появление test app windows;
- layout intent;
- Close/recover;
- fallback после остановки HWS.

## 14. Golden tests

Подходят для:

- tree projections;
- IPC schemas/examples;
- normalized layout plans;
- Explain rendering model;
- diagnostics summaries.

Golden files не должны использоваться для скрытия сложной семантики вместо точных assertions.

## 15. Race and concurrency

Go CI:

```text
go test ./...
go test -race ./...
```

Отдельно тестировать:

- simultaneous UI read + provider refresh;
- concurrent Activate requests одного workspace;
- Activate vs Close race;
- adapter event arriving during reconcile;
- daemon shutdown while activity pending.

## 16. Performance tests

После M1 ввести benchmark baseline:

- tree projection;
- search;
- desired/observed diff;
- reconcile planning;
- IPC serialization;
- Axiom dispatch без external latency;
- large hierarchy navigation dataset.

Производительность реального app startup измеряется отдельно от core overhead.

## 17. Security tests

Минимально:

- argv escaping / отсутствие implicit shell;
- path traversal;
- symlink-sensitive operations;
- malicious provider payload;
- IPC unauthorized peer where applicable;
- diagnostics redaction;
- secrets not serialized to history/logs.

## 18. CI gates

M1 target gate:

```text
fmt
vet
unit
property
fuzz smoke
race
Axiom contract
```

M2 добавляет:

```text
adapter contract
nested desktop E2E
restart/fault suite
```

Release gate добавляет vulnerability/security scans и packaging install/uninstall tests.

## 19. Правило regression test

Каждый найденный production/real-session defect сначала превращается в воспроизводимый test на самом низком возможном уровне, затем исправляется.

Если defect невозможно воспроизвести тестом, issue должен документировать причину и временный diagnostic probe.