import Clutter from 'gi://Clutter';
import GLib from 'gi://GLib';
import GObject from 'gi://GObject';
import Shell from 'gi://Shell';
import St from 'gi://St';

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as PanelMenu from 'resource:///org/gnome/shell/ui/panelMenu.js';
import * as PopupMenu from 'resource:///org/gnome/shell/ui/popupMenu.js';

import {AppCapsule} from './appCapsule.js';
import {DaemonClient} from './daemonClient.js';

const MAX_INLINE_WINDOWS = 4;
const SHELL_SNAPSHOT_SCHEMA = 1;

function needsAttention(window) {
    try {
        return Boolean(window.demands_attention || window.urgent);
    } catch (_error) {
        return false;
    }
}

function attentionRank(value) {
    if (value === 'urgent') return 3;
    if (value === 'attention') return 2;
    return 1;
}

function connectOptional(target, signal, callback, bucket) {
    try {
        bucket.push([target, target.connect(signal, callback)]);
    } catch (_error) {
        // GNOME minor versions may not expose every optional signal.
    }
}

function windowID(window) {
    return `window:${window.get_stable_sequence()}`;
}

function workspaceID(window) {
    try {
        const index = window.get_workspace?.()?.index?.();
        return Number.isInteger(index) && index >= 0 ? `workspace:${index}` : '';
    } catch (_error) {
        return '';
    }
}

function monitorRef(window) {
    try {
        const index = window.get_monitor?.();
        return Number.isInteger(index) && index >= 0 ? `monitor:${index}` : '';
    } catch (_error) {
        return '';
    }
}

function optionalIdentity(window, method) {
    try {
        const value = window?.[method]?.();
        return typeof value === 'string' ? value.trim() : '';
    } catch (_error) {
        return '';
    }
}

function identityHints(app, windows) {
    const values = new Set();
    const appID = app.get_id?.();
    if (appID)
        values.add(appID);
    for (const window of windows) {
        for (const method of ['get_sandboxed_app_id', 'get_gtk_application_id', 'get_wm_class', 'get_wm_class_instance']) {
            const value = optionalIdentity(window, method);
            if (value)
                values.add(value);
        }
    }
    return [...values];
}

