# HWS — исследование лучших практик и типовых отказов разработки

Статус: research-backed architecture input  
Дата: 2026-09-01  
Целевая среда: Ubuntu 26.04 LTS, GNOME Shell 50, Wayland

## 1. Зачем этот документ

HWS вмешивается в один из наиболее чувствительных слоёв пользовательской системы: GNOME Shell, управление окнами, запуск приложений, рабочие области, ввод и восстановление рабочего окружения. Ошибка обычного desktop-приложения обычно ломает только это приложение. Ошибка GNOME Shell extension способна испортить или аварийно завершить всю графическую сессию.

Цель документа — собрать практики и реальные классы проблем, с которыми сталкиваются авторы GNOME Shell extensions и тайлинговых оболочек, и превратить их в обязательные требования HWS.

Источники разделены на:

- официальную документацию GNOME/GJS/Mutter/GIO;
- Ubuntu 26.04 documentation/packages;
- реальные issue/release history зрелых GNOME window-management extensions;
- спецификации D-Bus/systemd/Wayland.

## 2. Главные выводы

### 2.1. Shell extension должен быть минимальным

Код `extension.js` работает внутри процесса `gnome-shell`. Ошибка, блокировка main loop, утечка signal handler или неверная работа с actor может повредить весь desktop.

Следствие для HWS:

```text
GNOME Shell extension
    = presentation + input + window/session adapter

hwsd
    = domain + search + persistence + orchestration + heavy work
```

Extension не должен превращаться в монолитный desktop application.

### 2.2. Ubuntu 26.04 — Wayland-only для GNOME session

Ubuntu 26.04 продолжает изменение, сделанное в 25.10: GNOME Shell больше не запускается как X.org session. X11-приложения продолжают работать через XWayland.

Следствие: HWS проектируется Wayland-first/Wayland-only на уровне GNOME adapter. X11-specific обходы не должны становиться архитектурой продукта.

### 2.3. Основная сложность — не layout-алгоритм, а lifecycle и взаимодействия

Реальные баги PaperWM/Tiling Shell показывают повторяющиеся классы:

- GNOME API changes;
- fractional scaling;
- multi-monitor geometry;
- focus policy;
- gesture state machines;
- stale input grabs;
- signal leaks/races;
- конфликт extensions;
- Wayland/XWayland различия;
- неверная идентификация новых окон.

### 2.4. Нельзя считать окно продолжением процесса

На Wayland PID и title не являются надёжной идентичностью приложения. GNOME сам использует application id / `.desktop` association / WM_CLASS heuristics через `Shell.WindowTracker`.

Для HWS идентификация ресурса должна быть multi-signal и вероятностно-подтверждаемой, а не `pid == launchedPid`.

### 2.5. Запуск и фокус — разные операции

Wayland защищает пользователя от focus stealing. Правильный app launch использует startup notification / XDG activation token. Запуск из фонового Go daemon без корректного activation context может создать окно, но не дать желаемого фокуса.

HWS должен разделить:

```text
EnsureApplicationRunning
EnsureWindowExists
PlaceWindow
RequestActivation
VerifyFocus
```

а не реализовывать всё одной функцией `Launch()`.

---

# 3. Практика №1 — минимальный Shell process

## Проблема

GNOME extension фактически загружается внутрь `gnome-shell`. Официальная документация прямо предупреждает: fatal errors и ошибки extension влияют на стабильность desktop.

GNOME рекомендует:

- маленький entry point;
- модули с одной ответственностью;
- тяжёлые задачи выносить во внешний процесс;
- использовать D-Bus вместо частого запуска shell-команд.

## Правило HWS

В shell process разрешены:

- построение St/Clutter UI;
- обработка input;
- чтение необходимых Shell/Meta объектов;
- короткие локальные projections;
- D-Bus communication;
- lightweight event coalescing.

Запрещены:

- Git scanning;
- filesystem crawling;
- network requests как часть navigation path;
- SQLite/Pebble operations;
- Axiom runtime;
- SSH;
- VPN/network orchestration;
- долгое JSON parsing/large transformations;
- синхронное ожидание subprocess.

