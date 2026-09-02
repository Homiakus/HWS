# HWS MASTER PLAN

Единый living execution plan проекта HWS.

Правило: код, тесты и фактическое поведение являются источником истины. `[x]` означает реализованный и проверенный результат. Наличие документации без работающего кода не закрывает пункт. Если новая находка меняет архитектуру или порядок работ, этот файл обновляется до следующего крупного среза.

## 0. Цель продукта

Создать иерархическую рабочую оболочку для Ubuntu, которая организует работу вокруг контекста и задачи, а не вокруг плоского списка приложений.

```text
Area → Context → Project → Task → Ready Workspace
```

HWS должен:

- быстро перемещаться по произвольно глубокой иерархии;
- превращать leaf/task в versioned desired workspace;
- проводить значимые переходы через Axiom;
- наблюдать реальное состояние GNOME/приложений до объявления успеха;
- безопасно владеть только теми ресурсами, ownership которых доказан;
- оставаться отключаемым слоем поверх стандартного GNOME Shell/Mutter.

## 1. Зафиксированные архитектурные решения

- [x] Основной daemon — Go.
- [x] GNOME Shell extension — тонкий GJS UI/compositor adapter.
- [x] Shell ↔ `hwsd` — versioned session D-Bus.
- [x] Первый desktop target — Ubuntu 26.04 LTS / GNOME Shell 50 / Wayland.
- [x] HWS не форкает Mutter/GNOME Shell в MVP.
- [x] Эфемерные hover/focus/animation состояния не проходят через Axiom.
- [x] Значимые workspace transitions проходят через Axiom.
- [x] Desired и observed state разделены.
- [x] Внешние side effects считаются at-least-once/idempotent, exactly-once не обещается.
- [x] PID/title не считаются durable window identity.
- [x] Desktop application identity опирается прежде всего на Shell/.desktop identity.
- [x] Window identity внутри Shell-сессии использует `Meta.Window.get_stable_sequence()`.
- [x] `launch`, `window observed`, `layout`, `focus` рассматриваются как разные состояния.
- [x] Layout domain использует logical/normalized coordinates.
- [x] Monitor topology должна иметь revision и инвалидировать устаревшие placements.
- [x] `managed/adopted/external` ownership является обязательной частью resource model.
- [x] HWS не закрывает adopted/external ресурсы как managed.
- [x] Application card представляет ApplicationSurface, а не автоматически отдельное окно.
- [x] Rich tabs/documents/sessions приходят от capability providers, а не угадываются из title.
- [x] Panel configuration — строгий HCL-based DSL → normalized Panel Model.
- [x] Panel DSL не исполняет arbitrary shell/filesystem/network code.
- [x] Lock-screen integration отсутствует по умолчанию.

## 2. Состояние `main` на 2026-09-02

Текущий рабочий vertical slice:

```text
Home Grid (GJS)
      ↓ ActivateWorkspace
session D-Bus
      ↓
hwsd (Go)
      ↓
versioned Workspace Catalog
      ↓
Axiom WorkspaceLifecycle + Pebble
      ↓
Desired ↔ Observed Reconciler
      ↓
Shell Desktop Adapter
      ↓ ShellActionRequested
GNOME Shell Executor
      ↓
Shell.App / Meta.Window
      ↓
Shell snapshot → hwsd → observed convergence
```

### Реализовано и проверено автоматическими тестами

