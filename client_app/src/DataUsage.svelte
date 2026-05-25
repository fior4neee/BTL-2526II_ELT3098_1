<script>
  import { onDestroy } from 'svelte';
  import { telemetry as netTelemetry, location as netLocation, activeDevice, DEVICES } from './network.js';

  let rateHistory = [];
  let alerts = [];

  $: dev = DEVICES.find(d => d.id === $activeDevice) || DEVICES[0];

  $: usage = $netLocation || {
    latitude: dev.home.lat,
    longitude: dev.home.lon,
    altitude_m: 14.0,
    geofence_status: 'inside',
    monthly_used_gb: 0,
    monthly_cap_gb: 1500,
    plan_name: dev.plan === 'Mobile' ? 'Mobile Unlimited 5G' : 'Fixed Business 300',
    plan_type: dev.plan,
    session_duration_s: 0,
    session_down_gb: 0,
    session_up_gb: 0,
    estimated_cost_usd: 0
  };

  $: telemetry = $netTelemetry || { data_down_mbps: 0, data_up_mbps: 0 };

  $: quotaPct = Math.min(100, (usage.monthly_used_gb / usage.monthly_cap_gb) * 100);
  $: downBars = rateHistory.slice(-96).map((point) => Math.max(4, Math.min(100, point.down / 120 * 100)));
  $: upBars = rateHistory.slice(-96).map((point) => Math.max(4, Math.min(100, point.up / 35 * 100)));
  $: fenceClass = usage.geofence_status === 'inside' ? 'connected' : (usage.geofence_status === 'breach' ? 'outage' : 'searching');
  $: positionStyle = `left:${50 + (usage.longitude - dev.home.lon) * 1200}%;top:${50 - (usage.latitude - dev.home.lat) * 1200}%`;

  const unsubTelemetry = netTelemetry.subscribe((val) => {
    if (val) {
      rateHistory = [...rateHistory.slice(-287), { down: val.data_down_mbps, up: val.data_up_mbps }];
    } else {
      rateHistory = [];
    }
  });

  const unsubLocation = netLocation.subscribe((val) => {
    if (val) {
      if (val.geofence_status !== 'inside') {
        const text = val.geofence_status === 'breach' ? 'Geo-fence BREACH' : 'Geo-fence warning';
        alerts = [{ id: val.timestamp_ms, text, detail: `${val.latitude.toFixed(4)}, ${val.longitude.toFixed(4)}` }, ...alerts].slice(0, 5);
      }
      if ((val.monthly_used_gb / val.monthly_cap_gb) > 0.85) {
        alerts = [{ id: val.timestamp_ms + 1, text: 'Data limit approaching', detail: `${quotaPct.toFixed(1)}% used` }, ...alerts].slice(0, 5);
      }
    } else {
      alerts = [];
    }
  });

  onDestroy(() => {
    unsubTelemetry();
    unsubLocation();
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
