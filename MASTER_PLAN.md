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
- [x] Классическая taskbar не является целевой моделью: HWS использует resource-oriented Activity Strip.
- [x] Одна application card по умолчанию представляет ApplicationSurface, а не отдельное окно.
- [x] Внутренние tabs/documents/sessions моделируются как provider-defined `SurfaceView`, а не угадываются из window title.
- [x] Rich view provider является capability; его отсутствие обязано деградировать к корректной window-only модели.
- [x] Пользовательская панель конфигурируется декларативным HCL-based DSL через normalized Panel Model, а не renderer-specific scripting.
- [x] Panel DSL не имеет arbitrary shell/filesystem/network side effects; действия идут как typed action IDs через `hwsd`.
- [x] Карточки используют semantic zoom `full → compact → micro` и constraint-based layout.

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

Документационно зафиксированы ApplicationSurface, Activity Strip и HCL-based Panel DSL, но их runtime-код ещё не реализован.

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
17. Focus Mode показывает window-only Activity Strip как минимум для активных desktop applications текущего контекста.
18. Несколько окон одного приложения представлены одной ApplicationSurface card и остаются индивидуально доступными через group navigator/MRU.
19. Отсутствие rich tab/view provider не приводит к ошибке Shell и явно отражается capability model.
20. Карточки имеют минимум `full/compact/micro` режимы и корректно деградируют при нехватке места.

Rich browser/IDE tabs и Panel DSL не блокируют первый desktop MVP, но обязательны до завершения M3.5.

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
- [ ] Surface provider registry.
- [ ] Surface aggregation service.
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
- [ ] Реализовать `GetSurfaces` / `GetSurface` window-only contract.
- [ ] Реализовать Surface changed/add/remove signals с bounded/coalesced payload.
- [ ] Реализовать typed `ActivateSurfaceWindow` / `ActivateSurfaceView` capability actions.
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
- [ ] Shell-side window event bridge для ApplicationSurface.
- [ ] user-context desktop app launch/activation executor.
- [ ] Meta.Window capability/geometry adapter.
- [ ] window preview capability/renderer с lazy creation.
- [ ] monitor topology observer.
- [ ] Multi-monitor behavior.
- [ ] window-only Activity Strip MVP.
- [ ] Application card semantic zoom full/compact/micro.
- [ ] window group navigator + MRU activation.
- [ ] `user` session mode only for initial releases.
- [ ] No filesystem/network/database/Axiom/provider-heavy work in Shell process.

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
- [ ] Никогда не сохранять tab/view index или provider UI handle как durable identity.
- [ ] Panel DSL schema/version/migration policy.

### W9 — Search/discovery

- [ ] Prefix/fuzzy search.
- [ ] Hierarchical result paths.
- [ ] Recent/frequent ranking.
- [ ] Project discovery providers.
- [ ] Action search.
- [ ] Capability-aware filtering.
- [ ] Direct-jump navigation.
- [ ] Surface/window/view search с privacy policy.
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
- [ ] ApplicationSurface 1 app / 1 window test.
- [ ] ApplicationSurface 1 app / N windows grouping test.
- [ ] N windows / N views provider aggregation test.
- [ ] provider disconnect/reconnect/stale snapshot tests.
- [ ] 20/100+ browser views stress tests.
- [ ] MRU determinism при close/reorder/focus races.
- [ ] no-provider window-only fallback test.
- [ ] invalid Panel DSL hot reload keeps previous valid layout.
- [ ] Activity Strip width matrix 800/1366/1920/4K + fractional scale.
- [ ] privacy/redaction tests для URLs/titles/document names.

### W11 — Observability

- [ ] Structured event model.
- [ ] Correlation chain `intent → Axiom execution → activity → adapter`.
- [ ] Reconcile timings.
- [ ] Activity retry metrics.
- [ ] D-Bus reconnect metrics.
- [ ] Surface provider health/staleness metrics.
- [ ] Surface aggregation/update timings.
- [ ] Explain panel/API.
- [ ] Diagnostics bundle.
- [ ] `hwsctl doctor`.
- [ ] Redaction policy implementation.
- [ ] Titles/URLs/document names не попадают в durable telemetry по умолчанию.

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
- [ ] browser extension минимальная permissions model.
- [ ] provider data schema validation / markup escaping.
- [ ] Panel DSL expression sandbox; v1 без arbitrary side effects.
- [ ] AT-SPI provider read-only до отдельного ADR на mutation.
- [ ] privileged action boundary, если вообще понадобится.
- [ ] Lock-screen integration отсутствует по умолчанию.

### W13 — Compatibility / release engineering