- [x] Go module / Go 1.26 / reproducible module graph.
- [x] Axiom закреплён конкретным pre-v1 pseudo-version.
- [x] Stable domain IDs и arbitrary-depth hierarchy.
- [x] Duplicate/missing-parent/cycle validation.
- [x] Deterministic ordering/path resolution.
- [x] Versioned hierarchy snapshots, hot reload и last-known-good.
- [x] `GetTree`, `GetPath`, `TreeChanged`.
- [x] Home Grid с динамическими строками, breadcrumbs, keyboard navigation и search/jump.
- [x] Desired/observed workspace state и deterministic reconcile.
- [x] Required/optional evaluation и degraded classification.
- [x] `managed/adopted/external` ownership.
- [x] Axiom Activate/Recover/Resume/Suspend/Close model.
- [x] Pebble + ProductionMode production path.
- [x] Durable store reopen test сохраняет lifecycle state.
- [x] Versioned `workspaces.json` catalog с active revision map.
- [x] Workspace catalog strict schema, hot reload и last-known-good.
- [x] Реальный `cmd/hwsd`.
- [x] D-Bus well-known name обеспечивает single-instance daemon policy.
- [x] Hello: protocol/server instance/revision epoch/capabilities.
- [x] Shell owner-change handling делает fresh handshake и snapshot refresh.
- [x] `ActivateWorkspace` и `GetWorkspaceState` по D-Bus.
- [x] Typed asynchronous `ShellActionRequested` / `CompleteShellAction` protocol.
- [x] Реальный session-bus test: Activate → signal → completion → Axiom Active.
- [x] Shell executor использует typed `desktopAppId` и session window ID.
- [x] Уже запущенное приложение не присваивается HWS как managed без launch evidence.
- [x] Managed close ждёт исчезновения именно HWS-tracked windows из observed state.
- [x] Late action completion после timeout отвергается.
- [x] ApplicationSurface domain, semantic snapshot/revision/diff.
- [x] Provider registry и aggregation вне Shell hot path.
- [x] GNOME Shell window snapshot provider.
- [x] Activity Strip с application grouping и window fallback.
- [x] Rich provider plumbing для browser/VS Code и вспомогательных sources.
- [x] MPRIS и LauncherEntry collectors.
- [x] AT-SPI/native-frame/status provider groundwork.
- [x] HCL Panel DSL parser/model/hot reload с last-known-good.
- [x] `hwsctl doctor`, health/panel/spec/tree/path/app diagnostics.
- [x] systemd --user service artifact.
- [x] CI: module graph, gofmt, unit, race, vet, real D-Bus round trip, JS syntax, Node model tests, manifests, daemon build, smoke.

### Ещё НЕ доказано

Ни один пункт ниже нельзя считать закрытым только потому, что unit/contract test зелёный:

- [ ] Полный E2E на Ubuntu 26.04 / GNOME Shell 50 / Wayland.
- [ ] Реальный запуск `.desktop` приложения через `Shell.App.activate()` в GNOME 50 qualification environment.
- [ ] Реальный `Meta.Window.delete()` + save/cancel dialog behavior.
- [ ] 100× enable/disable extension без signal/source/actor leaks.
- [ ] Nested GNOME 50 lifecycle harness в CI.
- [ ] Fractional scaling 100/125/150/200%.
- [ ] Mixed-scale multi-monitor + hotplug/rotation/primary change.
- [ ] Placement/layout относительно live topology.
- [ ] Focus request + observed focus confirmation/policy-blocked classification.
- [ ] Generic process/terminal production adapter.
- [ ] Crash-after-external-effect/before-durable-checkpoint recovery proof.
- [ ] History/replay compatibility gate и Explain API.
- [ ] Реальная совместимость с Ubuntu default extensions.

## 3. Definition of Done MVP

MVP готов только когда на реальном Ubuntu 26.04 / GNOME 50 / Wayland доказано:

1. HWS устанавливается и запускается как user-session component.
2. `Super` или официально выбранный primary shortcut открывает Home Grid без ломки Overview.
3. Навигация проходит минимум 3 уровня и поддерживает search/direct jump.
4. Leaf/task выбирает versioned workspace definition.
5. Activation проходит через D-Bus → Axiom → reconciler → Shell executor.
6. Минимум два ресурса одного workspace достигают observed ready state.
7. Уже существующее приложение корректно adopted и не дублируется.
8. HWS-owned окно корректно закрывается, adopted окно не закрывается.
9. Desktop app/window identity не зависит от PID/title как durable evidence.
10. Layout применён в logical coordinates против актуальной topology revision.
11. Focus подтверждается observed state, а policy denial не маскируется под success.
12. Suspend/Resume/Close/Recover имеют проверенную desktop semantics.
13. Restart `hwsd` сохраняет/объясняет durable lifecycle state.
14. D-Bus owner restart приводит к fresh handshake/capabilities/snapshots.
15. Extension переживает 100× enable/disable без утечек.
16. Disable HWS возвращает стандартную GNOME-сессию в рабочее состояние.
17. Activity Strip корректно группирует несколько окон одного приложения.
18. Rich provider failure деградирует к window-only состоянию.
19. full/compact/micro режимы проходят width/scale matrix.
20. `hwsctl doctor` различает healthy, degraded и stale/LKG configuration states.

## 4. Workstreams

### W0 — Repository foundation

- [x] Go module / Go 1.26.
- [x] Makefile.
- [x] CI baseline и расширенные race/D-Bus/JS gates.
- [x] ADR framework.
- [ ] Определить LICENSE и совместимость с extensions.gnome.org.
- [ ] Vulnerability/dependency scan.
- [ ] `CONTRIBUTING.md`.
- [ ] top-level `SECURITY.md`.
- [ ] CODEOWNERS / branch protection policy.

### W1 — Hierarchy / Home Grid model

- [x] Arbitrary-depth typed hierarchy.
- [x] Cycle/missing-parent/duplicate validation.
- [x] Deterministic ordering/path resolution.
- [x] Revisioned snapshots и last-known-good config.
- [x] Home Grid dynamic rows.
- [x] Breadcrumbs.
- [x] Keyboard navigation.
- [x] Search/jump с bounded results.
- [x] Vertical viewport для глубокой hierarchy.
- [ ] Context-stack back/forward history как отдельный service.
- [ ] DynamicQuery/provider-driven children.
- [ ] Property/fuzz tests для больших деревьев и revision churn.
- [ ] Primary Super/shortcut integration после qualification harness.

### W2 — Workspace / reconciliation

- [x] Desired/Observed state.
- [x] Required/optional resources.
- [x] Deterministic diff/evaluation.
- [x] managed/adopted/external ownership.
- [x] Normalized placement intent model.
- [x] Desktop app production adapter boundary.
- [x] Pre-existing managed-intent application превращается в adopted без ownership evidence.
- [x] Close подтверждается observed disappearance tracked windows.
- [ ] Readiness разложить на process/window/layout/focus sub-observations.
- [ ] Window association/evidence model сделать явным pure-Go contract.
- [ ] Live monitor topology revision model.
- [ ] Placement resolution/reconciliation.
- [ ] Focus request/observe state machine.
- [ ] Compensation policy для non-idempotent operations.

### W3 — Axiom integration

- [x] Pinned Axiom dependency.
- [x] Declarative `model` frontend.
- [x] Activate/Recover/Resume/Suspend/Close.
- [x] Inactive/Preparing/Active/Degraded/Recovering/Closing/Failed.
- [x] Typed activities.
- [x] Operation-key idempotency boundary.
- [x] ProductionMode + Pebble.
- [x] Reopen persistence test.
- [x] Runtime activation через real D-Bus action round trip.
- [ ] Уточнить Suspend semantics для реально оставленных running resources.
- [ ] Model-level ownership/capability claims там, где они не adapter-specific.
- [ ] Crash-after-effect/before-checkpoint test.
- [ ] History/replay compatibility test.
- [ ] `Explain` service/API.

### W4 — `hwsd`

