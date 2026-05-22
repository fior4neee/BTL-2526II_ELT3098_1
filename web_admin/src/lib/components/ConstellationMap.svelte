<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { gateways, satellites } from '$lib/stores';

  // ── Map viewport constants ─────────────────────────────────────────────────
  const MAP_W = 300;
  const MAP_H = 400;
  const LAT_MIN = 7;
  const LAT_MAX = 24;
  const LNG_MIN = 100;
  const LNG_MAX = 112;

  // ── LEO orbital simulation constants ──────────────────────────────────────
  // LEO altitude ~550 km → orbital period ≈ 95.5 min
  // We accelerate time so one full orbit = 60 seconds on screen (×95.5 speedup)
  const ORBIT_PERIOD_S = 120;          // simulated seconds per orbit (slowed for smoother motion)
  const REFRESH_MS = 200;             // redraw every 200 ms (smoother updates)
  const EARTH_R = 6371;               // km
  const ALT_KM = 550;                 // km

  // Walker-Delta 15/3/1 constellation: 15 sats, 3 planes, inclination 53°
  // Each plane spaced 60° apart in RAAN; sats within plane equally spaced
  const SAT_DEFS = Array.from({ length: 15 }, (_, i) => {
    const plane      = Math.floor(i / 5);         // 0,1,2
    const seat       = i % 5;                      // 0..4
    const inc        = 53 * (Math.PI / 180);       // inclination radians
    const raan       = (plane * 120) * (Math.PI / 180); // RAAN: 0°, 120°, 240°
    const m0         = (seat * 72) * (Math.PI / 180);   // mean anomaly offset
    return { id: `SAT-LEO-${String(i + 1).padStart(3, '0')}`, inc, raan, m0 };
  });

  // ── Reactive animated satellites (purely client-side) ─────────────────────
  interface AnimSat {
    id: string;
    lat: number;
    lon: number;
    inView: boolean;
  }

  let animSats: AnimSat[] = [];
  let tick = 0;
  let timer: ReturnType<typeof setInterval>;

  function computePositions(t: number): AnimSat[] {
    return SAT_DEFS.map(({ id, inc, raan, m0 }) => {
      // Mean motion: one orbit in ORBIT_PERIOD_S seconds
      const M = m0 + (2 * Math.PI * t) / ORBIT_PERIOD_S;

      // Simplified Keplerian → Cartesian (circular orbit)
      // In orbital plane: x = cos(M), y = sin(M)
      // Rotate by inclination then RAAN to get ECI, then to lat/lon
      const xOrb = Math.cos(M);
      const yOrb = Math.sin(M);

      // Rotate by inclination (around x-axis of orbital plane)
      const x3 = xOrb;
      const y3 = yOrb * Math.cos(inc);
      const z3 = yOrb * Math.sin(inc);

      // Rotate by RAAN (around z-axis)
      const x = x3 * Math.cos(raan) - y3 * Math.sin(raan);
      const y = x3 * Math.sin(raan) + y3 * Math.cos(raan);
      const z = z3;

      // Earth's rotation: ~360° per 86400 s, scaled to sim time
      const earthRot = (2 * Math.PI * t * 95.5) / 86400;
      // Longitude = atan2(y,x) - earth rotation → normalise to [-180, 180]
      let lon = (Math.atan2(y, x) - earthRot) * (180 / Math.PI);
      lon = ((lon + 540) % 360) - 180;

      const lat = Math.asin(Math.max(-1, Math.min(1, z))) * (180 / Math.PI);

      // Wider margin (±10°) to keep satellites visible longer and avoid flickering
      const inView =
        lat >= LAT_MIN - 10 && lat <= LAT_MAX + 10 &&
        lon >= LNG_MIN - 10 && lon <= LNG_MAX + 10;

      return { id, lat, lon, inView };
    });
  }

  onMount(() => {
    const startWall = Date.now() / 1000;
    timer = setInterval(() => {
      tick = Date.now() / 1000 - startWall;
      animSats = computePositions(tick);
    }, REFRESH_MS);
    // Initial render immediately
    animSats = computePositions(0);
  });

  onDestroy(() => clearInterval(timer));

  // ── Coordinate helpers ────────────────────────────────────────────────────
  function latToY(lat: number) {
    return MAP_H - ((lat - LAT_MIN) / (LAT_MAX - LAT_MIN)) * MAP_H;
  }
  function lngToX(lng: number) {
    return ((lng - LNG_MIN) / (LNG_MAX - LNG_MIN)) * MAP_W;
  }

  // ── Find nearest animated sat to a gateway (for link line) ───────────────
  function nearestSat(gwLat: number, gwLng: number): AnimSat | null {
    let best: AnimSat | null = null;
    let bestD = Infinity;
    for (const s of animSats) {
      if (!s.inView) continue;
      const d = Math.hypot(s.lat - gwLat, s.lon - gwLng);
      if (d < bestD) { bestD = d; best = s; }
    }
    return best;
  }

  // ── Tick-driven reactive vars for template ────────────────────────────────
  $: visibleSats = animSats.filter(s => s.inView);
  // Use animated positions; fall back to store if animation hasn't started yet
  $: displaySats = visibleSats.length > 0 ? visibleSats : $satellites.map(s => ({
    id: s.satelliteId, lat: s.latitude, lon: s.longitude, inView: true,
  }));
</script>

