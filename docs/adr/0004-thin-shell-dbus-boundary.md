# ADR-0004 — Thin GNOME Shell extension + D-Bus daemon boundary

Status: Accepted  
Date: 2026-09-01

## Context

GNOME Shell extensions execute inside the `gnome-shell` process. Blocking I/O, signal leaks, actor lifecycle errors and uncaught failures can degrade or crash the entire desktop session. Official GNOME extension guidance recommends keeping the entry point small, offloading heavy work, and preferring D-Bus communication with external background processes.

HWS needs substantially more functionality than is safe to place in Shell:

- context/project indexing;
- persistence;
- Axiom orchestration;
- Git/files/network integration;
- recovery/history;
- search;
- diagnostics.

At the same time, several operations are correctly performed in Shell context because only there HWS has direct access to `Shell`, `Meta`, current user timestamp, `Shell.App`, window objects and GNOME compositor state.

## Decision

HWS uses two primary processes/layers:

```text
GNOME Shell extension
- UI
- input
- Meta/Shell observation
- compositor-local actions
- user-context application activation
- D-Bus client

        ↕ versioned D-Bus

hwsd (Go)
- domain model
- Axiom
- desired/observed reconciliation
- persistence
- search/providers
- heavy integrations
- policy/history/diagnostics
```

The shell extension MUST remain a thin adapter/presentation layer.

`hwsd` owns durable intent and decision semantics. The Shell extension may execute a compositor-local action requested by `hwsd`, but that action is treated as an external activity whose result is verified from observed state.

## Consequences

### Positive

- Shell crash surface is reduced.
- Go code remains testable without GNOME.
- Axiom and storage do not run in compositor process.
- Daemon can restart independently.
- GNOME-specific APIs stay isolated.
- App activation can still use proper Shell/Wayland context.

### Negative

- IPC protocol becomes critical infrastructure.
- Need reconnect/owner-change handling.
- Some operations become multi-step distributed state machines across two processes.
- Need explicit correlation IDs and revisions.

## Required safeguards

- versioned D-Bus protocol;
- bounded async calls;
- fresh handshake after bus owner change;
- shell UI stays usable/degraded if daemon disappears;
- no synchronous filesystem/network/database work in Shell;
- extension lifecycle stress tests;
- observed-state verification after Shell actions.

## Rejected alternatives

### Put all HWS logic in GJS extension

Rejected because it maximizes Shell stability risk and makes persistence/orchestration harder to test and maintain.

### Put all desktop actions in Go daemon

Rejected because generic background processes do not have the same Shell/Meta access or reliable user activation context under Wayland.

### Custom compositor immediately

Rejected by ADR-0001; cost and compatibility risk are much higher than necessary for MVP.

## Evidence

See [`../DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](../DEVELOPER_PRACTICES_AND_FAILURE_MODES.md).