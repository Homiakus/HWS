export const SHELL_ACTION_SCHEMA = 1;

export const ShellActionKind = Object.freeze({
    ENSURE_DESKTOP_APP: 'ensure_desktop_app',
    CLOSE_WINDOW: 'close_window',
    PLACE_WINDOW: 'place_window',
});

function requireString(value, field) {
    if (typeof value !== 'string' || value.trim().length === 0)
        throw new Error(`${field} must be a non-empty string`);
    return value.trim();
}

function requireIndex(value, field) {
    if (!Number.isInteger(value) || value < 0)
        throw new Error(`${field} must be a non-negative integer`);
    return value;
}

function validateRect(raw) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw))
        throw new Error('rect must be an object');
    const rect = {
        x: Number(raw.x),
        y: Number(raw.y),
        width: Number(raw.width),
        height: Number(raw.height),
    };
    if (![rect.x, rect.y, rect.width, rect.height].every(Number.isFinite) || rect.width <= 0 || rect.height <= 0)
        throw new Error('rect must contain finite positive dimensions');
    return Object.freeze(rect);
}

export function validateShellAction(raw) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw))
        throw new Error('shell action must be an object');
    if (raw.schema !== SHELL_ACTION_SCHEMA)
        throw new Error(`unsupported shell action schema ${raw.schema}`);

    const action = {
        schema: SHELL_ACTION_SCHEMA,
        id: requireString(raw.id, 'id'),
        kind: requireString(raw.kind, 'kind'),
        workspaceId: requireString(raw.workspaceId, 'workspaceId'),
        resourceId: requireString(raw.resourceId, 'resourceId'),
        desktopAppId: typeof raw.desktopAppId === 'string' ? raw.desktopAppId.trim() : '',
        windowId: typeof raw.windowId === 'string' ? raw.windowId.trim() : '',
        topologyRevision: typeof raw.topologyRevision === 'string' ? raw.topologyRevision.trim() : '',
        monitorRef: typeof raw.monitorRef === 'string' ? raw.monitorRef.trim() : '',
        monitorIndex: Number.isInteger(raw.monitorIndex) ? raw.monitorIndex : -1,
        targetWorkspace: Number.isInteger(raw.targetWorkspace) ? raw.targetWorkspace : -1,
        rect: raw.rect ?? null,
    };

    switch (action.kind) {
    case ShellActionKind.ENSURE_DESKTOP_APP:
        requireString(action.desktopAppId, 'desktopAppId');
        break;
    case ShellActionKind.CLOSE_WINDOW:
        requireString(action.windowId, 'windowId');
        break;
    case ShellActionKind.PLACE_WINDOW:
        requireString(action.windowId, 'windowId');
        requireString(action.topologyRevision, 'topologyRevision');
        requireString(action.monitorRef, 'monitorRef');
        requireIndex(action.monitorIndex, 'monitorIndex');
        requireIndex(action.targetWorkspace, 'targetWorkspace');
        action.rect = validateRect(action.rect);
        break;
    default:
        throw new Error(`unsupported shell action kind ${action.kind}`);
    }
    return Object.freeze(action);
}

export function makeShellActionResult(action, {
    success,
    changed = false,
    code = '',
    message = '',
} = {}) {
    const valid = validateShellAction(action);
    if (typeof success !== 'boolean')
        throw new Error('success must be boolean');
    return Object.freeze({
        schema: SHELL_ACTION_SCHEMA,
        id: valid.id,
        success,
        changed: Boolean(changed),
        code: typeof code === 'string' ? code : '',
        message: typeof message === 'string' ? message : '',
    });
}

export function stableWindowId(sequence) {
    const value = Number(sequence);
    if (!Number.isSafeInteger(value) || value < 0)
        throw new Error('stable window sequence must be a non-negative safe integer');
    return `window:${value}`;
}
