import type { Handle, HandleFetch } from '@sveltejs/kit';

const CORE_API_TOKEN = process.env.CORE_API_TOKEN;

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.requestId = crypto.randomUUID();
  event.locals.authToken = CORE_API_TOKEN ?? null;
  return resolve(event);
};

export const handleFetch: HandleFetch = async ({ request, fetch, event }) => {
  const url = new URL(request.url);
  if (url.pathname.startsWith('/api/')) {
    const headers = new Headers(request.headers);
    if (event.locals.authToken) {
      headers.set('Authorization', `Bearer ${event.locals.authToken}`);
    }
    headers.set('X-Request-Id', event.locals.requestId);
    request = new Request(request, { headers });
  }
  return fetch(request);
};

declare global {
  namespace App {
    interface Locals {
      requestId: string;
      authToken: string | null;
    }
  }
}
