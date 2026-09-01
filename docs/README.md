# HWS Documentation

Навигация по архитектурной документации HWS.

## Core

- [`../README.md`](../README.md) — краткая концепция и границы проекта.
- [`../MASTER_PLAN.md`](../MASTER_PLAN.md) — единый living execution plan.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — архитектура процессов, слоёв, domain model, desired/observed state и integration boundaries.
- [`AXIOM_INTEGRATION.md`](AXIOM_INTEGRATION.md) — правила использования `Homiakus/axiom`, lifecycle, activities, retry/idempotency и recovery.
- [`UX_HIERARCHY.md`](UX_HIERARCHY.md) — динамическая иерархия, Home/Focus modes, поиск и UX-инварианты.
- [`INVARIANTS.md`](INVARIANTS.md) — правила, которые код HWS не имеет права нарушать без отдельного ADR.

## Architecture Decision Records

- [`adr/0001-shell-overlay-first.md`](adr/0001-shell-overlay-first.md) — сначала overlay/integration layer, не собственный compositor.
- [`adr/0002-axiom-orchestration-boundary.md`](adr/0002-axiom-orchestration-boundary.md) — Axiom для durable orchestration, не для эфемерного UI.
- [`adr/0003-desired-observed-reconciliation.md`](adr/0003-desired-observed-reconciliation.md) — desired/observed state и reconciliation.
- [`adr/TEMPLATE.md`](adr/TEMPLATE.md) — шаблон новых ADR.

## Документы, которые должны появиться следующими

1. `DOMAIN_MODEL.md` — точные типы Node/Workspace/Resource/Action и identity semantics.
2. `IPC_CONTRACT.md` — versioned локальный API UI ↔ `hwsd`.
3. `WORKSPACE_LIFECYCLE.md` — формальная Axiom state/event/activity модель.
4. `SECURITY_MODEL.md` — threat model и privilege boundaries.
5. `TEST_STRATEGY.md` — unit/property/fault/restart/E2E/mutation testing.
6. `GNOME_ADAPTER.md` — capability matrix и границы desktop integration.
7. `STORAGE.md` — durable state, snapshots, migrations, corruption/recovery.
8. `OBSERVABILITY.md` — correlation IDs, logs, traces, diagnostics bundle.
9. `PERFORMANCE.md` — budgets, benchmarks и profiling protocol.
10. `PACKAGING.md` — install/update/disable/uninstall/fallback.

## Правило качества документации

Документ не должен выдавать планируемую возможность за реализованную. Для будущих возможностей используются формулировки `target`, `planned`, `proposed` или явный статус документа.

Архитектурное изменение, нарушающее [`INVARIANTS.md`](INVARIANTS.md), требует ADR до или одновременно с изменением кода.