## Проверка

Добавить архитектурный lint/checklist: shell package не должен импортировать или реализовывать инфраструктуру daemon-domain уровня.

---

# 4. Практика №2 — lifecycle симметричен enable/disable

## Проблема

Официальные EGO guidelines называют cleanup одним из центральных требований. GNOME может вызвать `disable()` не только при ручном отключении extension, но и при смене session mode, например при блокировке экрана.

Все созданные extension ресурсы должны быть отменены/удалены:

- actors/widgets;
- signals;
- GLib sources/timeouts;
- cancellables;
- D-Bus subscriptions;
- method injections;
- keybindings;
- временные Shell modifications.

Реальные release notes Tiling Shell отдельно фиксируют исправления неправильного порядка disconnect, race condition в Alt-Tab и signal-handler leak.

## Правило HWS

Каждый компонент shell adapter владеет своим lifecycle:

```text
create/connect/start
        ↓
      owner
        ↓
disconnect/cancel/destroy
```

Нельзя создавать resource в одном классе, а уничтожать в глобальном `disable()` другого слоя.

## Обязательные тесты

Повторить минимум 100 циклов:

```text
enable → interact → disable → enable
```

и проверять:

- отсутствие дублирующихся callbacks;
- отсутствие оставшихся actors;
- отсутствие лишних shortcuts;
- отсутствие timeout/source leaks;
- отсутствие повторной подписки на hwsd events.

---

# 5. Практика №3 — никакого синхронного I/O в Shell hot path

## Проблема

GNOME Extensions review tooling отдельно предупреждает о synchronous file I/O в shell code. Shell делит main loop с rendering/input/compositor-side logic.

Даже «небольшая» синхронная операция может проявиться как:

- frame hitch;
- задержка открытия Overview;
- потерянный input;
- perceived freeze.

## Правило HWS

Navigation rendering работает только с уже доступным cache/snapshot.

```text
UI asks hwsd for refresh
      ↓
old snapshot remains usable
      ↓
new revision arrives asynchronously
      ↓
projection swapped atomically
```

Никакой dynamic provider не имеет права блокировать отрисовку текущего дерева.

---

# 6. Практика №4 — D-Bus как граница Shell ↔ hwsd

## Почему

GNOME best practices прямо рекомендуют внешний background process + D-Bus для heavy tasks.

D-Bus хорошо подходит HWS, потому что:

- session-scoped;
- имеет well-known bus name;
- поддерживает service activation;
- естественно обнаруживает restart через смену owner;
- хорошо интегрируется с GIO/GJS.

## Проблемы разработчиков

Типовые ошибки:

- считать существование proxy доказательством живого daemon;
- не обрабатывать disappearance/reappearance bus owner;
- использовать бесконечные sync calls;
- не version API;
- передавать по session bus данные, считая его privilege boundary.

`Gio.DBusProxy` умеет отслеживать `g-name-owner`; cache очищается при исчезновении owner и восстанавливается при появлении нового.

## Правило HWS

Shell UI имеет состояния:

```text
DaemonUnavailable
Connecting
Ready(protocolVersion, capabilities)
StaleSnapshot
ProtocolMismatch
```

После смены D-Bus owner UI обязан повторно:

1. выполнить handshake;
2. проверить API version;
3. получить capability snapshot;
4. получить fresh tree/workspace revision;
5. только затем принимать event stream как актуальный.

## systemd

`hwsd` должен быть user/session service, связанный с `graphical-session.target` или D-Bus activation, а не root system daemon.

---

# 7. Практика №5 — version pinning и compatibility matrix GNOME

## Проблема

GNOME Shell extensions не имеют гарантированно стабильного API на уровне внутренних Shell JS modules. GNOME публикует porting guides именно потому, что extension иногда требуется адаптировать к каждому major release.

Реальный пример GNOME 50:

- PaperWM ломался из-за исчезнувшего API `is_wayland_compositor`;
- `Gio.DesktopAppInfo` перемещён в `GioUnix`;
- GNOME 50 окончательно убрал Shell X11 restart path;
- GNOME 51 уже несёт дальнейшие изменения input/controller API и D-Bus proxy wrapper semantics.

