# HWS MASTER PLAN

Единый living execution plan проекта HWS.

Правило: если архитектурная находка, ограничение или новый риск меняет последовательность работ, сначала обновляется этот файл, затем код/документация. Отмечать `[x]` разрешено только для реализованного и проверенного результата, а не для одной документации о будущем поведении.

## 0. Цель продукта

Создать иерархическую рабочую оболочку для Ubuntu, которая организует работу вокруг контекстов и задач, а не вокруг списка приложений.

```text
Area → Context → Project → Task → Ready Workspace
```

HWS должен быстро навигировать по иерархии, восстанавливать рабочее окружение, управлять значимыми side effects через проверяемую модель состояния и оставаться безопасно отключаемым слоем поверх базовой графической системы.

## 1. Архитектурные решения

- [x] Основной daemon пишется на Go.
- [x] UI и durable orchestration разделены.
- [x] Иерархия имеет произвольную глубину.
- [x] Axiom используется для значимых проверяемых переходов и orchestration.
- [x] Эфемерное UI-состояние не проводится через Axiom.
- [x] На первом этапе HWS не форкает compositor/window manager.
- [x] Интеграции с ОС скрываются за adapter boundaries.
- [x] Desired state и observed state разделены.
- [x] Внешние side effects проектируются как idempotent/recoverable; exactly-once не обещается.
- [x] GNOME Shell extension является тонким UI/compositor adapter; тяжёлая логика живёт в `hwsd`.
- [x] Shell ↔ `hwsd` используют versioned D-Bus boundary.
- [x] Первый desktop target: Ubuntu 26.04 LTS / GNOME Shell 50 / Wayland.
- [x] X11-only tooling не используется как core GNOME integration path.
- [x] Desktop app launch/activation отделены от generic process launch.
- [x] PID/title не считаются durable window identity.
- [x] Layout domain использует logical/normalized coordinates.
- [x] Monitor topology рассматривается как versioned observed state.
- [x] Значимые компромиссы фиксируются ADR.

## 2. Текущее реализованное состояние

На `main` реализован первый headless vertical slice:

```text
Hierarchy/domain
      ↓
Desired ↔ Observed diff
      ↓
Reconciler
      ↓
Fake Desktop adapter
      ↓
Axiom WorkspaceLifecycle
      ↓
Headless hwsctl demo
```

Фактически присутствуют:

- Go module с минимальной версией Go 1.26;
- закреплённый pseudo-version Axiom на commit `e6a7991b5010bd39875b8f7256850d462c2a0bc6`;
- `go.sum` и воспроизводимый module graph;
- stable ID types;
- дерево произвольной глубины с cycle/missing-parent validation;
- deterministic child ordering и path resolution;
- desired/observed workspace state;
- resource ownership `managed/adopted/external`;
- normalized placement rectangles;
- deterministic reconcile plan/evaluation;
- desktop adapter interface;
- deterministic fake adapter с failure injection;
- reconciler `observe → diff → ensure → observe → evaluate`;
- Axiom `HWSWorkspaceLifecycle` с typed activities;
- operation-key based idempotency boundary;
- headless CLI smoke path;
- versioned IPC protocol primitives и D-Bus contract documentation;
- CI: module graph, gofmt, unit tests, race, vet, CLI smoke.

Ограничение: fake adapter доказывает domain/orchestration semantics, но **не** доказывает готовность GNOME integration.

## 3. Definition of Done для MVP

MVP считается готовым только если один реальный workspace проходит полный цикл в Ubuntu 26.04 / GNOME 50 / Wayland:

1. HWS запускается в пользовательской графической сессии.
2. `Super` открывает иерархическую сетку.
3. Пользователь проходит минимум 3 уровня иерархии.
4. Выбор task создаёт revisioned desired workspace state.
5. Переход проходит через Axiom model/claims.
6. HWS запускает минимум два разных ресурса.
7. Desktop app запускается/активируется через GNOME/Wayland semantics, а не только generic exec.
8. Уже существующие ресурсы распознаются без blind duplicate.
9. Window association использует Shell/application semantics.
10. Layout применяется относительно актуальной monitor topology либо возвращается объяснимая capability error.
11. Workspace можно suspend/resume/close/recover согласно формальной semantics.
12. История операции доступна для диагностики.
13. После рестарта `hwsd` durable operation/state остаются объяснимыми.
14. После D-Bus owner change Shell делает fresh handshake и snapshot.
15. Extension переживает многократные enable/disable без leaks.
16. Отключение HWS не ломает базовую GNOME-сессию.

