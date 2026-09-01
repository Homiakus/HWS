'use strict';

const net = require('net');
const os = require('os');
const path = require('path');
const vscode = require('vscode');

let revision = 0;
let timer = null;
let reconnectTimer = null;
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

function allTabs() {
  return vscode.window.tabGroups.all.flatMap((group, groupIndex) =>
    group.tabs.map((tab, tabIndex) => ({group, groupIndex, tab, tabIndex, id: tabId(groupIndex, tabIndex, tab)})));
}

function snapshot() {
  const tabs = [];
  allTabs().forEach(({groupIndex, tabIndex, tab, id}) => {
    if (tab.isActive) mru.set(id, ++mruCounter);
    tabs.push({
      id,
      groupId: String(groupIndex),
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

function scheduleReconnect() {
  if (reconnectTimer) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connect();
    schedule();
  }, 1200);
}

function connect() {
  if (socket && !socket.destroyed) return socket;
  socket = net.createConnection({path: socketPath()});
  socket.setEncoding('utf8');
  socket.on('connect', schedule);
  socket.on('data', chunk => {
    buffer += chunk;
    let index;
    while ((index = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, index); buffer = buffer.slice(index + 1);
      try { void handleCommand(JSON.parse(line)); } catch (_err) {}
    }
  });
  socket.on('error', () => {});
  socket.on('close', () => { socket = null; scheduleReconnect(); });
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

function findTarget(message) {
  const candidates = allTabs();
  if (typeof message.viewId === 'string') {
    const exact = candidates.find(candidate => candidate.id === message.viewId);
    if (exact) return exact;
  }
  if (typeof message.resource === 'string' && message.resource)
    return candidates.find(candidate => resourceFor(candidate.tab.input) === message.resource);
  return null;
}

async function handleCommand(message) {
  if (!message || typeof message !== 'object') return;
  const target = findTarget(message);
  if (!target) return;
  try {
    if (message.type === 'activateView') {
      const resource = resourceFor(target.tab.input);
      if (resource)
        await vscode.commands.executeCommand('vscode.open', vscode.Uri.parse(resource));
    } else if (message.type === 'closeView') {
      await vscode.window.tabGroups.close(target.tab);
    }
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
  context.subscriptions.push({dispose() {
    if (timer) clearTimeout(timer);
    if (reconnectTimer) clearTimeout(reconnectTimer);
    socket?.destroy();
  }});
  connect();
  schedule();
}

function deactivate() { socket?.destroy(); }
module.exports = {activate, deactivate};
