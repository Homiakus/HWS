# HWS MASTER PLAN

Единый living execution plan проекта HWS.

Правило: если архитектурная находка, ограничение или новый риск меняет последовательность работ, сначала обновляется этот файл, затем код/документация.

## 0. Цель продукта

Создать иерархическую рабочую оболочку для Ubuntu, которая организует работу вокруг контекстов и задач, а не вокруг списка приложений.

Целевой пользовательский цикл:

```text
Area → Context → Project → Task → Ready Workspace
```

HWS должен уметь быстро навигировать по иерархии, восстанавливать рабочее окружение, управлять значимыми side effects через проверяемую модель состояния и оставаться безопасно отключаемым слоем поверх базовой графической системы.

## 1. Архитектурные решения, уже принятые

- [x] Основной daemon пишется на Go.
- [x] UI и durable orchestration разделены.
- [x] Иерархия контекста имеет произвольную глубину.
- [x] Axiom используется для значимых проверяемых переходов и orchestration.
- [x] Эфемерное UI-состояние не проводится через Axiom.
- [x] На первом этапе HWS не форкает compositor/window manager.
- [x] Интеграции с ОС скрываются за адаптерами.
- [x] Desired state и observed state рабочего окружения разделяются.
- [x] Все системные side effects должны иметь идемпотентные границы или явное описание невозможности идемпотентности.
- [x] Значимые архитектурные компромиссы фиксируются ADR.

## 2. Definition of Done для MVP

MVP считается готовым только если один реальный workspace проходит полный цикл:

1. HWS запускается в пользовательской сессии Ubuntu.
2. `Super` открывает иерархическую сетку.
3. Пользователь проходит минимум 3 уровня иерархии.
4. Выбор task создаёт desired workspace state.
5. Переход проходит через Axiom model/claims.
6. HWS запускает минимум два разных ресурса (например editor + terminal).
7. HWS распознаёт уже запущенные ресурсы и не создаёт дубликаты без причины.
8. HWS применяет layout либо возвращает объяснимую ошибку capability.
9. Workspace можно закрыть и восстановить.
10. История операции доступна для диагностики.
11. После рестарта `hwsd` durable операция/состояние не становится неописанным.
12. Отключение HWS не делает базовую графическую сессию неработоспособной.

## 3. Workstreams

### W0 — Repository foundation

- [ ] Определить лицензию.
- [ ] Создать Go module.
- [ ] Зафиксировать минимальную поддерживаемую версию Go.
- [ ] Добавить Makefile/Taskfile.
- [ ] Добавить CI: fmt, vet, test, race, vulnerability scan.
- [ ] Добавить CONTRIBUTING.md.
- [ ] Добавить SECURITY.md.
- [ ] Добавить ADR template.
- [ ] Зафиксировать code ownership и правила ветвления.

### W1 — Domain model

- [ ] Определить `NodeID`, `WorkspaceID`, `ResourceID`, `ActionID`.
- [ ] Определить типы узлов: category/project/task/action/widget/query.
- [ ] Реализовать дерево произвольной глубины.
- [ ] Реализовать stable ordering.
- [ ] Реализовать selection path.
- [ ] Реализовать breadcrumbs/context stack.
- [ ] Определить правила динамических children providers.
- [ ] Определить snapshot/version модели дерева.
- [ ] Покрыть property tests для дерева и path resolution.

### W2 — Workspace state model

- [ ] Описать desired state.
- [ ] Описать observed state.
- [ ] Описать reconciliation result.
- [ ] Описать resource lifecycle.
- [ ] Определить partial/degraded states.
- [ ] Определить ownership ресурсов: managed/adopted/external.
- [ ] Определить правила закрытия workspace без убийства чужих процессов.
- [ ] Добавить deterministic diff desired ↔ observed.

### W3 — Axiom integration

