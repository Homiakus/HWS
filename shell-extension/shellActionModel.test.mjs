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