## Правило HWS

Не писать «универсальный слой на все GNOME» через десятки `typeof foo === function`.

Вместо этого:

```text
adapter/gnome50
adapter/gnome51   (когда понадобится)
```

или тонкий explicit compatibility layer с подтверждёнными capability branches.

## Release policy

- поддержка версии Shell объявляется только после тестирования;
- CI может тестировать development release заранее;
- metadata не должна заявлять будущие стабильные версии «на всякий случай»;
- обновление GNOME считается отдельным compatibility event.

---

# 8. Практика №6 — Wayland-first, без X11 architectural fallback

## Факт

Ubuntu 26.04 GNOME session работает только на Wayland. X11 applications идут через XWayland.

## Следствия

Запрещено строить core behaviour вокруг:

- `wmctrl`;
- `xdotool`;
- EWMH-only assumptions;
- X11 window IDs как durable identity;
- Alt+F2 `r` как development/recovery mechanism.

Подобные инструменты могут существовать только как optional legacy integration для других desktop/session, но не как основа HWS GNOME adapter.

---

# 9. Практика №7 — app/window identity должна быть многослойной

## Проблема

`Meta.Window.get_pid()` недостаточен:

- Wayland не обязан давать X11-style PID semantics;
- одно приложение может иметь несколько процессов;
- один процесс может создавать несколько окон;
- application can remote-activate already running instance;
- portals/sandboxing меняют process relationships.

GNOME предоставляет `Shell.WindowTracker` для mapping `Meta.Window → Shell.App` и использует application metadata/WM_CLASS heuristics.

## Рекомендуемая identity модель HWS

Приоритет сигналов:

1. `Shell.App` / desktop app id;
2. sandboxed app id;
3. GTK application id;
4. Startup ID / launch correlation token;
5. WM_CLASS / StartupWMClass для XWayland;
6. PID только как дополнительное доказательство;
7. title — только selector/heuristic, никогда durable ID.

## WindowObservation

```text
WindowObservation
- stableSequence (session-local)
- appID
- sandboxedAppID
- gtkApplicationID
- startupID
- wmClass
- pid?
- workspace
- monitor
- rect
- role/type
- title (non-identity)
```

`stableSequence` не переносится между logins/restarts как persistent ID.

---

# 10. Практика №8 — app launch должен учитывать XDG activation

## Проблема

На Wayland compositor может отказать фоновой программе в фокусе. XDG activation token связывает пользовательское действие с запуском/активацией surface.

GNOME Shell имеет `Shell.Global.create_app_launch_context(timestamp, workspace)`; `Shell.App.activate_full()` и launch path умеют учитывать current event timestamp/workspace.

## Следствие для HWS

Для desktop apps предпочтителен запуск через Shell/GIO application semantics, а не через голый `exec` в daemon.

Рекомендуемый split:

```text
hwsd decides WHAT must exist
        ↓
Shell adapter performs user-session launch/activation
        ↓
observed window is correlated back to hwsd
```

Это не означает перенос orchestration в extension: extension является capability executor для операций, которые требуют Shell interaction context.

## Отдельные activities

- `EnsureAppRunning`;
- `AwaitWindowMatch`;
- `MoveWindow`;
- `ResizeWindow`;
- `ActivateWindow`;
- `VerifyWindowState`.

Каждая имеет отдельный timeout/error reason.

---

# 11. Практика №9 — geometry только в логических координатах

## Проблема

Tiling Shell получил реальный GNOME 50 regression: snap assistant увеличивался пропорционально scale factor при >100% scaling. Причина — собственная логика определения fractional scaling после изменения GNOME semantics.

## Правило HWS

Нельзя вручную умножать geometry на monitor scale «потому что монитор 2x» без документированного boundary.

Внутренняя layout-модель HWS использует normalized/logical coordinates.

```text
LayoutIntent: fractions / logical units
              ↓
GNOME adapter
              ↓
Meta work area / protocol-stage conversion where required
```

