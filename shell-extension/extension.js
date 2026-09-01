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

function strongestAttention(a, b) {
    return attentionRank(a) >= attentionRank(b) ? a : b;
}

function connectOptional(target, signal, callback, bucket) {
    try {
        bucket.push([target, target.connect(signal, callback)]);
    } catch (_error) {
        // GNOME minor versions may not expose every optional signal.
    }
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
        this._richCards = [];

        connectOptional(global.display, 'notify::focus-window', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-created', () => this.queueRefresh(), this._signals);
        connectOptional(global.display, 'window-demands-attention', () => this.queueRefresh(), this._signals);
        connectOptional(global.workspace_manager, 'active-workspace-changed', () => this.queueRefresh(), this._signals);
        connectOptional(this._appSystem, 'app-state-changed', () => this.queueRefresh(), this._signals);

        this._daemon = new DaemonClient({
            onCardsChanged: cards => {
                this._richCards = cards;
                this.queueRefresh();
            },
            onAvailabilityChanged: available => {
                this.toggle_style_class_name('daemon-unavailable', !available);
                this.queueRefresh();
            },
        });
        this.queueRefresh();
    }

    queueRefresh() {
        if (this._refreshSource)
            return;
        this._refreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 40, () => {
            this._refreshSource = 0;
            this._refresh();
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
                id: app.get_id() || `window:${active.get_stable_sequence()}`,
                name: app.get_name?.() || 'Application',
                title: active.get_title?.() || app.get_name?.() || 'Application',
                subtitle: app.get_busy?.() ? 'Busy' : '',
                focused: windows.some(window => window === focused),
                attention,
                windowCount: windows.length,
                surfaceCount: 0,
                windows,
                app,
                iconActor: app.create_icon_texture?.(16) || null,
                segments: windows.slice(0, MAX_INLINE_WINDOWS).map(window => ({
                    id: `window:${window.get_stable_sequence()}`,
                    title: window.get_title?.() || '',
                    active: window === active,
                    dirty: false,
                    attention: needsAttention(window) ? 'attention' : 'normal',
                    nativeWindow: true,
                })),
                overflowCount: Math.max(0, windows.length - MAX_INLINE_WINDOWS),
            });
        }
        return cards;
    }

    _mergeCards(nativeCards, richCards) {
        const byID = new Map(nativeCards.map(card => [card.id, card]));
        for (const rich of richCards || []) {
            const native = byID.get(rich.id);
            if (!native) {
                byID.set(rich.id, {
                    ...rich,
                    windows: [],
                    app: null,
                    providerOnly: true,
                    focused: Boolean(rich.focused),
                    iconActor: null,
                });
                continue;
            }
            native.title = rich.title || native.title;
            native.subtitle = rich.subtitle || native.subtitle;
            native.progress = typeof rich.progress === 'number' ? rich.progress : native.progress;
            native.surfaceCount = rich.surfaceCount || native.surfaceCount;
            if (Array.isArray(rich.segments) && rich.segments.length > 0) {
                native.segments = rich.segments;
                native.overflowCount = rich.overflowCount || 0;
            }
            native.attention = strongestAttention(native.attention, rich.attention || 'normal');
            native.rich = true;
        }
        return [...byID.values()];
    }

    _refresh() {
        const cards = this._mergeCards(this._nativeCards(), this._richCards);
        cards.sort((a, b) =>
            Number(b.focused) - Number(a.focused) ||
            attentionRank(b.attention) - attentionRank(a.attention) ||
            (a.name || a.id).localeCompare(b.name || b.id));
        this._render(cards);
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
        if (target)
            this._daemon.activateView(card.id, target.id);
    }

    _activateSegment(card, segment) {
        if (segment.nativeWindow) {
            const window = (card.windows || []).find(candidate => `window:${candidate.get_stable_sequence()}` === segment.id);
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
