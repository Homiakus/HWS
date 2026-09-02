import Gio from 'gi://Gio';

import {ExtensionState} from 'resource:///org/gnome/shell/misc/extensionUtils.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as Scripting from 'resource:///org/gnome/shell/ui/scripting.js';

const UUID = 'hws@homiakus';
const BUS_NAME = 'org.homiakus.HWS1';
const OBJECT_PATH = '/org/homiakus/HWS1';
const INTERFACE_NAME = 'org.homiakus.HWS1';

export var METRICS = {};

function assert(condition, message) {
    if (!condition)
        throw new Error(message);
}

async function eventually(description, predicate, attempts = 100, delayMS = 100) {
    let lastError = null;
    for (let attempt = 0; attempt < attempts; attempt++) {
        try {
            const value = predicate();
            if (value)
                return value;
        } catch (error) {
            lastError = error;
        }
        await Scripting.sleep(delayMS);
    }
    const suffix = lastError ? `; last error: ${lastError.message}` : '';
    throw new Error(`Timed out waiting for ${description}${suffix}`);
}

async function waitForExtension() {
    return eventually('HWS extension to become active', () => {
        const extension = Main.extensionManager.lookup(UUID);
        if (!extension)
            return null;
        if (extension.state === ExtensionState.ERROR)
            throw new Error(`HWS extension failed: ${extension.error || 'unknown error'}`);
        if (extension.state !== ExtensionState.ACTIVE || !extension.stateObj)
            return null;
        return extension.stateObj;
    });
}

function callJSON(method) {
    const reply = Gio.DBus.session.call_sync(
        BUS_NAME,
        OBJECT_PATH,
        INTERFACE_NAME,
        method,
        null,
        null,
        Gio.DBusCallFlags.NONE,
        3000,
        null
    );
    const values = reply.deep_unpack();
    assert(Array.isArray(values) && typeof values[0] === 'string', `${method} returned an unexpected D-Bus payload`);
    return JSON.parse(values[0]);
}

function validateHealth(health) {
    assert(health && typeof health === 'object', 'GetHealth did not return an object');
    assert(health.configValid === true, `panel config is invalid: ${health.configError || 'unknown'}`);
    assert(health.hierarchyValid === true, `hierarchy is invalid: ${health.hierarchyError || 'unknown'}`);
    assert(health.workspaceCatalogValid === true, `workspace catalog is invalid: ${health.workspaceCatalogError || 'unknown'}`);
    assert(health.workspaceLifecycleReady === true, 'workspace lifecycle is unavailable');
}

function validateTopology(topology) {
    assert(topology && typeof topology === 'object', 'topology snapshot is unavailable');
    assert(/^topology:[0-9a-f]{16}$/.test(topology.revision || ''), `unexpected topology revision ${topology.revision}`);
    assert(Array.isArray(topology.monitors) && topology.monitors.length > 0, 'topology contains no monitors');
    const primary = topology.monitors.filter(monitor => Boolean(monitor.primary));
    assert(primary.length === 1, `expected one primary monitor, got ${primary.length}`);
    assert(primary[0].ref === topology.primaryMonitorRef, 'primary monitor ref does not match primary monitor');
    for (const monitor of topology.monitors) {
        assert(typeof monitor.ref === 'string' && monitor.ref.length > 0, 'monitor ref is missing');
        assert(Number.isInteger(monitor.index) && monitor.index >= 0, `invalid monitor index ${monitor.index}`);
        assert(Number.isFinite(monitor.scale) && monitor.scale > 0, `invalid monitor scale ${monitor.scale}`);
        assert(monitor.geometry?.width > 0 && monitor.geometry?.height > 0, `invalid monitor geometry for ${monitor.ref}`);
        assert(monitor.workArea?.width > 0 && monitor.workArea?.height > 0, `invalid monitor work area for ${monitor.ref}`);
    }
}

export async function run() {
    console.debug('HWS GNOME 50 qualification: start');
    await Scripting.waitLeisure();

    const state = await waitForExtension();
    assert(state._indicator?.get_parent(), 'HWS Activity Strip was not attached to the panel');
    assert(state._home?._indicator?.get_parent(), 'HWS Home Grid trigger was not attached to the panel');

    await eventually('Activity Strip daemon client', () => state._indicator?._daemon?.available === true);
    await eventually('Home Grid daemon client', () => state._home?._client?.available === true);

    const health = callJSON('GetHealth');
    validateHealth(health);

    const nativeCards = state._indicator._nativeCards();
    const payload = state._indicator._shellPayload(nativeCards);
    validateTopology(payload.topology);

    // The same native topology must have crossed the real extension -> D-Bus ->
    // daemon boundary. Retry briefly because publication is coalesced by the
    // production extension before SubmitShellSnapshot.
    await eventually('daemon to observe the GNOME topology', () => {
        const refreshed = callJSON('GetHealth');
        return refreshed.status === 'ok' || refreshed.status === 'degraded';
    });

    METRICS = {
        monitorCount: payload.topology.monitors.length,
        topologyRevision: payload.topology.revision,
    };
    console.debug(`HWS GNOME 50 qualification: topology=${payload.topology.revision} monitors=${payload.topology.monitors.length}`);
    console.debug('HWS GNOME 50 qualification: success');
}

export function finish() {
}
