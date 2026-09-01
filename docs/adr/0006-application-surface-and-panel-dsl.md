# ADR 0006 — Application Surface и декларативный Activity Strip

Status: Accepted

## Context

HWS организует работу вокруг контекстов и ресурсов, поэтому классическая taskbar-модель `иконка приложения + индикатор запуска` слишком бедна. Пользователю важно видеть активный документ/вкладку, количество окон, progress, attention, dirty state, терминальную/SSH-сессию и другие признаки текущей работы.

GNOME/Wayland надёжно предоставляет модель приложений и окон, но внутренние вкладки и документы принадлежат приложениям и не являются универсальным свойством compositor window. Попытка угадывать tabs из window title создаёт нестабильную identity model и тесно связывает Shell adapter с конкретными приложениями.

Панель также должна быть настраиваемой, но пользовательский config не должен исполнять arbitrary commands или напрямую создавать GNOME objects.

## Decision

1. Ввести first-class domain projection `ApplicationSurface`.
2. Одна ApplicationSurface агрегирует application state, окна и provider-defined внутренние `views`.
3. `SurfaceView` является общим понятием для browser tab, IDE document tab, terminal session/pane и подобных внутренних представлений.
4. Window observation поступает через GNOME Shell adapter; tabs/views поступают через независимые application providers.
5. Provider priority: native/app API → HWS plugin/extension → AT-SPI read-only fallback → metadata heuristic.
6. Activity Strip является resource-oriented panel, а не taskbar; позже он может отображать builds, SSH, AI agents, transfers и services.
7. Одна карточка по умолчанию соответствует одному ApplicationSurface; несколько окон раскрываются как group, несколько views показываются как active/recent/important + overflow.
8. Карточки имеют semantic zoom `full → compact → micro` и constraint-based sizing.
9. Пользовательская конфигурация панели использует HCL-based declarative DSL с собственной строгой schema HWS.
10. DSL компилируется в normalized Panel Model и не создаёт renderer objects напрямую.
11. DSL v1 не поддерживает arbitrary shell, filesystem/network calls или unrestricted expressions.
12. Любое действие с внешним effect проходит typed action/capability boundary через `hwsd`.
13. PID, window title и tab index не являются durable identity.
14. Browser URLs/titles, document names и другие potentially sensitive fields подчиняются отдельной redaction/privacy policy.

## Consequences

### Positive

- панель показывает рабочее состояние, а не только процессы;
- одна модель подходит browser/IDE/terminal и будущим resource cards;
- отсутствие tab provider корректно деградирует к window-only UI;
- GNOME extension остаётся тонким adapter-ом;
- panel config остаётся стабильным при смене renderer implementation;
- semantic zoom позволяет использовать одну модель на 800 px и 4K экранах;
- typed capabilities предотвращают UI affordance для неподдерживаемых действий.

### Negative

- rich tabs/views требуют отдельной provider ecosystem;
- browser/IDE integrations увеличивают compatibility matrix;
- HCL добавляет внешнюю dependency и требует schema/versioning policy;
- aggregation нескольких asynchronous providers усложняет stale-state/reconnect semantics;
- privacy model становится обязательной частью архитектуры.

## Rejected alternatives

### Обычная taskbar с иконками

Недостаточно информации для task-first HWS и плохо отражает windows/tabs/operations.

### Одна карточка на каждое окно

Плохо масштабируется при большом числе окон и теряет application-level grouping.

### Парсить заголовки окон для вкладок

Нестабильно, не даёт надёжной identity/capabilities и ломается при локализации/изменениях приложений.

### AT-SPI как основной источник tabs

Accessibility tree полезен как fallback, но не гарантирует устойчивую identity, полноту и одинаковую семантику во всех приложениях.

### JSON/YAML как единственный panel format

Подходит для простых settings, но быстро становится громоздким для вложенных blocks, constraints и будущих безопасных expressions.

### QML как публичный config API

Слишком тесно связывает user configuration с renderer/runtime и даёт слишком широкую поверхность выполнения логики.

## Validation

Решение считается подтверждённым после реализации AS-0..AS-4 из `docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md` и прохождения тестов:

- app grouping для нескольких окон;
- MRU window navigation;
- минимум один реальный view/tab provider;
- graceful window-only fallback;
- HCL parse/schema/hot-reload failure tests;
- privacy/redaction tests;
- 800/1366/1920/4K + fractional scaling checks;
- Shell main-loop performance budget.