- [ ] Добавить зависимость `github.com/Homiakus/axiom` через отдельный integration package.
- [ ] На ранней стадии закрепить конкретную совместимую версию/commit Axiom и обновлять осознанно.
- [ ] Использовать declarative Go `model` как основной frontend.
- [ ] Определить model `WorkspaceLifecycle`.
- [ ] Определить events: Activate, Reconcile, Suspend, Resume, Close, Recover.
- [ ] Определить states: Inactive, Preparing, Active, Degraded, Suspending, Recovering, Failed.
- [ ] Определить claims для ownership, capability и safety.
- [ ] Реализовать typed activities для OS side effects.
- [ ] В production path использовать transactional durable store.
- [ ] Зафиксировать idempotency keys для каждой activity.
- [ ] Добавить failure injection tests.
- [ ] Добавить replay/history tests.
- [ ] Добавить explain diagnostics в UI/API.

### W4 — `hwsd`

- [ ] Bootstrap daemon lifecycle.
- [ ] Single-instance policy.
- [ ] Structured logging.
- [ ] Health endpoint/IPC method.
- [ ] Config loader.
- [ ] State storage abstraction.
- [ ] Axiom engine lifecycle.
- [ ] Context service.
- [ ] Workspace service.
- [ ] Reconciler.
- [ ] Integration registry.
- [ ] Graceful shutdown.
- [ ] Crash recovery path.

### W5 — IPC contract

- [ ] Выбрать transport для локальной сессии; D-Bus является исходным предпочтением, но решение фиксируется ADR.
- [ ] Сформировать versioned API.
- [ ] `GetTree`.
- [ ] `SelectNode`.
- [ ] `ActivateWorkspace`.
- [ ] `GetWorkspaceState`.
- [ ] `GetHistory`.
- [ ] `Explain`.
- [ ] `Search`.
- [ ] event stream/subscriptions.
- [ ] capability discovery.
- [ ] contract tests.

### W6 — Shell UI

- [ ] Home/Grid mode.
- [ ] Focus mode.
- [ ] Dynamic row generation.
- [ ] Keyboard navigation.
- [ ] Touch/mouse navigation.
- [ ] Breadcrumbs.
- [ ] Global search.
- [ ] Loading/degraded/error states.
- [ ] Accessibility model.
- [ ] Animation budget.
- [ ] Focus handling.
- [ ] Multi-monitor behavior.

### W7 — OS integration

- [ ] Application discovery.
- [ ] Process launch/adoption.
- [ ] Terminal launch with cwd.
- [ ] Window discovery.
- [ ] Window intent/layout adapter.
- [ ] systemd user services.
- [ ] filesystem/project discovery.
- [ ] Git status provider.
- [ ] SSH integration.
- [ ] network/VPN capabilities только через безопасный adapter boundary.
- [ ] portals/screen sharing compatibility checks.

### W8 — Persistence & recovery

- [ ] Durable Axiom store adapter/configuration.
- [ ] User config schema.
- [ ] Context tree persistence.
- [ ] Workspace snapshots.
- [ ] Last active contexts.
- [ ] Crash markers.
- [ ] Recovery decision model.
- [ ] Migration strategy.
- [ ] Corruption handling.
- [ ] backup/export format.

### W9 — Search and discovery

- [ ] Prefix/fuzzy search.
- [ ] Hierarchical result paths.
- [ ] Recent/frequent ranking.
- [ ] Project discovery providers.
- [ ] Action search.
- [ ] Capability-aware filtering.
- [ ] Search performance budget.

### W10 — Testing

- [ ] Unit tests.
- [ ] Property tests for hierarchy/reconciler.
- [ ] Axiom model tests.
- [ ] Integration contract tests.
- [ ] Fake desktop/session adapter.
- [ ] Golden tests for layouts/tree projections.
- [ ] Race tests.
- [ ] Fuzz tests for config/IPC/state transitions.
- [ ] Mutation testing for critical pure Go packages.
- [ ] Fault injection for activities and storage.
- [ ] Restart/recovery tests.
- [ ] End-to-end nested-session tests where feasible.

### W11 — Observability

- [ ] Structured event model.
- [ ] Correlation ID: UI intent → Axiom execution → activities.
- [ ] Latency metrics.
- [ ] Reconcile timings.
- [ ] Activity retry metrics.
- [ ] Explain panel.
- [ ] Diagnostics bundle.
- [ ] Privacy/redaction policy.

### W12 — Security

