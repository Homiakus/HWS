# Application Surfaces и Activity Strip

## 1. Назначение

HWS не должен воспроизводить классическую taskbar-модель `иконка приложения + индикатор запуска`.

Основная единица представления активной работы — **Application Surface**: нормализованная проекция приложения, его окон, доступных внутренних представлений (tabs/documents/sessions) и значимых состояний.

Activity Strip — адаптивная панель HWS, которая показывает не только приложения, но и другие живые ресурсы рабочего контекста: терминальные/SSH-сессии, builds, AI agents, downloads, сервисы и операции workspace.

Главный принцип:

> Показывать пользователю «что сейчас происходит и где находится работа», а не просто «какой executable запущен».

## 2. Domain model

Целевая модель:

```text
ApplicationSurface
├── identity
├── application
├── lifecycle
├── attention
├── activity
├── resource_state
├── media_state
├── context
├── windows[]
│   ├── identity
│   ├── title
│   ├── workspace
│   ├── monitor
│   ├── focused
│   ├── preview_capability
│   └── views[]
│       ├── identity
│       ├── title
│       ├── active
│       ├── dirty
│       ├── progress
│       ├── resource_ref
│       └── provider_metadata
└── capabilities
```

`views[]` — обобщение browser tabs, IDE editor tabs, terminal sessions, document tabs и других внутренних представлений.

### 2.1 Identity

Запрещено считать PID, title окна или позицию вкладки durable identity.

Идентификаторы имеют scope:

- application identity — стабильная там, где её предоставляет desktop/application model;
- window identity — session-scoped;
- view/tab identity — provider-scoped и обычно session-scoped;
- resource/document identity может быть durable только если provider предоставляет устойчивый resource reference (например URI/path/project ID), а не только UI handle.

Минимальный ключ внутреннего view:

```text
provider_id + application_instance + window_session_id + view_session_id
```

## 3. Состояние приложения — несколько независимых осей

Нельзя сводить состояние к `running/green`.

### Lifecycle

```text
stopped
starting
running
suspended
stopping
crashed
unknown
```

### Attention

```text
normal
attention
urgent
```

### Activity

```text
idle
working
progress
waiting
blocked
```

### Resource state

```text
clean
dirty
syncing
conflict
error
unknown
```

### Media/privacy state

```text
audio_playing
microphone_active
camera_active
screen_sharing
```

UI выбирает только наиболее важные сигналы согласно приоритету, а не отображает все одновременно.

## 4. Источники наблюдения

Application Surface строится из нескольких независимых providers.

```text
GNOME Shell window observer
          │
          ├───────────────┐
          ▼               ▼
 Application/window    Application-specific providers
 observation               │
                           ├── browser extension/native messaging
                           ├── IDE plugin/API
                           ├── terminal/session adapter
                           ├── media/download/build providers
                           └── AT-SPI semantic fallback
                                  │
                                  ▼
                         Surface Aggregator
                                  │
                                  ▼
                         ApplicationSurface
```

### 4.1 GNOME Shell provider

Отвечает за:

- Shell application association;
- список окон;
- focus;
- workspace/monitor;
- transient/window-role сведения, когда доступны;
- возможность preview;
- события создания/закрытия/focus.

GNOME adapter не должен угадывать browser/IDE tabs по заголовкам окон.

### 4.2 Browser provider

Для Chromium/Chrome/Firefox предпочтителен отдельный HWS extension + Native Messaging/local IPC bridge.

Provider может сообщать:

- windows;
- tabs;
- active tab;
- title;
- URL/domain по разрешённой privacy policy;
- favicon reference;
- pinned/audible/muted state;
- download/progress при доступности;
- focus/activate/close capability.

URL и title могут быть чувствительными данными. По умолчанию они не должны попадать в telemetry/history без явной redaction policy.

### 4.3 IDE provider

Предпочтительный контракт:

- project/workspace;
- active document;
- open documents;
- dirty documents;
- git branch/status summary;
- running task/test/build;
- integrated terminal/session summary;
- activate view/document.

Сначала достаточно одного реального provider-а для целевого IDE, затем общего capability contract.

### 4.4 Terminal provider

Терминал может предоставлять более полезную семантику, чем одно окно:

- tabs/panes/sessions;
- cwd;
- remote host;
- foreground command class;
- long-running job status;
- attention bell.

Нельзя автоматически читать/логировать содержимое терминала как часть обычного status provider.

### 4.5 AT-SPI fallback

AT-SPI используется только как semantic fallback, если приложение корректно публикует доступные роли/иерархию.

Приоритет providers:

```text
native/application API
        ↓
HWS plugin/extension
        ↓
AT-SPI semantic adapter
        ↓
window metadata heuristic
```

AT-SPI discovery не является доказательством стабильной tab identity и не должен использоваться как durable source.

