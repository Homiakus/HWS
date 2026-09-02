# HWS MASTER PLAN

Единый living execution plan проекта HWS.

Правило: код, тесты и фактическое поведение являются источником истины. `[x]` означает реализованный и проверенный автоматическими тестами результат. Это **не** означает, что compositor-specific поведение доказано на реальном GNOME, если такой qualification gate отдельно не пройден. Если новая находка меняет архитектуру или порядок работ, этот файл обновляется до следующего крупного среза.

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
- работать в logical coordinates и никогда не replay устаревшую пиксельную раскладку после изменения monitor topology;
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
- [x] Внешние side effects считаются at-least-once/idempotent; exactly-once не обещается.
- [x] PID/title не считаются durable window identity.
- [x] Desktop application identity опирается прежде всего на Shell/.desktop identity.
- [x] Window identity внутри Shell-сессии использует `Meta.Window.get_stable_sequence()`.
- [x] `launch`, `window observed`, `layout`, `focus` — разные состояния.
- [x] Layout domain использует normalized coordinates; Shell/Mutter side использует logical coordinates.
- [x] Monitor topology имеет semantic revision; stale placement обязан fail-closed.
- [x] Placement action связывает topology revision, monitor ref и monitor index, а не доверяет одному индексу.
- [x] Shell action acknowledgement не считается observed convergence.
- [x] `managed/adopted/external` ownership является обязательной частью resource model.
- [x] HWS не закрывает adopted/external ресурсы как managed.
- [x] Application card представляет ApplicationSurface, а не автоматически отдельное окно.
- [x] Rich tabs/documents/sessions приходят от capability providers, а не угадываются из title.
- [x] Panel configuration — строгий HCL-based DSL → normalized Panel Model.
- [x] Panel DSL не исполняет arbitrary shell/filesystem/network code.
- [x] Lock-screen integration отсутствует по умолчанию.

## 2. Текущий проверенный vertical slice

```text
Home Grid (GJS)
      ↓ typed workspace intent
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
      ↓ typed ShellActionRequested
GNOME Shell Executor
      ↓
Shell.App / Meta.Window
      ↓
GNOME native snapshot
  ├─ applications/windows
  ├─ workspace/monitor identity
  ├─ logical frame
  └─ monitor topology + semantic revision
      ↓
hwsd observed convergence
      ↓
Axiom Active / Degraded / Failed
```

### Реализовано и проверено автоматическими тестами