- [x] Реальный user daemon.
- [x] Session D-Bus single-instance name.
- [x] Health JSON.
- [x] Panel/hierarchy/workspace config loaders.
- [x] LKG semantics для panel/hierarchy/workspaces.
- [x] XDG config/state directory policy.
- [x] Axiom lifecycle в daemon.
- [x] Provider registry и surface aggregation.
- [x] Graceful shutdown.
- [x] Workspace catalog maintenance.
- [x] systemd --user unit artifact.
- [ ] Structured event logging вместо преимущественно `log.Printf`.
- [ ] Crash recovery bootstrap report.
- [ ] D-Bus activation `.service` artifact, если нужен поверх systemd user startup.
- [ ] Явная graphical-session lifecycle binding/PartOf policy qualification.

### W5 — IPC / D-Bus

- [x] Protocol v1 constants.
- [x] Hello + instance + revision epoch + capabilities.
- [x] `GetTree` / `GetPath`.
- [x] `GetPanelSnapshot` / `GetPanelSpec` / `GetHealth`.
- [x] `GetApplicationSurface`.
- [x] `SubmitShellSnapshot`.
- [x] `ActivateView` / `CloseView`.
- [x] `ActivateWorkspace` / `GetWorkspaceState`.
- [x] `ShellActionRequested` / `CompleteShellAction`.
- [x] Panel/tree signals.
- [x] Real session-bus round-trip tests, включая async workspace action loop.
- [ ] `RecoverWorkspace` / `SuspendWorkspace` / `ResumeWorkspace` / `CloseWorkspace`.
- [ ] Operation object/ID + `GetOperation`.
- [ ] Explain/history methods.
- [ ] Surface change signal с bounded/coalesced payload.
- [ ] Typed activate/focus window action, отделённый от app ensure.
- [ ] Payload size limits и rejection tests.
- [ ] Explicit capability negotiation tests между protocol minors.

### W6 — GNOME 50 Shell adapter / UI

- [x] Thin extension entry point.
- [x] user-session-only metadata target GNOME 50.
- [x] Reversible actor/signal/source ownership в коде disable path.
- [x] D-Bus owner watcher + fresh handshake.
- [x] Home Grid modal.
- [x] Dynamic rows/breadcrumbs/search/keyboard/error states.
- [x] Window snapshot bridge.
- [x] Application grouping через Shell semantics.
- [x] Activity Strip window-only fallback.
- [x] Window group/MRU interaction groundwork.
- [x] Semantic density full/compact/micro renderer.
- [x] Typed desktop app ensure executor через `Shell.AppSystem.lookup_app` / `Shell.App.activate`.
- [x] Typed close executor через session-stable `Meta.Window` identity.
- [ ] Nested GNOME 50 runtime test.
- [ ] 100× enable/disable leak test.
- [ ] Super/primary shortcut ownership/conflict policy.
- [ ] Touch gesture navigation.
- [ ] Accessibility qualification with Orca/high-contrast/large text.
- [ ] Animation/main-loop profiling.
- [ ] Meta.Window geometry/placement executor.
- [ ] Monitor topology observer.
- [ ] Focus executor + observed confirmation.
- [ ] Lazy window previews.
- [ ] Multi-monitor qualification.

### W7 — OS integration

- [x] Desktop app discovery path through Shell AppSystem.
- [x] Desktop app ensure/adopt boundary.
- [x] Session window discovery through Shell snapshot.
- [ ] Generic process launch/adoption production adapter.
- [ ] Terminal launch with cwd production adapter.
- [ ] Focus request + confirmation.
- [ ] Logical placement → current topology geometry.
- [ ] Filesystem/project discovery provider.
- [ ] Git status provider.
- [ ] SSH provider.
- [ ] Network/VPN capability boundary.
- [ ] Portals/screen-sharing compatibility.
- [ ] Ubuntu default-extension compatibility scanner.
- [ ] Third-party extension conflict classification без silent disable.

### W8 — Persistence / recovery

