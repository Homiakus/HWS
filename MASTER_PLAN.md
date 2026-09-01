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
- [x] GNOME Shell extension является тонким UI/compositor adapter; тяжёлая логика живёт в `hwsd`.
- [x] Shell ↔ `hwsd` используют versioned D-Bus boundary.
- [x] Первый desktop target: Ubuntu 26.04 LTS / GNOME Shell 50 / Wayland.
- [x] X11-only tooling не используется как core GNOME integration path.
- [x] Desktop app launch/activation отделены от generic process launch.
- [x] PID/title не считаются durable window identity.
- [x] Layout domain использует logical/normalized coordinates.
- [x] Monitor topology является versioned observed state и вызывает reconcile при изменении.

## 2. Definition of Done для MVP

MVP считается готовым только если один реальный workspace проходит полный цикл:

1. HWS запускается в пользовательской сессии Ubuntu 26.04 / GNOME 50 / Wayland.
2. `Super` открывает иерархическую сетку.
3. Пользователь проходит минимум 3 уровня иерархии.
4. Выбор task создаёт desired workspace state.
5. Переход проходит через Axiom model/claims.
6. HWS запускает минимум два разных ресурса (например editor + terminal).
7. Desktop app запускается/активируется через корректную GNOME/Wayland semantics, а не только generic exec.
8. HWS распознаёт уже запущенные ресурсы и не создаёт дубликаты без причины.
9. HWS сопоставляет window через Shell/application semantics и подтверждает observed window state.
10. HWS применяет layout либо возвращает объяснимую ошибку capability.
11. Workspace можно закрыть и восстановить.
12. История операции доступна для диагностики.
13. После рестарта `hwsd` durable операция/состояние не становится неописанным.
14. После disappearance/reappearance D-Bus owner Shell выполняет fresh handshake и snapshot.
15. Extension переживает многократные enable/disable без signal/actor/source leaks.
16. Отключение HWS не делает базовую графическую сессию неработоспособной.

## 3. Workstreams

### W0 — Repository foundation

- [ ] Определить лицензию.
- [ ] Создать Go module.
- [ ] Зафиксировать минимальную поддерживаемую версию Go.
- [ ] Добавить Makefile/Taskfile.
- [ ] Добавить CI: fmt, vet, test, race, vulnerability scan.
- [ ] Добавить CONTRIBUTING.md.
- [ ] Добавить SECURITY.md.
- [x] Добавить ADR template.
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

- [x] Описать desired state на уровне архитектурного контракта.
- [x] Описать observed state на уровне архитектурного контракта.
- [x] Описать reconciliation result на уровне архитектурного контракта.
- [x] Описать resource lifecycle на уровне архитектурного контракта.
- [x] Определить partial/degraded states.
- [x] Определить ownership ресурсов: managed/adopted/external.
- [x] Определить правила закрытия workspace без убийства чужих процессов.
- [ ] Реализовать deterministic diff desired ↔ observed.
- [ ] Реализовать monitor topology revision semantics.
- [ ] Реализовать window observation identity model.

### W3 — Axiom integration

- [ ] Добавить зависимость `github.com/Homiakus/axiom` через отдельный integration package.
- [ ] На ранней стадии закрепить конкретную совместимую версию/commit Axiom и обновлять осознанно.
- [ ] Использовать declarative Go `model` как основной frontend.
- [x] Описать model `WorkspaceLifecycle`.
- [x] Описать events: Activate, Reconcile, Suspend, Resume, Close, Recover.
- [x] Описать states: Inactive, Preparing, Active, Degraded, Suspending, Recovering, Failed.
- [x] Описать claims для ownership, capability и safety.
- [ ] Реализовать typed activities для OS side effects.
- [ ] Разделить generic process activities и Shell desktop-app/window activities.
- [ ] В production path использовать transactional durable store.
- [ ] Зафиксировать idempotency keys для каждой activity.
- [ ] Добавить failure injection tests.
- [ ] Добавить replay/history tests.
- [ ] Добавить explain diagnostics в UI/API.