- [x] Go 1.26 module и reproducible module graph.
- [x] Axiom закреплён конкретным pre-v1 pseudo-version и изолирован orchestration boundary.
- [x] Stable domain IDs и arbitrary-depth hierarchy.
- [x] Duplicate/missing-parent/cycle validation.
- [x] Deterministic ordering/path resolution.
- [x] Versioned hierarchy snapshots, hot reload и last-known-good.
- [x] `GetTree`, `GetPath`, `TreeChanged`.
- [x] Home Grid: dynamic rows, breadcrumbs, keyboard navigation, search/jump.
- [x] Desired/observed workspace state и deterministic reconcile.
- [x] Required/optional evaluation и degraded classification.
- [x] `managed/adopted/external` ownership.
- [x] Axiom Activate/Recover/Resume/Suspend/Close model.
- [x] Pebble + ProductionMode production path.
- [x] Durable store reopen test.
- [x] Versioned `workspaces.json` catalog с active revision map.
- [x] Workspace catalog strict schema, hot reload и last-known-good.
- [x] Реальный `cmd/hwsd` и session D-Bus single-instance policy.
- [x] Hello: protocol/server instance/revision epoch/capabilities.
- [x] D-Bus owner-change handling делает fresh handshake и snapshot refresh.
- [x] Полный workspace lifecycle D-Bus API: Activate/State/Recover/Resume/Suspend/Close.
- [x] Batch `GetWorkspaceStates` + `WorkspaceChanged` без N запросов из Home Grid.
- [x] `hwsctl workspace activate/state/recover/resume/suspend/close`.
- [x] Typed asynchronous `ShellActionRequested` / `CompleteShellAction` protocol.
- [x] Реальный session-bus lifecycle test Activate → Active → Close → Inactive.
- [x] Home Grid показывает durable lifecycle state и typed actions.
- [x] Shell executor использует typed `desktopAppId` и session-stable window ID.
- [x] Уже запущенное приложение не присваивается HWS как managed без launch evidence.
- [x] Managed close ждёт observed disappearance именно HWS-tracked windows.
- [x] Late action completion после timeout отвергается.
- [x] ApplicationSurface domain, semantic snapshot/revision/diff.
- [x] Provider registry и aggregation вне Shell hot path.
- [x] GNOME Shell native window snapshot provider.
- [x] Activity Strip с application grouping и window fallback.
- [x] Rich provider plumbing для browser/VS Code и вспомогательных sources.
- [x] MPRIS и LauncherEntry collectors.
- [x] AT-SPI/native-frame/status provider groundwork.
- [x] HCL Panel DSL parser/model/hot reload с last-known-good.
- [x] `hwsctl doctor`, health/panel/spec/tree/path/app diagnostics.
- [x] systemd --user service artifact.
- [x] GNOME snapshot содержит optional live logical monitor topology и window frame.
- [x] Topology получает deterministic semantic revision, независимую от порядка monitor entries.
- [x] Normalized placement детерминированно разрешается против logical monitor work area.
- [x] `primary`, `secondary`, explicit `monitor:N` monitor roles имеют pure-Go resolver.
- [x] Typed `place_window` содержит topology revision + monitor ref/index + workspace + logical rect.
- [x] Stale topology отклоняется до placement; topology change во время операции инвалидирует результат.
- [x] Resource с placement считается Ready только после fresh observed monitor/workspace/frame convergence.
- [x] Headless mixed-scale resolver test подтверждает отсутствие повторного умножения geometry на scale.
- [x] Adapter integration test подтверждает `ensure_desktop_app → place_window → fresh observation → Ready`.
- [x] Реальный session-bus E2E подтверждает D-Bus/Axiom placement loop до Active.
- [x] CI: module graph, gofmt, unit, race, vet, D-Bus round trip, JS syntax, Node model tests, manifests, daemon build, smoke.

### Что ещё НЕ доказано на настоящем GNOME 50

Следующие пункты остаются открытыми независимо от зелёных headless/contract tests:

- [ ] Полный E2E на Ubuntu 26.04 / GNOME Shell 50 / Wayland.
- [ ] Реальный `Shell.App.activate()` в GNOME 50 qualification environment.
- [ ] Реальный `Meta.Window.delete()` + save/cancel dialog behavior.
- [ ] Реальный `Meta.Window.move_to_monitor()` / `change_workspace_by_index()` / `move_resize_frame()` convergence.
- [ ] 100× enable/disable extension без signal/source/actor leaks.
- [ ] Nested GNOME 50 lifecycle harness в CI.
- [ ] Fractional scaling 100/125/150/200%.
- [ ] Mixed-scale multi-monitor + hotplug/rotation/primary change.
- [ ] Topology revision semantics при реальном hotplug/reorder мониторов.
- [ ] Fullscreen/maximized/non-resizable placement policy на реальных клиентах.
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
8. HWS-owned окно корректно закрывается; adopted окно не закрывается.
9. Desktop app/window identity не зависит от PID/title как durable evidence.
10. Layout применяется в logical coordinates против актуальной topology revision.
11. Hotplug/topology change не приводит к replay stale placement.
12. Focus подтверждается observed state; policy denial не маскируется под success.
13. Suspend/Resume/Close/Recover имеют проверенную desktop semantics.
14. Restart `hwsd` сохраняет/объясняет durable lifecycle state.
15. D-Bus owner restart приводит к fresh handshake/capabilities/snapshots.
16. Extension переживает 100× enable/disable без утечек.
17. Disable HWS возвращает стандартную GNOME-сессию в рабочее состояние.
18. Activity Strip корректно группирует несколько окон одного приложения.
19. Rich provider failure деградирует к window-only состоянию.
20. full/compact/micro режимы проходят width/scale matrix.
21. `hwsctl doctor` различает healthy, degraded и stale/LKG states.

