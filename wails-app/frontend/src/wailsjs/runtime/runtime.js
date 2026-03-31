'use strict';

// Wails runtime bridge — real implementation injected by Wails webview at runtime.
// This stub allows Vite dev server to run outside of Wails (e.g. in a browser).

const noop = () => {};

function _wails(method, ...args) {
  if (window.runtime && typeof window.runtime[method] === 'function') {
    return window.runtime[method](...args);
  }
  // Running in a plain browser (outside Wails) — provide stub behaviour.
  console.debug('[wails-stub]', method, args);
  if (method === 'EventsOn') return noop;
  return Promise.resolve(undefined);
}

export const EventsOn    = (n, cb) => _wails('EventsOn', n, cb);
export const EventsOff   = (...args) => _wails('EventsOff', ...args);
export const EventsOnce  = (n, cb) => _wails('EventsOnce', n, cb);
export const EventsEmit  = (n, ...d) => _wails('EventsEmit', n, ...d);

export const LogPrint    = (m) => _wails('LogPrint', m);
export const LogTrace    = (m) => _wails('LogTrace', m);
export const LogDebug    = (m) => _wails('LogDebug', m);
export const LogInfo     = (m) => _wails('LogInfo', m);
export const LogWarning  = (m) => _wails('LogWarning', m);
export const LogError    = (m) => _wails('LogError', m);
export const LogFatal    = (m) => _wails('LogFatal', m);

export const WindowReload              = () => _wails('WindowReload');
export const WindowReloadApp           = () => _wails('WindowReloadApp');
export const WindowCenter              = () => _wails('WindowCenter');
export const WindowSetTitle            = (t) => _wails('WindowSetTitle', t);
export const WindowFullscreen          = () => _wails('WindowFullscreen');
export const WindowUnfullscreen        = () => _wails('WindowUnfullscreen');
export const WindowSetSize             = (w, h) => _wails('WindowSetSize', w, h);
export const WindowGetSize             = () => _wails('WindowGetSize');
export const WindowMinimise            = () => _wails('WindowMinimise');
export const WindowUnminimise          = () => _wails('WindowUnminimise');
export const WindowMaximise            = () => _wails('WindowMaximise');
export const WindowUnmaximise          = () => _wails('WindowUnmaximise');
export const WindowToggleMaximise      = () => _wails('WindowToggleMaximise');
export const WindowShow                = () => _wails('WindowShow');
export const WindowHide                = () => _wails('WindowHide');
export const WindowIsMaximised         = () => _wails('WindowIsMaximised');
export const WindowIsMinimised         = () => _wails('WindowIsMinimised');
export const WindowIsNormal            = () => _wails('WindowIsNormal');
export const WindowIsFullscreen        = () => _wails('WindowIsFullscreen');
export const WindowSetBackgroundColour = (r, g, b, a) => _wails('WindowSetBackgroundColour', r, g, b, a);
export const WindowSetAlwaysOnTop      = (b) => _wails('WindowSetAlwaysOnTop', b);
export const WindowSetPosition         = (x, y) => _wails('WindowSetPosition', x, y);
export const WindowGetPosition         = () => _wails('WindowGetPosition');
export const WindowSetMinSize          = (w, h) => _wails('WindowSetMinSize', w, h);
export const WindowSetMaxSize          = (w, h) => _wails('WindowSetMaxSize', w, h);

export const Quit        = () => _wails('Quit');
export const Hide        = () => _wails('Hide');
export const Show        = () => _wails('Show');

export const BrowserOpenURL = (url) => _wails('BrowserOpenURL', url);
export const Environment    = () => _wails('Environment');

export const IsFullscreen = () => _wails('IsFullscreen');
export const IsMaximised  = () => _wails('IsMaximised');
export const IsMinimised  = () => _wails('IsMinimised');
export const IsNormal     = () => _wails('IsNormal');
