# ADR-0001: Начинать HWS как overlay/integration layer, а не собственный compositor

- Статус: Accepted
- Дата: 2026-09-01

## Контекст

HWS требует глубокого управления пользовательским рабочим окружением: overview, иерархическая навигация, активация workspace и управление окнами. Один из вариантов — сразу создавать собственный compositor/window manager. Другой — строить HWS как слой поверх стандартной графической сессии Ubuntu и изолировать desktop-specific доступ через адаптер.

Собственный compositor резко увеличивает область ответственности проекта: ввод, окна, экраны, multi-monitor, accessibility, portals, screen sharing, драйверные особенности, lifecycle сессии и совместимость приложений.

Главная продуктовая гипотеза HWS — не новый compositor, а task-first hierarchical workspace UX.

## Решение

Первый production path HWS строится как overlay/integration layer над стандартной графической сессией Ubuntu.

Архитектура обязана скрывать desktop API за capability-driven adapter boundary.

HWS Core не зависит от конкретного compositor API.

## Следствия

Плюсы:

- быстрее проверяется продуктовая гипотеза;
- меньше системных рисков;
- сохраняется fallback к обычной среде;
- headless/domain части можно тестировать независимо;
- позднее можно заменить desktop adapter.

Минусы:

- часть window-management возможностей может быть ограничена API текущей среды;
- некоторые layouts/animations могут потребовать compromises;
- появится compatibility matrix.

## Guardrail

Переход к собственному compositor допускается только после отдельного ADR, содержащего:

1. подтверждённые невозможные/нестабильные ключевые сценарии текущего adapter;
2. прототип;
3. оценку maintenance cost;
4. compatibility plan;
5. migration path;
6. rollback/fallback plan.

## Не принято

Не принимается стратегия «сразу форкнуть desktop shell/compositor и затем строить продукт вокруг него», потому что она связывает проверку UX-гипотезы с самым дорогим системным слоем.