## 4. Workstreams

### W0 — Repository foundation

- [x] Go module / Go 1.26.
- [x] Makefile.
- [x] CI baseline + race/D-Bus/JS/model gates.
- [x] ADR framework.
- [ ] LICENSE и совместимость с extensions.gnome.org.
- [ ] Vulnerability/dependency scan.
- [ ] `CONTRIBUTING.md`.
- [ ] top-level `SECURITY.md`.
- [ ] CODEOWNERS / branch protection policy.

### W1 — Hierarchy / Home Grid

- [x] Arbitrary-depth typed hierarchy.
- [x] Cycle/missing-parent/duplicate validation.
- [x] Deterministic ordering/path resolution.
- [x] Revisioned snapshots и LKG config.
- [x] Dynamic rows, breadcrumbs, keyboard, search/jump.
- [x] Workspace lifecycle state projection без polling hot loop.
- [ ] Context-stack back/forward history.
- [ ] DynamicQuery/provider-driven children.
- [ ] Property/fuzz tests для больших деревьев и revision churn.
- [ ] Primary Super/shortcut integration после qualification harness.

### W2 — Workspace / reconciliation

- [x] Desired/Observed state.
- [x] Required/optional resources.
- [x] Deterministic diff/evaluation.
- [x] managed/adopted/external ownership.
- [x] Normalized placement intent.
- [x] Live topology revision contract.
- [x] Logical monitor/work-area placement resolver.
- [x] Observed placement readiness.
- [x] Stale-topology rejection.
- [x] Desktop app production adapter boundary.
- [x] Pre-existing managed-intent application становится adopted без ownership evidence.
- [x] Close подтверждается disappearance tracked windows.
- [ ] Явно разложить readiness на process/window/layout/focus sub-observations вместо одного bool в публичном domain API.
- [ ] Сделать window association/evidence model явным pure-Go contract.
- [ ] Window selector/cardinality policy для нескольких окон одного desktop app.
- [ ] Focus request/observe state machine.
- [ ] Compensation policy для partially-applied placement при topology change.

### W3 — Axiom integration

- [x] Pinned Axiom dependency.
- [x] Declarative `model` frontend.
- [x] Activate/Recover/Resume/Suspend/Close.
- [x] Inactive/Preparing/Active/Degraded/Recovering/Closing/Failed.
- [x] Typed activities.
- [x] Operation-key idempotency boundary.
- [x] ProductionMode + Pebble.
- [x] Reopen persistence test.
- [x] D-Bus action/placement loop reaches Active only after observation.
- [ ] Уточнить Suspend semantics для реально оставленных running resources.
- [ ] Model-level ownership/capability claims, где они не adapter-specific.
- [ ] Crash-after-effect/before-checkpoint test.
- [ ] History/replay compatibility test.
- [ ] `Explain` service/API.

### W4 — `hwsd`

- [x] Реальный user daemon.
- [x] Session D-Bus single-instance name.
- [x] Health JSON.
- [x] Panel/hierarchy/workspace config loaders + LKG.
- [x] XDG config/state policy.
- [x] Axiom lifecycle.
- [x] Provider registry/surface aggregation.
- [x] Graceful shutdown.
- [x] Workspace catalog maintenance.
- [x] systemd --user unit artifact.
- [x] Last-known-good authoritative GNOME topology/window snapshot retained separately from rich provider aggregation.
- [ ] Structured event logging вместо преимущественно `log.Printf`.
- [ ] Crash recovery bootstrap report.
- [ ] D-Bus activation `.service`, если нужен поверх systemd user startup.
- [ ] graphical-session lifecycle binding/PartOf qualification.

