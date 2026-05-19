<script lang="ts">
  import { gateways, satellites } from '$lib/stores';

  const MAP_W = 300;
  const MAP_H = 400;
  const LAT_MIN = 7;
  const LAT_MAX = 24;
  const LNG_MIN = 100;
  const LNG_MAX = 112;

  function latToY(lat: number) {
    return MAP_H - ((lat - LAT_MIN) / (LAT_MAX - LAT_MIN)) * MAP_H;
  }

  function lngToX(lng: number) {
    return ((lng - LNG_MIN) / (LNG_MAX - LNG_MIN)) * MAP_W;
  }
</script>

<div class="chart-container">
  <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">CONSTELLATION VIEW - VIETNAM CORRIDOR</div>
  <div class="relative flex justify-center min-h-[400px]">
    <svg viewBox="0 0 {MAP_W} {MAP_H}" width={MAP_W} height={MAP_H} class="rounded overflow-hidden">
      <rect width={MAP_W} height={MAP_H} fill="#060d14"/>

      {#each [8,10,12,14,16,18,20,22,24] as lat}
        <line x1="0" y1={latToY(lat)} x2={MAP_W} y2={latToY(lat)} stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1"/>
      {/each}
      {#each [102,104,106,108,110] as lng}
        <line x1={lngToX(lng)} y1="0" x2={lngToX(lng)} y2={MAP_H} stroke="#22d3ee" stroke-opacity="0.06" stroke-width="1"/>
      {/each}

      <path
        d="M 185 30 L 205 60 L 220 90 L 215 120 L 200 150 L 210 180 L 215 210 L 200 240 L 185 260 L 165 280 L 150 300 L 140 330 L 125 355 L 115 370 L 110 380 L 100 375 L 108 360 L 115 345 L 120 320 L 130 295 L 145 270 L 155 245 L 165 220 L 160 190 L 155 165 L 165 140 L 170 115 L 160 90 L 155 60 L 165 35 Z"
        fill="rgba(34,211,238,0.05)"
        stroke="rgba(34,211,238,0.2)"
        stroke-width="1.5"
      />

      {#each $gateways as gw}
        {@const x = lngToX(gw.lng)}
        {@const y = latToY(gw.lat)}
        {@const color = gw.status === 'alive' ? '#34d399' : gw.status === 'degraded' ? '#fbbf24' : '#f43f5e'}
        <circle cx={x} cy={y} r="42" fill={color} fill-opacity="0.04" stroke={color} stroke-opacity="0.15" stroke-width="1" stroke-dasharray="4 3"/>
        <circle cx={x} cy={y} r="5" fill={color} fill-opacity="0.9"/>
        <circle cx={x} cy={y} r="9" fill="none" stroke={color} stroke-opacity="0.5" stroke-width="1.5">
          <animate attributeName="r" values="9;14;9" dur="3s" repeatCount="indefinite"/>
          <animate attributeName="stroke-opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite"/>
        </circle>
        <text x={x + 10} y={y + 4} font-family="JetBrains Mono, monospace" font-size="8" fill={color} fill-opacity="0.9">
          {gw.id.replace('GW-', '').replace('-01', '')}
        </text>
      {/each}

      {#each $satellites as sat}
        {@const sx = lngToX(sat.longitude)}
        {@const sy = latToY(sat.latitude)}
        {#if sx >= 0 && sx <= MAP_W && sy >= 0 && sy <= MAP_H}
          <g>
            <line x1={sx-5} y1={sy} x2={sx+5} y2={sy} stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.7"/>
            <line x1={sx} y1={sy-5} x2={sx} y2={sy+5} stroke="#22d3ee" stroke-width="1.5" stroke-opacity="0.7"/>
            <circle cx={sx} cy={sy} r="3" fill="#22d3ee" fill-opacity="0.8"/>
            <circle cx={sx} cy={sy} r="18" fill="none" stroke="#22d3ee" stroke-opacity="0.1" stroke-width="1"/>
            <text x={sx+6} y={sy-4} font-family="JetBrains Mono, monospace" font-size="7" fill="#22d3ee" fill-opacity="0.6">
              {sat.satelliteId}
            </text>
          </g>
        {/if}
      {/each}

      <text x="4" y="395" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">8N</text>
      <text x="4" y="200" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">16N</text>
      <text x="4" y="8" font-family="JetBrains Mono, monospace" font-size="7" fill="rgba(34,211,238,0.3)">24N</text>
    </svg>

    {#if $satellites.length === 0}
      <div class="absolute inset-x-6 top-1/2 -translate-y-1/2 rounded border border-slate-700/60 bg-space-900/90 px-4 py-3 text-center">
        <div class="font-mono text-xs text-slate-500">NO SATELLITE EPHEMERIS LOADED</div>
      </div>
    {/if}

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
