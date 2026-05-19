<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import {
    gateways,
    handovers,
    lastError,
    selectedGateway,
    sessions,
    triggerHandoverForSession,
  } from '$lib/stores';
  import type { Gateway, HandoverEvent, Session, SessionState } from '$lib/types';

  let sessionFilter = 'all';
  let stateFilter: SessionState | 'all' = 'all';
  let sessionSearch = '';
  let hoSearch = '';
  let page = 0;
  let actionMessage = '';
  let actionError = '';
  const PAGE_SIZE = 20;
  const handoverTargets: Record<string, string> = {};

  $: filteredSessions = $sessions.filter((s) => {
    const gatewayMatch = sessionFilter === 'all' || s.currentGatewayId === sessionFilter;
    const stateMatch = stateFilter === 'all' || s.state === stateFilter;
    const search = sessionSearch.trim().toLowerCase();
    const searchMatch = !search ||
      s.id.toLowerCase().includes(search) ||
      s.routerMac.toLowerCase().includes(search) ||
      s.satelliteId.toLowerCase().includes(search);
    return gatewayMatch && stateMatch && searchMatch;
  });

  $: pagedSessions = filteredSessions.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);
  $: totalPages = Math.max(1, Math.ceil(filteredSessions.length / PAGE_SIZE));
  $: filteredHandovers = $handovers.filter((h) => {
    const search = hoSearch.trim().toLowerCase();
    return !search ||
      h.sessionId.toLowerCase().includes(search) ||
      h.fromGateway.toLowerCase().includes(search) ||
      h.toGateway.toLowerCase().includes(search);
  });

  function gatewayName(id: string) {
    return $gateways.find((g) => g.id === id)?.name ?? id;
  }

  function gatewayStatusClass(status: Gateway['status']) {
    if (status === 'alive') return 'badge-online';
    if (status === 'degraded') return 'badge-warning';
    return 'badge-offline';
  }

  function stateBadge(state: SessionState) {
    if (state === 'Hold') return 'badge-online';
    if (state === 'Prepare' || state === 'Execute') return 'badge-handover';
    if (state === 'Release') return 'badge-offline';
    return 'badge-warning';
  }

  function formatDuration(startTime: Date) {
    const seconds = Math.max(0, Math.floor((Date.now() - startTime.getTime()) / 1000));
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    const rest = seconds % 60;
    if (minutes < 60) return `${minutes}m ${String(rest).padStart(2, '0')}s`;
    return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
  }

  function formatTs(d: Date) {
    return `${d.toLocaleDateString('vi-VN')} ${d.toLocaleTimeString('en-GB', { hour12: false })}`;
  }

  function availableTargets(session: Session) {
    return $gateways.filter((g) => g.id !== session.currentGatewayId && g.status === 'alive');
  }

  function selectedTarget(session: Session) {
    return handoverTargets[session.id] || availableTargets(session)[0]?.id || '';
  }

  async function triggerSessionHandover(session: Session) {
    const target = selectedTarget(session);
    if (!target) return;

    actionMessage = '';
    actionError = '';
    try {
      const result = await triggerHandoverForSession(session.id, target);
      actionMessage = `Handover ${result.handoverId} accepted for ${session.id}`;
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Handover trigger failed';
    }
  }
</script>

<Topbar title="MONITORING" subtitle="Gateway state, sessions, handovers" />