### W5 — IPC / D-Bus

- [x] Protocol v1 constants.
- [x] Hello + instance + revision epoch + capabilities.
- [x] `GetTree` / `GetPath`.
- [x] `GetPanelSnapshot` / `GetPanelSpec` / `GetHealth`.
- [x] `GetApplicationSurface`.
- [x] `SubmitShellSnapshot`.
- [x] `ActivateView` / `CloseView`.
- [x] Activate/State/Recover/Suspend/Resume/Close workspace methods.
- [x] `GetWorkspaceStates` + `WorkspaceChanged`.
- [x] `ShellActionRequested` / `CompleteShellAction`.
- [x] Panel/tree/workspace signals.
- [x] Real session-bus async workspace action loop.
- [x] Real session-bus placement loop `ensure → place → observe → Active`.
- [ ] Operation object/ID + `GetOperation`.
- [ ] Explain/history methods.
- [ ] Surface change signal с bounded/coalesced payload.
- [ ] Typed focus window action, отделённый от app ensure/layout.
- [ ] Payload size/rate limits и rejection tests.
- [ ] Explicit capability negotiation tests между protocol minors.

### W6 — GNOME 50 Shell adapter / UI

- [x] Thin extension entry point.
- [x] user-session-only metadata target GNOME 50.
- [x] Reversible actor/signal/source ownership в disable path.
- [x] D-Bus owner watcher + fresh handshake.
- [x] Home Grid modal + dynamic rows/breadcrumbs/search/keyboard/error states.
- [x] Window snapshot bridge и Application grouping.
- [x] Activity Strip window-only fallback.
- [x] Semantic density full/compact/micro renderer.
- [x] Typed desktop app ensure executor.
- [x] Typed close executor.
- [x] Logical monitor topology observer/model.
- [x] Window frame observation.
- [x] Topology-guarded `place_window` executor.
- [ ] Nested GNOME 50 runtime test.
- [ ] 100× enable/disable leak test.
- [ ] Super/primary shortcut ownership/conflict policy.
- [ ] Accessibility qualification with Orca/high-contrast/large text.
- [ ] Animation/main-loop profiling.
- [ ] Real GNOME placement qualification.
- [ ] Focus executor + observed confirmation.
- [ ] Lazy window previews.
- [ ] Multi-monitor/fractional-scale qualification.

### W7 — OS integration

- [x] Desktop app discovery through Shell AppSystem.
- [x] Desktop app ensure/adopt boundary.
- [x] Session window discovery through Shell snapshot.
- [x] Logical placement → current topology work-area geometry contract.
- [ ] Generic process launch/adoption production adapter.
- [ ] Terminal launch with cwd production adapter.
- [ ] Focus request + confirmation.
- [ ] Filesystem/project discovery provider.
- [ ] Git status provider.
- [ ] SSH provider.
- [ ] Network/VPN capability boundary.
- [ ] Portals/screen-sharing compatibility.
- [ ] Ubuntu default-extension compatibility scanner.
- [ ] Third-party extension conflict classification без silent disable.

### W8 — Persistence / recovery

- [x] Pebble Axiom store + ProductionMode.
- [x] Durable lifecycle state reopen test.
- [x] Multiple workspace definition revisions retained for reopen/recovery.
- [ ] Crash after external effect / before checkpoint test.
- [ ] Store corruption/permission/disk-full tests.
- [ ] History/replay schema compatibility gate.
- [ ] Upgrade/migration policy.
- [ ] Backup/repair/doctor commands для durable state.

### W9 — Providers / ApplicationSurface

- [x] ApplicationSurface/Window/View/status/capability model.
- [x] Normalized snapshots, semantic revisions/generation и diffs.
- [x] Provider registry/health/staleness model.
- [x] GNOME Shell provider.
- [x] Browser/VS Code provider groundwork.
- [x] MPRIS + LauncherEntry collectors.
- [x] AT-SPI/native-frame/status groundwork.
- [x] Provider disconnect допускает window-only fallback.
- [ ] Provider protocol version negotiation hardening.
- [ ] Privacy/redaction URL/title/document names.
- [ ] 20/100/500-view stress profile.
- [ ] First rich provider real-app qualification.

