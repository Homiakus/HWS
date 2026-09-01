# ADR-0005 — Ubuntu 26.04 / GNOME 50 / Wayland is the first-class desktop target

Status: Accepted  
Date: 2026-09-01

## Context

Ubuntu 26.04 GNOME desktop runs on Wayland; GNOME Shell no longer provides an X.org session. X11 applications continue through XWayland.

Window-management extensions historically accumulate complexity by trying to support many GNOME major versions, X11 and Wayland simultaneously. Real projects such as Tiling Shell and PaperWM show recurring compatibility regressions around GNOME major upgrades, scaling, input/focus and compositor API changes.

HWS needs a narrow, testable first production target.

## Decision

The first-class target for HWS MVP and M2/M3 desktop integration is:

```text
Ubuntu 26.04 LTS
GNOME Shell 50
Mutter 50
Wayland session
```

HWS does not use X11-only tooling or EWMH assumptions as architectural dependencies.

XWayland is treated as a client compatibility case inside the GNOME adapter, not as a separate HWS compositor backend.

## Consequences

### Positive

- smaller compatibility matrix;
- aligns with actual Ubuntu 26.04 session architecture;
- avoids dual X11/Wayland implementations;
- makes Wayland focus/activation semantics explicit from day one;
- allows rigorous GNOME 50 nested testing.

### Negative

- HWS MVP is not a generic Linux desktop shell;
- other GNOME major versions require explicit adapter qualification;
- non-GNOME desktops are future integrations, not implicit compatibility.

## Rules

- no `wmctrl`/`xdotool` dependency in core GNOME path;
- no persistent X11 window ID assumptions;
- app/window matching uses Shell/Meta application semantics;
- app launch/activation respects XDG activation/user timestamp semantics;
- geometry is logical/normalized, not hand-scaled framebuffer coordinates;
- support for a new GNOME major is declared only after its adapter test matrix passes.

## Future versions

Development CI MAY test upcoming GNOME versions to discover breakage early. Metadata/product claims MUST NOT list a version as supported before qualification.

## Evidence

See [`../DEVELOPER_PRACTICES_AND_FAILURE_MODES.md`](../DEVELOPER_PRACTICES_AND_FAILURE_MODES.md) and [`../GNOME_ADAPTER.md`](../GNOME_ADAPTER.md).