import Clutter from 'gi://Clutter';
import GObject from 'gi://GObject';
import St from 'gi://St';

import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as ModalDialog from 'resource:///org/gnome/shell/ui/modalDialog.js';
import * as PanelMenu from 'resource:///org/gnome/shell/ui/panelMenu.js';

import {DaemonClient} from './daemonClient.js';
import {
    buildTreeModel,
    rowsForPath,
    sanitizePath,
    searchTree,
    selectPathNode,
} from './homeGridModel.js';
import {ShellExecutor} from './shellExecutor.js';
import {
    buildWorkspaceStateModel,
    workspacePresentation,
    workspaceStatusText,
} from './workspaceStateModel.js';

const HomeGridIndicator = GObject.registerClass(
class HomeGridIndicator extends PanelMenu.Button {
    _init(onActivate) {
        super._init(0.0, 'HWS Home', false);
        this.add_style_class_name('hws-home-trigger');
        this._onActivate = onActivate;

        const box = new St.BoxLayout({style_class: 'hws-home-trigger-content'});
        box.add_child(new St.Icon({icon_name: 'view-grid-symbolic', icon_size: 14}));
        box.add_child(new St.Label({text: 'HWS', y_align: Clutter.ActorAlign.CENTER}));
        this.add_child(box);

        this.connect('button-press-event', (_actor, event) => {
            if (event.get_button() !== 1)
                return Clutter.EVENT_PROPAGATE;
            this._onActivate?.();
            return Clutter.EVENT_STOP;
        });
    }
});

class HomeGridDialog extends ModalDialog.ModalDialog {
    constructor(onWorkspaceAction) {
        super({styleClass: 'hws-home-modal', destroyOnClose: false});
        this._onWorkspaceAction = onWorkspaceAction;
        this._available = false;
        this._model = null;
        this._workspaceStates = null;
        this._path = [];
        this._rowButtons = [];
        this._searchButtons = [];
        this._searchPaths = [];
        this._statusText = '';
        this._statusError = false;
        this._mutatingWorkspace = '';
        this._mutationToken = 0;

        this._root = new St.BoxLayout({
            vertical: true,
            style_class: 'hws-home-root',
            x_expand: true,
        });
        this._header = new St.BoxLayout({
            vertical: false,
            style_class: 'hws-home-header',
            x_expand: true,
        });
        this._title = new St.Label({
            text: 'Hierarchical Workspace',
            style_class: 'hws-home-title',
            x_expand: true,
            y_align: Clutter.ActorAlign.CENTER,
        });
        this._revision = new St.Label({
            style_class: 'hws-home-revision',
            y_align: Clutter.ActorAlign.CENTER,
        });
        this._header.add_child(this._title);
        this._header.add_child(this._revision);

        this._search = new St.Entry({
            style_class: 'hws-home-search',
            hint_text: 'Jump to context, project, task or workspace…',
            can_focus: true,
            x_expand: true,
        });
        this._search.clutter_text.connect('text-changed', () => this._renderSearch());
        this._search.clutter_text.connect('key-press-event', (_actor, event) => {
            const key = event.get_key_symbol();
            if (key === Clutter.KEY_Down && this._searchButtons.length > 0) {
                this._searchButtons[0].grab_key_focus();
                return Clutter.EVENT_STOP;
            }
            if ((key === Clutter.KEY_Return || key === Clutter.KEY_KP_Enter) && this._searchPaths.length > 0) {
                this._jumpToPath(this._searchPaths[0]);
                return Clutter.EVENT_STOP;
            }
            if (key === Clutter.KEY_Escape) {
                if (this._search.get_text())
                    this._search.set_text('');
                else
                    this.close(global.get_current_time());
                return Clutter.EVENT_STOP;
            }
            return Clutter.EVENT_PROPAGATE;
        });
        this._searchResults = new St.BoxLayout({
            vertical: true,
            style_class: 'hws-home-search-results',
            x_expand: true,
        });

        this._breadcrumbs = new St.BoxLayout({
            vertical: false,
            style_class: 'hws-home-breadcrumbs',
            x_expand: true,
        });
        this._rows = new St.BoxLayout({
            vertical: true,
            style_class: 'hws-home-rows',
            x_expand: true,
        });
        this._rowsViewport = new St.ScrollView({
            style_class: 'hws-home-rows-viewport',
            x_expand: true,
            y_expand: true,
            overlay_scrollbars: true,
        });
        this._rowsViewport.set_policy(St.PolicyType.NEVER, St.PolicyType.AUTOMATIC);
        this._rowsViewport.set_child(this._rows);

        this._status = new St.Label({
            style_class: 'hws-home-status',
            x_expand: true,
        });
        this._workspaceActions = new St.BoxLayout({
            vertical: false,
            style_class: 'hws-workspace-actions',
            x_expand: true,
        });
        this._hint = new St.Label({
            text: '←/→ choose · ↑/↓ level · Enter open · Backspace parent · Ctrl+K search · Esc close',
            style_class: 'hws-home-hint',
            x_expand: true,
        });

        this._root.add_child(this._header);
        this._root.add_child(this._search);
        this._root.add_child(this._searchResults);
        this._root.add_child(this._breadcrumbs);
        this._root.add_child(this._rowsViewport);
        this._root.add_child(this._status);
        this._root.add_child(this._workspaceActions);
        this._root.add_child(this._hint);
        this.contentLayout.add_child(this._root);
        this._render();
    }

