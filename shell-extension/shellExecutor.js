import Shell from 'gi://Shell';

import {
    makeShellActionResult,
    ShellActionKind,
    stableWindowId,
    validateShellAction,
} from './shellActionModel.js';

export class ShellExecutor {
    constructor(complete) {
        this._appSystem = Shell.AppSystem.get_default();
        this._completeResult = complete;
    }

    handle(raw) {
        let action;
        try {
            action = validateShellAction(raw);
        } catch (error) {
            // An invalid request may not have a trustworthy request id, so it
            // cannot be completed safely. The daemon-side request timeout is
            // the fail-closed recovery path.
            console.error(`HWS rejected shell action: ${error.message}`);
            return;
        }

        try {
            switch (action.kind) {
            case ShellActionKind.ENSURE_DESKTOP_APP:
                this._ensureDesktopApp(action);
                break;
            case ShellActionKind.CLOSE_WINDOW:
                this._closeWindow(action);
                break;
            default:
                this._complete(action, false, false, 'unsupported_action', `Unsupported action ${action.kind}`);
                break;
            }
        } catch (error) {
            this._complete(action, false, false, 'executor_error', error?.message || String(error));
        }
    }

    _ensureDesktopApp(action) {
        const app = this._appSystem.lookup_app(action.desktopAppId);
        if (!app) {
            this._complete(action, false, false, 'app_not_found', `Desktop application ${action.desktopAppId} was not found`);
            return;
        }

        const stopped = app.get_state() === Shell.AppState.STOPPED;
        if (stopped)
            app.activate();
        this._complete(action, true, stopped);
    }

    _closeWindow(action) {
        const window = this._findWindow(action.windowId);
        if (!window) {
            // Absence is already the desired state and must be idempotent.
            this._complete(action, true, false);
            return;
        }
        if (!window.can_close()) {
            this._complete(action, false, false, 'window_not_closeable', `Window ${action.windowId} cannot be closed`);
            return;
        }
        window.delete(global.get_current_time());
        // This only means Mutter accepted the close request. hwsd still waits
        // for the window to disappear from observed state before convergence.
        this._complete(action, true, true);
    }

    _findWindow(windowId) {
        for (const actor of global.get_window_actors()) {
            const window = actor.meta_window ?? actor.get_meta_window?.();
            if (!window)
                continue;
            try {
                if (stableWindowId(window.get_stable_sequence()) === windowId)
                    return window;
            } catch (_error) {
                // Ignore a disappearing window and continue the observation.
            }
        }
        return null;
    }

    _complete(action, success, changed = false, code = '', message = '') {
        try {
            this._completeResult?.(makeShellActionResult(action, {
                success,
                changed,
                code,
                message,
            }));
        } catch (error) {
            console.error(`HWS could not complete shell action ${action.id}: ${error.message}`);
        }
    }

    destroy() {
        this._completeResult = null;
        this._appSystem = null;
    }
}
