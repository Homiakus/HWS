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

const MAX_INLINE_WINDOWS = 4;

function needsAttention(window) {
    try {
        return Boolean(window.demands_attention || window.urgent);
    } catch (_error) {
        return false;
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

        this._signals.push([global.display, global.display.connect('notify::focus-window', () => this.queueRefresh())]);
        this._signals.push([global.display, global.display.connect('window-created', () => this.queueRefresh())]);
        this._signals.push([global.workspace_manager, global.workspace_manager.connect('active-workspace-changed', () => this.queueRefresh())]);
        this._signals.push([this._appSystem, this._appSystem.connect('app-state-changed', () => this.queueRefresh())]);
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

    _refresh() {
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
                    id: String(window.get_stable_sequence()),
                    title: window.get_title?.() || '',
                    active: window === active,
                    dirty: false,
                    attention: needsAttention(window) ? 'attention' : 'normal',
                })),
                overflowCount: Math.max(0, windows.length - MAX_INLINE_WINDOWS),
            });
        }
        cards.sort((a, b) => Number(b.focused) - Number(a.focused) || a.name.localeCompare(b.name));
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
                    cycle: (current, direction) => this._cycle(current, direction),
                    newWindow: current => current.app.can_open_new_window?.() && current.app.open_new_window(-1),
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
        if (!card.windows?.length)
            return;
        if (card.focused && card.windows.length > 1) {
            this._cycle(card, Clutter.ScrollDirection.DOWN);
            return;
        }
        card.app.activate_window(card.windows[0], global.get_current_time());
    }

    _cycle(card, direction) {
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
        this.menu.removeAll();
        for (const window of card.windows || []) {
            const item = new PopupMenu.PopupMenuItem(window.get_title?.() || card.name);
            item.connect('activate', () => card.app.activate_window(window, global.get_current_time()));
            this.menu.addMenuItem(item);
        }
        if (card.app?.can_open_new_window?.()) {
            this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());
            const item = new PopupMenu.PopupMenuItem('New window');
            item.connect('activate', () => card.app.open_new_window(-1));
            this.menu.addMenuItem(item);
        }
        this.menu.open();
    }

    destroy() {
        if (this._refreshSource) {
            GLib.source_remove(this._refreshSource);
            this._refreshSource = 0;
        }
        for (const [object, id] of this._signals)
            object.disconnect(id);
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
