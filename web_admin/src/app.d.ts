/// <reference types="svelte" />
/// <reference types="vite/client" />

declare global {
  namespace App {
    interface Locals {
      requestId: string;
      authToken: string | null;
    }
  }

  interface ImportMetaEnv {
    readonly VITE_CORE_API_BASE?: string;
    readonly VITE_TELEMETRY_TOKEN?: string; // trigger telemetry WebSocket connection if set
    readonly VITE_ENABLE_DEVICES?: string; // 'true' to enable device management features
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

export {};
