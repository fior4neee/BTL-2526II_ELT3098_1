<script>
  import { onDestroy, onMount } from 'svelte';
  import { createLocationSample, createTelemetrySample } from './simulator.js';
  import { invokeCommand, isTauri, listenEvent } from './tauriClient.js';
  import { location as realLocation, telemetry as realTelemetry, connectionState } from './network.js';

  let tick = 1;
  let usage = createLocationSample(tick);
  let telemetry = createTelemetrySample(tick);
  let rateHistory = [];
  let alerts = [];
  let timer;
  let unlistenUsage = () => {};
  let unlistenSignal = () => {};

  $: quotaPct = Math.min(100, (usage.monthly_used_gb / usage.monthly_cap_gb) * 100);
  $: downBars = rateHistory.slice(-96).map((point) => Math.max(4, Math.min(100, point.down / 120 * 100)));
  $: upBars = rateHistory.slice(-96).map((point) => Math.max(4, Math.min(100, point.up / 35 * 100)));
  $: fenceClass = usage.geofence_status === 'inside' ? 'connected' : 'searching';
  $: positionStyle = `left:${50 + (usage.longitude - 105.8342) * 1200}%;top:${50 - (usage.latitude - 21.0278) * 1200}%`;

  function ingestUsage(next) {
    if (!next) return;
    usage = next;
    if (next.geofence_status !== 'inside') {
      alerts = [{ id: next.timestamp_ms, text: 'Geo-fence boundary warning', detail: `${next.latitude.toFixed(4)}, ${next.longitude.toFixed(4)}` }, ...alerts].slice(0, 5);
    }
    if ((next.monthly_used_gb / next.monthly_cap_gb) > 0.85) {
      alerts = [{ id: next.timestamp_ms + 1, text: 'Data limit approaching', detail: `${quotaPct.toFixed(1)}% used` }, ...alerts].slice(0, 5);
    }
  }

  function ingestSignal(next) {
    if (!next) return;
    telemetry = next;
    rateHistory = [...rateHistory.slice(-287), { down: next.data_down_mbps, up: next.data_up_mbps }];
  }

  $: if ($connectionState === 'connected') {
    if ($realLocation) ingestUsage($realLocation);
    if ($realTelemetry) ingestSignal($realTelemetry);
  }

  onMount(async () => {
    const initialUsage = await invokeCommand('get_location_snapshot');
    if (initialUsage) ingestUsage(initialUsage);
    unlistenUsage = await listenEvent('location-telemetry', ingestUsage);
    unlistenSignal = await listenEvent('signal-telemetry', ingestSignal);
    if (!isTauri) {
      timer = setInterval(() => {
        if ($connectionState !== 'connected') {
          tick += 1;
          ingestUsage(createLocationSample(tick));
          ingestSignal(createTelemetrySample(tick));
        }
      }, 1000);
    }
  });

  onDestroy(() => {
    clearInterval(timer);
    unlistenUsage();
    unlistenSignal();
  });
</script>

<section class="usage-layout">
  <div class="panel usage-summary">
    <div class="panel-title">
      <span>Data Usage</span>
      <strong>{usage.plan_name}</strong>
    </div>
    <div class="rate-pair">
      <div><small>Downlink</small><strong>{telemetry.data_down_mbps.toFixed(1)}</strong><span>Mbps</span></div>
      <div><small>Uplink</small><strong>{telemetry.data_up_mbps.toFixed(1)}</strong><span>Mbps</span></div>
    </div>
    <div class="quota">
      <div class="metric-row"><span>Monthly quota</span><strong>{usage.monthly_used_gb.toFixed(1)} / {usage.monthly_cap_gb} GB</strong></div>
      <div class="quality-track"><i style:width={`${quotaPct}%`}></i></div>
    </div>
    <div class="diagnostics">
      <span>Duration <strong>{Math.floor(usage.session_duration_s / 60)}m</strong></span>
      <span>Session down <strong>{usage.session_down_gb.toFixed(2)} GB</strong></span>
      <span>Session up <strong>{usage.session_up_gb.toFixed(2)} GB</strong></span>
      <span>Cost <strong>${usage.estimated_cost_usd.toFixed(2)}</strong></span>
    </div>
  </div>

  <div class="panel rate-chart">
    <div class="panel-title">
      <span>24-hour Rate Histogram</span>
      <strong>rolling</strong>
    </div>
    <div class="dual-bars">
      {#each downBars as bar, index}
        <span>
          <i class="down" style:height={`${bar}%`}></i>
          <i class="up" style:height={`${upBars[index] || 4}%`}></i>
        </span>
      {/each}
    </div>
  </div>

  <div class="panel location-panel">
    <div class="panel-title">
      <span>Location</span>
      <strong class={fenceClass}>{usage.geofence_status}</strong>
    </div>
    <div class="map-surface">
      <div class="fence"></div>
      <div class="position-dot" style={positionStyle}></div>
      <span>Hanoi service fence</span>
    </div>
    <div class="metric-stack">
      <div>Latitude <strong>{usage.latitude.toFixed(5)}</strong></div>
      <div>Longitude <strong>{usage.longitude.toFixed(5)}</strong></div>
      <div>Altitude <strong>{usage.altitude_m.toFixed(0)} m</strong></div>
      <div>Plan type <strong>{usage.plan_type}</strong></div>
    </div>
  </div>

  <div class="panel alerts-panel">
    <div class="panel-title">
      <span>Alerts</span>
      <strong>{alerts.length}</strong>
    </div>
    <div class="event-log">
      {#if alerts.length === 0}
        <p>No active alerts</p>
      {:else}
        {#each alerts as alert}
          <div><span>{alert.text}</span><small>{alert.detail}</small></div>
        {/each}
      {/if}
    </div>
  </div>
</section>