const ActivityStripIndicator = GObject.registerClass(
class ActivityStripIndicator extends PanelMenu.Button {
    _init() {
        super._init(0.0, 'HWS Activity Strip', false);
        this.add_style_class_name('hws-activity-strip');
        this._box = new St.BoxLayout({style_class: 'hws-activity-strip-box'});
        this.add_child(this._box);
        this._appSystem = Shell.AppSystem.get_default();
        this._cards = new Map();
        this._signals = [];
        this._refreshSource = 0;
        this._heartbeatSource = 0;
        this._daemonCards = [];
        this._panelRender = null;
        this._shellRevision = 0;
        this._lastShellSemantic = '';
        this._forcePublish = false;

        connectOptional(global.display, 'notify::focus-window', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-created', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-demands-attention', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-marked-urgent', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-entered-monitor', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-left-monitor', () => this.queueRefresh(), this._signals);
        connectOptional(global.workspace_manager, 'active-workspace-changed', () => this.queueRefresh(), this._signals);
        connectOptional(this._appSystem, 'app-state-changed', () => this.queueRefresh(), this._signals);

        this._daemon = new DaemonClient({
            onCardsChanged: (cards, render) => {
                this._daemonCards = cards;
                this._panelRender = render;
                this.queueRefresh();
            },
            onAvailabilityChanged: available => {
                this.toggle_style_class_name('daemon-unavailable', !available);
                if (!available)
                    this._lastShellSemantic = '';
                this.queueRefresh();
            },
        });

        this._heartbeatSource = GLib.timeout_add_seconds(GLib.PRIORITY_DEFAULT, 2, () => {
            this.queueRefresh(true);
            return GLib.SOURCE_CONTINUE;
        });
        this.queueRefresh();
    }

    queueRefresh(forcePublish = false) {
        this._forcePublish = this._forcePublish || forcePublish;
        if (this._refreshSource)
            return;
        this._refreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 40, () => {
            this._refreshSource = 0;
            const force = this._forcePublish;
            this._forcePublish = false;
            this._refresh(force);
            return GLib.SOURCE_REMOVE;
        });
    }

    _nativeCards() {
        const focused = global.display.focus_window;
        const cards = [];
        for (const app of this._appSystem.get_running()) {
            const windows = app.get_windows().filter(window => !window.is_skip_taskbar?.());
            if (windows.length === 0)
                continue;
            const active = windows.find(window => window === focused) || windows[0];
            const attention = windows.some(needsAttention) ? 'attention' : 'normal';
            cards.push({
                id: app.get_id() || windowID(active),
                name: app.get_name?.() || 'Application',
                title: active.get_title?.() || app.get_name?.() || 'Application',
                subtitle: app.get_busy?.() ? 'Busy' : '',
                busy: Boolean(app.get_busy?.()),
                focused: windows.some(window => window === focused),
                attention,
                windowCount: windows.length,
                surfaceCount: 0,
                windows,
                app,
                identityHints: identityHints(app, windows),
                segments: windows.slice(0, MAX_INLINE_WINDOWS).map(window => ({
                    id: windowID(window),
                    kind: 'window',
                    title: window.get_title?.() || '',
                    active: window === focused,
                    dirty: false,
                    attention: needsAttention(window) ? 'attention' : 'normal',
                    nativeWindow: true,
                })),
                overflowCount: Math.max(0, windows.length - MAX_INLINE_WINDOWS),
            });
        }
        return cards;
    }

    _shellPayload(nativeCards) {
        return {
            schema: SHELL_SNAPSHOT_SCHEMA,
            apps: nativeCards.map(card => ({
                appId: card.id,
                name: card.name,
                desktopAppId: card.id,
                identityHints: card.identityHints || [],
                busy: Boolean(card.busy),
                windows: card.windows.map(window => ({
                    id: windowID(window),
                    title: window.get_title?.() || '',
                    workspaceId: workspaceID(window),
                    monitorRef: monitorRef(window),
                    focused: window === global.display.focus_window,
                    minimized: Boolean(window.minimized),
                    mru: Number(window.get_user_time?.() || 0),
                    attention: needsAttention(window) ? 'attention' : 'normal',
                })),
            })),
        };
    }

    _publishNative(nativeCards, force = false) {
        if (!this._daemon.available)
            return;
        const base = this._shellPayload(nativeCards);
        const semantic = JSON.stringify(base);
        const changed = semantic !== this._lastShellSemantic;
        if (!changed && !force)
            return;
        if (changed || this._shellRevision === 0)
            this._shellRevision++;
        this._lastShellSemantic = semantic;
        this._daemon.submitShellSnapshot({
            ...base,
            revision: this._shellRevision,
            capturedAt: new Date().toISOString(),
        }, () => this._daemon.queueRefresh(), () => {
            if (this._lastShellSemantic === semantic)
                this._lastShellSemantic = '';
        });
    }

    _enrichCards(daemonCards, nativeCards) {
        const nativeByID = new Map(nativeCards.map(card => [card.id, card]));
        const nativeByWindow = new Map();
        for (const card of nativeCards) {
            for (const window of card.windows || [])
                nativeByWindow.set(windowID(window), card);
        }
        const out = [];
        const seenNative = new Set();

        for (const daemonCard of daemonCards || []) {
            let native = nativeByID.get(daemonCard.id);
            if (!native) {
                const nativeWindowSegment = (daemonCard.segments || []).find(segment => segment.kind === 'window');
                native = nativeByWindow.get(nativeWindowSegment?.id);
            }
            const segments = (daemonCard.segments || []).map(segment => ({
                ...segment,
                nativeWindow: segment.kind === 'window',
            }));
            out.push({
                ...daemonCard,
                windows: native?.windows || [],
                app: native?.app || null,
                providerOnly: !native,
                segments,
            });
            if (native)
                seenNative.add(native.id);
        }

        // During daemon startup or a missed D-Bus push, never make an otherwise
        // healthy GNOME application disappear from the panel.
        for (const native of nativeCards) {
            if (!seenNative.has(native.id))
                out.push(native);
        }
        return out;
    }

    _refresh(forcePublish = false) {
        const nativeCards = this._nativeCards();
        this._publishNative(nativeCards, forcePublish);

        const cards = this._daemon.available
            ? this._enrichCards(this._daemonCards, nativeCards)
            : nativeCards;
        cards.sort((a, b) =>
            Number(b.focused) - Number(a.focused) ||
            attentionRank(b.attention) - attentionRank(a.attention) ||
            (a.name || a.id).localeCompare(b.name || b.id));
        this._applyRenderConfig();
        this._render(cards);
    }

    _applyRenderConfig() {
        const gap = Number(this._panelRender?.gap);
        if (Number.isFinite(gap) && gap >= 0)
            this._box.set_style(`spacing: ${Math.min(64, gap)}px;`);
        else
            this._box.set_style('');
    }

    _render(cards) {
        const next = new Set(cards.map(card => card.id));
        for (const [id, capsule] of this._cards) {
            if (!next.has(id)) {
                capsule.destroy();
                this._cards.delete(id);
            }
        }
        for (const card of cards) {
            // Icons are compositor-owned actors; create them only at render time
            // so daemon snapshots never contain GObjects.
            card.iconActor = card.app?.create_icon_texture?.(16) || null;
            let capsule = this._cards.get(card.id);
            if (!capsule) {
                capsule = new AppCapsule(card, {
                    activate: current => this._activate(current),
                    cycle: (current, direction, state) => this._cycle(current, direction, state),
                    activateSegment: (current, segment) => this._activateSegment(current, segment),
                    newWindow: current => current.app?.can_open_new_window?.() && current.app.open_new_window(-1),
                    openSwitcher: current => this._openSwitcher(current),
                });
                this._cards.set(card.id, capsule);
                this._box.add_child(capsule);
            } else {
                capsule.update(card);
            }
        }
        this.visible = cards.length > 0;
    }

    _activate(card) {
        if (card.windows?.length) {
            if (card.focused && card.windows.length > 1) {
                this._cycle(card, Clutter.ScrollDirection.DOWN, 0);
                return;
            }
            card.app.activate_window(card.windows[0], global.get_current_time());
            return;
        }
        const target = (card.segments || []).find(segment => segment.active) || card.segments?.[0];
        if (target && !target.nativeWindow)
            this._daemon.activateView(card.id, target.id);
    }

    _activateSegment(card, segment) {
        if (segment.nativeWindow) {
            const window = (card.windows || []).find(candidate => windowID(candidate) === segment.id);
            if (window)
                card.app?.activate_window(window, global.get_current_time());
            return;
        }
        this._daemon.activateView(card.id, segment.id);
    }

    _cycle(card, direction, state) {
        const shift = Boolean(state & Clutter.ModifierType.SHIFT_MASK);
        if (shift && (card.segments || []).some(segment => !segment.nativeWindow)) {
            const segments = card.segments.filter(segment => !segment.nativeWindow);
            if (segments.length === 0)
                return;
            let index = Math.max(0, segments.findIndex(segment => segment.active));
            const backwards = direction === Clutter.ScrollDirection.UP || direction === Clutter.ScrollDirection.LEFT;
            index = (index + (backwards ? -1 : 1) + segments.length) % segments.length;
            this._daemon.activateView(card.id, segments[index].id);
            return;
        }

        const windows = card.windows || [];
        if (windows.length < 1)
            return;
        const focused = global.display.focus_window;
        let index = Math.max(0, windows.indexOf(focused));
        const backwards = direction === Clutter.ScrollDirection.UP || direction === Clutter.ScrollDirection.LEFT;
        index = (index + (backwards ? -1 : 1) + windows.length) % windows.length;
        card.app.activate_window(windows[index], global.get_current_time());
    }

    _openSwitcher(card) {
        if (card.surfaceCount > 0 && this._daemon.available) {
            this._daemon.getApplicationSurface(card.id, surface => {
                if (surface)
                    this._renderSurfaceMenu(card, surface);
                else
                    this._renderWindowMenu(card);
            });
            return;
        }
        this._renderWindowMenu(card);
    }

    _renderSurfaceMenu(card, surface) {
        this.menu.removeAll();
        for (const window of surface.windows || []) {
            const views = window.views || [];
            if (views.length === 0)
                continue;
            if ((surface.windows || []).length > 1) {
                const heading = new PopupMenu.PopupMenuItem(window.title || 'Window', {reactive: false});
                heading.add_style_class_name('hws-menu-heading');
                this.menu.addMenuItem(heading);
            }
            for (const view of views) {
                const marks = `${view.active ? '● ' : ''}${view.dirty ? '• ' : ''}`;
                const item = new PopupMenu.PopupMenuItem(`${marks}${view.title || view.id}`);
                item.connect('activate', () => this._daemon.activateView(card.id, view.id));
                this.menu.addMenuItem(item);
            }
        }
        this._appendWindowItems(card);
        this.menu.open();
    }

    _renderWindowMenu(card) {
        this.menu.removeAll();
        this._appendWindowItems(card, false);
        this.menu.open();
    }

    _appendWindowItems(card, withSeparator = true) {
        const windows = card.windows || [];
        if (withSeparator && windows.length > 0)
            this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());
        for (const window of windows) {
            const item = new PopupMenu.PopupMenuItem(window.get_title?.() || card.name);
            item.connect('activate', () => card.app.activate_window(window, global.get_current_time()));
            this.menu.addMenuItem(item);
        }
        if (card.app?.can_open_new_window?.()) {
            if (windows.length > 0)
                this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());
            const item = new PopupMenu.PopupMenuItem('New window');
            item.connect('activate', () => card.app.open_new_window(-1));
            this.menu.addMenuItem(item);
        }
    }

    destroy() {
        if (this._refreshSource) {
            GLib.source_remove(this._refreshSource);
            this._refreshSource = 0;
        }
        if (this._heartbeatSource) {
            GLib.source_remove(this._heartbeatSource);
            this._heartbeatSource = 0;
        }
        this._daemon?.destroy();
        this._daemon = null;
        for (const [object, id] of this._signals) {
            try {
                object.disconnect(id);
            } catch (_error) {
                // Object may have disappeared during session teardown.
            }
        }
        this._signals = [];
        this._cards.clear();
        super.destroy();
    }
});

export default class HWSExtension extends Extension {
    enable() {
        this._indicator = new ActivityStripIndicator();
        Main.panel.addToStatusArea('hws-activity-strip', this._indicator, 0, 'center');
    }

    disable() {
        this._indicator?.destroy();
        this._indicator = null;
    }
}