## Матрица тестов

Минимум:

- 100%;
- 125%;
- 150%;
- 200%;
- два монитора с одинаковым scale;
- два монитора с разным scale;
- secondary слева/справа/сверху/снизу;
- hotplug;
- primary monitor change.

---

# 12. Практика №10 — multi-monitor является отдельной state machine

## Проблема

Monitor index нестабилен при hotplug/reconfiguration. Work area меняется из-за panels/docks/struts. Разные scale factors и расположение мониторов создают отрицательные global coordinates.

Реальные Mutter/GNOME 50 reports показывают баги, проявляющиеся только при external monitor слева от primary.

## Правило HWS

Workspace layout не хранит «window X always monitor 1» как единственную идентичность.

Нужна `MonitorSelector` policy:

```text
primary
focused
same-as-resource(X)
connector/name preference
relative role: left/right/external
fallback: any
```

После monitor topology change выполняется reconcile, а не blind replay старых rectangles.

---

# 13. Практика №11 — focus и input нельзя считать детерминированными

## Реальные проблемы

PaperWM issues включают:

- focus возвращается к окну под мышью при `mouse/sloppy focus`;
- fullscreen требует дополнительного клика;
- gesture transition может зависнуть;
- stale grab остаётся после race при создании Chrome window.

## Правило HWS

Focus является observed state, а не гарантированным side effect.

```text
RequestFocus(window)
      ↓
wait bounded interval
      ↓
ObserveFocus
      ↓
Reached | PolicyBlocked | Superseded
```

Никогда не удерживать input grab дольше минимального interaction scope.

На любом exceptional exit grab/controller state должен освобождаться через owner cleanup.

---

# 14. Практика №12 — extensions конфликтуют друг с другом

## Факт

PaperWM явно документирует несовместимость с extensions, которые меняют:

- desktop;
- window shapes;
- workspaces;
- touch gestures;
- panel/workspace behaviour.

Ubuntu сама поставляет собственные GNOME extensions; historical Ubuntu reports показывают реальные crash/conflict cases, например сочетание Ubuntu Dock и Dash-to-Panel.

## Следствие для HWS

HWS не может предполагать «чистый GNOME».

### Compatibility scanner

При старте adapter собирает:

- enabled extensions;
- known capability overlaps;
- known incompatible extensions;
- changed GNOME settings relevant to workspace/window behaviour.

Результат:

```text
Compatible
PotentialConflict
KnownConflict
UnsupportedCombination
```

HWS не должен автоматически отключать чужие extensions без явного consent.

## Особо важно для Ubuntu

Тестировать HWS как минимум в двух профилях:

1. чистый upstream-like GNOME;
2. стандартная Ubuntu 26.04 desktop session с Ubuntu extensions.

---

# 15. Практика №13 — lock/unlock/suspend являются нормальным lifecycle

## Проблема

GNOME может отключить/включить extension при изменении session mode. Это не exceptional event.

## Правило HWS

Shell extension по умолчанию работает только в `user` mode.

Не поддерживать `unlock-dialog`, если для product requirement нет строгой причины.

При lock:

- UI исчезает;
- input handlers освобождаются;
- extension-side subscriptions корректно закрываются;
- `hwsd` может сохранить durable orchestration, но не должен делать предположение, что Shell adapter доступен.

После unlock выполняется fresh capability/window observation.

---

# 16. Практика №14 — иерархия должна быть глубокой в модели, но неглубокой в когнитивной нагрузке

## Конфликт требований

HWS сознательно допускает дерево произвольной глубины. GNOME HIG при этом предупреждает против чрезмерно глубоких visible hierarchies и рекомендует progressive disclosure.

Это не означает отказ от модели HWS. Это означает разделение:

```text
model depth = arbitrary
visible decision depth = controlled
```

## UX правила

- показывать текущую полосу/уровень и следующий meaningful level;
- breadcrumbs всегда дают быстрый подъём;
- global search может перепрыгивать уровни;
- recent/favorite nodes могут быть shortcuts, не меняя canonical hierarchy;
- не требовать 7 кликов для часто используемой task;
- при длинном path показывать компактную breadcrumb projection.

