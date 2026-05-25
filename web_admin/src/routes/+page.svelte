<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import ConstellationMap from '$lib/components/ConstellationMap.svelte';
  import WorldMapModal from '$lib/components/WorldMapModal.svelte';
  import { gateways, handovers, lastError, lastUpdated, metrics, securityAlerts, worldMapOpen } from '$lib/stores';
  import type { Gateway } from '$lib/types';

  function statusClass(status: Gateway['status']) {
    if (status === 'alive') return 'badge-online';
    if (status === 'degraded') return 'badge-warning';
    return 'badge-offline';
  }

  function statusDot(status: Gateway['status']) {
    if (status === 'alive') return 'online';
    if (status === 'degraded') return 'warning';
    return 'offline';
  }

  function capacityPct(gateway: Gateway) {
    if (!gateway.maxSessions) return 0;
    return Math.min(100, (gateway.currentSessions / gateway.maxSessions) * 100);
  }

  function formatPct(value: number) {
    return `${value.toFixed(2)}%`;
  }

  $: criticalAlerts = [
    ...$securityAlerts.slice(0, 3).map((alert) => ({
      title: alert.type.replace('_', ' ').toUpperCase(),
      message: alert.message,
      time: alert.timestamp.toLocaleTimeString('en-GB', { hour12: false }),
      severity: alert.severity,
    })),
    ...$gateways
      .filter((g) => g.status === 'dead')
      .map((g) => ({
        title: 'GATEWAY DOWN',
        message: `${g.name} is offline`,
        time: $lastUpdated ? $lastUpdated.toLocaleTimeString('en-GB', { hour12: false }) : '--:--:--',
        severity: 'high',
      })),
    ...$handovers
      .filter((h) => !h.success)
      .slice(0, 2)
      .map((h) => ({
        title: 'HANDOVER FAIL',
        message: `${h.sessionId} ${h.fromGateway} → ${h.toGateway}`,
        time: h.timestamp.toLocaleTimeString('en-GB', { hour12: false }),
        severity: 'high',
      })),
  ].slice(0, 5);
</script>

