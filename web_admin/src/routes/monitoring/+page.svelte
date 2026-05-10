<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Topbar from '$lib/components/Topbar.svelte';
  import StatCard from '$lib/components/StatCard.svelte';
  import { gateways, sessions, handoverHistory, trafficHistory, sessionCountHistory, handoverFreqHistory, cnHistory, selectedGateway } from '$lib/stores';
  import type { Session, HandoverEvent } from '$lib/types';

  // ─── Chart.js (lazy imported to avoid SSR issues) ─────────────────
  let Chart: any;
  let trafficChart: any, sessionChart: any, cnChart: any, hoFreqChart: any;
  let trafficCanvas: HTMLCanvasElement, sessionCanvas: HTMLCanvasElement;
  let cnCanvas: HTMLCanvasElement, hoFreqCanvas: HTMLCanvasElement;

  // ─── Filters ──────────────────────────────────────────────────────
  let sessionFilter = 'all'; // 'all' | gateway id
  let sessionSearch = '';
  let hoSearch = '';
  let refreshRate = 5; // seconds
  let page = 0;
  const PAGE_SIZE = 20;

  $: filteredSessions = $sessions.filter(s => {
    const matchGw = sessionFilter === 'all' || s.gatewayId === sessionFilter;
    const matchSearch = !sessionSearch ||
      s.id.toLowerCase().includes(sessionSearch.toLowerCase()) ||
      s.deviceMac.toLowerCase().includes(sessionSearch.toLowerCase());
    return matchGw && matchSearch;
  });

  $: pagedSessions = filteredSessions.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);
  $: totalPages = Math.ceil(filteredSessions.length / PAGE_SIZE);

  $: filteredHandovers = $handoverHistory.filter(h =>
    !hoSearch || h.sessionId.toLowerCase().includes(hoSearch.toLowerCase()) ||
    h.fromGateway.toLowerCase().includes(hoSearch.toLowerCase()) ||
    h.toGateway.toLowerCase().includes(hoSearch.toLowerCase())
  );

  function formatDuration(s: number): string {
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    const r = s % 60;
    return `${m}m ${String(r).padStart(2,'0')}s`;
  }

  function formatTime(d: Date): string {
    return new Date(d).toLocaleTimeString('en-GB', { hour12: false });
  }

  function formatTs(d: Date): string {
    const dt = new Date(d);
    return `${dt.toLocaleDateString('vi-VN')} ${dt.toLocaleTimeString('en-GB', { hour12: false })}`;
  }

  const CHART_DEFAULTS = {
    borderColor: 'rgba(34,211,238,0.8)',
    backgroundColor: 'rgba(34,211,238,0.08)',
    tension: 0.4,
    pointRadius: 0,
    fill: true,
  };

  function buildLabels(points: {t: Date}[]) {
    return points.map(p => formatTime(new Date(p.t)));
  }

  async function initCharts() {
    const mod = await import('chart.js');
    const { Chart: C, registerables } = mod;
    C.register(...registerables);
    Chart = C;

    const gridColor = 'rgba(34,211,238,0.05)';
    const tickColor = 'rgba(148,163,184,0.5)';
    const baseOptions = {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 0 },
      plugins: { legend: { display: false }, tooltip: {
        backgroundColor: '#0a1628',
        borderColor: 'rgba(34,211,238,0.3)',
        borderWidth: 1,
        titleColor: '#22d3ee',
        bodyColor: '#94a3b8',
        titleFont: { family: 'JetBrains Mono', size: 10 },
        bodyFont: { family: 'JetBrains Mono', size: 10 },
      }},
      scales: {
        x: { grid: { color: gridColor }, ticks: { color: tickColor, font: { family: 'JetBrains Mono', size: 9 }, maxTicksLimit: 8 } },
        y: { grid: { color: gridColor }, ticks: { color: tickColor, font: { family: 'JetBrains Mono', size: 9 } } },
      },
    };

    trafficChart = new Chart(trafficCanvas, {
      type: 'line',
      data: {
        labels: buildLabels($trafficHistory),
        datasets: [{ ...CHART_DEFAULTS, label: 'Traffic (Gbps)', data: $trafficHistory.map(p => p.value) }],
      },
      options: { ...baseOptions },
    });

    sessionChart = new Chart(sessionCanvas, {
      type: 'line',
      data: {
        labels: buildLabels($sessionCountHistory),
        datasets: [{ ...CHART_DEFAULTS, borderColor: 'rgba(52,211,153,0.8)', backgroundColor: 'rgba(52,211,153,0.08)', label: 'Sessions', data: $sessionCountHistory.map(p => p.value) }],
      },
      options: { ...baseOptions },
    });

    const cnData = $cnHistory;
    cnChart = new Chart(cnCanvas, {
      type: 'line',
      data: {
        labels: buildLabels(cnData['Hanoi GW']),
        datasets: [
          { ...CHART_DEFAULTS, borderColor: 'rgba(34,211,238,0.8)', backgroundColor: 'transparent', fill: false, label: 'Hanoi', data: cnData['Hanoi GW'].map(p => p.value) },
          { ...CHART_DEFAULTS, borderColor: 'rgba(251,191,36,0.8)', backgroundColor: 'transparent', fill: false, label: 'Danang', data: cnData['Danang GW'].map(p => p.value) },
          { ...CHART_DEFAULTS, borderColor: 'rgba(244,63,94,0.8)', backgroundColor: 'transparent', fill: false, label: 'HCMC', data: cnData['HCMC GW'].map(p => p.value) },
        ],
      },
      options: {
        ...baseOptions,
        plugins: {
          ...baseOptions.plugins,
          legend: { display: true, labels: { color: '#94a3b8', font: { family: 'JetBrains Mono', size: 9 }, boxWidth: 12 } },
        },
      },
    });

    hoFreqChart = new Chart(hoFreqCanvas, {
      type: 'bar',
      data: {
        labels: buildLabels($handoverFreqHistory),
        datasets: [{
          label: 'Handovers/hr',
          data: $handoverFreqHistory.map(p => p.value),
          backgroundColor: 'rgba(251,191,36,0.25)',
          borderColor: 'rgba(251,191,36,0.7)',
          borderWidth: 1,
        }],
      },
      options: { ...baseOptions },
    });
  }

  // Reactive chart updates
  let updateTimer: ReturnType<typeof setInterval>;
  onMount(async () => {
    await initCharts();
    updateTimer = setInterval(() => {
      if (!trafficChart) return;
      trafficChart.data.labels = buildLabels($trafficHistory);
      trafficChart.data.datasets[0].data = $trafficHistory.map(p => p.value);
      trafficChart.update('none');

      sessionChart.data.labels = buildLabels($sessionCountHistory);
      sessionChart.data.datasets[0].data = $sessionCountHistory.map(p => p.value);
      sessionChart.update('none');
    }, refreshRate * 1000);
  });

  onDestroy(() => {
    clearInterval(updateTimer);
    [trafficChart, sessionChart, cnChart, hoFreqChart].forEach(c => c?.destroy());
  });

  function signalClass(cn: number): string {
    if (cn >= 22) return 'text-emerald-400';
    if (cn >= 16) return 'text-amber-400';
    return 'text-rose-400';
  }