## 5. Capability model

Каждый Surface provider сообщает capabilities, например:

```text
surface.observe
window.observe
window.preview
window.activate
view.observe
view.activate
view.close
view.reorder
view.progress
view.dirty
media.observe
```

UI никогда не показывает действие, которое provider не объявил доступным.

Capability может измениться во время сессии; UI обязан обновить affordance без перезапуска.

## 6. Activity Strip

Одна карточка по умолчанию соответствует **одному ApplicationSurface**, а не одному окну.

Пример:

```text
┌──────────────────────────┐
│ Zed                   ●2 │
│ HWS · panel.go            │
│ tests running 42%         │
└──────────────────────────┘

┌──────────────────────────┐
│ Firefox              12 ▾│
│ GitHub / HWS              │
│ [HWS] [Docs] [CI] +9      │
└──────────────────────────┘
```

### 6.1 Несколько окон

Модель:

```text
Application Card
      ↓
Window stack / preview
      ↓
Views/tabs текущего окна
```

В свернутом виде показывается количество окон. При раскрытии — MRU-упорядоченные окна с preview, если capability доступна.

Клик по карточке при одном окне активирует его. При нескольких окнах политика должна быть настраиваемой: activate-MRU либо раскрыть switcher.

### 6.2 Несколько вкладок/views

Нельзя пытаться показывать все вкладки на панели.

По умолчанию:

```text
active + recent + important + overflow counter
```

Пример:

```text
[HWS] [GNOME] [Axiom] +20
```

Раскрытый navigator:

```text
Application
├── Window A
│   ├── active view
│   ├── recent view
│   └── ...
└── Window B
    └── ...
```

Сортировка по умолчанию — MRU внутри окна, с возможностью provider-defined importance (pinned, dirty, attention, active media).

## 7. Semantic zoom

Карточка имеет минимум три presentation levels:

### Full

```text
application + context + status + progress + windows/views
```

### Compact

```text
application + active context/view + strongest status
```

### Micro

```text
icon/short-name + count + status glyph
```

Переключение уровня определяется layout constraints, а не ручным responsive CSS для каждого приложения.

## 8. Layout constraints

Каждый widget/card объявляет:

```text
min_width
ideal_width
max_width
priority
expand_weight
collapse rules
```

Панель решает недостаток места в порядке:

1. уменьшить flexible space;
2. перевести низкоприоритетные cards `full → compact`;
3. `compact → micro`;
4. перенести overflow в раскрываемый список;
5. никогда не скрывать urgent state без видимого aggregate indicator.

## 9. Panel DSL

Пользовательский DSL должен быть декларативным, валидируемым и независимым от конкретного renderer-а.

Выбранное направление — **HCL-based syntax** с собственной ограниченной schema HWS.

Пайплайн:

```text
*.hws.hcl
   ↓
HCL parser
   ↓
Panel AST
   ↓
Schema validation
   ↓
Normalized Panel Model
   ↓
Shell renderer
```

DSL никогда не создаёт GNOME objects напрямую.

### 9.1 Пример

```hcl
panel "workspace" {
  edge   = "bottom"
  height = 64
  margin = 8

  apps {
    source   = "current-context"
    grouping = "application"

    card {
      min_width   = 108
      ideal_width = 180
      max_width   = 280
      density     = "adaptive"

      show = [
        "icon",
        "title",
        "context",
        "status",
        "progress",
        "window-count"
      ]

      windows {
        mode        = "stack"
        max_visible = 3
      }

      views {
        mode        = "recent"
        max_visible = 3
        overflow    = "counter"
      }
    }
  }

  spacer {}

  widget "network" {
    variant = "mini"
  }

  widget "audio" {
    variant = "mini"
  }

  widget "clock" {
    format = "HH:mm"
  }
}
```

### 9.2 Expressions

Если expressions добавляются, они должны иметь ограниченную capability-safe среду, например:

```text
surface.window_count
surface.view_count
surface.lifecycle
surface.attention
context.id
workspace.status
```

Запрещены:

- arbitrary shell execution;
- filesystem reads;
- network calls;
- reflection над внутренними Go/GNOME objects;
- непредсказуемые side effects.

### 9.3 Hot reload

Hot reload должен быть транзакционным:

```text
read → parse → validate → build normalized model
                     ├─ success → atomic UI swap
                     └─ failure → keep previous valid model + diagnostics
```

Ошибка DSL не должна уничтожать текущую рабочую панель.

## 10. Interaction defaults

Рекомендуемая семантика:

- primary click: activate MRU window либо раскрыть group согласно policy;
- secondary click: contextual actions;
- middle click: new window/instance только если capability поддерживается;
- wheel: MRU windows, если пользователь явно включил такую policy;
- Shift+wheel: MRU views/tabs при наличии provider capability;
- keyboard focus: полный аналог pointer navigation;
- long press/touch: contextual actions без hover dependency.

