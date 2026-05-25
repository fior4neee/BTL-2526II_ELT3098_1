<script>
  import { onDestroy, onMount } from 'svelte';
  import { telemetry, events as networkEvents } from './network.js';
  import { invokeCommand, isTauri } from './tauriClient.js';

  const maxHistory = 3600;
  const thresholds = [10, 12, 15, 18, 22, 26];

  let history = [];
  let selectedSatellite = 'auto';
  let steeringMode = 'auto';
  let signalThreshold = 12;
  let logLevel = 'info';

  // Subscribe to telemetry store to maintain history array
  const unsubTelemetry = telemetry.subscribe((val) => {
    if (val) {
      history = [...history.slice(-(maxHistory - 1)), val];
    } else {
      history = [];
    }
  });

  // Reactive values derived from store states
  $: sample = $telemetry || {
    link_status: 'outage',
    azimuth_deg: 0,
    elevation_deg: 0,
    beam_quality: 0,
    c_n_ratio_db: 0,
    carrier_power_dbm: -120,
    eb_n0_db: 0,
    ber: 0.5,
    path_loss_db: 200,
    eirp_dbw: 48,
    modulation_scheme: 'None',
    time_to_horizon_s: 0,
    latency_ms: 0,
    jitter_ms: 0,
    packet_loss_pct: 0,
    satellite_name: 'None',
    gateway_name: 'None'
  };

  $: events = $networkEvents;

  $: cnPath = history
    .slice(-90)
    .map((point, index, rows) => {
      const x = rows.length <= 1 ? 0 : (index / (rows.length - 1)) * 100;
      const y = 96 - Math.min(92, Math.max(8, (point.c_n_ratio_db / 26) * 86));
      return `${x},${y}`;
    })
    .join(' ');

  $: berPath = history
    .slice(-90)
    .map((point, index, rows) => {
      const x = rows.length <= 1 ? 0 : (index / (rows.length - 1)) * 100;
      const value = Math.max(0, -Math.log10(point.ber || 1e-8));
      const y = 96 - Math.min(88, value * 12);
      return `${x},${y}`;
    })
    .join(' ');

  $: powerBars = history.slice(-32).map((point) => Math.max(8, Math.min(100, point.carrier_power_dbm + 112)));
  $: compassRotation = `rotate(${sample.azimuth_deg}deg)`;
  $: qualityPct = Math.round(sample.beam_quality * 100);
  $: statusClass = sample.link_status.toLowerCase();
  $: visibleSatellites = ['auto', 'VNU-LEO-Alpha', 'VNU-LEO-Beta', 'VNU-LEO-Gamma', 'VNU-LEO-Delta'];
  $: nextPasses = [
    { name: sample.satellite_name, eta: 'Now', elevation: sample.elevation_deg },
    { name: 'VNU-LEO-Epsilon', eta: '+07:20', elevation: 42 },
    { name: 'VNU-LEO-Lambda', eta: '+18:45', elevation: 57 }
  ];

  async function applySettings() {
    await invokeCommand('update_tracking_settings', {
      settings: {
        selected_satellite: selectedSatellite,
        steering_mode: steeringMode,
        signal_threshold_db: Number(signalThreshold),
        log_level: logLevel
      }
    });
  }

  onDestroy(() => {
    unsubTelemetry();
  });
</script>

