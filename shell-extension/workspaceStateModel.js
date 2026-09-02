const ALLOWED_STATUSES = new Set([
    'inactive',
    'preparing',
    'active',
    'degraded',
    'recovering',
    'closing',
    'failed',
]);

export function buildWorkspaceStateModel(payload) {
    if (!payload || typeof payload !== 'object')
        throw new Error('workspace state snapshot is required');
    if (payload.schema !== 1)
        throw new Error(`unsupported workspace state schema ${payload.schema}`);
    if (!Number.isSafeInteger(payload.revision) || payload.revision < 1)
        throw new Error('workspace state revision must be a positive integer');
    if (!Array.isArray(payload.states))
        throw new Error('workspace states must be an array');

    const byId = new Map();
    for (const raw of payload.states) {
        const state = normalizeWorkspaceState(raw);
        if (byId.has(state.workspaceId))
            throw new Error(`duplicate workspace state ${state.workspaceId}`);
        byId.set(state.workspaceId, state);
    }
    return Object.freeze({
        revision: payload.revision,
        catalogRevision: Number.isSafeInteger(payload.catalogRevision) ? payload.catalogRevision : 0,
        byId,
    });
}

export function normalizeWorkspaceState(raw) {
    if (!raw || typeof raw !== 'object')
        throw new Error('workspace state must be an object');
    const workspaceId = typeof raw.workspaceId === 'string' ? raw.workspaceId.trim() : '';
    if (!workspaceId)
        throw new Error('workspace state requires workspaceId');
    const status = typeof raw.status === 'string' ? raw.status.trim() : '';
    if (!ALLOWED_STATUSES.has(status))
        throw new Error(`workspace ${workspaceId} has invalid status ${status || '<empty>'}`);
    return Object.freeze({
        workspaceId,
        status,
        definitionRevision: typeof raw.definitionRevision === 'string' ? raw.definitionRevision : '',
        reachedRequired: safeCount(raw.reachedRequired),
        totalRequired: safeCount(raw.totalRequired),
        lastFailureCode: typeof raw.lastFailureCode === 'string' ? raw.lastFailureCode : '',
    });
}

export function workspacePresentation(state) {
    const status = state?.status || 'inactive';
    switch (status) {
    case 'active':
        return {status, className: 'workspace-active', badge: '●', primaryAction: 'activate', busy: false, label: 'Active'};
    case 'degraded':
        return {status, className: 'workspace-degraded', badge: '!', primaryAction: 'recover', busy: false, label: 'Degraded'};
    case 'failed':
        return {status, className: 'workspace-failed', badge: '×', primaryAction: 'recover', busy: false, label: 'Failed'};
    case 'preparing':
        return {status, className: 'workspace-busy', badge: '…', primaryAction: null, busy: true, label: 'Preparing'};
    case 'recovering':
        return {status, className: 'workspace-busy', badge: '…', primaryAction: null, busy: true, label: 'Recovering'};
    case 'closing':
        return {status, className: 'workspace-busy', badge: '…', primaryAction: null, busy: true, label: 'Closing'};
    default:
        return {status: 'inactive', className: 'workspace-inactive', badge: '', primaryAction: 'activate', busy: false, label: 'Inactive'};
    }
}

export function workspaceStatusText(state) {
    const view = workspacePresentation(state);
    const counts = state && state.totalRequired > 0
        ? ` · required ${state.reachedRequired}/${state.totalRequired}`
        : '';
    const failure = state?.lastFailureCode ? ` · ${state.lastFailureCode}` : '';
    return `${view.label}${counts}${failure}`;
}

function safeCount(value) {
    return Number.isSafeInteger(value) && value >= 0 ? value : 0;
}
