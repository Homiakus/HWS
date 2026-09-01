import assert from 'node:assert/strict';
import test from 'node:test';

import {
    buildTreeModel,
    pathToNode,
    rowsForPath,
    sanitizePath,
    selectPathNode,
} from './homeGridModel.js';

function fixture() {
    return {
        schema: 1,
        revision: 7,
        rootId: 'root',
        nodes: [
            {id: 'root', title: 'Home', kind: 'category', order: 0},
            {id: 'dev', parentId: 'root', title: 'Development', kind: 'category', order: 10},
            {id: 'cad', parentId: 'root', title: 'CAD', kind: 'category', order: 20},
            {id: 'projects', parentId: 'dev', title: 'Projects', kind: 'category', order: 10},
            {id: 'hws', parentId: 'projects', title: 'HWS', kind: 'project', order: 10},
            {id: 'develop', parentId: 'hws', title: 'Develop', kind: 'task', order: 10, workspaceId: 'hws-dev'},
        ],
    };
}

test('rows expand dynamically to arbitrary hierarchy depth', () => {
    const model = buildTreeModel(fixture());
    let path = ['root'];
    path = selectPathNode(model, path, 0, 'dev');
    path = selectPathNode(model, path, 1, 'projects');
    path = selectPathNode(model, path, 2, 'hws');
    path = selectPathNode(model, path, 3, 'develop');

    const projection = rowsForPath(model, path);
    assert.deepEqual(projection.path, ['root', 'dev', 'projects', 'hws', 'develop']);
    assert.equal(projection.rows.length, 4);
    assert.equal(projection.rows[0].selectedId, 'dev');
    assert.equal(projection.rows[3].selectedId, 'develop');
});

test('tree revision sanitizes a path at the first missing descendant', () => {
    const before = buildTreeModel(fixture());
    const oldPath = pathToNode(before, 'develop');
    const changed = fixture();
    changed.revision = 8;
    changed.nodes = changed.nodes.filter(node => node.id !== 'hws' && node.id !== 'develop');
    const after = buildTreeModel(changed);

    assert.deepEqual(sanitizePath(after, oldPath), ['root', 'dev', 'projects']);
});

test('malformed duplicate, missing-parent and cycle snapshots are rejected', () => {
    const duplicate = fixture();
    duplicate.nodes.push({...duplicate.nodes[1]});
    assert.throws(() => buildTreeModel(duplicate), /duplicate node/);

    const missing = fixture();
    missing.nodes = missing.nodes.map(node => node.id === 'projects' ? {...node, parentId: 'missing'} : node);
    assert.throws(() => buildTreeModel(missing), /missing parent/);

    const cycle = fixture();
    cycle.nodes = cycle.nodes.map(node => {
        if (node.id === 'dev')
            return {...node, parentId: 'projects'};
        if (node.id === 'projects')
            return {...node, parentId: 'dev'};
        return node;
    });
    assert.throws(() => buildTreeModel(cycle), /cycle detected/);
});

test('path lookup is fail-closed for unknown nodes', () => {
    const model = buildTreeModel(fixture());
    assert.deepEqual(pathToNode(model, 'hws'), ['root', 'dev', 'projects', 'hws']);
    assert.deepEqual(pathToNode(model, 'missing'), []);
});
