# HWS — GNOME 50 adapter contract

Статус: proposed architecture contract  
Target: Ubuntu 26.04 LTS / GNOME Shell 50 / Wayland

## 1. Назначение

`GNOMEAdapter` — единственная часть HWS, которой разрешено знать о `Shell`, `Meta`, `St`, `Clutter`, GNOME session state и конкретных особенностях Mutter.

Domain/application code не зависит от этих API.

```text
HWS domain
   ↓ abstract intents/observations
GNOMEAdapter
   ↓
GNOME Shell 50 / Mutter / Wayland
```

## 2. Разделение процесса

### Shell extension

Отвечает за:

- Home/Grid UI;
- keyboard/pointer input;
- Shell/Meta window observation;
- user-context application activation;
- window placement actions, требующие compositor access;
- monitor/workspace observation;
- D-Bus client к `hwsd`;
- projection daemon state в UI.

### `hwsd`

Отвечает за:

- domain model;
- Axiom lifecycle;
- desired/observed reconciliation;
- persistence;
- search/index;
- project discovery;
- Git/files/network/SSH;
- operation history;
- policy and capability decisions.

Shell process не импортирует daemon business logic.

## 3. Capability model

Минимальная capability schema:

```text
WindowObservation        Supported|Degraded|Unavailable
WindowMove               Supported|Constrained|Unavailable
WindowResize             Supported|Constrained|Unavailable
WindowActivate           Supported|Constrained|Unavailable
CrossMonitorMove         Supported|Constrained|Unavailable
CrossWorkspaceMove       Supported|Constrained|Unavailable
MonitorTopology          Supported|Degraded|Unavailable
WorkspaceObservation     Supported|Degraded|Unavailable
DesktopAppLaunch         Supported|Constrained|Unavailable
DesktopAppNewWindow      Supported|Constrained|Unavailable
FocusObservation         Supported|Degraded|Unavailable
ShellOverlay             Supported|Unavailable
SearchIntegration        Supported|Unavailable
```

Capability response содержит reason code и adapter version.

## 4. Window observation

Нормализованная структура:

```text
WindowObservation
- sessionWindowID
- stableSequence
- appID?
- sandboxedAppID?
- gtkApplicationID?
- startupID?
- wmClass?
- pid?                 // secondary hint only
- title                // non-identity
- windowType
- role?
- workspaceID?
- monitorRef?
- rectLogical
- maximized
- fullscreen
- minimized
- focused
- movable
- resizable
- timestamp/revision
```

### Identity rules

1. `title` никогда не является persistent identity.
2. PID не является основным application identity.
3. Для app association использовать `Shell.WindowTracker` / `Shell.App` там, где возможно.
4. `stableSequence` считать только session-local identifier.
5. После logout/restart старый window ID не восстанавливается как тот же физический объект.

## 5. Application launch

Desktop app и arbitrary process — разные resource types.

### Desktop app

Предпочтительный путь:

```text
hwsd: EnsureDesktopApp intent
        ↓
extension: Shell.App / app launch context
        ↓
Wayland startup/XDG activation semantics
        ↓
window observation
        ↓
hwsd: correlate + reconcile
```

### Generic process

Запускается `hwsd` через typed executable + argv[] policy.

Generic process launch не обещает появление/focus desktop window.

## 6. Focus model

`ActivateWindow` — request, а не гарантия.

Результат:

```text
Reached
DeniedByPolicy
Superseded
WindowGone
TimedOut
Unsupported
```

После запроса focus adapter обязан вернуть fresh focus observation или timeout.

## 7. Geometry model

В domain/layout planner используются logical coordinates и normalized fractions.

Запрещено распространять monitor scale multiplication за пределы GNOME adapter.

### LayoutIntent

```text
monitor selector
workspace selector
x/y/w/h in normalized or logical form
constraints
priority
```

Adapter преобразует intent относительно текущего `work area`, а не полного framebuffer bounds.

## 8. Monitor topology

Topology имеет monotonic revision внутри session.

```text
MonitorTopology
- revision
- primary
- monitors[]
```

Monitor ref должен позволять использовать стабильные доступные свойства (connector/name where exposed) и graceful fallback.

Изменение topology:

```text
observe change
→ increment revision
→ invalidate stale placements
→ notify hwsd
→ reconcile active workspaces
```

