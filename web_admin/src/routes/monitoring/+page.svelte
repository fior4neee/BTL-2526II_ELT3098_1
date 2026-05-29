<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import ChartPanel from '$lib/components/ChartPanel.svelte';
  import {
    gateways,
    handovers,
    lastError,
    selectedGateway,
    series,
    sessions,
    triggerHandoverForSession,
  } from '$lib/stores';
  import type { Gateway, HandoverEvent, Session, SessionState } from '$lib/types';
  import type { ChartConfiguration } from 'chart.js/auto';

  let sessionFilter = 'all';
  let stateFilter: SessionState | 'all' = 'all';
  let sessionSearch = '';
  let hoSearch = '';
  let page = 0;
  let actionMessage = '';
  let actionError = '';
  const PAGE_SIZE = 15;
  const handoverTargets: Record<string, string> = {};

  let gatewaySortKey: keyof Gateway = 'name';
  let gatewaySortDir: 'asc' | 'desc' = 'asc';
  let sessionSortKey: keyof Session = 'startTime';
  let sessionSortDir: 'asc' | 'desc' = 'desc';

  $: filteredSessions = $sessions.filter((s) => {
    const selectedId = $selectedGateway;
    const gatewayMatch = selectedId
      ? s.currentGatewayId === selectedId
      : sessionFilter === 'all' || s.currentGatewayId === sessionFilter;
    const stateMatch = stateFilter === 'all' || s.state === stateFilter;
    const search = sessionSearch.trim().toLowerCase();
    const searchMatch =
      !search ||
      s.id.toLowerCase().includes(search) ||
      s.routerMac.toLowerCase().includes(search) ||
      s.satelliteId.toLowerCase().includes(search);
    return gatewayMatch && stateMatch && searchMatch;
  });

  $: sortedSessions = [...filteredSessions].sort((a, b) => {
    const dir = sessionSortDir === 'asc' ? 1 : -1;
    const av = a[sessionSortKey];
    const bv = b[sessionSortKey];
    if (av instanceof Date && bv instanceof Date) return (av.getTime() - bv.getTime()) * dir;
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir;
    return String(av ?? '').localeCompare(String(bv ?? '')) * dir;
  });

  $: pagedSessions = sortedSessions.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);
  $: totalPages = Math.max(1, Math.ceil(sortedSessions.length / PAGE_SIZE));
  $: filteredHandovers = $handovers.filter((h) => {
    const search = hoSearch.trim().toLowerCase();
    return (
      !search ||
      h.sessionId.toLowerCase().includes(search) ||
      h.fromGateway.toLowerCase().includes(search) ||
      h.toGateway.toLowerCase().includes(search)
    );
  });

  $: sortedGateways = [...$gateways].sort((a, b) => {
    const dir = gatewaySortDir === 'asc' ? 1 : -1;
    const av = a[gatewaySortKey];
    const bv = b[gatewaySortKey];
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir;
    return String(av ?? '').localeCompare(String(bv ?? '')) * dir;
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

  function toggleSort(key: keyof Gateway) {
    if (gatewaySortKey === key) {
      gatewaySortDir = gatewaySortDir === 'asc' ? 'desc' : 'asc';
    } else {
      gatewaySortKey = key;
      gatewaySortDir = 'asc';
    }
  }

  function toggleSessionSort(key: keyof Session) {
    if (sessionSortKey === key) {
      sessionSortDir = sessionSortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sessionSortKey = key;
      sessionSortDir = 'desc';
    }
  }

  $: trafficChart = buildChart('Traffic Volume (Gbps)', $series.trafficGbps, '#22d3ee', 'rgba(34,211,238,0.15)', 'line');
  $: handoverChart = buildChart('Handover Frequency', $series.handoverCount, '#fbbf24', 'rgba(251,191,36,0.15)', 'bar', true);
  $: sessionChart = buildChart('Active Sessions', $series.sessionCount, '#34d399', 'rgba(52,211,153,0.15)', 'line', true);

  function buildChart(
    label: string,
    points: { ts: Date; value: number }[],
    border: string,
    fill: string,
    type: 'line' | 'bar',
    forceIntY = false
  ): ChartConfiguration {
    return {
      type,
      data: {
        labels: points.map((p) => p.ts.toLocaleTimeString('en-GB', { hour12: false })),
        datasets: [
          {
            label,
            data: points.map((p) => p.value),
            borderColor: border,
            backgroundColor: fill,
            fill: type === 'line',
            tension: 0.35,
            borderWidth: 2,
            pointRadius: 0,
          },
        ],
      },
      options: {
        responsive: true,
        plugins: {
          legend: { display: false },
        },
        scales: {
          x: {
            ticks: { color: '#64748b', maxTicksLimit: 6 },
            grid: { color: 'rgba(34,211,238,0.04)' },
          },
          y: {
            beginAtZero: true,
            ticks: {
              color: '#64748b',
              ...(forceIntY ? { stepSize: 1, precision: 0 } : {})
            },
            grid: { color: 'rgba(34,211,238,0.04)' },
          },
        },
      },
    };
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

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="font-mono text-xs text-slate-500 tracking-widest">GATEWAY STATUS ({sortedGateways.length})</div>
    </div>
    <div class="overflow-x-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSort('name')}>Gateway</button></th>
            <th>Location</th>
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSort('status')}>Status</button></th>
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSort('currentSessions')}>Active Sessions</button></th>
            <th>CPU%</th>
            <th>Memory%</th>
            <th>Bandwidth</th>
          </tr>
        </thead>
        <tbody>
          {#each sortedGateways as gw (gw.id)}
            <tr class="animate-fade-in cursor-pointer" on:click={() => selectedGateway.set(gw.id === $selectedGateway ? null : gw.id)}>
              <td class="text-cyan-400">{gw.name}</td>
              <td class="text-slate-500">{gw.lat.toFixed(2)}, {gw.lng.toFixed(2)}</td>
              <td><span class="{gatewayStatusClass(gw.status)}">{gw.status}</span></td>
              <td class="text-slate-300">{gw.currentSessions}</td>
              <td class="text-slate-400">{gw.cpuPct?.toFixed(1) ?? '-'}</td>
              <td class="text-slate-400">{gw.memoryPct?.toFixed(1) ?? '-'}</td>
              <td class="text-slate-400">{gw.bandwidthMbps ? `${gw.bandwidthMbps.toFixed(1)} Mbps` : '-'}</td>
            </tr>
          {:else}
            <tr>
              <td colspan="7" class="text-center text-slate-600 py-8">No gateways returned from API</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  <section class="grid grid-cols-1 xl:grid-cols-3 gap-3">
    <ChartPanel title="TRAFFIC VOLUME" config={trafficChart} />
    <ChartPanel title="HANDOVER FREQUENCY" config={handoverChart} />
    <ChartPanel title="ACTIVE SESSIONS" config={sessionChart} />
  </section>

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="font-mono text-xs text-slate-500 tracking-widest">SESSIONS ({sortedSessions.length})</div>
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
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSessionSort('id')}>Session ID</button></th>
            <th>Router MAC</th>
            <th>Gateway</th>
            <th>Satellite</th>
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSessionSort('startTime')}>Duration</button></th>
            <th>Data</th>
            <th>C/N</th>
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
              <td class="text-slate-400">{s.dataMb ? `${s.dataMb.toFixed(1)} MB` : '-'}</td>
              <td class="text-slate-400">{s.cnRatioDb ? `${s.cnRatioDb.toFixed(1)} dB` : '-'}</td>
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
              <td colspan="9" class="text-center text-slate-600 py-8">No sessions returned from API</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="flex items-center justify-between mt-3">
      <span class="font-mono text-xs text-slate-600">
        {sortedSessions.length === 0 ? 0 : page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, sortedSessions.length)} of {sortedSessions.length}
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