## 4. Workstreams

### W0 — Repository foundation

- [ ] Определить лицензию и проверить совместимость с GNOME Extensions distribution.
- [x] Создать Go module.
- [x] Зафиксировать Go 1.26 как минимальную версию текущего прототипа.
- [x] Добавить Makefile.
- [x] Добавить CI: module graph, fmt, unit, race, vet, smoke.
- [ ] Добавить vulnerability scan.
- [ ] Добавить CONTRIBUTING.md.
- [ ] Добавить top-level SECURITY.md.
- [x] Добавить ADR template.
- [ ] Зафиксировать CODEOWNERS/branch protection policy.

### W1 — Domain hierarchy

- [x] `NodeID`, `WorkspaceID`, `ResourceID`, `ActionID`.
- [x] Типы узлов category/project/task/action/widget/query.
- [x] Дерево произвольной глубины.
- [x] Проверка duplicate IDs, missing parents и cycles.
- [x] Stable deterministic ordering.
- [x] Selection/path resolution.
- [ ] Breadcrumb/context-stack service поверх path model.
- [ ] Dynamic children provider rules.
- [ ] Snapshot/revision дерева.
- [ ] Property tests для дерева/path invariants.

### W2 — Workspace state / reconciliation

- [x] Реализовать desired state.
- [x] Реализовать observed state.
- [x] Реализовать resource ownership managed/adopted/external.
- [x] Реализовать normalized placement intent.
- [x] Реализовать deterministic desired ↔ observed diff.
- [x] Реализовать reconcile evaluation required/optional resources.
- [x] Реализовать partial/degraded classification.
- [x] Close path не удаляет adopted/external ресурсы в fake adapter/reconciler semantics.
- [ ] Разделить readiness на process/window/layout/focus sub-observations.
- [ ] Реализовать monitor topology revision semantics, а не только поле в observed model.
- [ ] Реализовать window identity/association model.
- [ ] Определить compensation policy для non-idempotent actions.

### W3 — Axiom integration

- [x] Добавить `github.com/Homiakus/axiom`.
- [x] Закрепить конкретный pre-v1 commit через pseudo-version.
- [x] Использовать declarative Go `model` frontend.
- [x] Реализовать `HWSWorkspaceLifecycle` prototype.
- [x] Реализовать Activate/Recover/Resume/Suspend/Close events.
- [x] Реализовать states Inactive/Preparing/Active/Degraded/Recovering/Closing/Failed.
- [x] Реализовать базовые count/identity claims.
- [x] Реализовать typed fake-side-effect activities.
- [x] Использовать operation key как activity idempotency boundary.
- [x] Добавить lifecycle unit tests для activation и ownership-safe close.
- [x] Добавить recovery/idempotent repeated-activation tests.
- [ ] Уточнить formal Suspend semantics; текущий prototype не должен путать inactive и оставленные running resources.
- [ ] Добавить ownership/capability claims непосредственно в lifecycle model там, где они действительно model-level.
- [ ] Разделить generic process и Shell desktop/window activities.
- [ ] Перевести production path на transactional durable store.
- [ ] Добавить reopen/restart continuation tests.
- [ ] Добавить history/replay compatibility tests.
- [ ] Вывести `Run.Explain` в service/API.

### W4 — `hwsd`

- [ ] Создать реальный `cmd/hwsd` daemon.
- [ ] Single-instance policy через D-Bus well-known name.
- [ ] Structured logging.
- [ ] Health/status method.
- [ ] Config loader.
- [ ] Durable state directory policy.
- [ ] Axiom engine lifecycle в daemon.
- [ ] Context service.
- [ ] Workspace service.
- [x] Headless reconciler core реализован как application package.
- [ ] Integration registry.
- [ ] Graceful shutdown.
- [ ] Crash recovery bootstrap.
- [ ] systemd --user / D-Bus activation.
- [ ] graphical-session lifecycle integration.

### W5 — IPC / D-Bus