## Accessibility

Каждый tile:

- имеет accessible name;
- доступен keyboard focus;
- имеет визуальный focus state;
- активируется Enter/Space;
- grid поддерживает directional navigation;
- Escape выполняет предсказуемый Back/Close;
- UI тестируется keyboard-only, large text, high contrast, screen reader.

---

# 17. Практика №15 — predictable escape hatch обязателен

## Почему

Поскольку extension может влиять на Shell, пользователь должен иметь гарантированный путь выхода.

## Требования

HWS обязан иметь:

- команду отключения extension;
- user-service stop;
- safe mode, в котором daemon работает без window mutations;
- documented recovery из TTY;
- отсутствие irreversible изменений GNOME settings;
- restore ledger для временно изменённых settings;
- диагностическую команду `hwsctl doctor`;
- команду `hwsctl disable-shell-integration`.

При disabled extension базовый Ubuntu desktop должен оставаться полностью usable.

---

# 18. Практика №16 — nested GNOME tests до тестирования в основной сессии

GNOME 49+ предоставляет mutter-devkit/nested Wayland development. GNOME 50 также добавил `gnome-shell-test-tool --extension`.

## Пирамида HWS

### Layer A — pure Go

- domain;
- hierarchy;
- layout planner;
- reconciler;
- Axiom model;
- property/fuzz/mutation tests.

### Layer B — fake shell adapter

Детерминированные window/session events и fault injection.

### Layer C — nested GNOME 50

Автоматизированно:

- load extension;
- open grid;
- keyboard navigation;
- app launch;
- window observe/place;
- disable/enable;
- lock-like lifecycle where possible.

### Layer D — VM/image matrix

- Ubuntu 26.04 clean;
- Ubuntu 26.04 defaults;
- NVIDIA Wayland;
- AMD/Intel;
- mixed scaling;
- multi-monitor virtual topology where test harness позволяет.

### Layer E — canary/manual

Только после предыдущих слоёв — основная пользовательская desktop session.

---

# 19. Практика №17 — event-driven reconciliation вместо command scripts

## Проблема

Окна создаются асинхронно. События могут приходить в другом порядке. Приложение может открыть splash/dialog до main window. Existing instance может remote-activate instead of spawning.

Надёжная orchestration не выглядит так:

```text
launch
sleep 1
find by title
move
```

Она выглядит так:

```text
record intent
launch/activate
subscribe/observe
match candidate windows
wait until condition or deadline
apply layout
observe result
```

## HWS consequence

Axiom activity должна завершаться по наблюдаемому condition, а не по факту успешного syscall/DBus return.

---

# 20. Практика №18 — Axiom не должен управлять compositor frame-by-frame

Axiom подходит для:

- ActivateWorkspace;
- RecoverWorkspace;
- CloseWorkspace;
- resource lifecycle;
- retries;
- claims;
- history/explain.

Axiom не подходит для:

- drag pointer motion;
- hover;
- animation frames;
- focus ring updates;
- every Meta.Window signal;
- geometry preview during drag.

Правильная граница:

```text
high-frequency Shell events
        ↓ coalesce/project
ObservedState revision
        ↓
hwsd reconciler / Axiom decision
```

---

# 21. Практика №19 — capability model вместо скрытых assumptions

GNOME adapter должен явно сообщать возможности:

```text
CanObserveWindows
CanMoveWindows
CanResizeWindows
CanMoveAcrossMonitors
CanMoveAcrossWorkspaces
CanActivateWithUserContext
CanCreateWorkspace
CanObserveFocus
CanObserveMonitorTopology
CanLaunchDesktopApp
CanOpenNewWindow
```

Capability может иметь не только bool:

```text
Supported
SupportedWithConstraints
Unavailable
Degraded(reason)
```

Это позволяет HWS честно работать с приложениями, которые нельзя resize/move, modal dialogs, fullscreen и будущими GNOME changes.

---

# 22. Практика №20 — release engineering важнее «поддерживаем все версии»