    setAvailable(available) {
        this._available = Boolean(available);
        if (!this._available) {
            this._mutationToken++;
            this._mutatingWorkspace = '';
        }
        this._render();
    }

    setTree(payload) {
        if (!payload) {
            this._model = null;
            this._path = [];
            this._render();
            return;
        }
        try {
            const next = buildTreeModel(payload);
            this._model = next;
            this._path = sanitizePath(next, this._path);
            this._statusText = '';
            this._statusError = false;
        } catch (error) {
            this._statusText = `Hierarchy rejected: ${error.message}`;
            this._statusError = true;
        }
        this._render();
    }

    setWorkspaceStates(payload) {
        if (!payload) {
            this._workspaceStates = null;
            this._render();
            return;
        }
        try {
            this._workspaceStates = buildWorkspaceStateModel(payload);
        } catch (error) {
            this._statusText = `Workspace states rejected: ${error.message}`;
            this._statusError = true;
        }
        this._render();
    }

    resetPath() {
        if (!this._model)
            return;
        this._path = [this._model.rootId];
        this._search.set_text('');
        this._render();
        this._focusTile(0, 0);
    }

    open(timestamp = global.get_current_time()) {
        const opened = super.open(timestamp);
        if (opened !== false)
            this._focusTile(0, 0);
        return opened;
    }

    _render() {
        this._breadcrumbs.destroy_all_children();
        this._rows.destroy_all_children();
        this._workspaceActions.destroy_all_children();
        this._rowButtons = [];
        this._search.visible = this._available && Boolean(this._model);

        if (!this._available) {
            this._revision.text = '';
            this._searchResults.destroy_all_children();
            this._searchResults.visible = false;
            this._workspaceActions.visible = false;
            this._setStatus('hwsd is unavailable. Activity Strip fallback remains active.', true);
            this._hint.visible = false;
            return;
        }
        if (!this._model) {
            this._revision.text = '';
            this._searchResults.destroy_all_children();
            this._searchResults.visible = false;
            this._workspaceActions.visible = false;
            this._setStatus(this._statusText || 'Loading hierarchy…', this._statusError);
            this._hint.visible = false;
            return;
        }

        const projection = rowsForPath(this._model, this._path);
        this._path = projection.path;
        const workspaceRevision = this._workspaceStates ? ` · ws r${this._workspaceStates.revision}` : '';
        this._revision.text = `tree r${this._model.revision}${workspaceRevision}`;
        this._renderBreadcrumbs();

        projection.rows.forEach((row, rowIndex) => {
            const scroll = new St.ScrollView({
                style_class: 'hws-home-row-scroll',
                x_expand: true,
                overlay_scrollbars: true,
            });
            scroll.set_policy(St.PolicyType.AUTOMATIC, St.PolicyType.NEVER);
            const box = new St.BoxLayout({
                vertical: false,
                style_class: 'hws-home-row',
                x_expand: true,
            });
            scroll.set_child(box);

            const buttons = [];
            row.nodes.forEach((node, itemIndex) => {
                const state = this._workspaceState(node);
                const view = node.workspaceId ? workspacePresentation(state) : null;
                const label = view?.badge ? `${node.title}  ${view.badge}` : node.title;
                const styleClass = view
                    ? `hws-home-tile kind-${node.kind} ${view.className}`
                    : `hws-home-tile kind-${node.kind}`;
                const button = new St.Button({
                    label,
                    style_class: styleClass,
                    can_focus: true,
                    reactive: true,
                    track_hover: true,
                });
                button.accessible_name = view
                    ? `${node.title}, ${node.kind}, ${view.label}`
                    : `${node.title}, ${node.kind}`;
                button.toggle_style_class_name('selected', row.selectedId === node.id);
                button.toggle_style_class_name('mutating', Boolean(node.workspaceId) && node.workspaceId === this._mutatingWorkspace);
                button.connect('clicked', () => this._choose(rowIndex, itemIndex));
                button.connect('key-press-event', (_actor, event) =>
                    this._handleTileKey(event, rowIndex, itemIndex));
                box.add_child(button);
                buttons.push(button);
            });
            this._rowButtons.push(buttons);
            this._rows.add_child(scroll);
        });

        const selected = this._model.byId.get(this._path.at(-1));
        if (selected && this._path.length > 1 && (this._model.children.get(selected.id)?.length || 0) === 0) {
            if (selected.workspaceId) {
                const state = this._workspaceState(selected);
                this._setStatus(`${selected.title} · ${selected.workspaceId} · ${workspaceStatusText(state)}`, false);
                this._renderWorkspaceActions(selected, state);
            } else {
                this._setStatus(`${selected.kind}: ${selected.title}`, false);
            }
        } else {
            this._setStatus(this._statusText, this._statusError);
        }
        this._workspaceActions.visible = this._workspaceActions.get_n_children() > 0;
        this._hint.visible = true;
        this._renderSearch();
    }