- [x] Выбрать session D-Bus как основной Shell ↔ `hwsd` transport.
- [x] Зафиксировать `org.homiakus.HWS1`, object path и protocol v1 в контракте.
- [x] Реализовать pure-Go protocol constants/Hello/cache identity/mutation request validation.
- [x] Зафиксировать handshake: protocol + daemon instance + revision epoch + capabilities.
- [x] Зафиксировать daemon owner change/stale-cache semantics.
- [x] Зафиксировать operation key ≠ operation tracking ID.
- [ ] Реализовать D-Bus server transport.
- [ ] Реализовать `Hello`.
- [ ] Реализовать `GetTree` / `GetPath`.
- [ ] Реализовать `ActivateWorkspace` / `RecoverWorkspace` / `SuspendWorkspace` / `ResumeWorkspace` / `CloseWorkspace`.
- [ ] Реализовать `GetWorkspace` / `GetOperation` / `GetCapabilities`.
- [ ] Реализовать `Search`.
- [ ] Реализовать signals/subscriptions.
- [ ] Реализовать bounded payload limits.
- [ ] Contract tests на реальном session bus.

### W6 — GNOME 50 Shell adapter/UI

- [ ] Thin extension entry point.
- [ ] Полностью reversible enable/disable ownership model.
- [ ] D-Bus client/reconnect state machine.
- [ ] Home/Grid mode.
- [ ] Focus mode.
- [ ] Dynamic row generation.
- [ ] Keyboard/pointer/touch navigation.
- [ ] Breadcrumbs.
- [ ] Global search.
- [ ] Loading/degraded/error states.
- [ ] Accessibility names/focus states.
- [ ] Animation/main-loop budget.
- [ ] `Shell.WindowTracker` application association.
- [ ] user-context desktop app launch/activation executor.
- [ ] Meta.Window capability/geometry adapter.
- [ ] monitor topology observer.
- [ ] Multi-monitor behavior.
- [ ] `user` session mode only for initial releases.
- [ ] No filesystem/network/database/Axiom work in Shell process.

### W7 — OS integration

- [ ] Desktop application discovery.
- [ ] Generic process launch/adoption.
- [ ] Terminal launch with cwd.
- [ ] Desktop app launch/activation as separate path.
- [ ] Window discovery/association.
- [ ] Focus request + observed confirmation.
- [ ] Logical/normalized → current topology geometry resolution.
- [ ] filesystem/project discovery.
- [ ] Git status provider.
- [ ] SSH provider.
- [ ] network/VPN capability boundary.
- [ ] portals/screen-sharing compatibility.
- [ ] Ubuntu default-extension compatibility scanner.
- [ ] third-party conflict classification without silent disable.

### W8 — Persistence / recovery

- [ ] Pebble-backed Axiom store configuration.
- [ ] ProductionMode path with transactional store.
- [ ] Durable workspace definition/config schema.
- [ ] Context tree persistence.
- [ ] Workspace snapshots.
- [ ] Last active context.
- [ ] Restart bootstrap: fresh observed state before recovery decision.
- [ ] Crash-between-side-effect-and-checkpoint tests.
- [ ] Store reopen/retry tests.
- [ ] Migration/version policy.
- [ ] Corruption handling.
- [ ] backup/export.
- [ ] Никогда не сохранять session-local Meta window ID как durable identity.

### W9 — Search/discovery

- [ ] Prefix/fuzzy search.
- [ ] Hierarchical result paths.
- [ ] Recent/frequent ranking.
- [ ] Project discovery providers.
- [ ] Action search.
- [ ] Capability-aware filtering.
- [ ] Direct-jump navigation.
- [ ] Search performance baseline.

### W10 — Testing

- [x] Unit tests для текущего headless slice.
- [x] Fake desktop adapter.
- [x] Fake failure injection.
- [x] Axiom lifecycle tests.
- [x] Repeated reconcile/no-duplicate test.
- [x] Ownership-safe close test.
- [x] Race test в CI для текущего Go codebase.
- [x] IPC protocol primitive tests.
- [ ] Property tests hierarchy/reconciler.
- [ ] Fuzz config/IPC/state transitions.
- [ ] Mutation testing critical pure packages.
- [ ] Durable activity/storage fault injection.
- [ ] Restart/recovery tests.
- [ ] Real D-Bus contract tests.
- [ ] GNOME 50 nested Wayland tests.
- [ ] 100x extension enable/disable lifecycle stress.
- [ ] hwsd owner restart tests.
- [ ] splash/modal/delayed/main-window/close-during-action races.
- [ ] focus-policy matrix.
- [ ] scale 100/125/150/200% matrix.
- [ ] mixed-scale/multi-monitor topology tests.
- [ ] secondary-monitor-left-of-primary test.
- [ ] Ubuntu default extensions compatibility profile.
- [ ] keyboard-only/high-contrast/large-text checks.