### W4 — `hwsd`

- [ ] Bootstrap daemon lifecycle.
- [ ] Single-instance policy через D-Bus well-known name/service ownership.
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
- [ ] systemd --user/D-Bus activation integration.
- [ ] graphical-session lifecycle integration.

### W5 — IPC contract

- [x] Выбрать D-Bus как основной local session transport Shell ↔ `hwsd`.
- [ ] Сформировать versioned API.
- [ ] Handshake: protocol version + daemon instance + capabilities + revision epoch.
- [ ] `GetTree`.
- [ ] `SelectNode`.
- [ ] `ActivateWorkspace`.
- [ ] `GetWorkspaceState`.
- [ ] `GetHistory`.
- [ ] `Explain`.
- [ ] `Search`.
- [ ] event stream/subscriptions.
- [ ] capability discovery.
- [ ] daemon owner disappearance/reappearance semantics.
- [ ] stale snapshot invalidation rules.
- [ ] bounded timeout/cancellation rules.
- [ ] contract tests.

### W6 — Shell UI / GNOME 50 adapter

- [ ] Home/Grid mode.
- [ ] Focus mode.
- [ ] Dynamic row generation.
- [ ] Keyboard navigation.
- [ ] Touch/mouse navigation.
- [ ] Breadcrumbs.
- [ ] Global search.
- [ ] Loading/degraded/error states.
- [ ] Accessibility model.
- [ ] Accessible name/focus state for every interactive tile.
- [ ] Animation budget.
- [ ] Focus handling.
- [ ] Multi-monitor behavior.
- [ ] Thin entry point and process-isolated module structure.
- [ ] Per-component ownership of signals/sources/cancellables/actors.
- [ ] Full reversible `enable()`/`disable()`.
- [ ] No synchronous filesystem/network/database work in Shell process.
- [ ] D-Bus reconnect/handshake state machine.
- [ ] Shell application launch/activation executor with user timestamp/context.
- [ ] `Shell.WindowTracker`-based window/application association.
- [ ] Meta.Window capability/geometry adapter.
- [ ] monitor topology observer with revisions.
- [ ] `user` session mode only for initial releases.

### W7 — OS integration

- [ ] Application discovery via desktop application semantics.
- [ ] Generic process launch/adoption.
- [ ] Desktop app launch/activation as separate path.
- [ ] Terminal launch with cwd.
- [ ] Window discovery.
- [ ] Window observation model: appID/startupID/WM_CLASS/PID hints.
- [ ] Window intent/layout adapter.
- [ ] Focus request + observed confirmation.
- [ ] Logical/normalized geometry conversion.
- [ ] systemd user services.
- [ ] filesystem/project discovery.
- [ ] Git status provider.
- [ ] SSH integration.
- [ ] network/VPN capabilities только через безопасный adapter boundary.
- [ ] portals/screen sharing compatibility checks.
- [ ] Ubuntu default-extension compatibility scanner.
- [ ] third-party extension conflict classification without silent disable.

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
- [ ] Never persist session-local Meta window identifiers as durable identity.
- [ ] Fresh monitor/window observation after Shell or daemon restart.

### W9 — Search and discovery

- [ ] Prefix/fuzzy search.
- [ ] Hierarchical result paths.
- [ ] Recent/frequent ranking.
- [ ] Project discovery providers.
- [ ] Action search.
- [ ] Capability-aware filtering.
- [ ] Search performance budget.
- [ ] Direct-jump results to reduce cognitive depth of deep hierarchy.

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
- [ ] GNOME 50 nested Wayland tests through mutter-devkit/gnome-shell test tooling.
- [ ] 100x extension enable/disable lifecycle stress.
- [ ] hwsd D-Bus owner restart tests.
- [ ] window race fixtures: splash/modal/delayed/main-window/close-during-action.
- [ ] focus matrix: click/mouse/sloppy/fullscreen/modal.
- [ ] scale matrix: 100/125/150/200%.
- [ ] multi-monitor mixed-scale and topology-change tests.
- [ ] secondary-monitor-left-of-primary geometry test.
- [ ] Ubuntu 26.04 default extensions compatibility profile.
- [ ] known conflicting extension profile.
- [ ] keyboard-only/high-contrast/large-text/accessibility checks.