Interaction DSL не должен позволять arbitrary command strings. Действия с side effects представлены typed action IDs через `hwsd`.

## 11. Activity Strip как общий ресурсный слой

Панель не ограничивается desktop applications.

Целевые resource cards:

```text
application
terminal/ssh session
build/test task
AI agent
file transfer/download
service
network/VPN state
workspace operation
media session
```

Это позволяет HWS показывать:

```text
Zed          Firefox       dev-vps       Tests        Agent
editing      12 tabs       ssh active    42%          waiting
```

без превращения каждого ресурса в фиктивное приложение.

## 12. Privacy и security

Surface providers потенциально видят чувствительные UI-данные.

Инварианты:

1. Titles/URLs/document names не пишутся в durable telemetry без redaction policy.
2. Browser provider запрашивает минимально необходимые permissions.
3. Provider не получает права выполнять arbitrary OS commands.
4. Закрытие tab/window — typed capability action, а не shell command.
5. AT-SPI provider работает read-only до отдельного ADR на mutation.
6. Provider failure не должен ломать shell; соответствующая Surface деградирует.
7. Неизвестные provider fields не трактуются как trusted HTML/markup.

## 13. Performance budgets

Target до измеренного baseline:

- изменение focus/status → видимый card update p95 < 50 ms;
- shell main-loop work на одно coalesced surface update < 4 ms;
- bursts window/view events coalesce до bounded frame/update rate;
- provider reconnect не блокирует Shell;
- preview создаётся лениво, а не постоянно для всех окон;
- отсутствие tab provider не вызывает polling title/AT-SPI с высокой частотой;
- 50 surfaces / 500 views не должны приводить к линейному постоянному renderer cost на каждый frame.

## 14. Failure semantics

Различаются:

- application disappeared;
- window disappeared;
- provider disconnected;
- provider snapshot stale;
- provider returned partial data;
- action capability denied;
- view identity invalidated;
- preview unavailable;
- DSL parse/validation error;
- renderer error.

При provider failure окно/приложение остаётся доступным через базовый GNOME observation; просто исчезает richer tab/status projection.

## 15. Testing matrix

Обязательны:

- 1 app / 1 window / 1 view;
- 1 app / N windows;
- 1 app / N windows / N views;
- 20+ browser tabs;
- 100+ tabs stress;
- provider disconnect/reconnect;
- window closes во время navigator open;
- view closes/reorders во время activation;
- MRU determinism;
- no-tab-provider fallback;
- AT-SPI malformed/unstable hierarchy;
- dirty/urgent/progress priority collisions;
- panel 800/1366/1920/4K widths;
- fractional scale 125/150/200%;
- touch/no-hover path;
- keyboard-only path;
- hot reload invalid DSL retains previous layout;
- privacy/redaction tests.

## 16. Implementation sequence

### AS-0 — Domain contracts

- `ApplicationSurface`, `SurfaceWindow`, `SurfaceView`;
- state axes;
- provider capabilities;
- provider-scoped identity;
- normalized snapshot and diff.

### AS-1 — GNOME window projection

- Shell.WindowTracker/Meta.Window bridge;
- app grouping;
- window count/focus/MRU;
- preview capability;
- no tab assumptions.

### AS-2 — Activity Strip MVP

- full/compact/micro cards;
- adaptive constraints;
- one ApplicationSurface = one card;
- window group navigator;
- overflow.

### AS-3 — Panel DSL v1

- HCL parser;
- strict schema;
- normalized Panel Model;
- diagnostics;
- transactional hot reload;
- no arbitrary expressions/commands in v1.

### AS-4 — First rich view provider

- browser extension or target IDE provider;
- views/tabs;
- MRU;
- active/dirty/progress state;
- activate action.

### AS-5 — Provider ecosystem

- browser family;
- IDE providers;
- terminal/session provider;
- AT-SPI read-only fallback.

### AS-6 — General resource cards

- builds/tests;
- SSH;
- AI agents;
- transfers;
- system/network resources.

## 17. Definition of Done

Application Surface / Activity Strip capability считается зрелой, когда:

1. несколько окон одного приложения представлены одной карточкой без потери доступа к каждому окну;
2. минимум один реальный provider показывает внутренние views/tabs;
3. отсутствие provider-а корректно деградирует к window-only модели;
4. panel DSL проходит schema validation и безопасный hot reload;
5. full/compact/micro semantic zoom работает на нескольких разрешениях и scale factors;
6. никакой title/PID/tab-index не используется как durable identity;
7. provider data redaction проверена тестами;
8. Shell main loop не выполняет provider/network/filesystem heavy work;
9. UI action выполняется только при объявленной capability;
10. все side-effect actions проходят typed `hwsd` boundary, а не arbitrary commands.
