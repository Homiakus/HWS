# HWS — Hierarchical Workspace Shell

HWS — проект контекстной иерархической рабочей оболочки для Ubuntu. Цель HWS — заменить модель «найди приложение → найди окно → восстанови контекст» на модель «выбери область → проект → задачу → получи готовое рабочее окружение».

> Статус: ранняя архитектурная стадия. Документация является living specification и должна изменяться вместе с подтверждёнными решениями и реализацией.

## Главная идея

Главный экран HWS представляет собой динамическую иерархическую сетку. Верхний ряд содержит области работы, следующий — контекст выбранной области, затем проект/подсистему, затем действия и рабочие представления. Число уровней не фиксировано.

```text
DEV        CAD        RESEARCH        SYSTEM
 ↓
Projects   Agents     Servers         Tools
 ↓
HWS        Axiom      CYTOLOGY-AI     NXGO
 ↓
Develop    Test       Inspect         Run
```

При выборе узла перестраиваются все уровни ниже него. Выбор конечного рабочего контекста может восстановить приложения, окна, каталоги, терминалы, соединения, состояние проекта и раскладку рабочего пространства.

## Базовая архитектура

HWS не начинает с форка оконного менеджера или compositor. Первый этап строится как управляемый слой над стандартной графической сессией Ubuntu:

```text
┌─────────────────────────────────────────────────────────────┐
│                    HWS Shell UI                             │
│ hierarchy grid · focus · Activity Strip · search · context │
└──────────────────────────────┬──────────────────────────────┘
                               │ IPC / D-Bus
┌──────────────────────────────▼──────────────────────────────┐
│                      hwsd (Go)                             │
│ context · workspace · surfaces · actions · persistence     │
│ git · processes · systemd · network · providers            │
└──────────────┬───────────────────────┬──────────────────────┘
               │                       │
        ┌──────▼──────┐         ┌──────▼────────────────┐
        │    Axiom    │         │ OS integration layer │
        │ verified FSM│         │ GNOME / system APIs  │
        └──────┬──────┘         └───────────────────────┘
               │
        ┌──────▼──────┐
        │ durable data│
        │ + history   │
        └─────────────┘
```

## Application Surfaces и Activity Strip

HWS не использует классическую taskbar как основную модель активной работы. Вместо «иконка приложения + точка запуска» оболочка строит `ApplicationSurface` — проекцию приложения, его окон, доступных внутренних views/tabs и значимых состояний.

Одна карточка Activity Strip по умолчанию соответствует одному приложению и адаптивно показывает текущий контекст, состояние, progress, количество окон и, если provider это умеет, последние/важные вкладки или документы.

```text
Zed                 Firefox              Terminal
HWS · panel.go       GitHub / HWS         ssh · dev-vps
Tests 42%            [HWS][CI][Docs]+9    build running
```

Несколько окон раскрываются как группа. Rich tabs/views не угадываются из заголовков окон: они приходят от browser/IDE/terminal providers, а отсутствие такого provider-а корректно деградирует к window-only модели.

Пользовательская компоновка панели проектируется как декларативный HCL-based DSL, который компилируется в renderer-neutral `Panel Model`. DSL не выполняет arbitrary shell commands и не получает прямого доступа к GNOME objects.

Подробности: [`docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md`](docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md).

## Роль Axiom

HWS закладывает `github.com/Homiakus/axiom` как обязательный архитектурный компонент для проверяемых значимых переходов состояния.

Axiom **не используется для каждого hover/click/animation кадра UI**. Быстрое эфемерное состояние интерфейса остаётся внутри UI-процесса.

Через Axiom должны проходить операции, для которых нужны инварианты, объяснимость, история, retry/idempotency и безопасное восстановление, например:

- открытие и восстановление workspace;
- активация рабочего контекста;
- запуск набора приложений и сервисов;
- применение раскладки окон;
- подключение к удалённой машине;
- операции с VPN/сетевым профилем;
- переключение project mode;
- восстановление после частично выполненной операции;
- системные действия, имеющие внешний эффект.

Интеграционная граница описана в [`docs/AXIOM_INTEGRATION.md`](docs/AXIOM_INTEGRATION.md).

## Ключевые принципы

