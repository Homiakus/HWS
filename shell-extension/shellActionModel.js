export const SHELL_ACTION_SCHEMA = 1;

export const ShellActionKind = Object.freeze({
    ENSURE_DESKTOP_APP: 'ensure_desktop_app',
    CLOSE_WINDOW: 'close_window',
});

function requireString(value, field) {
    if (typeof value !== 'string' || value.trim().length === 0)
        throw new Error(`${field} must be a non-empty string`);
    return value.trim();
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
    };

    switch (action.kind) {
    case ShellActionKind.ENSURE_DESKTOP_APP:
        requireString(action.desktopAppId, 'desktopAppId');
        break;
    case ShellActionKind.CLOSE_WINDOW:
        requireString(action.windowId, 'windowId');
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
