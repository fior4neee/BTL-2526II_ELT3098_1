<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { telemetryConnected, telemetryMode } from '$lib/stores';

  export let title: string;
  export let subtitle: string = '';

  let now = new Date();
  let tick: ReturnType<typeof setInterval>;

  onMount(() => {
    tick = setInterval(() => (now = new Date()), 1000);
  });
  onDestroy(() => clearInterval(tick));

  function formatTime(d: Date) {
    return d.toLocaleTimeString('en-GB', { hour12: false });
  }
  function formatDate(d: Date) {
    return d.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
  }
</script>

<header class="h-14 border-b border-slate-800/60 flex items-center px-6 gap-4 shrink-0 bg-space-900/50 backdrop-blur">
  <div class="flex-1">
    <h1 class="font-display text-sm text-slate-200 tracking-wide">{title}</h1>
    {#if subtitle}
      <p class="font-mono text-xs text-slate-500 mt-0.5">{subtitle}</p>
    {/if}
  </div>

  <div class="flex items-center gap-2 px-3 py-1.5 rounded bg-space-800 border border-slate-700/50">
    <span class="pulse-dot {$telemetryConnected ? 'online' : 'warning'}"></span>
    <span class="font-mono text-xs {$telemetryConnected ? 'text-emerald-400' : 'text-amber-400'}">
      {$telemetryMode}
    </span>
  </div>

  <div class="text-right">
    <div class="font-mono text-sm text-cyan-400 tabular-nums">{formatTime(now)}</div>
    <div class="font-mono text-xs text-slate-600 tabular-nums">{formatDate(now)} UTC+7</div>
  </div>

  <div class="flex items-center gap-2 pl-4 border-l border-slate-800">
    <div class="w-7 h-7 rounded bg-cyan-400/10 border border-cyan-400/20 flex items-center justify-center">
      <span class="font-mono text-xs text-cyan-400">SA</span>
    </div>
    <span class="font-sans text-xs text-slate-400">Super Admin</span>
  </div>
</header>