Tiling Shell публично описывает рост стоимости тестирования множества GNOME versions и hardware configurations и перешёл к release-candidate модели.

Forge показывает другой риск: mature extension может остаться без активного maintainer.

## Правило HWS

Лучше качественно поддерживать:

```text
Ubuntu 26.04 / GNOME 50
```

чем заявлять 45–51 без реального тестового покрытия.

### Release gates

Перед поддержкой новой GNOME major:

1. прочитать official porting guide;
2. собрать adapter;
3. запустить nested tests;
4. прогнать Ubuntu default-extension compatibility;
5. прогнать scaling/multi-monitor matrix;
6. провести canary period;
7. только затем обновить supported shell metadata.

---

# 23. Risk matrix

| Риск | Вероятность | Влияние | Основная защита |
|---|---:|---:|---|
| Shell main-loop blocking | средняя | критическое | minimal extension, async D-Bus |
| Leak signal/timeout/actor | высокая | высокое | ownership + lifecycle stress test |
| GNOME major API break | высокая | высокое | versioned adapter + porting CI |
| Wrong window identity | высокая | высокое | WindowTracker + multi-signal matching |
| Focus denied by Wayland policy | высокая | среднее | activation context + observed focus |
| Fractional scaling geometry bug | высокая | высокое | logical coordinates + matrix tests |
| Multi-monitor hotplug race | средняя | высокое | topology revision + reconcile |
| Extension conflict | высокая | высокое | compatibility scanner + safe degrade |
| hwsd restart | средняя | среднее | D-Bus owner handshake + fresh snapshot |
| Shell extension restart/lock | высокая | среднее | fully symmetric enable/disable |
| Stale input grab/gesture state | средняя | критическое | scoped ownership + forced cleanup |
| Axiom retry duplicates side effect | средняя | высокое | idempotency + observed-state reconcile |
| User cannot recover desktop | низкая при правильном дизайне | критическое | escape hatch + overlay-first architecture |
| Too-deep navigation | средняя | среднее | search, breadcrumbs, shortcuts, progressive disclosure |

---

# 24. Обязательные архитектурные решения после исследования

1. Ubuntu 26.04 GNOME adapter считается Wayland-only.
2. Shell extension остаётся thin client/adapter, `hwsd` — основной application process.
3. D-Bus становится предпочтительным IPC и должен переживать daemon owner changes.
4. Desktop application launch/activation split проектируется отдельно от generic process launch.
5. GNOME app/window identity использует Shell/desktop application semantics, PID — secondary signal.
6. Layout хранится в logical/normalized coordinates.
7. Monitor topology имеет revision и вызывает reconcile.
8. Extension lifecycle должен быть полностью reversible.
9. Compatibility с Ubuntu default extensions является release gate.
10. GNOME version support проверяется per-major; не заявляется заранее.
11. Nested GNOME tests являются обязательным CI layer.
12. HWS предоставляет safe disable/recovery path до первого desktop MVP.

---

# 25. Предлагаемые новые тесты

## Shell lifecycle

- 100x enable/disable;
- daemon disappears/reappears;
- lock/unlock-like disable/enable;
- extension disabled во время workspace activation;
- hwsd restart during active UI.

## Window races

- main window delayed;
- splash first;
- modal first;
- two matching app windows appear simultaneously;
- app already running;
- app remote-activates existing process;
- window closes between observation and mutation.

## Focus

- click-to-focus;
- mouse focus;
- sloppy focus;
- fullscreen;
- modal dialog;
- activation without fresh user timestamp.

## Display

- 100/125/150/200%;
- mixed DPI;
- monitor left of primary;
- hotplug;
- primary change;
- dock/work-area change while active.

## Compatibility

- Ubuntu default extensions;
- HWS + one known workspace modifier;
- HWS + one dock/panel modifier;
- conflict detection without automatic destructive action.

---

# 26. Source map

## Official GNOME / GJS

