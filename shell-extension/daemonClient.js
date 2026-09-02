import Gio from 'gi://Gio';
import GLib from 'gi://GLib';

const BUS_NAME = 'org.homiakus.HWS1';
const OBJECT_PATH = '/org/homiakus/HWS1';
const INTERFACE_NAME = 'org.homiakus.HWS1';
const PROTOCOL_VERSION = 1;

export class DaemonClient {
    constructor({
        onCardsChanged = null,
        onTreeChanged = null,
        onWorkspaceStatesChanged = null,
        onShellAction = null,
        onAvailabilityChanged = null,
    } = {}) {
        this._onCardsChanged = onCardsChanged;
        this._onTreeChanged = onTreeChanged;
        this._onWorkspaceStatesChanged = onWorkspaceStatesChanged;
        this._onShellAction = onShellAction;
        this._onAvailabilityChanged = onAvailabilityChanged;
        this._proxy = null;
        this._signals = [];
        this._cancellable = new Gio.Cancellable();
        this._refreshSource = 0;
        this._treeRefreshSource = 0;
        this._workspaceRefreshSource = 0;
        this._cards = [];
        this._render = null;
        this._tree = null;
        this._workspaceStates = null;
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

    get tree() {
        return this._tree;
    }

    get workspaceStates() {
        return this._workspaceStates;
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
                this._signals.push(this._proxy.connect('g-signal', (_proxy, _sender, signalName, parameters) => {
                    if (signalName === 'PanelChanged' || signalName === 'PanelConfigChanged') {
                        this.queueRefresh();
                    } else if (signalName === 'TreeChanged') {
                        this.queueTreeRefresh();
                    } else if (signalName === 'WorkspaceChanged') {
                        this.queueWorkspaceStatesRefresh();
                    } else if (signalName === 'ShellActionRequested') {
                        this._dispatchShellAction(parameters);
                    }
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
            this._tree = null;
            this._workspaceStates = null;
            this._setAvailable(false);
            this._onCardsChanged?.(this._cards, this._render);
            this._onTreeChanged?.(null);
            this._onWorkspaceStatesChanged?.(null);
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
            this.queueTreeRefresh();
            this.queueWorkspaceStatesRefresh();
        }, () => this._setAvailable(false));
    }

    _dispatchShellAction(parameters) {
        if (!this._onShellAction)
            return;
        try {
            const values = parameters?.deep_unpack?.();
            const payload = Array.isArray(values) ? values[0] : null;
            if (typeof payload !== 'string')
                return;
            const action = JSON.parse(payload);
            if (action && typeof action === 'object')
                this._onShellAction(action);
        } catch (error) {
            console.error(`HWS could not decode ShellActionRequested: ${error.message}`);
        }
    }

    _setAvailable(value) {
        if (this._available === value)
            return;
        this._available = value;
        this._onAvailabilityChanged?.(value);
    }

    _call(method, parameters, done = null, failed = null, timeoutMS = 1500) {
        if (!this._proxy || !this._proxy.get_name_owner?.()) {
            failed?.(new Error('hwsd is unavailable'));
            return;
        }
        this._proxy.call(
            method,
            parameters,
            Gio.DBusCallFlags.NONE,
            timeoutMS,
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
        if (!this._available || !this._onCardsChanged || this._refreshSource)
            return;
        this._refreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 30, () => {
            this._refreshSource = 0;
            this.refresh();
            return GLib.SOURCE_REMOVE;
        });
    }

    queueTreeRefresh() {
        if (!this._available || !this._onTreeChanged || this._treeRefreshSource)
            return;
        this._treeRefreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 30, () => {
            this._treeRefreshSource = 0;
            this.refreshTree();
            return GLib.SOURCE_REMOVE;
        });
    }

    queueWorkspaceStatesRefresh() {
        if (!this._available || !this._onWorkspaceStatesChanged || this._workspaceRefreshSource)
            return;
        this._workspaceRefreshSource = GLib.timeout_add(GLib.PRIORITY_DEFAULT, 30, () => {
            this._workspaceRefreshSource = 0;
            this.refreshWorkspaceStates();
            return GLib.SOURCE_REMOVE;
        });
    }

    refresh() {
        if (!this._available || !this._onCardsChanged)
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

    refreshTree() {
        if (!this._available || !this._onTreeChanged)
            return;
        this._call('GetTree', null, values => {
            try {
                const payload = JSON.parse(values[0]);
                this._tree = payload && typeof payload === 'object' ? payload : null;
                this._onTreeChanged?.(this._tree);
            } catch (_error) {
                // Keep the last valid tree on malformed daemon output.
            }
        });
    }

    refreshWorkspaceStates() {
        if (!this._available || !this._onWorkspaceStatesChanged)
            return;
        this._call('GetWorkspaceStates', null, values => {
            try {
                const payload = JSON.parse(values[0]);
                if (!payload || typeof payload !== 'object')
                    return;
                this._workspaceStates = payload;
                this._onWorkspaceStatesChanged?.(payload);
            } catch (_error) {
                // Keep the last valid workspace-state snapshot.
            }
        });
    }

    getPath(nodeID, done) {
        this._call('GetPath', new GLib.Variant('(s)', [nodeID]), values => {
            try {
                const payload = JSON.parse(values[0]);
                done?.(Array.isArray(payload) ? payload : null);
            } catch (_error) {
                done?.(null);
            }
        }, () => done?.(null));
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

    completeShellAction(result, done = null, failed = null) {
        if (!result || typeof result !== 'object') {
            failed?.(new Error('shell action result is required'));
            return;
        }
        this._call(
            'CompleteShellAction',
            new GLib.Variant('(s)', [JSON.stringify(result)]),
            done,
            failed
        );
    }

    activateWorkspace(workspaceID, done = null, failed = null) {
        this._workspaceMutation('ActivateWorkspace', 'activate', workspaceID, done, failed);
    }

    recoverWorkspace(workspaceID, done = null, failed = null) {
        this._workspaceMutation('RecoverWorkspace', 'recover', workspaceID, done, failed);
    }

    resumeWorkspace(workspaceID, done = null, failed = null) {
        this._workspaceMutation('ResumeWorkspace', 'resume', workspaceID, done, failed);
    }

    suspendWorkspace(workspaceID, done = null, failed = null) {
        const id = this._workspaceID(workspaceID, failed);
        if (!id)
            return;
        this._call(
            'SuspendWorkspace',
            new GLib.Variant('(s)', [id]),
            values => this._decodeWorkspaceState(values, done, failed),
            failed,
            7000
        );
    }

    closeWorkspace(workspaceID, done = null, failed = null) {
        this._workspaceMutation('CloseWorkspace', 'close', workspaceID, done, failed);
    }

    _workspaceMutation(method, action, workspaceID, done, failed) {
        const id = this._workspaceID(workspaceID, failed);
        if (!id)
            return;
        const operationKey = `shell:${action}:${this._instance}:${GLib.uuid_string_random()}`;
        this._call(
            method,
            new GLib.Variant('(ss)', [id, operationKey]),
            values => this._decodeWorkspaceState(values, done, failed),
            failed,
            22000
        );
    }

    _workspaceID(workspaceID, failed) {
        if (typeof workspaceID !== 'string' || !workspaceID.trim()) {
            failed?.(new Error('workspace id is required'));
            return '';
        }
        return workspaceID.trim();
    }

    _decodeWorkspaceState(values, done, failed) {
        try {
            const state = JSON.parse(values[0]);
            done?.(state && typeof state === 'object' ? state : null);
        } catch (error) {
            failed?.(error);
        }
    }

    getWorkspaceState(workspaceID, done = null, failed = null) {
        this._call('GetWorkspaceState', new GLib.Variant('(s)', [workspaceID]), values =>
            this._decodeWorkspaceState(values, done, failed), failed);
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
        if (this._treeRefreshSource) {
            GLib.source_remove(this._treeRefreshSource);
            this._treeRefreshSource = 0;
        }
        if (this._workspaceRefreshSource) {
            GLib.source_remove(this._workspaceRefreshSource);
            this._workspaceRefreshSource = 0;
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
        this._tree = null;
        this._workspaceStates = null;
        this._onWorkspaceStatesChanged = null;
        this._onShellAction = null;
    }
}