    _renderWorkspaceActions(node, state) {
        const view = workspacePresentation(state);
        if (this._mutatingWorkspace === node.workspaceId || view.busy)
            return;

        if (view.primaryAction)
            this._addWorkspaceAction(view.primaryAction === 'recover' ? 'Recover' : view.status === 'active' ? 'Focus / Reconcile' : 'Activate', node, view.primaryAction);
        if (view.status === 'active' || view.status === 'degraded')
            this._addWorkspaceAction('Suspend', node, 'suspend');
        if (view.status === 'active' || view.status === 'degraded' || view.status === 'failed')
            this._addWorkspaceAction('Close', node, 'close', true);
    }

    _addWorkspaceAction(label, node, action, danger = false) {
        const button = new St.Button({
            label,
            style_class: danger ? 'hws-workspace-action danger' : 'hws-workspace-action',
            can_focus: true,
            reactive: true,
            track_hover: true,
        });
        button.accessible_name = `${label} workspace ${node.title}`;
        button.connect('clicked', () => this._mutateWorkspace(node, action));
        this._workspaceActions.add_child(button);
    }

    _renderSearch() {
        this._searchResults.destroy_all_children();
        this._searchButtons = [];
        this._searchPaths = [];
        if (!this._available || !this._model || !this._search.visible) {
            this._searchResults.visible = false;
            return;
        }
        const query = this._search.get_text();
        const results = searchTree(this._model, query, 8);
        if (!query.trim() || results.length === 0) {
            this._searchResults.visible = false;
            return;
        }
        for (const result of results) {
            const pathLabel = result.path
                .map(id => this._model.byId.get(id)?.title)
                .filter(Boolean)
                .join(' › ');
            const state = this._workspaceState(result.node);
            const status = result.node.workspaceId ? ` · ${workspacePresentation(state).label}` : '';
            const button = new St.Button({
                label: `${pathLabel}   · ${result.node.kind}${status}`,
                style_class: 'hws-home-search-result',
                can_focus: true,
                x_expand: true,
                x_align: Clutter.ActorAlign.FILL,
            });
            button.accessible_name = `Jump to ${pathLabel}${status}`;
            button.connect('clicked', () => this._jumpToPath(result.path));
            button.connect('key-press-event', (_actor, event) => {
                const key = event.get_key_symbol();
                if (key === Clutter.KEY_Up || key === Clutter.KEY_Down) {
                    const index = this._searchButtons.indexOf(button);
                    const delta = key === Clutter.KEY_Up ? -1 : 1;
                    const next = index + delta;
                    if (next < 0)
                        this._search.grab_key_focus();
                    else if (next < this._searchButtons.length)
                        this._searchButtons[next].grab_key_focus();
                    return Clutter.EVENT_STOP;
                }
                if (key === Clutter.KEY_Escape) {
                    this._search.set_text('');
                    this._search.grab_key_focus();
                    return Clutter.EVENT_STOP;
                }
                return Clutter.EVENT_PROPAGATE;
            });
            this._searchResults.add_child(button);
            this._searchButtons.push(button);
            this._searchPaths.push(result.path);
        }
        this._searchResults.visible = true;
    }

    _jumpToPath(path) {
        this._path = sanitizePath(this._model, path);
        const selected = this._path.at(-1);
        this._search.set_text('');
        this._render();
        this._focusSelected(Math.max(0, this._path.length - 2), selected);
    }

