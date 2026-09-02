import assert from 'node:assert/strict';
import test from 'node:test';

import {topologyRevision, topologySemantic} from './topologyModel.js';

function topology() {
    return {
        primaryMonitorRef: 'monitor:0',
        monitors: [
            {ref: 'monitor:1', index: 1, primary: false, scale: 1.5, geometry: {x: 1920, y: 0, width: 1707, height: 960}, workArea: {x: 1920, y: 0, width: 1707, height: 960}},
            {ref: 'monitor:0', index: 0, primary: true, scale: 1, geometry: {x: 0, y: 0, width: 1920, height: 1080}, workArea: {x: 0, y: 32, width: 1920, height: 1048}},
        ],
    };
}

test('topology revision is stable across monitor input order', () => {
    const first = topology();
    const second = topology();
    second.monitors.reverse();
    assert.equal(topologySemantic(first), topologySemantic(second));
    assert.equal(topologyRevision(first), topologyRevision(second));
});

test('topology revision changes for geometry, scale and primary changes', () => {
    const base = topologyRevision(topology());
    const geometry = topology();
    geometry.monitors[0].workArea.width -= 1;
    assert.notEqual(topologyRevision(geometry), base);
    const scale = topology();
    scale.monitors[0].scale = 2;
    assert.notEqual(topologyRevision(scale), base);
    const primary = topology();
    primary.primaryMonitorRef = 'monitor:1';
    assert.notEqual(topologyRevision(primary), base);
});