Нельзя replay старые absolute rectangles без re-resolution.

## 9. Workspace semantics

HWS не должен предполагать фиксированное количество workspaces.

Каждая операция проверяет current workspace objects/revision перед mutation.

Window move between workspaces является отдельной operation от geometry placement.

## 10. Lifecycle ownership

Каждый shell component отвечает за cleanup собственных:

- actors;
- signals;
- timeout/source IDs;
- cancellables;
- keybindings;
- D-Bus subscriptions;
- method injections;
- input controllers/grabs.

`disable()` extension должен делать shell state эквивалентным состоянию до `enable()` за исключением внешних ресурсов, принадлежащих daemon/user.

## 11. Session modes

HWS extension по умолчанию поддерживает только обычный `user` session mode.

Не включать lock-screen mode без отдельного ADR и threat review.

При extension re-enable:

1. создать новый D-Bus proxy/client;
2. handshake с `hwsd`;
3. получить fresh capability snapshot;
4. получить current tree revision;
5. получить current active workspace state;
6. заново наблюдать windows/topology.

## 12. D-Bus reconnect

Extension отслеживает owner well-known name `hwsd`.

При owner loss:

- не отправлять mutations;
- пометить cached daemon state stale;
- UI остаётся responsive;
- показать degraded/unavailable state без shell crash.

При новом owner:

- protocol handshake;
- revision reset;
- resubscribe;
- fresh snapshot.

## 13. Extension conflicts

Adapter предоставляет:

```text
GetEnvironmentCompatibility()
```

Содержит:

- enabled extension IDs;
- Ubuntu-specific bundled extensions where observable;
- relevant GNOME settings;
- known conflict rules;
- warnings.

HWS не отключает extension автоматически.

## 14. Error taxonomy

Минимальные коды:

```text
GNOME_UNSUPPORTED_VERSION
GNOME_CAPABILITY_MISSING
GNOME_WINDOW_GONE
GNOME_WINDOW_NOT_MOVABLE
GNOME_WINDOW_NOT_RESIZABLE
GNOME_FOCUS_DENIED
GNOME_MONITOR_GONE
GNOME_TOPOLOGY_CHANGED
GNOME_WORKSPACE_GONE
GNOME_APP_NOT_FOUND
GNOME_APP_LAUNCH_FAILED
GNOME_WINDOW_MATCH_TIMEOUT
GNOME_DBUS_DAEMON_UNAVAILABLE
GNOME_PROTOCOL_MISMATCH
GNOME_EXTENSION_CONFLICT
GNOME_SESSION_NOT_USER
```

Ошибки возвращают structured context без секретов.

## 15. Performance rules

Shell hot path:

- no synchronous filesystem I/O;
- no network;
- no database;
- no unbounded traversal per frame;
- no polling when signal/event exists;
- event storms coalesced;
- large snapshots diffed outside rendering path where possible.

Target: UI interaction should not wait for `hwsd` mutation completion.

## 16. Testing contract

### Mandatory

- GNOME 50 nested Wayland session;
- 100x extension enable/disable lifecycle stress;
- D-Bus daemon restart;
- app already running/new window;
- window disappears during operation;
- mixed window types;
- focus policies;
- 100/125/150/200% scaling;
- multi-monitor geometry;
- topology changes;
- Ubuntu default extensions;
- known conflicting extension fixture.

### Release gate

Новая GNOME major version не считается supported до прохождения adapter suite.

## 17. Explicit non-goals for GNOME 50 adapter

На первом этапе adapter не обязан:

- заменять Mutter;
- реализовывать собственный Wayland compositor;
- поддерживать GNOME X11 session;
- гарантировать независимые workspaces на каждом monitor;
- принудительно фокусировать окно вопреки Wayland policy;
- исправлять поведение произвольных несовместимых third-party extensions.

## 18. Связанные документы

- [`DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](DEVELOPER_PRACTICES_AND_FAILURE_MODES.md)
- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`INVARIANTS.md`](INVARIANTS.md)
- [`WORKSPACE_LIFECYCLE.md`](WORKSPACE_LIFECYCLE.md)
- [`AXIOM_INTEGRATION.md`](AXIOM_INTEGRATION.md)
- [`TEST_STRATEGY.md`](TEST_STRATEGY.md)