### W11 — Observability

- [ ] Structured event model.
- [ ] Correlation ID: UI intent → Axiom execution → activities → Shell adapter action.
- [ ] Latency metrics.
- [ ] Reconcile timings.
- [ ] Activity retry metrics.
- [ ] Shell lifecycle/reconnect metrics without excessive Shell logging.
- [ ] Explain panel.
- [ ] Diagnostics bundle.
- [ ] `hwsctl doctor`.
- [ ] Privacy/redaction policy.

### W12 — Security

- [x] Начальный threat model.
- [x] Начальный privilege boundary document.
- [x] No root daemon by default как invariant.
- [ ] Explicit handling of privileged actions.
- [x] Command execution policy: argv arrays, no implicit shell.
- [ ] Environment sanitization.
- [ ] Path traversal protections.
- [ ] Symlink/race considerations.
- [x] Secrets never stored in workspace definitions.
- [ ] IPC peer/session validation.
- [ ] plugin/provider trust model.
- [ ] Lock-screen integration отсутствует по умолчанию.

### W13 — Compatibility & release engineering

- [x] Зафиксировать Ubuntu 26.04 / GNOME 50 / Wayland как first-class target.
- [ ] Создать `COMPATIBILITY_MATRIX.md`.
- [ ] Зафиксировать Ubuntu default extension set для test profile.
- [ ] Ввести known-conflicts database/rules.
- [ ] Добавить GNOME porting-guide review checklist.
- [ ] CI MAY тестировать next GNOME development version без product support claim.
- [ ] Добавление GNOME major в supported metadata возможно только после qualification suite.
- [ ] Release-candidate/canary stage перед desktop production release.
- [ ] Safe mode / disable path до первой публичной alpha.

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

- editor как desktop application;
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

Desktop activation sequence:

```text
Desired resource
→ ensure/activate desktop app
→ await matching window
→ resolve monitor/workspace against current topology
→ place window
→ request focus if policy requires
→ observe result
→ report reached/degraded/failed
```

Acceptance tests:

- [ ] repeated Activate не создаёт лишние процессы;
- [ ] activity retry не запускает неидемпотентный side effect повторно без key;
- [ ] missing capability приводит к Degraded/Failed согласно policy;
- [ ] crash `hwsd` между activities обнаруживается после restart;
- [ ] Explain сообщает, какая activity/claim остановила переход;
- [ ] Close не завершает adopted/external resource без явного ownership;
- [ ] already-running editor корректно adopt/activate без blind duplicate;
- [ ] PID/title mismatch не ломает window matching;
- [ ] focus denial не маскируется как success;
- [ ] monitor topology change во время activation вызывает re-resolution;
- [ ] daemon restart не заставляет extension использовать stale snapshot.

## 5. Performance budgets — начальные цели

Это проектные цели, а не измеренные характеристики.

- Grid navigation, локальный cached path: p95 < 16 ms до обновления UI state.
- Поиск по локальному индексу: p95 < 50 ms для типичного пользовательского дерева.
- IPC read query: p95 < 20 ms без внешних providers.
- Активация workspace не должна блокировать UI thread.
- Shell process не выполняет synchronous disk/network/database I/O в interaction path.
- Event storms от window/monitor signals должны coalesce до bounded update rate.
- Долгие activities отображаются как асинхронный progress/state.
- Idle `hwsd` должен избегать polling loops там, где доступны события/watchers.

После появления прототипа цели заменяются измеренными baseline.

## 6. Architectural guardrails

