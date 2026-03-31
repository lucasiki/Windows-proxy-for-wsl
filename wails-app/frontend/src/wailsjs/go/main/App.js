'use strict';

// Wails Go binding stubs — real implementation injected by Wails webview.
// These stubs allow Vite dev server to load without errors.

const _pkg = 'main';

function call(method, ...args) {
  if (window.go && window.go[_pkg] && typeof window.go[_pkg].App[method] === 'function') {
    return window.go[_pkg].App[method](...args);
  }
  // Fallback stubs for plain-browser development.
  console.debug('[wails-go-stub]', _pkg + '.App.' + method, args);
  if (method === 'GetMappings') return Promise.resolve([]);
  if (method === 'GetState')    return Promise.resolve('stopped');
  if (method === 'GetWSLInfo')  return Promise.resolve(null);
  return Promise.resolve();
}

export const GetWSLInfo    = ()    => call('GetWSLInfo');
export const GetMappings   = ()    => call('GetMappings');
export const GetState      = ()    => call('GetState');
export const AddMapping    = (s)   => call('AddMapping', s);
export const RemoveMapping = (id)  => call('RemoveMapping', id);
export const Start         = ()    => call('Start');
export const Pause         = ()    => call('Pause');
export const Stop          = ()    => call('Stop');
export const ClearLog      = ()    => call('ClearLog');
