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

  // Vietnam outline — real geometry from worldHigh.json (ISO3166-1-Alpha-2=VN)
  // Main polygon: 326 pts extracted from the TopoJSON GADM ring
  const VN_MAIN: [number,number][] = [[107.522,14.705],[107.489,14.804],[107.555,14.876],[107.447,15.025],[107.577,15.047],[107.656,15.205],[107.659,15.283],[107.574,15.411],[107.493,15.42],[107.447,15.508],[107.369,15.495],[107.16,15.758],[107.176,15.849],[107.392,15.935],[107.437,16.068],[107.3,16.065],[107.153,16.166],[107.118,16.254],[106.968,16.301],[106.824,16.531],[106.752,16.429],[106.655,16.47],[106.641,16.587],[106.534,16.684],[106.534,16.973],[106.42,16.989],[106.26,17.29],[106.185,17.257],[106.019,17.394],[105.846,17.598],[105.735,17.664],[105.595,17.888],[105.605,17.963],[105.477,18.127],[105.37,18.151],[105.158,18.32],[105.079,18.455],[105.171,18.63],[105.112,18.698],[104.936,18.731],[104.874,18.778],[104.721,18.792],[104.545,18.907],[104.479,18.978],[104.147,19.183],[103.974,19.226],[103.857,19.318],[104.039,19.413],[104.091,19.484],[104.088,19.651],[104.228,19.705],[104.382,19.685],[104.506,19.603],[104.623,19.619],[104.656,19.698],[104.806,19.792],[104.779,19.909],[104.936,19.984],[104.956,20.093],[104.819,20.22],[104.656,20.213],[104.682,20.341],[104.574,20.413],[104.434,20.411],[104.444,20.534],[104.541,20.536],[104.6,20.66],[104.388,20.772],[104.297,20.88],[104.056,20.959],[103.775,20.835],[103.759,20.75],[103.664,20.658],[103.426,20.82],[103.364,20.788],[103.116,20.869],[103.015,21.04],[102.95,21.069],[102.888,21.227],[102.895,21.496],[102.973,21.575],[102.947,21.737],[102.787,21.741],[102.65,21.658],[102.591,21.901],[102.477,21.957],[102.484,22.022],[102.226,22.229],[102.119,22.398],[102.22,22.411],[102.252,22.496],[102.376,22.616],[102.441,22.766],[102.536,22.696],[102.692,22.67],[102.846,22.586],[102.895,22.487],[102.989,22.438],[103.142,22.537],[103.142,22.607],[103.253,22.679],[103.309,22.787],[103.403,22.737],[103.472,22.591],[103.55,22.641],[103.589,22.768],[103.648,22.798],[103.866,22.575],[103.984,22.528],[104.023,22.719],[104.072,22.782],[104.212,22.825],[104.248,22.728],[104.339,22.687],[104.554,22.836],[104.727,22.84],[104.828,22.955],[104.799,23.086],[104.871,23.164],[104.946,23.16],[105.06,23.232],[105.334,23.319],[105.432,23.268],[105.516,23.167],[105.552,23.059],[105.689,23.043],[105.852,22.904],[105.999,22.975],[106.201,22.948],[106.315,22.854],[106.488,22.926],[106.668,22.867],[106.73,22.8],[106.736,22.696],[106.681,22.58],[106.57,22.575],[106.527,22.427],[106.664,22.224],[106.671,22.092],[106.723,22.007],[107.02,21.892],[107.065,21.797],[107.173,21.716],[107.274,21.719],[107.349,21.599],[107.613,21.606],[107.744,21.658],[107.832,21.64],[107.992,21.485],[107.881,21.534],[107.78,21.503],[107.747,21.411],[107.649,21.377],[107.6,21.299],[107.392,21.28],[107.333,21.006],[107.225,20.999],[107.16,20.932],[106.948,20.954],[106.866,20.883],[106.756,20.939],[106.749,20.804],[106.788,20.73],[106.628,20.617],[106.566,20.525],[106.596,20.447],[106.57,20.231],[106.394,20.204],[106.162,19.984],[106.025,19.999],[105.957,19.925],[105.924,19.767],[105.81,19.599],[105.784,19.394],[105.81,19.28],[105.745,19.237],[105.738,19.109],[105.644,19.066],[105.647,18.893],[105.784,18.747],[105.826,18.594],[105.976,18.403],[106.097,18.289],[106.27,18.205],[106.403,18.106],[106.524,17.949],[106.446,17.871],[106.469,17.749],[106.635,17.47],[106.765,17.335],[107.111,17.088],[107.215,16.908],[107.603,16.596],[107.672,16.497],[107.76,16.463],[107.812,16.313],[107.897,16.277],[107.946,16.357],[108.151,16.218],[108.142,16.119],[108.259,16.07],[108.305,15.955],[108.415,15.867],[108.455,15.748],[108.702,15.445],[108.882,15.337],[108.924,15.256],[108.898,15.153],[108.944,14.98],[109.081,14.73],[109.074,14.579],[109.182,14.305],[109.302,13.759],[109.227,13.743],[109.296,13.577],[109.231,13.393],[109.309,13.341],[109.309,13.134],[109.465,12.914],[109.205,12.646],[109.205,12.55],[109.338,12.41],[109.208,12.386],[109.214,12.067],[109.198,11.949],[109.126,11.864],[109.244,11.736],[109.136,11.57],[109.029,11.518],[109.019,11.356],[108.934,11.311],[108.82,11.323],[108.728,11.181],[108.578,11.179],[108.471,11.053],[108.344,10.954],[108.106,10.918],[107.998,10.7],[107.884,10.716],[107.581,10.571],[107.532,10.513],[107.326,10.443],[107.271,10.378],[107.176,10.477],[106.964,10.502],[106.883,10.652],[106.772,10.603],[106.733,10.522],[106.788,10.394],[106.775,10.275],[106.72,10.199],[106.791,10.109],[106.668,10.032],[106.615,9.938],[106.612,9.818],[106.394,9.974],[106.576,9.648],[106.485,9.547],[106.397,9.542],[106.074,9.747],[106.198,9.542],[106.195,9.369],[106.019,9.315],[105.539,9.129],[105.445,9.041],[105.334,8.839],[105.184,8.74],[105.102,8.634],[104.858,8.566],[104.851,8.73],[104.776,8.818],[104.806,9.045],[104.835,9.556],[104.894,9.852],[104.969,9.854],[105.089,9.996],[105.004,10.091],[104.9,10.097],[104.806,10.207],[104.59,10.261],[104.45,10.419],[104.554,10.518],[104.854,10.531],[105.066,10.747],[105.031,10.915],[105.252,10.884],[105.327,10.848],[105.412,10.963],[105.474,10.942],[105.755,11.015],[105.862,10.839],[105.931,10.897],[106.019,10.819],[106.156,10.785],[106.133,10.969],[106.159,11.053],[106.064,11.093],[105.843,11.304],[105.872,11.414],[105.852,11.547],[106.015,11.77],[106.156,11.747],[106.286,11.675],[106.4,11.747],[106.439,11.864],[106.394,11.97],[106.7,11.97],[106.749,12.053],[106.977,12.098],[107.153,12.276],[107.32,12.329],[107.414,12.255],[107.515,12.343],[107.564,12.606],[107.535,12.799],[107.46,13.022],[107.607,13.37],[107.597,13.536],[107.444,13.784],[107.431,13.984],[107.343,14.02],[107.32,14.119],[107.388,14.422],[107.457,14.434],[107.506,14.541],[107.522,14.705]];
  const VN_ISLANDS: [number,number][][] = [
    [[104.082,10.371],[104.085,10.248],[104.01,10.03],[103.938,10.266],[103.857,10.32],[104.003,10.452],[104.082,10.371]],
    [[107.082,20.8],[107.007,20.732],[106.909,20.819],[107.023,20.862],[107.082,20.8]],
    [[107.581,21.211],[107.47,21.096],[107.418,21.197],[107.467,21.276],[107.581,21.211]],
  ];
  function ringToPath(ring: [number,number][]): string {
    return ring.map((c, i) => `${i === 0 ? 'M' : 'L'} ${lngToX(c[0]).toFixed(1)} ${latToY(c[1]).toFixed(1)}`).join(' ') + ' Z';
  }
  const vietnamPath = ringToPath(VN_MAIN);
  const islandPaths = VN_ISLANDS.map(ringToPath);

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

      <!-- Vietnam silhouette (from worldHigh.json TopoJSON) -->
      <path
        d={vietnamPath}
        fill="rgba(34,211,238,0.07)"
        stroke="rgba(34,211,238,0.30)"
        stroke-width="1.5"
        stroke-linejoin="round"
      />
      {#each islandPaths as ip}
        <path d={ip}
          fill="rgba(34,211,238,0.07)"
          stroke="rgba(34,211,238,0.22)"
          stroke-width="1.2"
          stroke-linejoin="round" />
      {/each}

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