Запрещено без ADR:

- переносить эфемерное UI-состояние в durable Axiom execution;
- давать UI прямой доступ к system side effects в обход `hwsd` intent/orchestration boundary;
- превращать GNOME extension в основной business/infrastructure process;
- выполнять синхронный filesystem/network/database I/O в Shell hot path;
- выполнять произвольные shell-строки из workspace config;
- связывать domain model напрямую с GNOME-specific типами;
- строить GNOME 50 path на X11-only tooling;
- считать PID/title durable identity окна;
- считать успешный launch доказательством window/focus/layout success;
- вручную распространять monitor scale multiplication в domain/layout logic;
- считать успешным desired state без проверки observed state для критичных ресурсов;
- replay absolute window rectangles после topology change без re-resolution;
- удалять/убивать неуправляемые ресурсы ради «синхронизации»;
- использовать process-local lock как гарантию межпроцессной эксклюзивности;
- обещать exactly-once внешний side effect;
- скрывать partial failure под общим статусом `Active`;
- молча отключать чужие GNOME extensions;
- заявлять поддержку GNOME major до qualification tests;
- включать lock-screen session mode без отдельного ADR/security review;
- добавлять AI-автоорганизацию, способную молча менять пользовательскую иерархию.

## 7. Milestones

### M0 — Architecture bootstrap

- [x] базовая документация;
- [x] ADR foundation;
- [x] domain contracts;
- [x] Axiom boundary;
- [x] GNOME developer failure-mode research;
- [x] GNOME 50 adapter contract;
- [ ] skeleton Go module.

### M1 — Headless prototype

- дерево;
- workspace model;
- Axiom lifecycle;
- fake integrations;
- CLI demo.

### M2 — Desktop vertical slice

- thin Shell UI;
- versioned D-Bus;
- GNOME 50 app/window adapter;
- nested Wayland CI;
- VS-1 end-to-end.

### M3 — Durable/recovery

- production durable store;
- restart recovery;
- daemon owner reconnect;
- diagnostics/history/explain.

### M4 — Productization

- packaging;
- installer/uninstaller/safe mode;
- compatibility matrix;
- Ubuntu default-extension test profile;
- CI integration tests;
- accessibility;
- performance baselines.

### M5 — Extensibility

- providers/plugins;
- dynamic nodes;
- sync/export;
- optional recommendation layer.

## 8. Current next actions

1. Создать Go module и package skeleton.
2. Реализовать pure Go hierarchy/domain types из `DOMAIN_MODEL.md`.
3. Реализовать deterministic desired ↔ observed diff.
4. Подключить pinned Axiom и реализовать первую `WorkspaceLifecycle` model с fake activities.
5. Создать versioned D-Bus `IPC_CONTRACT.md` до написания Shell client.
6. Создать fake GNOME adapter и failure/race fixtures.
7. Реализовать минимальный GNOME 50 extension только после прохождения headless tests.
8. Поднять nested GNOME 50 test harness и 100x lifecycle test до подключения сложного tiling UI.
9. Реализовать safe disable/doctor path до активного window mutation MVP.

## 9. Change discipline

Каждая завершённая итерация должна:

1. синхронизировать этот план с реальным состоянием;
2. иметь тестируемый результат;
3. не объявлять неподтверждённые возможности реализованными;
4. фиксировать новые архитектурные ограничения;
5. по возможности оставлять `main` в собираемом и проверяемом состоянии.

## 10. Research inputs

Архитектурные решения после анализа реальных GNOME проектов и официальных guidance зафиксированы в:

- [`docs/DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](docs/DEVELOPER_PRACTICES_AND_FAILURE_MODES.md);
- [`docs/GNOME_ADAPTER.md`](docs/GNOME_ADAPTER.md);
- [`docs/adr/0004-thin-shell-dbus-boundary.md`](docs/adr/0004-thin-shell-dbus-boundary.md);
- [`docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md`](docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md).
