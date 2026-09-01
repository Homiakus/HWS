import Adw from 'gi://Adw';
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import Gtk from 'gi://Gtk?version=4.0';

import {ExtensionPreferences} from 'resource:///org/gnome/Shell/Extensions/js/extensions/prefs.js';

const DEFAULT_DSL = `panel "main" {
  edge = "top"
  height = 40
  gap = 6
  overflow = "popover"

  group "applications" {
    source = "running"
    app {
      density = "adaptive"
      min_width = 64
      preferred_width = 156
      max_width = 240
      surfaces { mode = "segments" max_visible = 4 overflow = "count" }
    }
    on "click" { action = "focus_or_cycle" }
    on "scroll" { action = "cycle_surface" }
    on "middle_click" { action = "new_window" }
  }

  group "system" {
    widget "network" { variant = "mini" }
    widget "clock" { format = "HH:mm" }
  }
}
`;

function configPath() {
    return GLib.build_filenamev([GLib.get_user_config_dir(), 'hws', 'panel.hws.hcl']);
}

function loadText() {
    const file = Gio.File.new_for_path(configPath());
    try {
        const [ok, bytes] = file.load_contents(null);
        if (ok)
            return new TextDecoder().decode(bytes);
    } catch (_error) {}
    return DEFAULT_DSL;
}

function saveText(text) {
    const path = configPath();
    GLib.mkdir_with_parents(GLib.path_get_dirname(path), 0o700);
    const file = Gio.File.new_for_path(path);
    file.replace_contents(text, null, false, Gio.FileCreateFlags.REPLACE_DESTINATION, null);
}

export default class HWSPreferences extends ExtensionPreferences {
    fillPreferencesWindow(window) {
        window.set_default_size(860, 680);
        window.search_enabled = true;

        const page = new Adw.PreferencesPage({
            title: 'Activity Strip',
            icon_name: 'view-grid-symbolic',
        });
        window.add(page);

        const quick = new Adw.PreferencesGroup({
            title: 'Panel configuration',
            description: 'Edit the declarative HWS panel DSL. Invalid files never replace the daemon last-known-good model.',
        });
        page.add(quick);

        const hint = new Adw.ActionRow({
            title: 'Configuration file',
            subtitle: configPath(),
        });
        quick.add(hint);

        const editorGroup = new Adw.PreferencesGroup({title: 'DSL editor'});
        page.add(editorGroup);

        const frame = new Gtk.Frame({hexpand: true, vexpand: true});
        const scroller = new Gtk.ScrolledWindow({min_content_height: 420, hexpand: true, vexpand: true});
        const view = new Gtk.TextView({monospace: true, wrap_mode: Gtk.WrapMode.NONE, left_margin: 12, right_margin: 12, top_margin: 10, bottom_margin: 10});
        view.buffer.set_text(loadText(), -1);
        scroller.set_child(view);
        frame.set_child(scroller);
        editorGroup.add(frame);

        const actions = new Adw.PreferencesGroup({title: 'Actions'});
        page.add(actions);
        const row = new Adw.ActionRow({title: 'Save configuration', subtitle: 'hwsd will validate and hot-reload this file.'});
        const save = new Gtk.Button({label: 'Save', valign: Gtk.Align.CENTER});
        save.connect('clicked', () => {
            const [start, end] = view.buffer.get_bounds();
            saveText(view.buffer.get_text(start, end, true));
            row.subtitle = `Saved: ${configPath()}`;
        });
        row.add_suffix(save);

        const reset = new Gtk.Button({label: 'Reset', valign: Gtk.Align.CENTER});
        reset.connect('clicked', () => view.buffer.set_text(DEFAULT_DSL, -1));
        row.add_suffix(reset);
        actions.add(row);
    }
}
