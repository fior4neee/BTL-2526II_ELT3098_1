<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { gateways } from '$lib/stores';
  import ConstellationModal from './ConstellationModal.svelte';

  // ── Mini-map viewport (Vietnam only) ─────────────────────────────────────
  const MAP_W = 300;
  const MAP_H = 420;
  const LAT_MIN = 7;
  const LAT_MAX = 24;
  const LNG_MIN = 100;
  const LNG_MAX = 113;

  // ── Coordinate helpers ────────────────────────────────────────────────────
  function latToY(lat: number) {
    return MAP_H - ((lat - LAT_MIN) / (LAT_MAX - LAT_MIN)) * MAP_H;
  }
  function lngToX(lng: number) {
    return ((lng - LNG_MIN) / (LNG_MAX - LNG_MIN)) * MAP_W;
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

  // ── Hover + modal state ───────────────────────────────────────────────────
  let isHovered = false;
  let showConstellation = false;

  function openConstellation() { showConstellation = true; }
  function closeConstellation() { showConstellation = false; }
</script>

{#if showConstellation}
  <ConstellationModal onClose={closeConstellation} />
{/if}

<div class="chart-container">
  <div class="flex items-center justify-between mb-3">
    <div class="font-mono text-xs text-slate-500 tracking-widest">CONSTELLATION VIEW — VIETNAM CORRIDOR</div>
    <div class="font-mono text-xs text-slate-600">{$gateways.length} GW</div>
  </div>

  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="mini-map-wrapper"
    on:mouseenter={() => isHovered = true}
    on:mouseleave={() => isHovered = false}
    on:click={openConstellation}
    role="button"
    tabindex="0"
    aria-label="Click to open Constellation Map"
    on:keydown={(e) => e.key === 'Enter' && openConstellation()}
  >
    <!-- SVG map: only gateways, no satellites -->
    <svg viewBox="0 0 {MAP_W} {MAP_H}" width={MAP_W} height={MAP_H}
         class="rounded overflow-hidden block transition-all duration-300"
         style="filter: {isHovered ? 'blur(3px) brightness(0.5)' : 'none'};">
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

      <!-- Accurate Vietnam silhouette -->
      <path
        d={vietnamPath}
        fill="rgba(34,211,238,0.07)"
        stroke="rgba(34,211,238,0.30)"
        stroke-width="1.5"
        stroke-linejoin="round"
      />

      <!-- Gateways only (no satellites) -->
      {#each $gateways as gw}
        {@const x = lngToX(gw.lng)}
        {@const y = latToY(gw.lat)}
        {@const color = gw.status === 'alive' ? '#34d399' : gw.status === 'degraded' ? '#fbbf24' : '#f43f5e'}
        <circle cx={x} cy={y} r="42" fill={color} fill-opacity="0.04"
                stroke={color} stroke-opacity="0.15" stroke-width="1" stroke-dasharray="4 3" />
        <circle cx={x} cy={y} r="5" fill={color} fill-opacity="0.9" />
        <circle cx={x} cy={y} r="9" fill="none" stroke={color} stroke-opacity="0.5" stroke-width="1.5">
          <animate attributeName="r" values="9;14;9" dur="3s" repeatCount="indefinite" />
          <animate attributeName="stroke-opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite" />
        </circle>
        <text x={x + 10} y={y + 4} font-family="JetBrains Mono, monospace" font-size="8"
              fill={color} fill-opacity="0.9">
          {gw.id.replace('GW-', '').replace('-01', '')}
        </text>
      {/each}

      <!-- Lat labels -->
      <text x="4" y="415" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">8N</text>
      <text x="4" y="210" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">16N</text>
      <text x="4" y="8"   font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">24N</text>
    </svg>

    <!-- Hover overlay: blur + click prompt -->
    <div class="hover-overlay" class:hover-overlay--visible={isHovered}>
      <div class="hover-overlay__icon">
        <!-- Expand icon -->
        <svg width="28" height="28" viewBox="0 0 28 28" fill="none" aria-hidden="true">
          <polyline points="18,4 24,4 24,10" stroke="#22d3ee" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="15" y1="13" x2="24" y2="4" stroke="#22d3ee" stroke-width="2" stroke-linecap="round"/>
          <polyline points="10,24 4,24 4,18" stroke="#22d3ee" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="13" y1="15" x2="4" y2="24" stroke="#22d3ee" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </div>
      <div class="hover-overlay__text">Click to full-screen<br/>Constellation Map</div>
    </div>

    <!-- Legend (visible when not hovered) -->
    <div class="absolute bottom-2 right-2 space-y-1 transition-opacity duration-300"
         style="opacity: {isHovered ? 0 : 1}; pointer-events: none;">
      <div class="flex items-center gap-1.5">
        <div class="w-2 h-2 rounded-full bg-emerald-400"></div>
        <span class="font-mono text-xs text-slate-500">Alive</span>
      </div>
      <div class="flex items-center gap-1.5">
        <div class="w-2 h-2 rounded-full bg-amber-400"></div>
        <span class="font-mono text-xs text-slate-500">Degraded</span>
      </div>
    </div>
  </div>
</div>

<style>
  .mini-map-wrapper {
    position: relative;
    display: flex;
    justify-content: center;
    min-height: 420px;
    cursor: pointer;
    border-radius: 6px;
    overflow: hidden;
  }

  /* Hover overlay — blur + text, invisible until hovered */
  .hover-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.25s ease;
    z-index: 5;
  }
  .hover-overlay--visible {
    opacity: 1;
  }
  .hover-overlay__icon {
    filter: drop-shadow(0 0 8px rgba(34,211,238,0.6));
  }
  .hover-overlay__text {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: rgba(34, 211, 238, 0.9);
    text-align: center;
    line-height: 1.6;
    letter-spacing: 0.05em;
    text-shadow: 0 0 12px rgba(34, 211, 238, 0.5);
  }
</style>
