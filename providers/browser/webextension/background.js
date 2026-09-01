const api = globalThis.browser ?? globalThis.chrome;
const NATIVE_HOST = 'org.homiakus.hws.browser';
let port = null;
let revision = 0;
let timer = null;
let reconnectTimer = null;

function browserKind() {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes('firefox')) return 'firefox';
  if (ua.includes('chromium')) return 'chromium';
  return 'chrome';
}

async function desktopAppId() {
  const stored = await api.storage.local.get('desktopAppId');
  if (stored.desktopAppId) return stored.desktopAppId;
  switch (browserKind()) {
    case 'firefox': return 'firefox.desktop';
    case 'chromium': return 'chromium.desktop';
    default: return 'google-chrome.desktop';
  }
}

function ensurePort() {
  if (port) return port;
  try {
    port = api.runtime.connectNative(NATIVE_HOST);
    port.onMessage.addListener(handleCommand);
    port.onDisconnect.addListener(() => {
      port = null;
      if (!reconnectTimer)
        reconnectTimer = setTimeout(() => { reconnectTimer = null; ensurePort(); schedule(); }, 1500);
    });
    return port;
  } catch (_err) {
    return null;
  }
}

async function publish() {
  timer = null;
  const currentPort = ensurePort();
  if (!currentPort) return;
  const tabs = await api.tabs.query({});
  const payload = {
    schema: 1,
    browser: browserKind(),
    appId: await desktopAppId(),
    revision: ++revision,
    capturedAt: new Date().toISOString(),
    tabs: tabs
      .filter(tab => !tab.incognito)
      .map(tab => ({
        id: tab.id,
        windowId: tab.windowId,
        title: tab.title || '',
        active: Boolean(tab.active),
        pinned: Boolean(tab.pinned),
        audible: Boolean(tab.audible),
        muted: Boolean(tab.mutedInfo?.muted),
        incognito: false,
        mru: Number(tab.lastAccessed || Date.now()),
      })),
  };
  currentPort.postMessage(payload);
}

function schedule() {
  if (timer) clearTimeout(timer);
  timer = setTimeout(() => publish().catch(() => {}), 80);
}

async function handleCommand(message) {
  if (!message || typeof message !== 'object') return;
  if (message.type === 'activateView' && Number.isInteger(message.tabId)) {
    const tab = await api.tabs.get(message.tabId);
    if (tab.incognito) return;
    await api.tabs.update(message.tabId, {active: true});
    await api.windows.update(tab.windowId, {focused: true});
  } else if (message.type === 'closeView' && Number.isInteger(message.tabId)) {
    const tab = await api.tabs.get(message.tabId);
    if (!tab.incognito) await api.tabs.remove(message.tabId);
  }
}

for (const event of [api.tabs.onCreated, api.tabs.onRemoved, api.tabs.onUpdated, api.tabs.onActivated, api.tabs.onMoved, api.tabs.onAttached, api.tabs.onDetached, api.windows.onFocusChanged, api.windows.onCreated, api.windows.onRemoved])
  event?.addListener(schedule);

ensurePort();
schedule();