    _renderBreadcrumbs() {
        this._path.forEach((id, index) => {
            const node = this._model.byId.get(id);
            if (!node)
                return;
            if (index > 0)
                this._breadcrumbs.add_child(new St.Label({text: '›', style_class: 'hws-home-chevron'}));
            const crumb = new St.Button({
                label: node.title,
                style_class: 'hws-home-crumb',
                can_focus: true,
            });
            crumb.toggle_style_class_name('current', index === this._path.length - 1);
            crumb.connect('clicked', () => {
                this._path = this._path.slice(0, index + 1);
                this._render();
                this._focusTile(Math.min(index, this._rowButtons.length - 1), 0);
            });
            this._breadcrumbs.add_child(crumb);
        });
    }

    _choose(rowIndex, itemIndex) {
        const projection = rowsForPath(this._model, this._path);
        const row = projection.rows[rowIndex];
        const node = row?.nodes?.[itemIndex];
        if (!node)
            return;
        this._path = selectPathNode(this._model, projection.path, rowIndex, node.id);
        const hasChildren = (this._model.children.get(node.id)?.length || 0) > 0;
        this._render();
        if (hasChildren) {
            this._focusTile(rowIndex + 1, 0);
        } else if (node.workspaceId) {
            const view = workspacePresentation(this._workspaceState(node));
            if (view.primaryAction)
                this._mutateWorkspace(node, view.primaryAction);
            else
                this._focusSelected(rowIndex, node.id);
        } else {
            this._focusSelected(rowIndex, node.id);
        }
    }

    _workspaceState(node) {
        if (!node?.workspaceId)
            return null;
        return this._workspaceStates?.byId.get(node.workspaceId) ?? {
            workspaceId: node.workspaceId,
            status: 'inactive',
            definitionRevision: '',
            reachedRequired: 0,
            totalRequired: 0,
            lastFailureCode: '',
        };
    }

    _mutateWorkspace(node, action) {
        if (!this._onWorkspaceAction) {
            this._setStatus(`Workspace control is unavailable for ${node.title}`, true);
            return;
        }
        if (this._mutatingWorkspace)
            return;

        const token = ++this._mutationToken;
        this._mutatingWorkspace = node.workspaceId;
        this._render();
        this._setStatus(`${workspaceActionLabel(action)} ${node.title}…`, false);

        this._onWorkspaceAction(
            action,
            node.workspaceId,
            state => {
                if (token !== this._mutationToken)
                    return;
                this._mutatingWorkspace = '';
                const status = typeof state?.status === 'string' ? state.status : 'unknown';
                if ((action === 'activate' || action === 'recover' || action === 'resume') && status === 'active') {
                    this.close(global.get_current_time());
                    return;
                }
                this._render();
                const failure = state?.lastFailureCode ? ` · ${state.lastFailureCode}` : '';
                this._setStatus(`Workspace ${node.title}: ${status}${failure}`, status === 'failed');
            },
            error => {
                if (token !== this._mutationToken)
                    return;
                this._mutatingWorkspace = '';
                this._render();
                this._setStatus(`${workspaceActionLabel(action)} failed: ${error?.message || String(error)}`, true);
            }
        );
    }

    _handleTileKey(event, rowIndex, itemIndex) {
        const key = event.get_key_symbol();
        const state = event.get_state();
        if ((state & Clutter.ModifierType.CONTROL_MASK) && (key === Clutter.KEY_k || key === Clutter.KEY_K)) {
            this._search.grab_key_focus();
            return Clutter.EVENT_STOP;
        }
        switch (key) {
        case Clutter.KEY_Left:
            this._focusTile(rowIndex, itemIndex - 1);
            return Clutter.EVENT_STOP;
        case Clutter.KEY_Right:
            this._focusTile(rowIndex, itemIndex + 1);
            return Clutter.EVENT_STOP;
        case Clutter.KEY_Up:
            this._focusTile(rowIndex - 1, itemIndex);
            return Clutter.EVENT_STOP;
        case Clutter.KEY_Down:
            this._focusTile(rowIndex + 1, itemIndex);
            return Clutter.EVENT_STOP;
        case Clutter.KEY_Return:
        case Clutter.KEY_KP_Enter:
        case Clutter.KEY_space:
            this._choose(rowIndex, itemIndex);
            return Clutter.EVENT_STOP;
        case Clutter.KEY_BackSpace:
            this._goParent();
            return Clutter.EVENT_STOP;
        case Clutter.KEY_Escape:
            this.close(global.get_current_time());
            return Clutter.EVENT_STOP;
        default:
            break;
        }
        if (key >= Clutter.KEY_1 && key <= Clutter.KEY_9) {
            const quickIndex = key - Clutter.KEY_1;
            if (quickIndex < (this._rowButtons[rowIndex]?.length || 0))
                this._choose(rowIndex, quickIndex);
            return Clutter.EVENT_STOP;
        }
        return Clutter.EVENT_PROPAGATE;
    }