### W11 — Observability

- [ ] Structured event model.
- [ ] Correlation chain `intent → Axiom execution → activity → adapter`.
- [ ] Reconcile timings.
- [ ] Activity retry metrics.
- [ ] D-Bus reconnect metrics.
- [ ] Explain panel/API.
- [ ] Diagnostics bundle.
- [ ] `hwsctl doctor`.
- [ ] Redaction policy implementation.

### W12 — Security

- [x] Начальный threat model.
- [x] Начальный privilege boundary.
- [x] No-root-by-default invariant.
- [x] argv arrays / no implicit shell invariant.
- [x] Secrets excluded from workspace definition invariant.
- [ ] Environment sanitization implementation.
- [ ] Path traversal/symlink race protections.
- [ ] D-Bus peer/session validation.
- [ ] plugin/provider trust model.
- [ ] privileged action boundary, если вообще понадобится.
- [ ] Lock-screen integration отсутствует по умолчанию.

### W13 — Compatibility / release engineering

- [x] Ubuntu 26.04 / GNOME 50 / Wayland first-class target.
- [ ] `COMPATIBILITY_MATRIX.md`.
- [ ] Ubuntu default extension test profile.
- [ ] known-conflicts database.
- [ ] GNOME porting-guide review checklist.
- [ ] qualification gate перед добавлением нового GNOME major в metadata.
- [ ] RC/canary stage.
- [ ] Safe mode / emergency disable path.

## 5. VS-1 — Local Development Workspace

Target hierarchy:

```text
DEV
└── Projects
    └── HWS
        └── Develop
```

Target resources:

- editor как desktop application;
- terminal в repo cwd;
- optional documentation/browser window.

### Уже доказано headless tests

- [x] desired state валидируется;
- [x] repeated reconcile не вызывает лишний Ensure для уже reached resource;
- [x] repeated activation не должна дублировать resource ensures;
- [x] partial required-resource failure классифицируется как Degraded;
- [x] explicit Recover повторно reconciles и может вернуть workspace в Active;
- [x] Close удаляет только managed resources;
- [x] Axiom lifecycle реально компилируется/исполняется на Go 1.26;
- [x] CLI smoke достигает `workspace=local-dev status=active required=2/2` на fake adapter.

### Ещё не доказано на настоящем desktop

- [ ] already-running editor корректно adopt/activate;
- [ ] PID/title mismatch не ломает window matching;
- [ ] focus denial не маскируется как success;
- [ ] fractional scaling/layout корректны;
- [ ] monitor topology change во время activation вызывает re-resolution;
- [ ] daemon restart не оставляет stale Shell snapshot;
- [ ] crash между внешним effect и durable checkpoint корректно восстанавливается;
- [ ] `Explain` показывает причину остановки transition.

## 6. Performance budgets

Пока это target, не измеренный baseline:

- Grid cached navigation p95 < 16 ms.
- Local search p95 < 50 ms.
- IPC local read p95 < 20 ms без внешних providers.
- Workspace activation не блокирует Shell main loop.
- Shell не выполняет synchronous filesystem/network/database I/O.
- Window/monitor event storms coalesce до bounded update rate.
- Idle `hwsd` избегает polling там, где есть events/watchers.

После появления GNOME prototype эти цели заменяются измеренными baseline/threshold.

## 7. Architectural guardrails

Без ADR запрещено:

- переносить hover/focus/animation в durable Axiom execution;
- выполнять system side effects напрямую из UI в обход intent/orchestration boundary;
- превращать GNOME extension в основной business process;
- делать sync filesystem/network/database I/O в Shell hot path;
- выполнять arbitrary shell strings из workspace config;
- импортировать GNOME-specific types в domain;
- строить GNOME 50 integration на X11-only tooling;
- использовать PID/title как durable window identity;
- считать launch доказательством window/focus/layout success;
- вручную размножать scale-factor math по domain code;
- объявлять Active без observed confirmation required resources;
- replay absolute rectangles после topology change без re-resolution;
- закрывать adopted/external resources как managed;
- использовать process-local lock как межпроцессную гарантию;
- обещать exactly-once внешний effect;
- скрывать partial failure под Active;
- молча отключать чужие extensions;
- заявлять GNOME major support до qualification suite;
- включать lock-screen mode без ADR/security review;
- позволять automation/AI молча менять canonical user hierarchy.

