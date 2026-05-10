<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import StatCard from '$lib/components/StatCard.svelte';
  import ConstellationMap from '$lib/components/ConstellationMap.svelte';
  import AlertPanel from '$lib/components/AlertPanel.svelte';
  import { gateways, networkStats } from '$lib/stores';

  function fmtDuration(s: number): string {
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${m}m ${sec}s`;
  }
</script>

<Topbar title="OVERVIEW" subtitle="VNU-LEO Satellite Constellation — ISP Control Center" />

<main class="flex-1 overflow-y-auto p-6">
  <!-- Stat Cards Row -->
  <div class="grid grid-cols-2 xl:grid-cols-4 gap-3 mb-6">
    <StatCard
      label="Active Sessions"
      value={$networkStats.activeSessions}
      accent="cyan"
      sublabel="{$networkStats.gatewaysOnline}/{$networkStats.gatewaysTotal} gateways online"
    />
    <StatCard
      label="Traffic Volume"
      value={$networkStats.totalTrafficGbps}
      unit="Gbps"
      accent="emerald"
      sublabel="current throughput"
    />
    <StatCard
      label="Avg Handover"
      value={$networkStats.avgHandoverMs}
      unit="ms"
      accent="amber"
      sublabel="target &lt;100ms"
    />
    <StatCard
      label="Packet Loss"
      value={($networkStats.avgPacketLoss * 100).toFixed(3)}
      unit="%"
      accent="rose"
      sublabel="target &lt;0.1%"
    />
  </div>

  <!-- Second row -->
  <div class="grid grid-cols-2 xl:grid-cols-4 gap-3 mb-6">
    <StatCard
      label="Handover Rate"
      value={$networkStats.handoverRate}
      unit="/min"
      accent="cyan"
      sublabel="events per minute"
    />
    <StatCard
      label="Active Satellites"
      value={$networkStats.activeSatellites}
      accent="emerald"
      sublabel="in Vietnam corridor"
    />
    <StatCard
      label="Gateway Uptime"
      value={($gateways.reduce((s,g) => s + g.uptime, 0) / $gateways.length).toFixed(2)}
      unit="%"
      accent="amber"
      sublabel="3-gateway average"
    />
    <StatCard
      label="Data Today"
      value={(Math.random() * 200 + 100).toFixed(0)}
      unit="GB"
      accent="cyan"
      sublabel="total transferred"
    />
  </div>

  <!-- Main Content Grid -->
  <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
    <!-- Gateway Status Cards -->
    <div class="xl:col-span-2 space-y-3">
      <div class="font-mono text-xs text-slate-500 tracking-widest mb-1">GATEWAY STATUS</div>
      {#each $gateways as gw (gw.id)}
        <div class="stat-card animate-fade-in">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <span class="pulse-dot {gw.status === 'online' ? 'online' : gw.status === 'degraded' ? 'warning' : 'offline'}"></span>
              <div>
                <div class="font-display text-sm text-slate-200">{gw.name}</div>
                <div class="font-mono text-xs text-slate-600">{gw.location} · {gw.antennaModel}</div>
              </div>
            </div>
            <div class="text-right">
              <div class="font-mono text-xs {gw.status === 'online' ? 'text-emerald-400' : gw.status === 'degraded' ? 'text-amber-400' : 'text-rose-400'} uppercase">
                {gw.status}
              </div>
              <div class="font-mono text-xs text-slate-600 mt-0.5">el. {gw.elevation}°</div>
            </div>
          </div>

          <!-- Metrics row -->
          <div class="grid grid-cols-4 gap-3">
            <div>
              <div class="font-mono text-xs text-slate-600">SESSIONS</div>
              <div class="font-mono text-sm text-cyan-400">{gw.activeSessions}</div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">CPU</div>
              <div class="font-mono text-sm {gw.cpu > 70 ? 'text-rose-400' : gw.cpu > 50 ? 'text-amber-400' : 'text-emerald-400'}">
                {gw.cpu.toFixed(0)}%
              </div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">MEM</div>
              <div class="font-mono text-sm {gw.memory > 80 ? 'text-rose-400' : gw.memory > 60 ? 'text-amber-400' : 'text-emerald-400'}">
                {gw.memory.toFixed(0)}%
              </div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">BW</div>
              <div class="font-mono text-sm text-cyan-400">{gw.bwUsage} Gbps</div>
            </div>
          </div>

          <!-- CPU Bar -->
          <div class="mt-3">
            <div class="h-1 bg-slate-800 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all duration-500 {gw.cpu > 70 ? 'bg-rose-500' : gw.cpu > 50 ? 'bg-amber-400' : 'bg-emerald-400'}"
                style="width: {gw.cpu}%"
              ></div>
            </div>
          </div>
        </div>
      {/each}

      <!-- Link to Monitoring -->
      <a href="/monitoring" class="flex items-center justify-center gap-2 p-3 rounded border border-dashed border-slate-700/50 hover:border-cyan-400/30 transition-colors text-slate-500 hover:text-cyan-400">
        <span class="font-mono text-xs">→ DETAILED MONITORING</span>
      </a>
    </div>

    <!-- Right column: Map + Alerts -->
    <div class="space-y-4">
      <ConstellationMap />
      <AlertPanel />
    </div>
  </div>
</main>
