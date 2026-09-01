# UX: иерархическая рабочая оболочка HWS

## 1. Цель

HWS должен сокращать расстояние между намерением пользователя и готовым рабочим окружением.

Пользователь не должен помнить:

- где лежит проект;
- какие приложения нужно открыть;
- какие терминалы нужны;
- какую раскладку восстановить;
- к какому серверу подключиться;
- какие вспомогательные панели обычно используются.

Он выбирает контекст, а HWS восстанавливает его.

## 2. Основная навигационная модель

Иерархия имеет произвольную глубину:

```text
Area
└── Domain
    └── Project
        └── Task
            └── Action
```

Это не жёсткая схема типов. У одного раздела может быть 2 уровня, у другого — 6.

Пример:

```text
DEV
└── Projects
    └── HWS
        ├── Develop
        ├── Test
        └── Release
```

```text
CAD
└── Siemens NX
    └── NXGO
        └── Automation
            └── Run tests
```

## 3. Динамические уровни

Экран строится как последовательность рядов/слоёв.

```text
[ DEV ] [ CAD ] [ RESEARCH ] [ SYSTEM ]
   ▲

[ Projects ] [ Agents ] [ Servers ] [ Tools ]
      ▲

[ HWS ] [ Axiom ] [ NXGO ] [ CYTOLOGY-AI ]
   ▲

[ Develop ] [ Test ] [ Docs ] [ Release ]
```

При изменении выбора на уровне `N`:

1. выбор уровней `> N` инвалидируется;
2. запрашиваются/строятся новые children;
3. UI сохраняет верхнюю часть path;
4. нижние ряды заменяются;
5. фокус перемещается предсказуемо;
6. никакая системная операция не выполняется, пока пользователь не активировал конечный intent/action согласно типу узла.

## 4. Home Mode

`Super` открывает основной HWS Overview.

Home Mode показывает:

- hierarchy rows;
- текущий path;
- recent contexts;
- active/degraded workspaces;
- быстрый поиск;
- системные indicators, если они помогают действию.

Home Mode не должен превращаться в перегруженный dashboard.

Приоритет:

1. текущая иерархия;
2. следующий выбор;
3. активные проблемы;
4. вторичная телеметрия.

## 5. Focus Mode

После активации workspace основная сетка скрывается.

Focus Mode может показывать минимальную context bar:

```text
DEV › Projects › HWS › Develop
```

и небольшие состояния:

```text
Active · 3 resources · Git clean
```

`Super` возвращает Home Mode.

## 6. Context Stack

HWS ведёт историю навигации независимо от истории workspace lifecycle.

Пример:

```text
DEV/HWS/Develop
→ SYSTEM/Network
→ DEV/HWS/Test
```

Навигационные команды:

- Back;
- Forward;
- Parent;
- Root;
- Last active context.

Это UI/navigation history и не обязано быть Axiom execution history.

## 7. Keyboard-first interaction

Базовая цель — любой frequent path должен проходиться без мыши.

Возможные взаимодействия:

```text
Super                открыть HWS
Arrows / hjkl         перемещение
Enter                 выбрать/активировать
Esc                   уровень вверх/закрыть
Backspace             parent
Ctrl+K или Super+Space search
Alt+Left              context back
Alt+Right             context forward
```

Цифровые быстрые последовательности допускаются как optional mode:

```text
Super → 1 → 2 → 1 → 3
```

но UI обязан показывать hints и не требовать запоминания кодов.

## 8. Search

Поиск возвращает не только имя, но и полный path:

```text
HWS
DEV › Projects › HWS

Run tests
DEV › Projects › HWS › Test › Run
```

Типы результатов:

- contexts;
- projects;
- actions;
- applications;
- documents;
- remote targets;
- system settings;
- active workspace operations.

Search result обязан явно показывать, приведёт ли Enter к навигации или к side effect.

## 9. Типы узлов

Начальный набор:

### Category