## 8. Milestones

### M0 — Architecture bootstrap — DONE

- [x] базовая документация;
- [x] ADR foundation;
- [x] domain contracts;
- [x] Axiom boundary;
- [x] GNOME failure-mode research;
- [x] GNOME 50 adapter contract;
- [x] Go module/package skeleton;
- [x] baseline CI.

### M1 — Headless prototype — FUNCTIONAL

- [x] hierarchy;
- [x] workspace model;
- [x] deterministic reconcile;
- [x] fake desktop integration;
- [x] Axiom lifecycle;
- [x] CLI demo;
- [x] unit/race/vet baseline;
- [ ] durable restart/reopen proof — переносится в M1.5 перед real desktop mutation.

### M1.5 — Durable orchestration gate

Этот milestone обязателен **до** подключения реального управления окнами.

- [ ] Pebble transactional store;
- [ ] `WithProductionMode`;
- [ ] restart/reopen test;
- [ ] crash-after-effect/before-checkpoint scenario;
- [ ] history/replay test;
- [ ] Explain service;
- [ ] Axiom pinned-version compatibility test.

### M2 — Local daemon / D-Bus

- [ ] `hwsd`;
- [ ] D-Bus well-known name + Hello;
- [ ] workspace mutations and snapshots;
- [ ] operation tracking/signals;
- [ ] systemd user activation;
- [ ] owner restart/reconnect tests.

### M3 — Desktop vertical slice

- [ ] thin GNOME 50 extension;
- [ ] app/window observation/activation adapter;
- [ ] normalized layout against live topology;
- [ ] nested Wayland CI;
- [ ] VS-1 real desktop end-to-end.

### M4 — Productization

- [ ] packaging/install/uninstall/safe mode;
- [ ] compatibility matrix;
- [ ] Ubuntu default-extension profile;
- [ ] accessibility;
- [ ] performance baselines;
- [ ] security hardening.

### M5 — Extensibility

- [ ] providers/plugins;
- [ ] dynamic nodes;
- [ ] sync/export;
- [ ] optional recommendation layer.

## 9. Current next actions

1. Зафиксировать final green CI после расширенных lifecycle/IPC tests и module graph lock.
2. Добавить durable Pebble Axiom engine path и restart/reopen tests.
3. Добавить history/replay/Explain contract tests для закреплённого Axiom commit.
4. Создать `cmd/hwsd` и real session D-Bus server без GNOME-specific логики.
5. Реализовать Hello/revision epoch/owner-restart semantics.
6. Реализовать workspace mutation/read operations поверх существующего lifecycle.
7. Только после durable + D-Bus tests создать минимальный GNOME 50 extension.
8. Поднять nested GNOME 50 / Wayland lifecycle harness до сложного tiling UI.
9. Добавить safe disable/doctor path до активного window mutation MVP.

## 10. Change discipline

Каждая итерация должна:

1. синхронизировать этот plan с кодом;
2. иметь тестируемый результат;
3. не выдавать proposed capability за implemented;
4. фиксировать обнаруженные ограничения;
5. оставлять `main` собираемым/проверяемым либо явно фиксировать известный CI blocker до его устранения.

## 11. Research inputs

- [`docs/DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](docs/DEVELOPER_PRACTICES_AND_FAILURE_MODES.md)
- [`docs/GNOME_ADAPTER.md`](docs/GNOME_ADAPTER.md)
- [`docs/AXIOM_INTEGRATION.md`](docs/AXIOM_INTEGRATION.md)
- [`docs/IPC_CONTRACT.md`](docs/IPC_CONTRACT.md)
- [`docs/INVARIANTS.md`](docs/INVARIANTS.md)
- [`docs/adr/0004-thin-shell-dbus-boundary.md`](docs/adr/0004-thin-shell-dbus-boundary.md)
- [`docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md`](docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md)