<section class="dashboard-grid">
  <aside class="panel left-panel">
    <div class="panel-title">
      <span>Antenna Pointing</span>
      <strong>{Math.round(sample.azimuth_deg)} deg</strong>
    </div>
    <div class="compass" aria-label="Antenna compass">
      <span class="north">N</span><span class="east">E</span><span class="south">S</span><span class="west">W</span>
      <div class="needle" style:transform={compassRotation}></div>
      <div class="compass-core">{Math.round(sample.elevation_deg)} deg</div>
    </div>
    <div class="elevation-dial">
      <div class="elevation-fill" style:height={`${Math.min(100, sample.elevation_deg)}%`}></div>
      <span>Elevation</span>
    </div>
    <div class="metric-row">
      <span>Beam quality</span>
      <strong>{qualityPct}%</strong>
    </div>
    <div class="quality-track"><i style:width={`${qualityPct}%`}></i></div>
    <div class="metric-row">
      <span>Mode</span>
      <strong>{steeringMode}</strong>
    </div>
  </aside>

  <section class="panel center-panel">
    <div class="panel-title">
      <span>Signal Quality</span>
      <strong class={statusClass}>{sample.link_status}</strong>
    </div>
    <div class="gauge-row">
      <div class="cn-gauge" style:--value={`${Math.min(100, (sample.c_n_ratio_db / 26) * 100)}%`}>
        <span>{sample.c_n_ratio_db.toFixed(1)}</span>
        <small>C/N dB</small>
      </div>
      <div class="metric-stack">
        <div>Carrier power <strong>{sample.carrier_power_dbm.toFixed(1)} dBm</strong></div>
        <div>Eb/N0 <strong>{sample.eb_n0_db.toFixed(1)} dB</strong></div>
        <div>BER <strong>{sample.ber.toExponential(2)}</strong></div>
        <div>Modulation <strong>{sample.modulation_scheme}</strong></div>
      </div>
    </div>
    <div class="chart-block">
      <div class="chart-head"><span>C/N rolling window</span><small>last 90 samples</small></div>
      <svg viewBox="0 0 100 100" preserveAspectRatio="none">
        <polyline points={cnPath} />
        {#each thresholds as threshold}
          <line x1="0" x2="100" y1={96 - (threshold / 26) * 86} y2={96 - (threshold / 26) * 86} />
        {/each}
      </svg>
    </div>
    <div class="histogram" aria-label="Received power histogram">
      {#each powerBars as bar}
        <i style:height={`${bar}%`}></i>
      {/each}
    </div>
    <div class="chart-block compact">
      <div class="chart-head"><span>BER trend</span><small>log scale</small></div>
      <svg viewBox="0 0 100 100" preserveAspectRatio="none"><polyline points={berPath} /></svg>
    </div>
  </section>

  <aside class="panel right-panel">
    <div class="panel-title">
      <span>Satellite Timeline</span>
      <strong>{sample.gateway_name}</strong>
    </div>
    <div class="sat-card">
      <span>{sample.satellite_name}</span>
      <strong>{Math.floor(sample.time_to_horizon_s / 60)}m {sample.time_to_horizon_s % 60}s</strong>
      <small>time to horizon</small>
    </div>
    <div class="timeline">
      {#each nextPasses as pass}
        <div>
          <span>{pass.name}</span>
          <strong>{pass.eta}</strong>
          <small>{Math.round(pass.elevation)} deg peak</small>
        </div>
      {/each}
    </div>
    <div class="settings-grid">
      <label>Satellite
        <select bind:value={selectedSatellite} on:change={applySettings}>
          {#each visibleSatellites as sat}<option value={sat}>{sat}</option>{/each}
        </select>
      </label>
      <label>Steering
        <select bind:value={steeringMode} on:change={applySettings}>
          <option value="auto">auto</option>
          <option value="manual">manual</option>
          <option value="null-steering">null steering</option>
        </select>
      </label>
      <label>Threshold
        <input type="number" min="6" max="24" bind:value={signalThreshold} on:change={applySettings} />
      </label>
      <label>Log
        <select bind:value={logLevel} on:change={applySettings}>
          <option>info</option><option>debug</option><option>warn</option>
        </select>
      </label>
    </div>
    <div class="diagnostics">
      <span>Latency <strong>{sample.latency_ms.toFixed(0)} ms</strong></span>
      <span>Jitter <strong>{sample.jitter_ms.toFixed(1)} ms</strong></span>
      <span>Loss <strong>{sample.packet_loss_pct.toFixed(2)}%</strong></span>
    </div>
    <div class="event-log">
      {#if events.length === 0}
        <p>No handovers in current window</p>
      {:else}
        {#each events as event}
          <div><span>{event.text}</span><small>{event.packetLoss.toFixed(2)}% loss, {event.latency.toFixed(0)} ms</small></div>
        {/each}
      {/if}
    </div>
  </aside>
</section>