<main class="flex-1 overflow-y-auto p-6 space-y-6">
  {#if $lastError}
    <div class="rounded border border-rose-400/30 bg-rose-400/8 px-4 py-3 font-mono text-xs text-rose-300">
      API refresh failed: {$lastError}
    </div>
  {/if}
  {#if actionMessage}
    <div class="rounded border border-emerald-400/30 bg-emerald-400/8 px-4 py-3 font-mono text-xs text-emerald-300">
      {actionMessage}
    </div>
  {/if}
  {#if actionError}
    <div class="rounded border border-rose-400/30 bg-rose-400/8 px-4 py-3 font-mono text-xs text-rose-300">
      {actionError}
    </div>
  {/if}

  <section class="grid grid-cols-1 xl:grid-cols-3 gap-3">
    {#each $gateways as gw (gw.id)}
      <button
        type="button"
        class="stat-card text-left cursor-pointer"
        on:click={() => selectedGateway.set(gw.id === $selectedGateway ? null : gw.id)}
      >
        <div class="flex items-center justify-between gap-3 mb-3">
          <div class="flex items-center gap-2 min-w-0">
            <span class="pulse-dot {gw.status === 'alive' ? 'online' : gw.status === 'degraded' ? 'warning' : 'offline'}"></span>
            <span class="font-display text-xs text-slate-300 truncate">{gw.name}</span>
          </div>
          <span class="{gatewayStatusClass(gw.status)}">{gw.status}</span>
        </div>
        <div class="grid grid-cols-2 gap-2 text-xs font-mono">
          <div><span class="text-slate-600">SESS </span><span class="text-cyan-400">{gw.currentSessions}</span></div>
          <div><span class="text-slate-600">MAX </span><span class="text-slate-300">{gw.maxSessions.toLocaleString()}</span></div>
          <div><span class="text-slate-600">MIN EL </span><span class="text-emerald-400">{gw.minElevationDeg.toFixed(1)}</span></div>
          <div><span class="text-slate-600">GAIN </span><span class="text-amber-400">{gw.antennaGainDbi.toFixed(1)}</span></div>
        </div>
        {#if $selectedGateway === gw.id}
          <div class="absolute inset-0 rounded-lg border border-cyan-400/40 pointer-events-none"></div>
        {/if}
      </button>
    {/each}
  </section>

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="font-mono text-xs text-slate-500 tracking-widest">SESSIONS ({filteredSessions.length})</div>
      <div class="flex gap-2 flex-wrap">
        <input
          bind:value={sessionSearch}
          placeholder="Search MAC / session..."
          class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-52"
        />
        <select bind:value={sessionFilter} class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-300 focus:outline-none focus:border-cyan-400/40">
          <option value="all">All gateways</option>
          {#each $gateways as gw}
            <option value={gw.id}>{gw.name}</option>
          {/each}
        </select>
        <select bind:value={stateFilter} class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-300 focus:outline-none focus:border-cyan-400/40">
          <option value="all">All states</option>
          {#each ['Acquire','Hold','Prepare','Execute','Release'] as state}
            <option value={state}>{state}</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Session ID</th>
            <th>Router MAC</th>
            <th>Gateway</th>
            <th>Satellite</th>
            <th>Duration</th>
            <th>State</th>
            <th>Manual Handover</th>
          </tr>
        </thead>
        <tbody>
          {#each pagedSessions as s (s.id)}
            <tr class="animate-fade-in">
              <td class="text-cyan-400">{s.id}</td>
              <td class="text-slate-400">{s.routerMac}</td>
              <td class="text-slate-400">{gatewayName(s.currentGatewayId)}</td>
              <td class="text-slate-500">{s.satelliteId}</td>
              <td class="text-slate-400">{formatDuration(s.startTime)}</td>
              <td><span class="{stateBadge(s.state)}">{s.state}</span></td>
              <td>
                <div class="flex gap-2">
                  <select bind:value={handoverTargets[s.id]} class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs text-slate-300">
                    {#each availableTargets(s) as gw}
                      <option value={gw.id}>{gw.name}</option>
                    {/each}
                  </select>
                  <button
                    type="button"
                    on:click={() => triggerSessionHandover(s)}
                    disabled={availableTargets(s).length === 0 || s.state !== 'Hold'}
                    class="px-2 py-1 font-mono text-xs text-cyan-400 border border-cyan-400/20 rounded hover:bg-cyan-400/10 transition-colors disabled:opacity-30 disabled:hover:bg-transparent"
                  >
                    TRIGGER
                  </button>
                </div>
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="7" class="text-center text-slate-600 py-8">No sessions returned from API</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="flex items-center justify-between mt-3">
      <span class="font-mono text-xs text-slate-600">
        {filteredSessions.length === 0 ? 0 : page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, filteredSessions.length)} of {filteredSessions.length}
      </span>
      <div class="flex gap-1">
        <button type="button" on:click={() => page = Math.max(0, page - 1)} disabled={page === 0} class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors">PREV</button>
        <button type="button" on:click={() => page = Math.min(totalPages - 1, page + 1)} disabled={page >= totalPages - 1} class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors">NEXT</button>
      </div>
    </div>
  </section>

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3">
      <div class="font-mono text-xs text-slate-500 tracking-widest">HANDOVERS ({filteredHandovers.length})</div>
      <input
        bind:value={hoSearch}
        placeholder="Search session / gateway..."
        class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-56"
      />
    </div>

    <div class="overflow-x-auto max-h-80 overflow-y-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Session</th>
            <th>From</th>
            <th>To</th>
            <th>Duration</th>
            <th>Packet Loss</th>
            <th>Result</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredHandovers as h (h.id)}
            <tr class="animate-fade-in">
              <td class="text-slate-500 whitespace-nowrap">{formatTs(h.timestamp)}</td>
              <td class="text-cyan-400">{h.sessionId}</td>
              <td class="text-slate-400">{gatewayName(h.fromGateway)}</td>
              <td class="text-slate-400">{gatewayName(h.toGateway)}</td>
              <td class="{h.durationMs > 100 ? 'text-amber-400' : 'text-emerald-400'}">{h.durationMs}ms</td>
              <td class="{h.packetLoss > 0.001 ? 'text-rose-400' : 'text-emerald-400'}">{(h.packetLoss * 100).toFixed(3)}%</td>
              <td><span class="{h.success ? 'badge-online' : 'badge-offline'}">{h.success ? 'OK' : 'FAIL'}</span></td>
            </tr>
          {:else}
            <tr>
              <td colspan="7" class="text-center text-slate-600 py-8">No handovers recorded</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>
</main>
