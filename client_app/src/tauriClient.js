const isTauri = typeof window !== 'undefined' && Boolean(window.__TAURI_INTERNALS__);

export async function invokeCommand(command, args = {}) {
  if (!isTauri) return null;
  const { invoke } = await import('@tauri-apps/api/core');
  return invoke(command, args);
}

export async function listenEvent(eventName, handler) {
  if (!isTauri) return () => {};
  const { listen } = await import('@tauri-apps/api/event');
  return listen(eventName, (event) => handler(event.payload));
}

export { isTauri };