1. **Task-first, а не app-first.** Пользователь выбирает намерение и контекст, а не вручную собирает окружение из приложений.
2. **Иерархия без фиксированной глубины.** Любой узел может иметь дочерние узлы и собственные действия.
3. **Контекст как объект первого класса.** Он имеет идентификатор, состояние, layout, ресурсы, историю и правила восстановления.
4. **Мгновенный UI.** Навигация по сетке не должна блокироваться durable runtime или внешними сервисами.
5. **Проверяемые side effects.** Значимые переходы идут через Axiom и явные activity boundaries.
6. **Fail-safe восстановление.** Частично запущенный workspace не должен оставлять систему в неописанном состоянии.
7. **Explainability.** Пользователь и разработчик должны понимать, почему HWS выполнил действие и в каком состоянии находится контекст.
8. **Keyboard-first, но не keyboard-only.** Полноценная мышь/тач/клавиатура, быстрые последовательности и поиск.
9. **Не ломать базовую Ubuntu на первом этапе.** HWS должен быть отключаемым/восстанавливаемым слоем.
10. **Архитектурные решения фиксируются ADR.** Значимые компромиссы нельзя хранить только в коде или переписке.
11. **Active work — это ресурсы и состояния, не иконки.** Activity Strip показывает текущую деятельность и доступные окна/views.
12. **Progressive disclosure применяется и к панели.** Карточки деградируют `full → compact → micro`, а не перегружают экран.

## Основные подсистемы

| Подсистема | Ответственность |
|---|---|
| `shell-ui` | Grid, Focus Mode, Activity Strip, Context Stack, search, widgets, keyboard navigation |
| `hwsd` | Главный Go daemon и application layer |
| `context` | Иерархия узлов, selection path, разрешение действий |
| `workspace` | Desired/observed state рабочего окружения |
| `surface` | ApplicationSurface aggregation, windows/views, provider capabilities |
| `panel` | Renderer-neutral Panel Model и безопасная DSL-конфигурация |
| `orchestrator` | Выполнение операций через Axiom |
| `integrations` | GNOME, systemd, processes, Git, SSH, network, portals |
| `providers` | Browser/IDE/terminal и другие rich-state adapters |
| `storage` | Настройки, индекс, durable state, history |
| `discovery` | Поиск проектов, приложений, репозиториев и системных возможностей |
| `policy` | Инварианты, permissions, safety boundaries |
| `telemetry` | Логи, диагностика, timings, health состояния |

## Документация

- [`MASTER_PLAN.md`](MASTER_PLAN.md) — единый living execution plan.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — целевая архитектура и границы компонентов.
- [`docs/AXIOM_INTEGRATION.md`](docs/AXIOM_INTEGRATION.md) — модель использования Axiom.
- [`docs/UX_HIERARCHY.md`](docs/UX_HIERARCHY.md) — иерархическая сетка, режимы и навигация.
- [`docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md`](docs/APPLICATION_SURFACES_AND_ACTIVITY_STRIP.md) — ApplicationSurface, окна/views, Activity Strip и Panel DSL.
- [`docs/INVARIANTS.md`](docs/INVARIANTS.md) — системные инварианты HWS.
- [`docs/adr/0006-application-surface-and-panel-dsl.md`](docs/adr/0006-application-surface-and-panel-dsl.md) — решение о resource-oriented панели и HCL DSL.
- [`docs/adr/`](docs/adr/) — архитектурные решения.

## Ближайший вертикальный срез

Первый демонстрируемый end-to-end сценарий:

```text
Super
  → DEV
  → Projects
  → HWS
  → Develop
```

После выбора `Develop` система должна:

1. сформировать desired workspace state;
2. проверить переход через Axiom;
3. открыть/найти нужные процессы;
4. применить раскладку;
5. зафиксировать observed state;
6. показать пользователю состояние и возможные ошибки;
7. позволить закрыть и восстановить контекст повторно;
8. построить window-only ApplicationSurface projection для запущенных приложений;
9. показать их в Activity Strip с корректной группировкой нескольких окон.

До появления этого вертикального среза расширение набора rich providers считается вторичным.

## Репозитории

- HWS: `https://github.com/Homiakus/HWS`
- Axiom: `https://github.com/Homiakus/axiom`

## Лицензия

Будет определена отдельным ADR до первого публичного релиза.