- [x] Pebble Axiom store.
- [x] ProductionMode.
- [x] Durable lifecycle state reopen test.
- [x] Workspace definition history хранит несколько revisions для reopen/recovery.
- [ ] Crash after external effect / before checkpoint test.
- [ ] Store corruption/permission/disk-full tests.
- [ ] History/replay schema compatibility gate.
- [ ] Upgrade/migration policy.
- [ ] Backup/repair/doctor commands для durable state.

### W9 — Providers / ApplicationSurface

- [x] ApplicationSurface/Window/View/status/capability model.
- [x] Immutable-ish normalized snapshots, semantic revision/generation и diffs.
- [x] Provider registry/health/staleness model.
- [x] GNOME Shell provider.
- [x] Browser provider/bridge groundwork.
- [x] VS Code provider groundwork.
- [x] MPRIS collector.
- [x] LauncherEntry collector.
- [x] AT-SPI/native-frame/status groundwork.
- [x] Provider disconnect допускает window-only fallback architecture.
- [ ] Provider protocol compatibility/version negotiation hardening.
- [ ] Privacy/redaction implementation для URL/title/document names.
- [ ] 20/100/500-view stress profile.
- [ ] First rich provider end-to-end qualification на реальном app.

### W10 — Panel DSL / Activity Strip

- [x] HCL dependency и strict HWS schema.
- [x] Parser → validation → normalized Panel Model.
- [x] Invalid hot reload сохраняет previous-good layout.
- [x] Typed renderer boundary.
- [x] Application cards и surface segments/chips.
- [x] full/compact/micro density modes.
- [x] Rich provider views могут добавляться без уничтожения window fallback.
- [ ] Width matrix 800/1366/1920/4K.
- [ ] Fractional scaling matrix.
- [ ] Touch/pointer/keyboard parity qualification.
- [ ] Renderer performance baseline и update coalescing measurement.
- [ ] Config schema version/migration policy.

### W11 — Observability / CLI

- [x] `hwsctl health`.
- [x] `hwsctl doctor`.
- [x] panel/spec/tree/path/app diagnostics.
- [x] provider health/stale/partial reporting.
- [ ] Workspace activate/state/close/recover CLI commands.
- [ ] Structured correlation `intent → execution → activity → adapter → observation`.
- [ ] Reconcile timings/retry metrics.
- [ ] D-Bus reconnect metrics.
- [ ] Explain/history CLI.
- [ ] Redacted diagnostics bundle.

### W12 — Security

- [x] Threat model и no-root-by-default boundary.
- [x] No implicit shell command invariant.
- [x] Workspace/Panel config strict schemas.
- [x] Shell executor принимает только typed action kinds.
- [x] Workspace result/action broker fail-closed при отсутствии executor/timeout.
- [x] Adopted resources не закрываются как managed.
- [ ] Environment sanitization для будущего generic process adapter.
- [ ] Path traversal/symlink race protections.
- [ ] D-Bus payload size/rate limiting.
- [ ] Provider trust/permission model.
- [ ] Browser extension minimum-permissions audit.
- [ ] Privacy/redaction implementation.
- [ ] Privileged helper boundary только при доказанной необходимости.

### W13 — Compatibility / release engineering

- [x] Ubuntu 26.04 / GNOME 50 / Wayland — first-class target.
- [x] GNOME 50-only extension metadata на текущем этапе.
- [x] systemd --user service artifact.
- [ ] `COMPATIBILITY_MATRIX.md`.
- [ ] Ubuntu default extension test profile.
- [ ] GNOME 50 nested CI image/harness.
- [ ] Known-conflicts database.
- [ ] Qualification gate перед новым GNOME major.
- [ ] Install/uninstall/safe-mode tooling.
- [ ] RC/canary process.

## 5. Текущий критический путь

Headless архитектура уже достаточно зрелая, поэтому главный риск сместился с domain design на **реальное compositor/session поведение**.

Следующая последовательность обязательна:

```text
1. CLI + example workspace definitions
      ↓
2. D-Bus Close/Recover/Suspend/Resume + workspace status visibility
      ↓
3. Nested GNOME 50 / Wayland harness
      ↓
4. 100× extension lifecycle + daemon restart tests
      ↓
5. Real desktop app ensure/adopt/close qualification
      ↓
6. Monitor topology observation
      ↓
7. Logical placement executor
      ↓
8. Focus request + observed confirmation
      ↓
9. VS-1: editor + terminal + optional browser
      ↓
10. Compatibility/scale/performance matrix
```

Не начинать сложный tiling/preview/gesture layer до прохождения пунктов 3–5.

## 6. Следующие атомарные действия

1. Добавить `examples/hierarchy.json` и `examples/workspaces.json`, связанные одинаковыми `workspaceId`.
2. Добавить `hwsctl workspace activate/state/close/recover` поверх D-Bus.
3. Экспортировать D-Bus Close/Recover/Suspend/Resume с contract tests.
4. Добавить Workspace state markers в Home Grid без polling hot loop.
5. Добавить explicit operation result/error codes для UI вместо разбора свободного текста.
6. Создать nested GNOME 50 / Wayland qualification harness.
7. Добавить 100× enable/disable test и daemon owner restart test.
8. Доказать `Shell.App.activate()` и `Meta.Window.delete()` на GNOME 50.
9. Добавить live monitor topology snapshot/revision.
10. Реализовать placement только после topology tests.
11. Реализовать focus как `Request → Observe → Reached/PolicyBlocked/Superseded/WindowGone/TimedOut`.
12. После VS-1 добавить measured performance baseline и compatibility matrix.

## 7. VS-1 — Local Development Workspace

Целевая hierarchy:

```text
Development
└── Projects
    └── HWS
        └── Develop
```

Минимальный workspace:

- editor: desktop application;
- terminal: repo cwd;
- optional browser/documentation.

Уже доказано:

- [x] versioned definition lookup;
- [x] Axiom activation;
- [x] durable reopen;
- [x] repeated reconcile не делает лишний ensure для reached resource;
- [x] degraded → Recover path;
- [x] adopted ownership protection;
- [x] D-Bus → Shell action → observed snapshot → Axiom Active integration test;
- [x] Home Grid leaf вызывает `ActivateWorkspace`.

Осталось доказать на настоящем desktop:

- [ ] editor запускается через Shell и корректно ассоциируется;
- [ ] already-running editor adopted без duplicate launch;
- [ ] terminal production adapter;
- [ ] optional browser placement;
- [ ] close с реальными save/cancel dialogs;
- [ ] placement/focus;
- [ ] restart/recovery при реальном external effect.

## 8. Performance budgets

До измеренного GNOME baseline это targets:

- cached Home Grid navigation p95 < 16 ms;
- local hierarchy search p95 < 50 ms;
- local read IPC p95 < 20 ms без external providers;
- Shell main loop никогда не выполняет synchronous filesystem/network/database/Axiom work;
- one coalesced Surface update target < 4 ms Shell CPU work;
- visible Surface state update p95 < 50 ms после получения provider event;
- event storms обязаны coalesce до bounded rate;
- 50 surfaces / 500 views не должны давать per-frame O(total views) работу;
- preview создаётся лениво;
- workspace activation может быть долгой, но D-Bus/GJS callback не блокирует Shell main loop.

После nested GNOME harness эти цели заменяются измеренными thresholds.

## 9. Непереступаемые guardrails

Без нового ADR запрещено:

- выполнять filesystem/network/database/Axiom работу в Shell hot path;
- выполнять arbitrary command strings из user config;
- импортировать GNOME types в domain;
- использовать PID/title как durable identity;
- считать Shell action acknowledgement доказательством observed convergence;
- считать `window.delete()` доказательством закрытия до observed disappearance;
- считать launch доказательством focus/layout success;
- закрывать adopted/external resource как managed;
- молча отключать сторонние GNOME extensions;
- replay absolute pixel placements после topology change;
- обещать exactly-once external effect;
- скрывать partial failure под Active;
- разрешать provider data обходить schema validation/escaping;
- писать URL/document title в durable telemetry без redaction policy;
- включать lock-screen session mode без security ADR;
- позволять AI/automation молча менять canonical hierarchy/workspace config.

## 10. Milestones

### M0 — Architecture bootstrap — DONE

- [x] architecture/docs/ADRs/invariants;
- [x] Go skeleton/CI;
- [x] GNOME 50 research and adapter contract.

### M1 — Headless domain/orchestration — DONE

- [x] hierarchy/domain/reconcile/fake adapter;
- [x] Axiom lifecycle;
- [x] tests/race/smoke.

### M1.5 — Durable orchestration gate — FUNCTIONAL

- [x] Pebble + ProductionMode;
- [x] state reopen test;
- [x] versioned workspace definitions retained across active-revision changes;
- [ ] crash-after-effect checkpoint test;
- [ ] history/replay compatibility;
- [ ] Explain.

### M2 — Local daemon / D-Bus — FUNCTIONAL

- [x] `hwsd`;
- [x] Hello/owner epoch/capabilities;
- [x] hierarchy/panel/surface reads;
- [x] workspace Activate/State;
- [x] async Shell action protocol;
- [x] real session-bus round-trip test;
- [x] systemd user artifact;
- [ ] full workspace mutation API;
- [ ] operation/history API;
- [ ] owner-restart E2E.

### M3 — Desktop vertical slice — IN PROGRESS

- [x] thin GNOME 50 extension;
- [x] Home Grid;
- [x] window observation/ApplicationSurface bridge;
- [x] Activity Strip groundwork;
- [x] typed Shell app ensure/window close executor;
- [x] Home Grid → Axiom activation wiring;
- [ ] nested GNOME 50 qualification;
- [ ] real desktop VS-1;
- [ ] topology/layout/focus;
- [ ] lifecycle leak tests.

### M3.5 — Programmable Activity Strip / rich providers — FUNCTIONAL BUT UNQUALIFIED

- [x] HCL Panel DSL runtime;
- [x] normalized Panel Model;
- [x] LKG hot reload;
- [x] provider framework/browser/VS Code groundwork;
- [x] window-only fallback architecture;
- [ ] real provider qualification;
- [ ] privacy/redaction tests;
- [ ] width/fractional-scale/performance matrix.

### M4 — Productization

- [ ] install/uninstall/safe mode;
- [ ] compatibility matrix;
- [ ] accessibility qualification;
- [ ] performance baselines;
- [ ] security hardening;
- [ ] config migration policy;
- [ ] release/canary process.

### M5 — Extensibility

- [ ] dynamic hierarchy providers;
- [ ] terminal/SSH/network/resource cards;
- [ ] provider SDK/compatibility tooling;
- [ ] sync/export;
- [ ] optional recommendation layer.

## 11. Change discipline

Каждая значимая итерация должна:

1. начинаться от актуального `main` и этого плана;
2. иметь атомарный observable outcome;
3. не оставлять промежуточно некомпилируемый `main`;
4. проходить unit/race/vet и релевантные protocol/UI tests;
5. для D-Bus изменений добавлять реальный session-bus contract test, где это возможно;
6. для Shell изменений отделять pure model tests от GJS runtime code;
7. для external effects проверять observed convergence, а не только acknowledgement;
8. обновлять этот plan при изменении критического пути;
9. merge в `main` делать только после зелёного verification gate.

Текущий интеграционный checkpoint `main`: `acb020828871d7d53053d5c2ca0c7b07f829e6b5` — Home Grid workspace activation через durable Axiom + typed GNOME Shell executor.