- [ ] Threat model.
- [ ] Privilege boundary document.
- [ ] No root daemon by default.
- [ ] Explicit handling of privileged actions.
- [ ] Command execution policy: argv arrays, no implicit shell.
- [ ] Environment sanitization.
- [ ] Path traversal protections.
- [ ] Symlink/race considerations.
- [ ] Secrets never stored in workspace definitions.
- [ ] IPC peer/session validation.
- [ ] plugin/provider trust model.

## 4. Первый vertical slice

### VS-1: Local Development Workspace

Иерархия:

```text
DEV
└── Projects
    └── HWS
        └── Develop
```

Ресурсы:

- editor;
- terminal в repo cwd;
- optional browser/documentation window.

Axiom lifecycle:

```text
Inactive
  → Preparing
  → Active
  → Suspending
  → Inactive
```

Error branches:

```text
Preparing → Degraded
Preparing → Failed
Active → Degraded
Degraded → Recovering → Active
```

Acceptance tests:

- [ ] repeated Activate не создаёт лишние процессы;
- [ ] activity retry не запускает неидемпотентный side effect повторно без key;
- [ ] missing capability приводит к Degraded/Failed согласно policy;
- [ ] crash `hwsd` между activities обнаруживается после restart;
- [ ] Explain сообщает, какая activity/claim остановила переход;
- [ ] Close не завершает adopted/external resource без явного ownership.

## 5. Performance budgets — начальные цели

Это проектные цели, а не измеренные характеристики.

- Grid navigation, локальный cached path: p95 < 16 ms до обновления UI state.
- Поиск по локальному индексу: p95 < 50 ms для типичного пользовательского дерева.
- IPC read query: p95 < 20 ms без внешних providers.
- Активация workspace не должна блокировать UI thread.
- Долгие activities отображаются как асинхронный progress/state.
- Idle `hwsd` должен избегать polling loops там, где доступны события/watchers.

После появления прототипа цели заменяются измеренными baseline.

## 6. Architectural guardrails

Запрещено без ADR:

- переносить эфемерное UI-состояние в durable Axiom execution;
- давать UI прямой доступ к system side effects в обход `hwsd`;
- выполнять произвольные shell-строки из workspace config;
- связывать domain model напрямую с GNOME-specific типами;
- считать успешным desired state без проверки observed state для критичных ресурсов;
- удалять/убивать неуправляемые ресурсы ради «синхронизации»;
- использовать process-local lock как гарантию межпроцессной эксклюзивности;
- обещать exactly-once внешний side effect;
- скрывать partial failure под общим статусом `Active`;
- добавлять AI-автоорганизацию, способную молча менять пользовательскую иерархию.

## 7. Milestones

### M0 — Architecture bootstrap

- документация;
- ADR;
- domain contracts;
- Axiom boundary;
- skeleton Go module.

### M1 — Headless prototype

- дерево;
- workspace model;
- Axiom lifecycle;
- fake integrations;
- CLI demo.

### M2 — Desktop vertical slice

- shell UI;
- IPC;
- app/window adapter;
- VS-1 end-to-end.

### M3 — Durable/recovery

- production durable store;
- restart recovery;
- diagnostics/history/explain.

### M4 — Productization

- packaging;
- installer/uninstaller;
- compatibility matrix;
- CI integration tests;
- accessibility;
- performance baselines.

### M5 — Extensibility

- providers/plugins;
- dynamic nodes;
- sync/export;
- optional recommendation layer.

## 8. Current next actions

1. Завершить начальный documentation pack.
2. Зафиксировать ADR по shell-overlay стратегии.
3. Зафиксировать ADR по Axiom orchestration boundary.
4. Создать Go module и package skeleton.
5. Реализовать headless hierarchy prototype.
6. Реализовать первую Axiom workspace model с fake activities.
7. Проверить recovery/idempotency до подключения реальных OS integrations.

## 9. Change discipline

Каждая завершённая итерация должна:

1. синхронизировать этот план с реальным состоянием;
2. иметь тестируемый результат;
3. не объявлять неподтверждённые возможности реализованными;
4. фиксировать новые архитектурные ограничения;
5. по возможности оставлять `main` в собираемом и проверяемом состоянии.