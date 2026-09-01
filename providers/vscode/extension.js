'use strict';

const net = require('net');
const os = require('os');
const path = require('path');
const vscode = require('vscode');

let revision = 0;
let timer = null;
let socket = null;
let buffer = '';
const mru = new Map();
let mruCounter = 0;

function socketPath() {
  if (process.env.HWS_PROVIDER_SOCKET) return process.env.HWS_PROVIDER_SOCKET;
  const runtime = process.env.XDG_RUNTIME_DIR || os.tmpdir();
  return path.join(runtime, 'hws', 'providers.sock');
}

function resourceFor(input) {
  if (input instanceof vscode.TabInputText) return input.uri.toString();
  if (input instanceof vscode.TabInputTextDiff) return input.modified.toString();
  if (input instanceof vscode.TabInputNotebook) return input.uri.toString();
  if (input instanceof vscode.TabInputNotebookDiff) return input.modified.toString();
  if (input instanceof vscode.TabInputCustom) return input.uri.toString();
  return '';
}

function kindFor(input) {
  if (input instanceof vscode.TabInputWebview) return 'custom';
  if (input instanceof vscode.TabInputNotebook || input instanceof vscode.TabInputNotebookDiff) return 'document';
  return 'editor';
}

function tabId(groupIndex, tabIndex, tab) {
  const resource = resourceFor(tab.input);
  return resource ? `resource:${resource}` : `session:${groupIndex}:${tabIndex}:${tab.label}`;
}

function snapshot() {
  const tabs = [];
  vscode.window.tabGroups.all.forEach((group, gi) => {
    group.tabs.forEach((tab, ti) => {
      const id = tabId(gi, ti, tab);
      if (tab.isActive) mru.set(id, ++mruCounter);
      tabs.push({
        id,
        groupId: String(gi),
        title: tab.label,
        resource: resourceFor(tab.input),
        active: tab.isActive,
        dirty: tab.isDirty,
        pinned: tab.isPinned,
        preview: tab.isPreview,
        kind: kindFor(tab.input),
        mru: mru.get(id) || 0,
      });
    });
  });
  return {
    schema: 1,
    appId: vscode.workspace.getConfiguration('hws').get('desktopAppId', 'code.desktop'),
    revision: ++revision,
    capturedAt: new Date().toISOString(),
    workspace: vscode.workspace.name || '',
    tabs,
  };
}

function envelope() {
  return JSON.stringify({schema: 1, type: 'vscode.snapshot', source: 'vscode-extension', receivedAt: new Date().toISOString(), payload: snapshot()}) + '\n';
}

function connect() {
  if (socket && !socket.destroyed) return socket;
  socket = net.createConnection({path: socketPath()});
  socket.setEncoding('utf8');
  socket.on('data', chunk => {
    buffer += chunk;
    let index;
    while ((index = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, index); buffer = buffer.slice(index + 1);
      try { handleCommand(JSON.parse(line)); } catch (_err) {}
    }
  });
  socket.on('error', () => {});
  socket.on('close', () => { socket = null; });
  return socket;
}

function schedule() {
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => {
    timer = null;
    const s = connect();
    if (s && !s.destroyed) s.write(envelope());
  }, 80);
}

async function handleCommand(message) {
  if (!message || message.type !== 'activateView' || typeof message.resource !== 'string') return;
  const target = vscode.window.tabGroups.all.flatMap(group => group.tabs).find(tab => resourceFor(tab.input) === message.resource);
  if (!target) return;
  const resource = resourceFor(target.input);
  if (!resource) return;
  try {
    await vscode.commands.executeCommand('vscode.open', vscode.Uri.parse(resource));
  } catch (_err) {}
}

function activate(context) {
  const subscriptions = [
    vscode.window.tabGroups.onDidChangeTabs(schedule),
    vscode.window.tabGroups.onDidChangeTabGroups(schedule),
    vscode.window.onDidChangeWindowState(schedule),
    vscode.window.onDidChangeActiveTextEditor(schedule),
    vscode.workspace.onDidChangeWorkspaceFolders(schedule),
    vscode.workspace.onDidChangeConfiguration(event => { if (event.affectsConfiguration('hws.desktopAppId')) schedule(); }),
  ];
  subscriptions.forEach(d => context.subscriptions.push(d));
  context.subscriptions.push({dispose() { if (timer) clearTimeout(timer); socket?.destroy(); }});
  schedule();
}

function deactivate() { socket?.destroy(); }
module.exports = {activate, deactivate};