<div class="chart-container">
  <div class="flex items-center justify-between mb-3">
    <div class="font-mono text-xs text-slate-500 tracking-widest">CONSTELLATION VIEW — VIETNAM CORRIDOR</div>
    <div class="font-mono text-xs text-slate-600">{visibleSats.length} SAT IN VIEW</div>
  </div>

  <div class="relative flex justify-center min-h-[400px]">
    <svg viewBox="0 0 {MAP_W} {MAP_H}" width={MAP_W} height={MAP_H} class="rounded overflow-hidden">
      <!-- Background -->
      <rect width={MAP_W} height={MAP_H} fill="#060d14" />

      <!-- Grid lines -->
      {#each [8,10,12,14,16,18,20,22,24] as lat}
        <line x1="0" y1={latToY(lat)} x2={MAP_W} y2={latToY(lat)}
              stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1" />
      {/each}
      {#each [102,104,106,108,110] as lng}
        <line x1={lngToX(lng)} y1="0" x2={lngToX(lng)} y2={MAP_H}
              stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1" />
      {/each}

      <!-- Vietnam silhouette -->
      <path
        d="M 185 30 L 205 60 L 220 90 L 215 120 L 200 150 L 210 180 L 215 210 L 200 240 L 185 260 L 165 280 L 150 300 L 140 330 L 125 355 L 115 370 L 110 380 L 100 375 L 108 360 L 115 345 L 120 320 L 130 295 L 145 270 L 155 245 L 165 220 L 160 190 L 155 165 L 165 140 L 170 115 L 160 90 L 155 60 L 165 35 Z"
        fill="rgba(34,211,238,0.05)"
        stroke="rgba(34,211,238,0.2)"
        stroke-width="1.5"
      />

      <!-- Gateway → nearest satellite link lines -->
      {#each $gateways as gw}
        {@const nearest = nearestSat(gw.lat, gw.lng)}
        {#if nearest}
          {@const x1 = lngToX(gw.lng)}
          {@const y1 = latToY(gw.lat)}
          {@const x2 = lngToX(nearest.lon)}
          {@const y2 = latToY(nearest.lat)}
          <line {x1} {y1} {x2} {y2}
                stroke="#22d3ee" stroke-opacity="0.25" stroke-width="0.8"
                stroke-dasharray="3 3" />
        {/if}
      {/each}

      <!-- Gateways -->
      {#each $gateways as gw}
        {@const x = lngToX(gw.lng)}
        {@const y = latToY(gw.lat)}
        {@const color = gw.status === 'alive' ? '#34d399' : gw.status === 'degraded' ? '#fbbf24' : '#f43f5e'}
        <!-- Coverage ring -->
        <circle cx={x} cy={y} r="42" fill={color} fill-opacity="0.04"
                stroke={color} stroke-opacity="0.15" stroke-width="1" stroke-dasharray="4 3" />
        <!-- Dot -->
        <circle cx={x} cy={y} r="5" fill={color} fill-opacity="0.9" />
        <!-- Pulse ring -->
        <circle cx={x} cy={y} r="9" fill="none" stroke={color} stroke-opacity="0.5" stroke-width="1.5">
          <animate attributeName="r" values="9;14;9" dur="3s" repeatCount="indefinite" />
          <animate attributeName="stroke-opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite" />
        </circle>
        <text x={x + 10} y={y + 4} font-family="JetBrains Mono, monospace" font-size="8"
              fill={color} fill-opacity="0.9">
          {gw.id.replace('GW-', '').replace('-01', '')}
        </text>
      {/each}

      <!-- Animated satellites -->
      {#each displaySats as sat (sat.id)}
        {@const sx = lngToX(sat.lon)}
        {@const sy = latToY(sat.lat)}
        <g>
          <!-- Direction cross -->
          <line x1={sx - 5} y1={sy} x2={sx + 5} y2={sy}
                stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.8" />
          <line x1={sx} y1={sy - 5} x2={sx} y2={sy + 5}
                stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.8" />
          <!-- Core dot -->
          <circle cx={sx} cy={sy} r="3" fill="#22d3ee" fill-opacity="0.9" />
          <!-- Footprint circle -->
          <circle cx={sx} cy={sy} r="18" fill="none"
                  stroke="#22d3ee" stroke-opacity="0.12" stroke-width="1" />
          <!-- Label -->
          <text x={sx + 5} y={sy - 5} font-family="JetBrains Mono, monospace"
                font-size="6" fill="#22d3ee" fill-opacity="0.55">
            {sat.id.replace('SAT-LEO-', '')}
          </text>
        </g>
      {/each}

      <!-- Lat labels -->
      <text x="4" y="395" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">8N</text>
      <text x="4" y="200" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">16N</text>
      <text x="4" y="8"   font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">24N</text>
    </svg>

    <!-- Legend -->
    <div class="absolute bottom-2 right-2 space-y-1">
      <div class="flex items-center gap-1.5">
        <div class="w-2 h-2 rounded-full bg-emerald-400"></div>
        <span class="font-mono text-xs text-slate-500">Alive</span>
      </div>
      <div class="flex items-center gap-1.5">
        <div class="w-2 h-2 rounded-full bg-amber-400"></div>
        <span class="font-mono text-xs text-slate-500">Degraded</span>
      </div>
      <div class="flex items-center gap-1.5">
        <div class="w-2 h-2 rounded-full bg-cyan-400"></div>
        <span class="font-mono text-xs text-slate-500">Satellite</span>
      </div>
    </div>
  </div>
</div>
