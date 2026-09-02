import assert from 'node:assert/strict';
import test from 'node:test';

import {
    makeShellActionResult,
    ShellActionKind,
    stableWindowId,
    validateShellAction,
} from './shellActionModel.js';

function ensureAction() {
    return {
        schema: 1,
        id: 'request-1',
        kind: ShellActionKind.ENSURE_DESKTOP_APP,
        workspaceId: 'dev',
        resourceId: 'editor',
        desktopAppId: 'dev.zed.Zed.desktop',
    };
}

function placementAction() {
    return {
        schema: 1,
        id: 'request-place',
        kind: ShellActionKind.PLACE_WINDOW,
        workspaceId: 'dev',
        resourceId: 'editor',
        windowId: 'window:42',
        topologyRevision: 'topology:abc',
        monitorRef: 'monitor:1',
        monitorIndex: 1,
        targetWorkspace: 2,
        rect: {x: 1920, y: 0, width: 854, height: 960},
    };
}

test('ensure desktop action validates typed desktop identity', () => {
    const action = validateShellAction(ensureAction());
    assert.equal(action.kind, 'ensure_desktop_app');
    assert.equal(action.desktopAppId, 'dev.zed.Zed.desktop');
});

test('close window action validates stable window identity', () => {
    const action = validateShellAction({
        schema: 1,
        id: 'request-2',
        kind: ShellActionKind.CLOSE_WINDOW,
        workspaceId: 'dev',
        resourceId: 'editor',
        windowId: stableWindowId(42),
    });
    assert.equal(action.windowId, 'window:42');
});

test('placement action requires topology, monitor identity and logical rect', () => {
    const action = validateShellAction(placementAction());
    assert.equal(action.kind, 'place_window');
    assert.equal(action.monitorRef, 'monitor:1');
    assert.equal(action.targetWorkspace, 2);
    assert.deepEqual(action.rect, {x: 1920, y: 0, width: 854, height: 960});
    assert.throws(() => validateShellAction({...placementAction(), topologyRevision: ''}), /topologyRevision/);
    assert.throws(() => validateShellAction({...placementAction(), monitorIndex: -1}), /monitorIndex/);
    assert.throws(() => validateShellAction({...placementAction(), rect: {x: 0, y: 0, width: 0, height: 10}}), /rect/);
});

test('invalid schemas, kinds and missing identities fail closed', () => {
    assert.throws(() => validateShellAction({...ensureAction(), schema: 2}), /unsupported shell action schema/);
    assert.throws(() => validateShellAction({...ensureAction(), kind: 'exec'}), /unsupported shell action kind/);
    assert.throws(() => validateShellAction({...ensureAction(), desktopAppId: ''}), /desktopAppId/);
    assert.throws(() => validateShellAction({
        schema: 1,
        id: 'request-3',
        kind: ShellActionKind.CLOSE_WINDOW,
        workspaceId: 'dev',
        resourceId: 'editor',
    }), /windowId/);
});

test('results echo the request identity and preserve changed evidence', () => {
    const result = makeShellActionResult(ensureAction(), {
        success: true,
        changed: true,
    });
    assert.deepEqual(result, {
        schema: 1,
        id: 'request-1',
        success: true,
        changed: true,
        code: '',
        message: '',
    });
});

test('stable window ids reject ambiguous numeric input', () => {
    assert.equal(stableWindowId(0), 'window:0');
    assert.throws(() => stableWindowId(-1), /non-negative/);
    assert.throws(() => stableWindowId(Number.NaN), /safe integer/);
});
