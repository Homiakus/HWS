import assert from 'node:assert/strict';
import test from 'node:test';

import {
    buildWorkspaceStateModel,
    workspacePresentation,
    workspaceStatusText,
} from './workspaceStateModel.js';

test('buildWorkspaceStateModel indexes valid states', () => {
    const model = buildWorkspaceStateModel({
        schema: 1,
        revision: 7,
        catalogRevision: 3,
        states: [
            {workspaceId: 'dev', status: 'active', definitionRevision: 'v1', reachedRequired: 2, totalRequired: 2},
            {workspaceId: 'docs', status: 'degraded', lastFailureCode: 'browser_missing'},
        ],
    });
    assert.equal(model.revision, 7);
    assert.equal(model.byId.get('dev').status, 'active');
    assert.equal(model.byId.get('docs').lastFailureCode, 'browser_missing');
});

test('workspace presentation maps lifecycle to safe primary actions', () => {
    assert.equal(workspacePresentation({status: 'inactive'}).primaryAction, 'activate');
    assert.equal(workspacePresentation({status: 'active'}).primaryAction, 'activate');
    assert.equal(workspacePresentation({status: 'degraded'}).primaryAction, 'recover');
    assert.equal(workspacePresentation({status: 'failed'}).primaryAction, 'recover');
    assert.equal(workspacePresentation({status: 'preparing'}).primaryAction, null);
    assert.equal(workspacePresentation({status: 'recovering'}).busy, true);
    assert.equal(workspacePresentation({status: 'closing'}).busy, true);
});

test('status text includes required progress and failure code', () => {
    assert.equal(
        workspaceStatusText({status: 'degraded', reachedRequired: 1, totalRequired: 2, lastFailureCode: 'terminal_missing'}),
        'Degraded · required 1/2 · terminal_missing'
    );
});

test('invalid and duplicate states fail closed', () => {
    assert.throws(() => buildWorkspaceStateModel({schema: 2, revision: 1, states: []}), /unsupported/);
    assert.throws(() => buildWorkspaceStateModel({
        schema: 1,
        revision: 1,
        states: [
            {workspaceId: 'dev', status: 'active'},
            {workspaceId: 'dev', status: 'inactive'},
        ],
    }), /duplicate/);
    assert.throws(() => buildWorkspaceStateModel({
        schema: 1,
        revision: 1,
        states: [{workspaceId: 'dev', status: 'mystery'}],
    }), /invalid status/);
});
