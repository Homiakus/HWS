const MAX_NODES = 4096;
const MAX_DEPTH = 4096;
const MAX_SEARCH_RESULTS = 50;

function requireString(value, name) {
    if (typeof value !== 'string' || value.length === 0)
        throw new Error(`${name} must be a non-empty string`);
    return value;
}

function compareNodes(a, b) {
    return (Number(a.order) || 0) - (Number(b.order) || 0) ||
        a.title.localeCompare(b.title) ||
        a.id.localeCompare(b.id);
}

export function buildTreeModel(payload) {
    if (!payload || typeof payload !== 'object')
        throw new Error('tree snapshot is required');
    if (payload.schema !== 1)
        throw new Error(`unsupported tree schema ${payload.schema}`);
    const rootId = requireString(payload.rootId, 'rootId');
    if (!Array.isArray(payload.nodes) || payload.nodes.length < 1 || payload.nodes.length > MAX_NODES)
        throw new Error(`nodes must contain 1..${MAX_NODES} entries`);

    const byId = new Map();
    for (const raw of payload.nodes) {
        if (!raw || typeof raw !== 'object')
            throw new Error('node must be an object');
        const node = Object.freeze({
            id: requireString(raw.id, 'node.id'),
            parentId: typeof raw.parentId === 'string' ? raw.parentId : '',
            kind: typeof raw.kind === 'string' && raw.kind ? raw.kind : 'category',
            title: requireString(raw.title, 'node.title'),
            order: Number.isInteger(raw.order) ? raw.order : 0,
            workspaceId: typeof raw.workspaceId === 'string' ? raw.workspaceId : '',
        });
        if (byId.has(node.id))
            throw new Error(`duplicate node ${node.id}`);
        byId.set(node.id, node);
    }

    const root = byId.get(rootId);
    if (!root)
        throw new Error(`root ${rootId} is missing`);
    if (root.parentId)
        throw new Error('root must not have a parent');

    const children = new Map();
    for (const node of byId.values()) {
        if (node.id === rootId)
            continue;
        if (!node.parentId || !byId.has(node.parentId))
            throw new Error(`node ${node.id} has a missing parent ${node.parentId}`);
        if (!children.has(node.parentId))
            children.set(node.parentId, []);
        children.get(node.parentId).push(node);
    }
    for (const values of children.values())
        values.sort(compareNodes);

    // Validate that every node reaches the declared root. This also catches
    // cycles without recursion, so malformed provider data cannot overflow the
    // Shell stack.
    for (const node of byId.values()) {
        const seen = new Set();
        let current = node;
        for (let depth = 0; depth <= byId.size; depth++) {
            if (seen.has(current.id))
                throw new Error(`cycle detected at ${current.id}`);
            seen.add(current.id);
            if (!current.parentId) {
                if (current.id !== rootId)
                    throw new Error(`node ${node.id} does not reach root ${rootId}`);
                current = null;
                break;
            }
            current = byId.get(current.parentId);
            if (!current)
                throw new Error(`node ${node.id} has a missing ancestor`);
        }
        if (current)
            throw new Error(`path for ${node.id} exceeds safe depth`);
    }

    return Object.freeze({
        schema: 1,
        revision: Number(payload.revision) || 0,
        rootId,
        byId,
        children,
    });
}

export function childrenOf(model, nodeId) {
    return model?.children?.get(nodeId) || [];
}

export function sanitizePath(model, inputPath) {
    if (!model)
        return [];
    const source = Array.isArray(inputPath) ? inputPath : [];
    const path = [model.rootId];
    let parent = model.rootId;
    for (const candidate of source.slice(source[0] === model.rootId ? 1 : 0)) {
        const id = typeof candidate === 'string' ? candidate : candidate?.id;
        if (!id)
            break;
        const node = model.byId.get(id);
        if (!node || node.parentId !== parent)
            break;
        path.push(id);
        parent = id;
        if (path.length >= MAX_DEPTH)
            break;
    }
    return path;
}

export function selectPathNode(model, currentPath, rowIndex, nodeId) {
    const path = sanitizePath(model, currentPath);
    const parentId = path[rowIndex];
    const node = model?.byId?.get(nodeId);
    if (!parentId || !node || node.parentId !== parentId)
        return path;
    return [...path.slice(0, rowIndex + 1), node.id];
}

export function rowsForPath(model, inputPath) {
    const path = sanitizePath(model, inputPath);
    const rows = [];
    for (let index = 0; index < path.length; index++) {
        const parentId = path[index];
        const nodes = childrenOf(model, parentId);
        if (nodes.length === 0)
            break;
        rows.push({
            parentId,
            selectedId: path[index + 1] || '',
            nodes,
        });
    }
    return {path, rows};
}

export function pathToNode(model, nodeId) {
    if (!model?.byId?.has(nodeId))
        return [];
    const reverse = [];
    const seen = new Set();
    let current = model.byId.get(nodeId);
    for (let depth = 0; current && depth < MAX_DEPTH; depth++) {
        if (seen.has(current.id))
            return [];
        seen.add(current.id);
        reverse.push(current.id);
        if (!current.parentId)
            break;
        current = model.byId.get(current.parentId);
    }
    reverse.reverse();
    return reverse[0] === model.rootId ? reverse : [];
}

function subsequenceScore(value, query) {
    let position = 0;
    let gaps = 0;
    for (const char of query) {
        const next = value.indexOf(char, position);
        if (next < 0)
            return 0;
        gaps += next - position;
        position = next + 1;
    }
    return Math.max(1, 180 - gaps);
}

function scoreNode(node, query) {
    const title = node.title.toLocaleLowerCase();
    const id = node.id.toLocaleLowerCase();
    const workspace = node.workspaceId.toLocaleLowerCase();
    if (title === query)
        return 1000;
    if (title.startsWith(query))
        return 850;
    if (title.split(/\s+/).some(word => word.startsWith(query)))
        return 760;
    if (title.includes(query))
        return 620;
    if (id === query || workspace === query)
        return 560;
    if (id.includes(query))
        return 430;
    if (workspace.includes(query))
        return 400;
    return subsequenceScore(title, query);
}

export function searchTree(model, rawQuery, limit = 12) {
    if (!model)
        return [];
    const query = String(rawQuery || '').trim().toLocaleLowerCase();
    if (!query)
        return [];
    const boundedLimit = Math.max(1, Math.min(MAX_SEARCH_RESULTS, Number(limit) || 12));
    const ranked = [];
    for (const node of model.byId.values()) {
        if (node.id === model.rootId)
            continue;
        const score = scoreNode(node, query);
        if (score > 0)
            ranked.push({node, score});
    }
    ranked.sort((a, b) =>
        b.score - a.score ||
        compareNodes(a.node, b.node));
    return ranked.slice(0, boundedLimit).map(result => ({
        ...result,
        path: pathToNode(model, result.node.id),
    }));
}
