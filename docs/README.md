# HWS Documentation

Навигация по архитектурной документации HWS.

## Core

- [`../README.md`](../README.md) — краткая концепция и границы проекта.
- [`../MASTER_PLAN.md`](../MASTER_PLAN.md) — единый living execution plan.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — архитектура процессов, слоёв, domain model, desired/observed state и integration boundaries.
- [`DOMAIN_MODEL.md`](DOMAIN_MODEL.md) — типы Node/Workspace/Resource/Action, identity, desired/observed и reconcile values.
- [`WORKSPACE_LIFECYCLE.md`](WORKSPACE_LIFECYCLE.md) — формальная lifecycle-модель первой реализации на Axiom.
- [`AXIOM_INTEGRATION.md`](AXIOM_INTEGRATION.md) — правила использования `Homiakus/axiom`, activities, retry/idempotency, durable storage и recovery.
- [`UX_HIERARCHY.md`](UX_HIERARCHY.md) — динамическая иерархия, Home/Focus modes, поиск и UX-инварианты.
- [`GNOME_ADAPTER.md`](GNOME_ADAPTER.md) — контракт GNOME 50 adapter: capabilities, window identity, launch/activation, focus, geometry, topology и D-Bus reconnect.
- [`DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](DEVELOPER_PRACTICES_AND_FAILURE_MODES.md) — исследование официальных практик GNOME и реальных проблем PaperWM/Tiling Shell/Ubuntu с архитектурными выводами для HWS.
- [`INVARIANTS.md`](INVARIANTS.md) — правила, которые код HWS не имеет права нарушать без отдельного ADR.
- [`TEST_STRATEGY.md`](TEST_STRATEGY.md) — unit/property/fuzz/mutation/Axiom-contract/fault/restart/E2E testing.
- [`SECURITY_MODEL.md`](SECURITY_MODEL.md) — threat model, privilege boundaries, typed command execution и ownership.

## Architecture Decision Records

- [`adr/0001-shell-overlay-first.md`](adr/0001-shell-overlay-first.md) — сначала overlay/integration layer, не собственный compositor.
- [`adr/0002-axiom-orchestration-boundary.md`](adr/0002-axiom-orchestration-boundary.md) — Axiom для durable orchestration, не для эфемерного UI.
- [`adr/0003-desired-observed-reconciliation.md`](adr/0003-desired-observed-reconciliation.md) — desired/observed state и reconciliation.
- [`adr/0004-thin-shell-dbus-boundary.md`](adr/0004-thin-shell-dbus-boundary.md) — минимальный Shell process, тяжёлая логика в `hwsd`, связь через versioned D-Bus.
- [`adr/0005-ubuntu-2604-gnome50-wayland-target.md`](adr/0005-ubuntu-2604-gnome50-wayland-target.md) — Ubuntu 26.04 / GNOME 50 / Wayland как первый production target.
- [`adr/TEMPLATE.md`](adr/TEMPLATE.md) — шаблон новых ADR.

## Документы следующей очереди

1. `IPC_CONTRACT.md` — versioned локальный API UI ↔ `hwsd`, handshake/revisions/owner changes.
2. `STORAGE.md` — durable state, snapshots, migrations, corruption/recovery.
3. `OBSERVABILITY.md` — correlation IDs, logs, traces, diagnostics bundle.
4. `PERFORMANCE.md` — budgets, benchmarks и profiling protocol, включая Shell main-loop budget.
5. `PACKAGING.md` — install/update/disable/uninstall/fallback/safe mode.
6. `CONFIG_SCHEMA.md` — пользовательская иерархия и workspace definitions.
7. `PROVIDER_MODEL.md` — dynamic providers, permissions и lifecycle.
8. `COMPATIBILITY_MATRIX.md` — Ubuntu defaults, GNOME major, scaling, multi-monitor и third-party extension conflicts.

## Правило качества документации

Документ не должен выдавать планируемую возможность за реализованную. Для будущих возможностей используются формулировки `target`, `planned`, `proposed` или явный статус документа.

Архитектурное изменение, нарушающее [`INVARIANTS.md`](INVARIANTS.md), требует ADR до или одновременно с изменением кода.