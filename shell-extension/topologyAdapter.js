import * as Main from 'resource:///org/gnome/shell/ui/main.js';

import {topologyRevision} from './topologyModel.js';

function rectValue(rect) {
    return {
        x: Math.trunc(Number(rect?.x ?? 0)),
        y: Math.trunc(Number(rect?.y ?? 0)),
        width: Math.trunc(Number(rect?.width ?? 0)),
        height: Math.trunc(Number(rect?.height ?? 0)),
    };
}

function validRect(rect) {
    return rect.width > 0 && rect.height > 0;
}

export function captureTopology() {
    const display = global.display;
    const count = Number(display.get_n_monitors?.() ?? 0);
    const primary = Number(display.get_primary_monitor?.() ?? 0);
    const monitors = [];
    for (let index = 0; index < count; index++) {
        const geometry = rectValue(display.get_monitor_geometry(index));
        let workArea = geometry;
        try {
            const candidate = rectValue(Main.layoutManager.getWorkAreaForMonitor(index));
            if (validRect(candidate))
                workArea = candidate;
        } catch (_error) {
            // Monitor geometry remains a valid logical-coordinate fallback.
        }
        const ref = `monitor:${index}`;
        monitors.push({
            ref,
            index,
            primary: index === primary,
            scale: Number(display.get_monitor_scale?.(index) ?? 1),
            geometry,
            workArea,
        });
    }
    const topology = {
        primaryMonitorRef: `monitor:${primary}`,
        monitors,
    };
    return Object.freeze({
        ...topology,
        revision: topologyRevision(topology),
    });
}

export function windowFrame(window) {
    try {
        return rectValue(window.get_frame_rect());
    } catch (_error) {
        return {x: 0, y: 0, width: 0, height: 0};
    }
}