<!-- WorldMapModal renders here — at page root, outside ALL layout stacking contexts -->
{#if $worldMapOpen}
  <WorldMapModal onClose={() => worldMapOpen.set(false)} />
{/if}

<Topbar title="OVERVIEW" subtitle="VNU-LEO ISP control center" />

<main class="flex-1 overflow-y-auto p-6 space-y-5">
  {#if $lastError}
    <div class="rounded border border-rose-400/30 bg-rose-400/8 px-4 py-3 font-mono text-xs text-rose-300">
      API refresh failed: {$lastError}
    </div>
  {/if}

  <section class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-600">GATEWAYS ONLINE</div>
      <div class="font-display text-2xl text-emerald-400">{$metrics.gatewaysOnline}</div>
      <div class="font-mono text-xs text-slate-600">OFFLINE {$metrics.gatewaysOffline}</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-600">ACTIVE SESSIONS</div>
      <div class="font-display text-2xl text-cyan-400">{$metrics.activeSessions}</div>
      <div class="font-mono text-xs text-slate-600">SATELLITES {$metrics.activeSatellites}</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-600">TOTAL TRAFFIC</div>
      <div class="font-display text-2xl text-amber-400">{$metrics.totalTrafficGbps} Gbps</div>
      <div class="font-mono text-xs text-slate-600">LAST HOUR</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-600">AVG HANDOVER</div>
      <div class="font-display text-2xl text-slate-200">{$metrics.avgHandoverMs} ms</div>
      <div class="font-mono text-xs text-slate-600">LOSS {formatPct($metrics.packetLossPct)}</div>
    </div>
  </section>

  <div class="grid grid-cols-1 xl:grid-cols-3 gap-4">
    <section class="xl:col-span-2 space-y-3">
      <div class="flex items-center justify-between">
        <div class="font-mono text-xs text-slate-500 tracking-widest">GATEWAYS ({$gateways.length})</div>
        {#if $lastUpdated}
          <div class="font-mono text-xs text-slate-600">UPDATED {$lastUpdated.toLocaleTimeString('en-GB', { hour12: false })}</div>
        {/if}
      </div>

      {#each $gateways as gw (gw.id)}
        <article class="stat-card animate-fade-in">
          <div class="flex items-center justify-between gap-3 mb-3">
            <div class="flex items-center gap-3 min-w-0">
              <span class="pulse-dot {statusDot(gw.status)}"></span>
              <div class="min-w-0">
                <div class="font-display text-sm text-slate-200 truncate">{gw.name}</div>
                <div class="font-mono text-xs text-slate-600">
                  {gw.id} / {gw.lat.toFixed(3)}, {gw.lng.toFixed(3)}
                </div>
              </div>
            </div>
            <span class="{statusClass(gw.status)}">{gw.status}</span>
          </div>

          <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
            <div>
              <div class="font-mono text-xs text-slate-600">SESSIONS</div>
              <div class="font-mono text-sm text-cyan-400">{gw.currentSessions.toLocaleString()} / {gw.maxSessions.toLocaleString()}</div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">MIN ELEVATION</div>
              <div class="font-mono text-sm text-emerald-400">{gw.minElevationDeg.toFixed(1)} deg</div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">ANTENNA GAIN</div>
              <div class="font-mono text-sm text-amber-400">{gw.antennaGainDbi.toFixed(1)} dBi</div>
            </div>
            <div>
              <div class="font-mono text-xs text-slate-600">BEAM WIDTH</div>
              <div class="font-mono text-sm text-slate-300">{gw.beamWidthDeg.toFixed(1)} deg</div>
            </div>
          </div>

          <div class="mt-3 h-1 bg-slate-800 rounded-full overflow-hidden">
            <div class="h-full rounded-full bg-cyan-400 transition-all duration-500" style="width: {capacityPct(gw)}%"></div>
          </div>
        </article>
      {:else}
        <div class="chart-container py-10 text-center">
          <div class="font-mono text-xs text-slate-600">NO GATEWAYS RETURNED FROM API</div>
        </div>
      {/each}

      <a href="/monitoring" class="flex items-center justify-center gap-2 p-3 rounded border border-dashed border-slate-700/50 hover:border-cyan-400/30 transition-colors text-slate-500 hover:text-cyan-400">
        <span class="font-mono text-xs">OPEN MONITORING</span>
      </a>
    </section>

    <section class="space-y-4">
      <ConstellationMap />

      <div class="chart-container">
        <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">ALERTS</div>
        <div class="space-y-2 max-h-72 overflow-y-auto pr-1">
          {#each criticalAlerts as alert, index}
            <div class="rounded border border-slate-700/50 bg-space-900/60 p-3" class:animate-fade-in={index < 2}>
              <div class="flex items-center justify-between gap-3">
                <div class="font-mono text-xs text-rose-400">{alert.title}</div>
                <div class="font-mono text-xs text-slate-600">{alert.time}</div>
              </div>
              <div class="mt-2 text-xs text-slate-400">{alert.message}</div>
              <div class="mt-2">
                <span class="{alert.severity === 'high' ? 'badge-offline' : alert.severity === 'medium' ? 'badge-warning' : 'badge-online'}">
                  {alert.severity.toUpperCase()}
                </span>
              </div>
            </div>
          {:else}
            <div class="py-8 text-center font-mono text-xs text-slate-600">
              No critical alerts
            </div>
          {/each}
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div class="stat-card">
          <div class="font-mono text-xs text-slate-600">ACTIVE CONNECTIONS</div>
          <div class="font-display text-xl text-cyan-400">{$metrics.activeSessions}</div>
        </div>
        <div class="stat-card">
          <div class="font-mono text-xs text-slate-600">DATA THROUGHPUT</div>
          <div class="font-display text-xl text-amber-400">{$metrics.totalTrafficGbps} Gbps</div>
        </div>
        <div class="stat-card">
          <div class="font-mono text-xs text-slate-600">GATEWAY UPTIME</div>
          <div class="font-display text-xl text-emerald-400">{formatPct($metrics.gatewayUptimePct)}</div>
        </div>
        <div class="stat-card">
          <div class="font-mono text-xs text-slate-600">HANDOVER RATE</div>
          <div class="font-display text-xl text-slate-200">{$metrics.handoverRatePerMin}/min</div>
        </div>
      </div>
    </section>
  </div>
</main>
