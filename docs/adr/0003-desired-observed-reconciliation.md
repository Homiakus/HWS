# ADR-0003: Разделять desired и observed state и использовать reconciliation

- Статус: Accepted
- Дата: 2026-09-01

## Контекст

Запуск приложения, отправка команды window manager или старт user service не доказывают, что требуемое рабочее окружение реально достигнуто.

Внешняя среда может:

- проигнорировать запрос;
- выполнить его частично;
- изменить состояние сразу после выполнения;
- уже содержать подходящий ресурс;
- потерять ресурс после рестарта daemon;
- не поддерживать requested capability.

Если HWS хранит только «что мы попросили сделать», интерфейс начинает показывать ложный успех.

## Решение

HWS разделяет:

- `DesiredState` — что должно быть;
- `ObservedState` — что реально обнаружено;
- `ReconcileResult` — различие и решение о дальнейших действиях.

Успешное выполнение side effect не является достаточным условием перехода workspace в `Active`.

Для required ресурсов HWS подтверждает observed state после выполнения действий.

## Алгоритм

```text
Workspace definition
      ↓
Desired state
      ↓
Observe environment
      ↓
Observed state
      ↓
Diff
      ↓
Plan intents
      ↓
Axiom activities
      ↓
Observe again
      ↓
Converged / Degraded / Failed
```

## Ownership

Reconciliation не означает «убрать всё лишнее».

Ресурс имеет ownership mode:

- managed;
- adopted;
- external.

HWS вправе автоматически удалять/закрывать только те managed resources, для которых это разрешает shutdown policy.

## После restart

Сохранённый `ObservedState` считается только историческим snapshot.

После рестарта `hwsd` фактическое состояние перечитывается до принятия recovery-решения.

## Следствия

Плюсы:

- меньше ложных success-state;
- корректная работа с частичными отказами;
- естественная recovery модель;
- адаптация к уже запущенным приложениям;
- better diagnostics.

Цена:

- требуется identity matching;
- нужно повторное наблюдение;
- integration adapters становятся двусторонними: command + observe;
- некоторые desktop APIs могут давать неполный observed state, что должно выражаться через capability/confidence semantics.

## Нельзя

Нельзя использовать флаг вроде `launched=true` как доказательство того, что окно найдено, размещено и workspace готов.