</script>

<Topbar title="MONITORING" subtitle="Real-time gateway telemetry · session tracking · handover analysis" />

<main class="flex-1 overflow-y-auto p-6 space-y-6">

  <!-- Quick Stats -->
  <div class="grid grid-cols-2 xl:grid-cols-4 gap-3">
    {#each $gateways as gw}
      <div class="stat-card cursor-pointer" on:click={() => selectedGateway.set(gw.id === $selectedGateway ? null : gw.id)}>
        <div class="flex items-center gap-2 mb-2">
          <span class="pulse-dot {gw.status === 'online' ? 'online' : gw.status === 'degraded' ? 'warning' : 'offline'}"></span>
          <span class="font-display text-xs text-slate-300">{gw.name}</span>
        </div>
        <div class="grid grid-cols-2 gap-2 text-xs font-mono">
          <div><span class="text-slate-600">CPU </span><span class="{gw.cpu>70?'text-rose-400':gw.cpu>50?'text-amber-400':'text-emerald-400'}">{gw.cpu.toFixed(0)}%</span></div>
          <div><span class="text-slate-600">MEM </span><span class="{gw.memory>80?'text-rose-400':gw.memory>60?'text-amber-400':'text-emerald-400'}">{gw.memory.toFixed(0)}%</span></div>
          <div><span class="text-slate-600">SESS </span><span class="text-cyan-400">{gw.activeSessions}</span></div>
          <div><span class="text-slate-600">BW </span><span class="text-cyan-400">{gw.bwUsage}G</span></div>
        </div>
        <div class="mt-2 h-1 bg-slate-800 rounded-full">
          <div class="h-full rounded-full transition-all duration-500 {gw.cpu>70?'bg-rose-500':gw.cpu>50?'bg-amber-400':'bg-emerald-400'}" style="width:{gw.cpu}%"></div>
        </div>
        {#if $selectedGateway === gw.id}
          <div class="absolute inset-0 rounded-lg border border-cyan-400/40 pointer-events-none"></div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Charts Grid -->
  <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-2 tracking-widest">TRAFFIC VOLUME — 24H (Gbps)</div>
      <div class="h-40"><canvas bind:this={trafficCanvas}></canvas></div>
    </div>
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-2 tracking-widest">ACTIVE SESSIONS — 24H</div>
      <div class="h-40"><canvas bind:this={sessionCanvas}></canvas></div>
    </div>
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-2 tracking-widest">C/N RATIO BY GATEWAY — 24H (dB)</div>
      <div class="h-40"><canvas bind:this={cnCanvas}></canvas></div>
    </div>
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-2 tracking-widest">HANDOVER FREQUENCY — 24H (events/hr)</div>
      <div class="h-40"><canvas bind:this={hoFreqCanvas}></canvas></div>
    </div>
  </div>

  <!-- Session List -->
  <div class="chart-container">
    <div class="flex items-center justify-between mb-3">
      <div class="font-mono text-xs text-slate-500 tracking-widest">SESSION LIST ({filteredSessions.length})</div>
      <div class="flex gap-2">
        <input
          bind:value={sessionSearch}
          placeholder="Search MAC / session..."
          class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-48"
        />
        <select
          bind:value={sessionFilter}
          class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-300 focus:outline-none focus:border-cyan-400/40"
        >
          <option value="all">All gateways</option>
          {#each $gateways as gw}
            <option value={gw.id}>{gw.name}</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Session ID</th>
            <th>Device MAC</th>
            <th>Gateway</th>
            <th>Duration</th>
            <th>Data (MB)</th>
            <th>C/N (dB)</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {#each pagedSessions as s (s.id)}
            <tr class="animate-fade-in">
              <td class="text-cyan-400">{s.id}</td>
              <td class="text-slate-400">{s.deviceMac}</td>
              <td class="text-slate-400">{s.gatewayName}</td>
              <td class="text-slate-400">{formatDuration(s.duration)}</td>
              <td class="text-slate-400">{s.dataMb.toLocaleString()}</td>
              <td class="{signalClass(s.signalCN)}">{s.signalCN}</td>
              <td>
                {#if s.status === 'connected'}
                  <span class="badge-online">connected</span>
                {:else if s.status === 'handover'}
                  <span class="badge-handover">handover</span>
                {:else}
                  <span class="badge-offline">disconnecting</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between mt-3">
      <span class="font-mono text-xs text-slate-600">
        {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, filteredSessions.length)} of {filteredSessions.length}
      </span>
      <div class="flex gap-1">
        <button
          on:click={() => page = Math.max(0, page - 1)}
          disabled={page === 0}
          class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors"
        >←</button>
        <button
          on:click={() => page = Math.min(totalPages - 1, page + 1)}
          disabled={page >= totalPages - 1}
          class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors"
        >→</button>
      </div>
    </div>
  </div>

  <!-- Handover History -->
  <div class="chart-container">
    <div class="flex items-center justify-between mb-3">
      <div class="font-mono text-xs text-slate-500 tracking-widest">HANDOVER HISTORY (last {filteredHandovers.length})</div>
      <input
        bind:value={hoSearch}
        placeholder="Search session / gateway..."
        class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-52"
      />
    </div>

    <div class="overflow-x-auto max-h-72 overflow-y-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Session</th>
            <th>From</th>
            <th>To</th>
            <th>Duration</th>
            <th>Pkt Loss</th>
            <th>Result</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredHandovers as h (h.id)}
            <tr class="animate-fade-in">
              <td class="text-slate-500 whitespace-nowrap">{formatTs(h.timestamp)}</td>
              <td class="text-cyan-400">{h.sessionId}</td>
              <td class="text-slate-400">{h.fromGateway}</td>
              <td class="text-slate-400">{h.toGateway}</td>
              <td class="{h.durationMs > 80 ? 'text-amber-400' : 'text-emerald-400'}">{h.durationMs}ms</td>
              <td class="{h.packetLoss > 0.05 ? 'text-rose-400' : 'text-emerald-400'}">{(h.packetLoss * 100).toFixed(3)}%</td>
              <td>
                {#if h.success}
                  <span class="badge-online">OK</span>
                {:else}
                  <span class="badge-offline">FAIL</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>

</main>