- [x] Ubuntu 26.04 / GNOME 50 / Wayland first-class target.
- [ ] `COMPATIBILITY_MATRIX.md`.
- [ ] Ubuntu default extension test profile.
- [ ] known-conflicts database.
- [ ] GNOME porting-guide review checklist.
- [ ] browser/IDE/terminal provider compatibility matrix.
- [ ] provider protocol versioning/compatibility policy.
- [ ] qualification gate перед добавлением нового GNOME major в metadata.
- [ ] RC/canary stage.
- [ ] Safe mode / emergency disable path.

### W14 — Application Surfaces / Activity Strip / Panel DSL

Detailed specification: [`docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md`](docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md).
ADR: [`docs/adr/0006-application-surface-and-panel-dsl.md`](docs/adr/0006-application-surface-and-panel-dsl.md).

#### AS-0 — Domain contracts [P0 before Activity Strip runtime]

- [ ] `ApplicationSurface`, `SurfaceWindow`, `SurfaceView`, `SurfaceProviderID`.
- [ ] Lifecycle/attention/activity/resource/media independent state axes.
- [ ] Provider-scoped/session-scoped identity model.
- [ ] Normalized immutable SurfaceSnapshot + revision/generation.
- [ ] Deterministic merge/diff нескольких providers.
- [ ] Capability model `window.*`, `view.*`, `media.observe`.
- [ ] Stale/partial/provider-disconnected semantics.
- [ ] Fake surface providers и property tests merge invariants.

#### AS-1 — GNOME window projection [P0 for MVP]

- [ ] `Shell.WindowTracker` → application grouping.
- [ ] Meta.Window → session window projection.
- [ ] focused/MRU/workspace/monitor observed state.
- [ ] deterministic ordering при event storms.
- [ ] lazy preview capability.
- [ ] отсутствие каких-либо tab assumptions в GNOME adapter.
- [ ] Shell bridge отправляет bounded/coalesced events, а aggregation живёт вне hot path.

#### AS-2 — Activity Strip MVP [P0 for MVP]

- [ ] одна card = одна ApplicationSurface.
- [ ] app title/icon/context + strongest status.
- [ ] window count и active-window context.
- [ ] multiple-window group navigator.
- [ ] MRU activation.
- [ ] `full/compact/micro` semantic zoom.
- [ ] layout constraints `min/ideal/max/priority/expand/collapse`.
- [ ] overflow policy без скрытия urgent aggregate state.
- [ ] keyboard/pointer/touch parity.
- [ ] Focus Mode integration без превращения Home Mode в dashboard.

#### AS-3 — Panel DSL v1 [P1, M3.5]

- [ ] Pin HCL dependency/version и проверить license/supply-chain policy.
- [ ] HWS-specific strict HCL schema.
- [ ] Parser → AST → validation → normalized Panel Model.
- [ ] Renderer полностью отделён от DSL types.
- [ ] schema diagnostics с source spans.
- [ ] transactional hot reload: invalid config сохраняет previous-good layout.
- [ ] v1 без arbitrary expressions/commands; только декларативные blocks/attributes.
- [ ] typed widget/action registry вместо command strings.
- [ ] config migration/version policy.
- [ ] sample configs для desktop, compact/laptop и touch layouts.

#### AS-4 — First rich view provider [P1, M3.5]

- [ ] Выбрать первый provider: browser либо основной IDE по качеству доступного API.
- [ ] Provider protocol versioning + capability handshake.
- [ ] Views/tabs snapshot.
- [ ] active + MRU + pinned/important ordering.
- [ ] dirty/progress/attention, где source достоверно предоставляет это состояние.
- [ ] typed activate action.
- [ ] provider disconnect → graceful window-only fallback.
- [ ] privacy/redaction для title/URL/document name.

#### AS-5 — Provider ecosystem [P2]

- [ ] Chromium/Chrome extension + Native Messaging/local bridge.
- [ ] Firefox extension/provider.
- [ ] IDE provider contract и минимум один IDE plugin.
- [ ] terminal/session provider.
- [ ] AT-SPI read-only semantic fallback.
- [ ] provider priority `native API → HWS plugin → AT-SPI → heuristic`.
- [ ] provider health/compatibility diagnostics.

#### AS-6 — General resource cards [P2]

- [ ] build/test cards.
- [ ] SSH/terminal session cards.
- [ ] AI agent cards.
- [ ] file transfer/download cards.
- [ ] service/network/VPN cards.
- [ ] workspace operation cards.
- [ ] единый priority/attention/overflow contract для app и non-app resources.

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
- [ ] `Explain` показывает причину остановки transition;
- [ ] editor/browser/terminal корректно группируются как ApplicationSurfaces;
- [ ] несколько окон одного приложения доступны через Activity Strip без дублирования app card;
- [ ] Surface provider failure не ломает базовую workspace activation.

## 6. Performance budgets

