<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { gateways, satellites, sessions } from '$lib/stores';
  import WorldMapModal from './WorldMapModal.svelte';

  export let onClose: () => void;

  // ── Viewport: locked to Vietnam ───────────────────────────────────────────
  const MAP_W = 560;
  const MAP_H = 780;
  const LAT_MIN = 7;
  const LAT_MAX = 24;
  const LNG_MIN = 100;
  const LNG_MAX = 113;

  // ── LEO orbital simulation ────────────────────────────────────────────────
  const ORBIT_PERIOD_S = 120;
  const REFRESH_MS = 200;

  const SAT_DEFS = Array.from({ length: 15 }, (_, i) => {
    const plane = Math.floor(i / 5);
    const seat = i % 5;
    const inc = 53 * (Math.PI / 180);
    const raan = (plane * 120) * (Math.PI / 180);
    const m0 = (seat * 72) * (Math.PI / 180);
    return { id: `SAT-LEO-${String(i + 1).padStart(3, '0')}`, inc, raan, m0 };
  });

  interface AnimSat { id: string; lat: number; lon: number; inView: boolean; }
  let animSats: AnimSat[] = [];
  let tick = 0;
  let timer: ReturnType<typeof setInterval>;

  function computePositions(t: number): AnimSat[] {
    return SAT_DEFS.map(({ id, inc, raan, m0 }) => {
      const M = m0 + (2 * Math.PI * t) / ORBIT_PERIOD_S;
      const xOrb = Math.cos(M), yOrb = Math.sin(M);
      const x3 = xOrb, y3 = yOrb * Math.cos(inc), z3 = yOrb * Math.sin(inc);
      const x = x3 * Math.cos(raan) - y3 * Math.sin(raan);
      const y = x3 * Math.sin(raan) + y3 * Math.cos(raan);
      const z = z3;
      const earthRot = (2 * Math.PI * t * 95.5) / 86400;
      let lon = (Math.atan2(y, x) - earthRot) * (180 / Math.PI);
      lon = ((lon + 540) % 360) - 180;
      const lat = Math.asin(Math.max(-1, Math.min(1, z))) * (180 / Math.PI);
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

  $: visibleSats = animSats.filter(s => s.inView);
  $: displaySats = visibleSats.length > 0 ? visibleSats : $satellites.map(s => ({
    id: s.satelliteId, lat: s.latitude, lon: s.longitude, inView: true,
  }));

  function nearestSat(gwLat: number, gwLng: number): AnimSat | null {
    let best: AnimSat | null = null, bestD = Infinity;
    for (const s of animSats) {
      if (!s.inView) continue;
      const d = Math.hypot(s.lat - gwLat, s.lon - gwLng);
      if (d < bestD) { bestD = d; best = s; }
    }
    return best;
  }

  // ── Accurate Vietnam SVG path (GADM-derived) ──────────────────────────────
  const vietnamPath = buildVietnamPath();
  function buildVietnamPath(): string {
    const coords: [number, number][] = [
      [102.14, 22.90], [103.00, 22.97], [103.53, 22.59], [104.26, 22.64],
      [104.73, 22.82], [105.28, 23.31], [105.88, 23.01], [106.72, 22.88],
      [107.37, 21.65], [107.72, 21.60], [107.96, 21.49],
      [108.05, 21.18], [108.22, 20.92],
      [107.37, 20.47], [107.01, 20.07], [106.74, 20.17],
      [106.58, 20.55], [106.42, 20.67],
      [106.03, 20.90], [105.77, 20.96],
      [105.65, 20.68], [105.64, 20.25], [105.90, 20.09],
      [106.07, 19.75], [106.35, 19.47], [106.51, 19.11],
      [107.05, 18.42], [107.46, 17.74],
      [107.91, 17.11], [108.11, 16.73],
      [108.26, 16.08], [108.09, 15.87],
      [108.47, 15.47], [108.94, 15.24],
      [109.21, 14.80], [109.45, 13.74],
      [109.33, 13.26], [109.21, 12.72],
      [109.46, 12.38], [109.32, 11.81],
      [108.88, 11.31], [108.67, 11.08],
      [108.25, 10.61], [107.89, 10.34], [107.06, 10.41],
      [106.64, 10.47], [106.16, 10.30],
      [105.61,  9.98], [105.06, 10.10],
      [104.86, 10.53], [104.51, 10.58],
      [104.28, 10.43], [104.23, 10.72],
      [104.75, 11.00], [105.07, 11.07],
      [105.25, 11.51], [104.86, 11.99],
      [104.51, 12.38], [104.15, 12.76],
      [103.50, 12.54], [102.98, 13.18],
      [102.57, 14.26], [102.50, 14.72],
      [102.62, 15.18], [103.03, 15.47],
      [103.21, 15.90], [102.89, 16.48],
      [102.54, 16.89], [102.20, 17.44],
      [102.15, 17.88], [102.43, 18.46],
      [103.11, 18.71], [103.57, 18.76],
      [103.94, 19.27], [104.25, 19.63],
      [104.34, 20.07], [103.76, 20.45],
      [103.43, 20.73], [103.25, 20.93],
      [103.09, 21.65], [102.93, 21.92],
      [102.50, 22.29], [102.14, 22.90],
    ];
    return coords
      .map((c, i) => `${i === 0 ? 'M' : 'L'} ${lngToX(c[0]).toFixed(1)} ${latToY(c[1]).toFixed(1)}`)
      .join(' ') + ' Z';
  }

  // ── World Map modal ───────────────────────────────────────────────────────
  let showWorldMap = false;
  function openWorldMap() { showWorldMap = true; }
  function closeWorldMap() { showWorldMap = false; }

  // Escape closes constellation (not world map when it's open)
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !showWorldMap) onClose();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if showWorldMap}
  <WorldMapModal onClose={closeWorldMap} />
{/if}

<!-- Full-screen constellation modal (Vietnam locked view) -->
<div
  class="fixed inset-0 z-40 bg-space-950 flex flex-col"
  role="dialog"
  aria-label="Constellation Map — Vietnam"
  aria-modal="true"
>
  <!-- Header bar with two buttons -->
  <div class="flex items-center justify-between px-5 py-3 border-b border-slate-700/40 flex-shrink-0">
    <div class="flex items-center gap-3">
      <div class="font-display text-xs tracking-widest text-cyan-400">CONSTELLATION MAP — VIETNAM CORRIDOR</div>
      <div class="font-mono text-xs text-slate-500">
        {visibleSats.length} SAT IN VIEW &nbsp;·&nbsp; {$gateways.length} GATEWAY
      </div>
    </div>

    <!-- Two top-right buttons -->
    <div class="flex items-center gap-2">
      <!-- Expand to World Map button -->
      <button
        type="button"
        id="constellation-expand"
        on:click={openWorldMap}
        class="map-icon-btn map-icon-btn--cyan"
        aria-label="Expand to World Map"
        title="Show full world constellation map"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <polyline points="10,2 14,2 14,6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="8.5" y1="7.5" x2="14" y2="2" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
          <polyline points="6,14 2,14 2,10" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="7.5" y1="8.5" x2="2" y2="14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
        </svg>
      </button>

      <!-- Red X — close constellation, back to Overview -->
      <button
        type="button"
        id="constellation-close"
        on:click={onClose}
        class="map-icon-btn map-icon-btn--rose"
        aria-label="Close constellation map"
        title="Exit Constellation Map"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <line x1="3" y1="3" x2="13" y2="13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="13" y1="3" x2="3" y2="13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
  </div>

  <!-- Map area — centred, scrollable if needed -->
  <div class="flex-1 overflow-auto flex items-center justify-center bg-space-950 p-4">
    <div class="relative">
      <svg
        viewBox="0 0 {MAP_W} {MAP_H}"
        width={MAP_W}
        height={MAP_H}
        class="rounded-lg overflow-hidden"
        style="max-height: calc(100vh - 100px); width: auto;"
      >
        <!-- Background -->
        <rect width={MAP_W} height={MAP_H} fill="#060d14" />

        <!-- Grid lines -->
        {#each [8,10,12,14,16,18,20,22,24] as lat}
          <line x1="0" y1={latToY(lat)} x2={MAP_W} y2={latToY(lat)}
                stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1" />
        {/each}
        {#each [102,104,106,108,110,112] as lng}
          <line x1={lngToX(lng)} y1="0" x2={lngToX(lng)} y2={MAP_H}
                stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1" />
        {/each}

        <!-- Vietnam silhouette -->
        <path d={vietnamPath} fill="rgba(34,211,238,0.07)" stroke="rgba(34,211,238,0.30)"
              stroke-width="1.5" stroke-linejoin="round" />

        <!-- Gateway → nearest sat link lines -->
        {#each $gateways as gw}
          {@const nearest = nearestSat(gw.lat, gw.lng)}
          {#if nearest}
            <line
              x1={lngToX(gw.lng)} y1={latToY(gw.lat)}
              x2={lngToX(nearest.lon)} y2={latToY(nearest.lat)}
              stroke="#22d3ee" stroke-opacity="0.25" stroke-width="1"
              stroke-dasharray="4 4"
            />
          {/if}
        {/each}

        <!-- Gateways -->
        {#each $gateways as gw}
          {@const x = lngToX(gw.lng)}
          {@const y = latToY(gw.lat)}
          {@const color = gw.status === 'alive' ? '#34d399' : gw.status === 'degraded' ? '#fbbf24' : '#f43f5e'}
          <circle cx={x} cy={y} r="60" fill={color} fill-opacity="0.04"
                  stroke={color} stroke-opacity="0.15" stroke-width="1" stroke-dasharray="4 3" />
          <circle cx={x} cy={y} r="7" fill={color} fill-opacity="0.9" />
          <circle cx={x} cy={y} r="12" fill="none" stroke={color} stroke-opacity="0.5" stroke-width="2">
            <animate attributeName="r" values="12;20;12" dur="3s" repeatCount="indefinite" />
            <animate attributeName="stroke-opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite" />
          </circle>
          <text x={x + 14} y={y + 5} font-family="JetBrains Mono, monospace" font-size="11"
                fill={color} fill-opacity="0.9">
            {gw.id.replace('GW-', '').replace('-01', '')}
          </text>
        {/each}

        <!-- Animated satellites -->
        {#each displaySats as sat (sat.id)}
          {@const sx = lngToX(sat.lon)}
          {@const sy = latToY(sat.lat)}
          <g>
            <line x1={sx-8} y1={sy} x2={sx+8} y2={sy} stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.8" />
            <line x1={sx} y1={sy-8} x2={sx} y2={sy+8} stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.8" />
            <circle cx={sx} cy={sy} r="4" fill="#22d3ee" fill-opacity="0.9" />
            <circle cx={sx} cy={sy} r="26" fill="none" stroke="#22d3ee" stroke-opacity="0.12" stroke-width="1" />
            <text x={sx+7} y={sy-8} font-family="JetBrains Mono, monospace" font-size="9"
                  fill="#22d3ee" fill-opacity="0.6">
              {sat.id.replace('SAT-LEO-', '')}
            </text>
          </g>
        {/each}

        <!-- Lat labels -->
        <text x="6" y={latToY(8)-4} font-family="JetBrains Mono, monospace" font-size="10" fill="rgba(34,211,238,0.3)">8°N</text>
        <text x="6" y={latToY(16)-4} font-family="JetBrains Mono, monospace" font-size="10" fill="rgba(34,211,238,0.3)">16°N</text>
        <text x="6" y={latToY(24)-4} font-family="JetBrains Mono, monospace" font-size="10" fill="rgba(34,211,238,0.3)">24°N</text>
      </svg>

      <!-- Legend -->
      <div class="absolute bottom-3 right-3 bg-space-800/80 backdrop-blur-sm border border-slate-700/40 rounded px-3 py-2 space-y-1">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-emerald-400"></div>
          <span class="font-mono text-xs text-slate-400">Alive</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-amber-400"></div>
          <span class="font-mono text-xs text-slate-400">Degraded</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-cyan-400"></div>
          <span class="font-mono text-xs text-slate-400">Satellite</span>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .map-icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: 6px;
    background: transparent;
    border: 1px solid transparent;
    cursor: pointer;
    padding: 0;
    transition: color 0.15s ease, background 0.15s ease, border-color 0.15s ease;
  }
  .map-icon-btn svg {
    display: block;
    pointer-events: none;
  }

  /* Cyan expand button */
  .map-icon-btn--cyan {
    color: rgba(34, 211, 238, 0.55);
    border-color: rgba(34, 211, 238, 0.12);
  }
  .map-icon-btn--cyan:hover {
    color: #22d3ee;
    background: rgba(34, 211, 238, 0.08);
    border-color: rgba(34, 211, 238, 0.25);
  }

  /* Rose X close button */
  .map-icon-btn--rose {
    color: rgba(251, 113, 133, 0.6);
    border-color: rgba(251, 113, 133, 0.12);
  }
  .map-icon-btn--rose:hover {
    color: #fb7185;
    background: rgba(251, 113, 133, 0.08);
    border-color: rgba(251, 113, 133, 0.25);
  }
</style>