### W10 — Panel DSL / Activity Strip

- [x] HCL dependency и strict schema.
- [x] Parser → validation → normalized Panel Model.
- [x] Invalid hot reload сохраняет previous-good layout.
- [x] Typed renderer boundary.
- [x] Application cards и surface segments/chips.
- [x] full/compact/micro density modes.
- [ ] Width matrix 800/1366/1920/4K.
- [ ] Fractional scaling matrix.
- [ ] Touch/pointer/keyboard parity qualification.
- [ ] Renderer performance baseline/update coalescing measurement.
- [ ] Config schema migration policy.

### W11 — Observability / CLI

- [x] `hwsctl health`.
- [x] `hwsctl doctor`.
- [x] panel/spec/tree/path/app diagnostics.
- [x] provider health/stale/partial reporting.
- [x] Workspace activate/state/close/recover/resume/suspend CLI commands.
- [x] Generated или explicit operation keys для idempotency testing.
- [ ] Structured correlation `intent → execution → activity → adapter → observation`.
- [ ] Reconcile timings/retry metrics.
- [ ] D-Bus reconnect metrics.
- [ ] Explain/history CLI.
- [ ] Redacted diagnostics bundle.

### W12 — Security

- [x] Threat model и no-root-by-default boundary.
- [x] No implicit shell command invariant.
- [x] Workspace/Panel strict schemas.
- [x] Shell executor принимает только typed action kinds.
- [x] Action broker fail-closed при отсутствии executor/timeout.
- [x] Adopted resources не закрываются как managed.
- [x] Placement fail-closed при stale/mismatched topology/monitor identity.
- [ ] Environment sanitization для generic process adapter.
- [ ] Path traversal/symlink race protections.
- [ ] D-Bus payload size/rate limiting.
- [ ] Provider trust/permission model.
- [ ] Browser extension minimum-permissions audit.
- [ ] Privacy/redaction implementation.

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

Headless/domain/D-Bus архитектура теперь достаточно зрелая. Главный риск — **реальное compositor/session поведение GNOME 50**, особенно lifecycle, placement, scaling и races.

Обязательная последовательность:

```text
1. Nested GNOME 50 / Wayland qualification harness
      ↓
2. 100× extension enable/disable + daemon owner restart
      ↓
3. Real Shell.App ensure/adopt/Meta.Window close
      ↓
4. Real logical placement on one monitor
      ↓
5. 100/125/150/200% fractional scale matrix
      ↓
6. Mixed-scale two-monitor placement
      ↓
7. Hotplug/primary/work-area/topology-change races
      ↓
8. Focus Request → Observe state machine
      ↓
9. Generic terminal/process adapter + VS-1
      ↓
10. Ubuntu default-extension compatibility + performance gate
```

Не начинать сложный tiling/preview/gesture layer до прохождения пунктов 1–4.

## 6. Следующие атомарные действия

1. Создать nested GNOME 50 / Wayland qualification harness, запускаемый воспроизводимо локально и из CI/runner, где доступны необходимые compositor capabilities.
2. Добавить минимальный test extension lifecycle runner и 100× enable/disable leak assertions.
3. Добавить daemon owner restart E2E с fresh Hello/capabilities/tree/workspace/topology snapshots.
4. Доказать реальный `Shell.App.activate()` и association появившегося окна.
5. Доказать реальный `Meta.Window.delete()` включая save/cancel client behavior.
6. Доказать `place_window` на GNOME 50: monitor/workspace/frame before → action → frame after.
7. Добавить test fixture для topology change между resolve и apply; ожидать `topology_changed` без ложного Active.
8. Прогнать scale 1.0/1.25/1.5/2.0 и запретить любое manual scale multiplication в domain/reconciler.
9. Прогнать два mixed-scale монитора, primary switch и work-area change.
10. После layout qualification реализовать focus как `Request → Observe → Reached | PolicyBlocked | Superseded | WindowGone | TimedOut`.
11. Затем добавить production terminal/process adapter и закончить VS-1.
12. После VS-1 зафиксировать measured performance baseline и compatibility matrix.

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

