import Clutter from 'gi://Clutter';
import GObject from 'gi://GObject';
import St from 'gi://St';

export const AppCapsule = GObject.registerClass(
class AppCapsule extends St.Button {
    _init(card, handlers = {}) {
        super._init({
            style_class: 'hws-app-capsule',
            can_focus: true,
            reactive: true,
            track_hover: true,
            x_expand: false,
        });
        this._handlers = handlers;
        this._card = null;

        this._root = new St.BoxLayout({vertical: true, style_class: 'hws-app-capsule-content'});
        this._top = new St.BoxLayout({vertical: false, x_expand: true});
        this._iconBin = new St.Bin({style_class: 'hws-app-icon'});
        this._title = new St.Label({style_class: 'hws-app-title', y_align: Clutter.ActorAlign.CENTER, x_expand: true});
        this._badge = new St.Label({style_class: 'hws-app-badge', y_align: Clutter.ActorAlign.CENTER});
        this._top.add_child(this._iconBin);
        this._top.add_child(this._title);
        this._top.add_child(this._badge);

        this._subtitle = new St.Label({style_class: 'hws-app-subtitle'});
        this._segments = new St.BoxLayout({vertical: false, style_class: 'hws-surface-segments'});
        this._progressTrack = new St.Widget({style_class: 'hws-progress-track', x_expand: true});
        this._progressFill = new St.Widget({style_class: 'hws-progress-fill'});
        this._progressTrack.add_child(this._progressFill);

        this._root.add_child(this._top);
        this._root.add_child(this._subtitle);
        this._root.add_child(this._segments);
        this._root.add_child(this._progressTrack);
        this.set_child(this._root);

        this.connect('clicked', () => this._handlers.activate?.(this._card));
        this.connect('button-press-event', (_actor, event) => {
            const button = event.get_button();
            if (button === 2) {
                this._handlers.newWindow?.(this._card);
                return Clutter.EVENT_STOP;
            }
            if (button === 3) {
                this._handlers.openSwitcher?.(this._card, this);
                return Clutter.EVENT_STOP;
            }
            return Clutter.EVENT_PROPAGATE;
        });
        this.connect('scroll-event', (_actor, event) => {
            this._handlers.cycle?.(this._card, event.get_scroll_direction());
            return Clutter.EVENT_STOP;
        });
        this.update(card);
    }

    update(card) {
        this._card = card;
        this.toggle_style_class_name('focused', Boolean(card.focused));
        this.toggle_style_class_name('urgent', card.attention === 'urgent');
        this.toggle_style_class_name('attention', card.attention === 'attention');
        this._title.text = card.title || card.name || card.id;
        this._subtitle.text = card.subtitle || '';
        this._subtitle.visible = Boolean(card.subtitle);
        const count = card.surfaceCount > 0 ? card.surfaceCount : card.windowCount;
        this._badge.text = count > 1 ? String(count) : '';
        this._badge.visible = count > 1;

        this._iconBin.destroy_all_children();
        if (card.iconActor)
            this._iconBin.set_child(card.iconActor);
        else
            this._iconBin.set_child(new St.Icon({icon_name: card.iconName || 'application-x-executable-symbolic', icon_size: 16}));

        this._segments.destroy_all_children();
        for (const segment of (card.segments || []).slice(0, 4)) {
            const dot = new St.Widget({style_class: 'hws-surface-segment'});
            dot.toggle_style_class_name('active', Boolean(segment.active));
            dot.toggle_style_class_name('dirty', Boolean(segment.dirty));
            dot.toggle_style_class_name('urgent', segment.attention === 'urgent');
            dot.accessible_name = segment.title || 'surface';
            this._segments.add_child(dot);
        }
        if ((card.overflowCount || 0) > 0)
            this._segments.add_child(new St.Label({style_class: 'hws-segment-overflow', text: `+${card.overflowCount}`}));
        this._segments.visible = this._segments.get_n_children() > 0;

        const progress = typeof card.progress === 'number' ? Math.max(0, Math.min(1, card.progress)) : null;
        this._progressTrack.visible = progress !== null;
        if (progress !== null) {
            const width = Math.max(1, Math.round(100 * progress));
            this._progressFill.set_style(`width: ${width}%;`);
        }
    }
});