- GNOME Extensions architecture and guides: https://gjs.guide/extensions/
- Best practices: https://gjs.guide/extensions/review-guidelines/best-practices.html
- Review guidelines: https://gjs.guide/extensions/review-guidelines/review-guidelines.html
- Updates and breakage: https://gjs.guide/extensions/overview/updates-and-breakage.html
- Anatomy/process isolation: https://gjs.guide/extensions/overview/anatomy.html
- Session modes: https://gjs.guide/extensions/topics/session-modes.html
- GNOME 50 porting: https://gjs.guide/extensions/upgrading/gnome-shell-50.html
- GNOME 51 porting (forward compatibility watch): https://gjs.guide/extensions/upgrading/gnome-shell-51.html
- Extension development/nested shell: https://gjs.guide/extensions/development/creating.html
- Debugging: https://gjs.guide/extensions/development/debugging.html
- Preferences/process split: https://gjs.guide/extensions/development/preferences.html
- `Shell.WindowTracker`: https://gnome.pages.gitlab.gnome.org/gnome-shell/shell/class.WindowTracker.html
- `Shell.App`: https://gnome.pages.gitlab.gnome.org/gnome-shell/shell/class.App.html
- `Shell.Global.create_app_launch_context`: https://gnome.pages.gitlab.gnome.org/gnome-shell/shell/method.Global.create_app_launch_context.html
- `Meta.Window`: https://gnome.pages.gitlab.gnome.org/mutter/meta/class.Window.html
- `Meta.Workspace`: https://gnome.pages.gitlab.gnome.org/mutter/meta/class.Workspace.html
- `St.Widget`: https://gnome.pages.gitlab.gnome.org/gnome-shell/st/class.Widget.html

## Ubuntu

- Ubuntu 26.04 LTS summary / Wayland-only GNOME session: https://documentation.ubuntu.com/release-notes/26.04/summary-for-lts-users/
- Ubuntu 26.04 GNOME extension packages: https://packages.ubuntu.com/search?keywords=gnome-shell&suite=resolute

## D-Bus / systemd / Wayland

- D-Bus specification: https://dbus.freedesktop.org/doc/dbus-specification.html
- D-Bus API design: https://dbus.freedesktop.org/doc/dbus-api-design.html
- `Gio.DBusProxy`: https://docs.gtk.org/gio/class.DBusProxy.html
- graphical session service target: https://man7.org/linux/man-pages/man7/systemd.special.7.html
- XDG activation: https://wayland.app/protocols/xdg-activation-v1
- GIO startup notification: https://gnome.pages.gitlab.gnome.org/gtk/gio/method.AppLaunchContext.get_startup_notify_id.html

## Real-world GNOME window manager extensions

- PaperWM: https://github.com/paperwm/PaperWM
- PaperWM issues: https://github.com/paperwm/PaperWM/issues
- Tiling Shell: https://github.com/domferr/tilingshell
- Tiling Shell releases: https://github.com/domferr/tilingshell/releases
- GNOME 50 scaling regression example: https://github.com/domferr/tilingshell/issues/541
- GNOME 50 rendering/tiling issue example: https://github.com/domferr/tilingshell/issues/579
- Forge: https://github.com/forge-ext/forge

## GNOME HIG

- Keyboard: https://developer.gnome.org/hig/guidelines/keyboard.html
- Navigation: https://developer.gnome.org/hig/guidelines/navigation.html
- Accessibility: https://developer.gnome.org/hig/guidelines/accessibility.html
- Design principles: https://developer.gnome.org/hig/principles.html

---

# 27. Вывод для HWS

Самая безопасная траектория разработки не такая:

```text
сразу написать большой GNOME extension
→ добавить тайлинг
→ потом бороться с race/leaks/conflicts
```

а такая:

```text
pure domain + reconciler
→ Axiom lifecycle
→ fake shell adapter
→ thin GNOME 50 adapter
→ nested Wayland tests
→ Ubuntu default-extension compatibility
→ scaling/multi-monitor matrix
→ canary
→ production desktop use
```

HWS должен рассматривать GNOME Shell как высокорисковую real-time-ish UI boundary: минимум кода, минимум блокирующей работы, полностью обратимый lifecycle и максимальная проверяемость вне процесса Shell.