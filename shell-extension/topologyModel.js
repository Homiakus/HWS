function normalizeRect(rect) {
    return {
        x: Number(rect?.x ?? 0),
        y: Number(rect?.y ?? 0),
        width: Number(rect?.width ?? 0),
        height: Number(rect?.height ?? 0),
    };
}

export function topologySemantic(topology) {
    const monitors = [...(topology?.monitors || [])]
        .map(monitor => ({
            ref: String(monitor.ref || ''),
            index: Number(monitor.index),
            primary: Boolean(monitor.primary),
            scale: Number(monitor.scale),
            geometry: normalizeRect(monitor.geometry),
            workArea: normalizeRect(monitor.workArea),
        }))
        .sort((a, b) => a.index - b.index || a.ref.localeCompare(b.ref));
    return JSON.stringify({
        primaryMonitorRef: String(topology?.primaryMonitorRef || ''),
        monitors,
    });
}

function fnv32(text, seed) {
    let hash = seed >>> 0;
    for (let i = 0; i < text.length; i++) {
        hash ^= text.charCodeAt(i);
        hash = Math.imul(hash, 0x01000193) >>> 0;
    }
    return hash >>> 0;
}

export function topologyRevision(topology) {
    const semantic = topologySemantic(topology);
    const a = fnv32(semantic, 0x811c9dc5).toString(16).padStart(8, '0');
    const b = fnv32(semantic, 0x9e3779b9).toString(16).padStart(8, '0');
    return `topology:${a}${b}`;
}