    _goParent() {
        if (!this._model || this._path.length <= 1)
            return;
        const removed = this._path.at(-1);
        const rowIndex = this._path.length - 2;
        this._path = this._path.slice(0, -1);
        this._render();
        this._focusSelected(rowIndex, removed);
    }

    _focusSelected(rowIndex, nodeId) {
        const projection = rowsForPath(this._model, this._path);
        const index = projection.rows[rowIndex]?.nodes?.findIndex(node => node.id === nodeId) ?? -1;
        this._focusTile(rowIndex, index >= 0 ? index : 0);
    }

    _focusTile(rowIndex, itemIndex) {
        if (rowIndex < 0 || rowIndex >= this._rowButtons.length)
            return;
        const row = this._rowButtons[rowIndex];
        if (!row || row.length === 0)
            return;
        const index = Math.max(0, Math.min(row.length - 1, itemIndex));
        row[index].grab_key_focus();
        this.setInitialKeyFocus(row[index]);
    }

    _setStatus(text, error) {
        this._status.text = text || '';
        this._status.visible = Boolean(text);
        this._status.toggle_style_class_name('error', Boolean(error));
    }
}

class HomeGridController {
    constructor() {
        this._tree = null;
        this._workspaceStates = null;
        this._available = false;
        this._dialog = null;
        this._open = false;
        this._executor = new ShellExecutor(result => {
            this._client?.completeShellAction(result, null, error =>
                console.error(`HWS shell action completion failed: ${error?.message || error}`));
        });

        this._client = new DaemonClient({
            onAvailabilityChanged: available => {
                this._available = available;
                this._dialog?.setAvailable(available);
                if (available) {
                    this._client.queueTreeRefresh();
                    this._client.queueWorkspaceStatesRefresh();
                }
            },
            onTreeChanged: tree => {
                this._tree = tree;
                this._dialog?.setTree(tree);
            },
            onWorkspaceStatesChanged: states => {
                this._workspaceStates = states;
                this._dialog?.setWorkspaceStates(states);
            },
            onShellAction: action => this._executor?.handle(action),
        });

        this._indicator = new HomeGridIndicator(() => this.toggle());
        Main.panel.addToStatusArea('hws-home-grid', this._indicator, 0, 'left');
    }

    _ensureDialog() {
        if (this._dialog)
            return this._dialog;
        this._dialog = new HomeGridDialog((action, workspaceID, done, failed) =>
            this._runWorkspaceAction(action, workspaceID, done, failed));
        this._dialog.connect('opened', () => {
            this._open = true;
        });
        this._dialog.connect('closed', () => {
            this._open = false;
        });
        return this._dialog;
    }

    _runWorkspaceAction(action, workspaceID, done, failed) {
        switch (action) {
        case 'activate':
            this._client.activateWorkspace(workspaceID, done, failed);
            break;
        case 'recover':
            this._client.recoverWorkspace(workspaceID, done, failed);
            break;
        case 'resume':
            this._client.resumeWorkspace(workspaceID, done, failed);
            break;
        case 'suspend':
            this._client.suspendWorkspace(workspaceID, done, failed);
            break;
        case 'close':
            this._client.closeWorkspace(workspaceID, done, failed);
            break;
        default:
            failed?.(new Error(`unsupported workspace action ${action}`));
            break;
        }
    }

    open() {
        const dialog = this._ensureDialog();
        dialog.setAvailable(this._available);
        dialog.setTree(this._tree);
        dialog.setWorkspaceStates(this._workspaceStates);
        this._client.queueTreeRefresh();
        this._client.queueWorkspaceStatesRefresh();
        dialog.open(global.get_current_time());
    }

    close() {
        this._dialog?.close(global.get_current_time());
    }

    toggle() {
        if (this._open)
            this.close();
        else
            this.open();
    }

    destroy() {
        this._dialog?.destroy();
        this._dialog = null;
        this._indicator?.destroy();
        this._indicator = null;
        this._client?.destroy();
        this._client = null;
        this._executor?.destroy();
        this._executor = null;
    }
}

function workspaceActionLabel(action) {
    switch (action) {
    case 'recover':
        return 'Recovering';
    case 'resume':
        return 'Resuming';
    case 'suspend':
        return 'Suspending';
    case 'close':
        return 'Closing';
    default:
        return 'Activating';
    }
}

export function installHomeGrid() {
    return new HomeGridController();
}