Пока это target, не измеренный baseline:

- Grid cached navigation p95 < 16 ms.
- Local search p95 < 50 ms.
- IPC local read p95 < 20 ms без внешних providers.
- Workspace activation не блокирует Shell main loop.
- Shell не выполняет synchronous filesystem/network/database I/O.
- Window/monitor event storms coalesce до bounded update rate.
- Idle `hwsd` избегает polling там, где есть events/watchers.
- Surface focus/status change → видимый card update p95 < 50 ms.
- Shell main-loop work на один coalesced Surface update target < 4 ms.
- Window/view event storms coalesce до bounded update rate.
- Window preview создаётся лениво.
- 50 surfaces / 500 views не должны создавать постоянную работу renderer-а пропорционально числу views на каждый frame.

После появления GNOME prototype эти цели заменяются измеренными baseline/threshold.

## 7. Architectural guardrails

Без ADR запрещено:

- переносить hover/focus/animation в durable Axiom execution;
- выполнять system side effects напрямую из UI в обход intent/orchestration boundary;
- превращать GNOME extension в основной business process;
- делать sync filesystem/network/database I/O в Shell hot path;
- выполнять arbitrary shell strings из workspace config или Panel DSL;
- импортировать GNOME-specific types в domain;
- строить GNOME 50 integration на X11-only tooling;
- использовать PID/title как durable window identity;
- использовать tab/view index или transient provider handle как durable identity;
- угадывать rich tabs/views из window title как основной provider path;
- считать launch доказательством window/focus/layout success;
- вручную размножать scale-factor math по domain code;
- объявлять Active без observed confirmation required resources;
- replay absolute rectangles после topology change без re-resolution;
- закрывать adopted/external resources как managed;
- использовать process-local lock как межпроцессную гарантию;
- обещать exactly-once внешний effect;
- скрывать partial failure под Active;
- выполнять provider-heavy работу или AT-SPI polling в Shell hot path;
- позволять provider-данным обходить escaping/schema validation;
- писать browser URL/document title в durable telemetry без redaction policy;
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

### M3 — Desktop vertical slice + window-only Activity Strip

- [ ] thin GNOME 50 extension;
- [ ] app/window observation/activation adapter;
- [ ] AS-0 ApplicationSurface domain contracts;
- [ ] AS-1 GNOME window projection;
- [ ] AS-2 window-only Activity Strip;
- [ ] multiple-window grouping/MRU navigator;
- [ ] full/compact/micro semantic zoom;
- [ ] normalized layout against live topology;
- [ ] nested Wayland CI;
- [ ] VS-1 real desktop end-to-end.

### M3.5 — Programmable Activity Strip + rich views

- [ ] AS-3 HCL-based Panel DSL v1;
- [ ] normalized Panel Model + renderer boundary;
- [ ] transactional hot reload;
- [ ] AS-4 minimum one real rich view/tab provider;
- [ ] provider disconnect → window-only fallback;
- [ ] privacy/redaction tests;
- [ ] 800/1366/1920/4K + fractional-scale Activity Strip tests;
- [ ] performance baseline для Surface update path.

### M4 — Productization

- [ ] packaging/install/uninstall/safe mode;
- [ ] compatibility matrix;
- [ ] Ubuntu default-extension profile;
- [ ] accessibility;
- [ ] performance baselines;
- [ ] security hardening;
- [ ] Panel DSL schema/version migration policy;
- [ ] provider compatibility diagnostics.

### M5 — Extensibility

- [ ] providers/plugins;
- [ ] AS-5 browser/IDE/terminal provider ecosystem;
- [ ] AS-6 general resource cards;
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
7. Реализовать AS-0 pure-Go ApplicationSurface contracts и fake provider tests до Shell UI.
8. Только после durable + D-Bus tests создать минимальный GNOME 50 extension.
9. Реализовать AS-1 window observation/grouping и доказать MRU/application association в nested Wayland.
10. Реализовать AS-2 window-only Activity Strip с semantic zoom до rich tabs.
11. Поднять nested GNOME 50 / Wayland lifecycle harness до сложного tiling/Activity Strip UI.
12. Добавить safe disable/doctor path до активного window mutation MVP.
13. После green M3 реализовать AS-3 HCL Panel DSL с transactional hot reload.
14. После стабильного Panel Model реализовать AS-4 первый rich view/tab provider.

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
- [`docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md`](docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md)
- [`docs/adr/0004-thin-shell-dbus-boundary.md`](docs/adr/0004-thin-shell-dbus-boundary.md)
- [`docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md`](docs/adr/0005-ubuntu-2604-gnome50-wayland-target.md)
- [`docs/adr/0006-application-surface-and-panel-dsl.md`](docs/adr/0006-application-surface-and-panel-dsl.md)