Только группирует children.

### Project

Связан с проектным контекстом, репозиторием, каталогом или набором ресурсов.

### Workspace

Активирует полноценное рабочее окружение.

### Action

Выполняет явное действие.

### Widget

Показывает состояние и может иметь drill-down.

### Dynamic Query

Children вычисляются provider-ом, например recent projects или active servers.

Тип должен влиять на визуальную семантику и подтверждение действия.

## 10. Action safety

Пользователь должен отличать:

- навигацию;
- безопасную activation;
- потенциально destructive action.

Destructive/system-sensitive actions не должны выглядеть как обычный переход по меню.

Пример:

```text
Restart service
Disconnect VPN
Close all managed workspace processes
```

Для них UI отображает явный action affordance и, где нужно, подтверждение.

## 11. Progressive disclosure

Не показывать все возможности одновременно.

На верхнем уровне:

```text
DEV  CAD  RESEARCH  SYSTEM
```

а не сотни приложений и команд.

Детали появляются по мере углубления.

Это основная причина иерархии, а не декоративная сетка.

## 12. Размер и компоновка плиток

Плитка может быть:

- compact;
- standard;
- status-rich.

Но размер не должен разрушать навигационную геометрию.

Статусные элементы:

```text
HWS
main · clean
CI ✓
```

не должны перетягивать внимание с названия и следующего действия.

## 13. Dynamic status

Статусы обновляются независимо от path navigation.

Примеры:

- Git dirty;
- workspace degraded;
- VPN active;
- server unreachable;
- tests failed.

Изменение статуса не должно самовольно менять пользовательский selection path.

## 14. Self-organization / recommendations

HWS может вычислять:

- recent;
- frequent;
- suggested workspace;
- common app combinations.

Но автоматическая система не имеет права молча менять пользовательскую canonical hierarchy.

Рекомендации живут в отдельном слое и требуют явного принятия для постоянного изменения структуры.

## 15. Loading semantics

Навигация по уже загруженному дереву должна быть мгновенной.

Dynamic provider не должен блокировать весь экран.

Вместо:

```text
[spinner на весь UI]
```

используется локальное состояние ряда:

```text
Servers
├── local cached entries
└── Loading remote…
```

Старый валидный path остаётся видимым до разрешения нового.

## 16. Error semantics

Ошибка должна быть привязана к уровню/операции.

Плохо:

```text
Something went wrong
```

Хорошо:

```text
HWS › Develop
Workspace activated with limitations
- editor: ready
- terminal: ready
- layout: unavailable
Reason: window.move capability is not available
```

## 17. Multi-monitor

Первый принцип: один логический Home Mode, независимо от количества мониторов.

Возможные policies workspace layout:

- current display;
- primary display;
- named role (`main`, `reference`);
- relative multi-monitor layout.

Нельзя сохранять только абсолютные пиксельные координаты как единственную форму layout definition.

## 18. Touch

На touch interface:

- targets достаточно крупные;
- hover-only функции запрещены;
- длинное нажатие не является единственным способом критичного действия;
- rows могут прокручиваться независимо только если это не ломает понимание path;
- breadcrumb всегда доступен.

## 19. Accessibility

С самого начала нужны:

- semantic labels;
- predictable focus order;
- keyboard traversal;
- high contrast compatibility;
- reduced motion mode;
- отсутствие зависимости только от цвета;
- корректные screen reader names для динамических rows.

## 20. Неподвижные UX-инварианты

1. Изменение выбора верхнего уровня всегда объяснимо меняет нижние уровни.
2. Навигация сама по себе не выполняет скрытый destructive side effect.
3. Пользователь всегда видит текущий path.
4. Активный workspace отличим от просто выбранного узла.
5. Degraded/Failed не маскируются как нормальный Active.
6. Система не меняет canonical hierarchy без явного согласия.
7. Возврат к базовой Ubuntu должен оставаться возможным даже при неисправности HWS.