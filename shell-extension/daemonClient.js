import Gio from 'gi://Gio';
import GLib from 'gi://GLib';

const BUS_NAME = 'org.homiakus.HWS1';
const OBJECT_PATH = '/org/homiakus/HWS1';
const INTERFACE_NAME = 'org.homiakus.HWS1';
const PROTOCOL_VERSION = 1;

export class DaemonClient {
    constructor({onCardsChanged = null, onAvailabilityChanged = null} = {}) {
        this._onCardsChanged = onCardsChanged;
        this._onAvailabilityChanged = onAvailabilityChanged;
        this._proxy = null;
        this._signals = [];
        this._cancellable = new Gio.Cancellable();
        this._refreshSource = 0;
        this._cards = [];
        this._render = null;
        this._available = false;
        this._instance = GLib.uuid_string_random();
        this._connect();
    }

    get cards() {
        return this._cards;
    }

    get render() {
        return this._render;
    }

    get available() {
        return this._available;
    }

    _connect() {
        Gio.DBusProxy.new_for_bus(
            Gio.BusType.SESSION,
            Gio.DBusProxyFlags.DO_NOT_LOAD_PROPERTIES,
            null,
            BUS_NAME,
            OBJECT_PATH,
            INTERFACE_NAME,
            this._cancellable,
            (_source, result) => {
                if (this._cancellable.is_cancelled())
                    return;
                try {
                    this._proxy = Gio.DBusProxy.new_for_bus_finish(result);
                } catch (_error) {
                    this._setAvailable(false);
                    return;
                }
                this._signals.push(this._proxy.connect('notify::g-name-owner', () => this._ownerChanged()));
                this._signals.push(this._proxy.connect('g-signal', (_proxy, _sender, signalName) => {
                    if (signalName === 'PanelChanged' || signalName === 'PanelConfigChanged')
                        this.queueRefresh();
                }));
                this._ownerChanged();
            }
        );
    }

    _ownerChanged() {
        const owner = this._proxy?.get_name_owner?.();
        if (!owner) {
            this._cards = [];
            this._render = null;
            this._setAvailable(false);
            this._onCardsChanged?.(this._cards, this._render);
            return;
        }
        this._call('Hello', new GLib.Variant('(us)', [PROTOCOL_VERSION, this._instance]), values => {
            const [serverProtocol] = values;
            if (serverProtocol !== PROTOCOL_VERSION) {
                this._setAvailable(false);
                return;
            }
            this._setAvailable(true);
            this.queueRefresh();
        }, () => this._setAvailable(false));
    }

    _setAvailable(value) {
        if (this._available === value)
            return;
        this._available = value;
        this._onAvailabilityChanged?.(value);
    }

    _call(method, parameters, done = null, failed = null) {
        if (!this._proxy || !this._proxy.get_name_owner?.()) {
            failed?.();
            return;
        }
        this._proxy.call(
            method,
            parameters,
            Gio.DBusCallFlags.NONE,
            1500,
            this._cancellable,
            (proxy, result) => {
                if (this._cancellable.is_cancelled())
                    return;
                try {
                    const value = proxy.call_finish(result);
                    done?.(value.deep_unpack());
                } catch (error) {
                    failed?.(error);
                }
            }
        );
    }

    queueRefresh() {
        if (!this._available || this._refreshSource)
            return;
        this._refreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 30, () => {
            this._refreshSource = 0;
            this.refresh();
            return GLib.SOURCE_REMOVE;
        });
    }

    refresh() {
        if (!this._available)
            return;
        this._call('GetPanelSnapshot', null, values => {
            try {
                const payload = JSON.parse(values[0]);
                this._cards = Array.isArray(payload.cards) ? payload.cards : [];
                this._render = payload.render && typeof payload.render === 'object' ? payload.render : null;
                this._onCardsChanged?.(this._cards, this._render);
            } catch (_error) {
                // Keep the last valid snapshot on malformed daemon output.
            }
        });
    }

    submitShellSnapshot(snapshot, done = null, failed = null) {
        if (!snapshot || typeof snapshot !== 'object') {
            failed?.(new Error('shell snapshot is required'));
            return;
        }
        this._call(
            'SubmitShellSnapshot',
            new GLib.Variant('(s)', [JSON.stringify(snapshot)]),
            done,
            failed
        );
    }

    getHealth(done) {
        this._call('GetHealth', null, values => {
            try {
                done?.(JSON.parse(values[0]));
            } catch (_error) {
                done?.(null);
            }
        }, () => done?.(null));
    }

    getApplicationSurface(appID, done) {
        this._call('GetApplicationSurface', new GLib.Variant('(s)', [appID]), values => {
            try {
                done?.(JSON.parse(values[0]));
            } catch (_error) {
                done?.(null);
            }
        }, () => done?.(null));
    }

    activateView(appID, viewID) {
        this._call('ActivateView', new GLib.Variant('(ss)', [appID, viewID]));
    }

    closeView(appID, viewID) {
        this._call('CloseView', new GLib.Variant('(ss)', [appID, viewID]));
    }

    destroy() {
        if (this._refreshSource) {
            GLib.source_remove(this._refreshSource);
            this._refreshSource = 0;
        }
        this._cancellable.cancel();
        if (this._proxy) {
            for (const id of this._signals)
                this._proxy.disconnect(id);
        }
        this._signals = [];
        this._proxy = null;
        this._cards = [];
        this._render = null;
    }
}