Уже доказано headless/contract tests:

- [x] versioned definition lookup;
- [x] Axiom activation/recover/close;
- [x] durable reopen;
- [x] repeated reconcile не делает лишний effect для reached resource;
- [x] adopted ownership protection;
- [x] D-Bus → Shell ensure → observed snapshot → Axiom Active;
- [x] normalized placement → typed place_window → observed frame → Active;
- [x] Home Grid leaf lifecycle integration.

Осталось доказать на настоящем desktop:

- [ ] editor запускается через Shell и корректно ассоциируется;
- [ ] already-running editor adopted без duplicate launch;
- [ ] terminal production adapter;
- [ ] optional browser placement;
- [ ] close с реальными save/cancel dialogs;
- [ ] placement/focus на GNOME 50;
- [ ] restart/recovery при реальном external effect.

## 8. Performance budgets

До измеренного GNOME baseline это targets:

- cached Home Grid navigation p95 < 16 ms;
- local hierarchy search p95 < 50 ms;
- local read IPC p95 < 20 ms без external providers;
- Shell main loop никогда не выполняет synchronous filesystem/network/database/Axiom work;
- one coalesced Surface update target < 4 ms Shell CPU work;
- visible Surface state update p95 < 50 ms после provider event;
- event storms обязаны coalesce до bounded rate;
- 50 surfaces / 500 views не должны давать per-frame O(total views) работу;
- preview создаётся лениво;
- workspace activation может быть долгой, но D-Bus/GJS callback не блокирует Shell main loop.

После nested GNOME harness targets заменяются measured thresholds.

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
- вручную домножать normalized/logical layout на monitor scale в domain/reconciler;
- применять placement, если topology revision или monitor identity не совпадает;
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
- [x] versioned workspace definitions retained;
- [ ] crash-after-effect checkpoint test;
- [ ] history/replay compatibility;
- [ ] Explain.

### M2 — Local daemon / D-Bus — FUNCTIONAL

- [x] `hwsd`;
- [x] Hello/owner epoch/capabilities;
- [x] hierarchy/panel/surface reads;
- [x] full workspace lifecycle API;
- [x] batch workspace state + change signal;
- [x] async Shell action protocol;
- [x] session-bus lifecycle/placement round trips;
- [x] systemd user artifact;
- [ ] operation/history API;
- [ ] owner-restart E2E.

### M3 — Desktop vertical slice — IN PROGRESS

- [x] thin GNOME 50 extension;
- [x] Home Grid;
- [x] window observation/ApplicationSurface bridge;
- [x] Activity Strip groundwork;
- [x] typed app ensure/window close executor;
- [x] Home Grid → Axiom lifecycle wiring;
- [x] topology observation contract;
- [x] normalized/logical placement resolver;
- [x] topology-guarded place_window executor contract;
- [x] observed placement convergence through D-Bus/Axiom tests;
- [ ] nested GNOME 50 qualification;
- [ ] real desktop VS-1;
- [ ] real fractional/multi-monitor placement;
- [ ] focus;
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
5. для D-Bus изменений добавлять real session-bus contract test, где это возможно;
6. для Shell изменений отделять pure model tests от GJS runtime code;
7. для external effects проверять observed convergence, а не acknowledgement;
8. для topology/layout изменений иметь stale-topology negative test;
9. обновлять этот plan при изменении критического пути;
10. merge в `main` делать только после зелёного verification gate.

Текущий следующий gate: **nested Ubuntu 26.04 / GNOME Shell 50 / Wayland qualification harness**. Headless placement contract считается реализованным, но реальное Mutter placement/scaling поведение — ещё нет.
