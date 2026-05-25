<script lang="ts">
  // ─────────────────────────────────────────────────────────────────────────────
  // WorldMapModal.svelte — Real-time LEO constellation world tracker
  //
  // Map:        worldHigh.json (TopoJSON) rendered via D3 geoEquirectangular
  // Sats:       TLE-derived SGP4-lite for VNU-LEO internet (24) + weather (18)
  // Interaction: D3 zoom/pan; all stroke-widths & radii divided by k
  // z-index:    9999 (breaks out of parent stacking context)
  // ─────────────────────────────────────────────────────────────────────────────

  import { onDestroy, onMount } from 'svelte';
  import { gateways, sessions, satellites } from '$lib/stores';
  import * as d3Geo from 'd3-geo';
  import * as d3Zoom from 'd3-zoom';
  import * as d3Selection from 'd3-selection';
  import * as topojson from 'topojson-client';
  import worldTopoRaw from '$lib/data/worldHigh.json';

  // ── TopoJSON → GeoJSON (module-level, once) ──────────────────────────────
  const worldTopo = worldTopoRaw as any;
  const _worldGeo = topojson.feature(worldTopo, worldTopo.objects.countries) as any;
  const WORLD_FEATURES: any[] = _worldGeo.type === 'FeatureCollection'
    ? _worldGeo.features
    : [{ type: 'Feature', geometry: _worldGeo, properties: {} }];

  // ── Props ─────────────────────────────────────────────────────────────────
  export let onClose: () => void;

  // ── Map styling constants (per spec) ─────────────────────────────────────
  const BG_COLOR       = '#060d14';
  const COUNTRY_FILL   = '#081b23';
  const COUNTRY_STROKE = '#0e4855';
  const COUNTRY_SW     = 1.5;     // base px, divided by k at render time
  const GRAT_COLOR     = '#25d8f4';
  const GRAT_OPACITY   = 0.14;
  const GRAT_SW        = 0.35;    // base px

  // ── TLE-derived constellation definitions ─────────────────────────────────
  // Parsed from orbit_calc/outputs/simulation/internet/vnu_leo.tle  (24 sats)
  //            orbit_calc/outputs/simulation/weather/vnu_leo.tle   (18 sats)
  // Circular orbit approximation: e≈0, argPerigee=0

  interface SatDef {
    id: string;
    type: 'internet' | 'weather';
    inc: number;   // radians
    raan: number;  // radians
    m0: number;    // mean anomaly at epoch (radians)
    n: number;     // mean motion (rad/s)
  }

  const D2R = Math.PI / 180;
  // Earth sidereal rotation rate (rad/s)
  const EARTH_ROT = 7.2921150e-5;

  // Internet constellation: n = 15.24308387 rev/day
  const N_INT = (15.24308387 * 2 * Math.PI) / 86400;
  // Weather constellation: n = 13.71870588 rev/day
  const N_WEA = (13.71870588 * 2 * Math.PI) / 86400;

  // [id-suffix, RAAN°, M0°]
  const INT_PARAMS: [string, number, number][] = [
    ['0101',  0,   0], ['0102',  0,  90], ['0103',  0, 180], ['0104',  0, 270],
    ['0201', 60,  15], ['0202', 60, 105], ['0203', 60, 195], ['0204', 60, 285],
    ['0301',120,  30], ['0302',120, 120], ['0303',120, 210], ['0304',120, 300],
    ['0401',180,  45], ['0402',180, 135], ['0403',180, 225], ['0404',180, 315],
    ['0501',240,  60], ['0502',240, 150], ['0503',240, 240], ['0504',240, 330],
    ['0601',300,  75], ['0602',300, 165], ['0603',300, 255], ['0604',300, 345],
  ];
  const WEA_PARAMS: [string, number, number][] = [
    ['0101',  0,   0], ['0102',  0, 120], ['0103',  0, 240],
    ['0201', 60,  20], ['0202', 60, 140], ['0203', 60, 260],
    ['0301',120,  40], ['0302',120, 160], ['0303',120, 280],
    ['0401',180,  60], ['0402',180, 180], ['0403',180, 300],
    ['0501',240,  80], ['0502',240, 200], ['0503',240, 320],
    ['0601',300, 100], ['0602',300, 220], ['0603',300, 340],
  ];

  const INC = 53 * D2R;

  function makeDefs(
    params: [string, number, number][],
    n: number,
    type: 'internet' | 'weather'
  ): SatDef[] {
    return params.map(([suffix, raanDeg, m0Deg]) => ({
      id: `${type === 'internet' ? 'I' : 'W'}-VNU-LEO-${suffix}`,
      type, inc: INC,
      raan: raanDeg * D2R,
      m0: m0Deg * D2R,
      n,
    }));
  }

  const ALL_DEFS: SatDef[] = [
    ...makeDefs(INT_PARAMS, N_INT, 'internet'),
    ...makeDefs(WEA_PARAMS, N_WEA, 'weather'),
  ];

  // ── SGP4-lite (circular orbit) ────────────────────────────────────────────
  function satPos(def: SatDef, t: number): [number, number] {
    const M = def.m0 + def.n * t;
    const xOrb = Math.cos(M), yOrb = Math.sin(M);
    const x3 = xOrb, y3 = yOrb * Math.cos(def.inc), z3 = yOrb * Math.sin(def.inc);
    const xEci = x3 * Math.cos(def.raan) - y3 * Math.sin(def.raan);
    const yEci = x3 * Math.sin(def.raan) + y3 * Math.cos(def.raan);
    const theta = EARTH_ROT * t;
    const xEcef =  xEci * Math.cos(theta) + yEci * Math.sin(theta);
    const yEcef = -xEci * Math.sin(theta) + yEci * Math.cos(theta);
    const zEcef = z3;
    let lon = Math.atan2(yEcef, xEcef) * (180 / Math.PI);
    lon = ((lon + 540) % 360) - 180;
    const lat = Math.asin(Math.max(-1, Math.min(1, zEcef))) * (180 / Math.PI);
    return [lon, lat];
  }

  // ── Orbital trail (full period, STEPS segments) ──────────────────────────
  const TRAIL_STEPS = 90;
  function buildTrail(def: SatDef, t0: number): [number, number][] {
    const T = (2 * Math.PI) / def.n;
    const pts: [number, number][] = [];
    for (let i = 0; i <= TRAIL_STEPS; i++) {
      pts.push(satPos(def, t0 + (i / TRAIL_STEPS) * T));
    }
    return pts;
  }

  // ── Ground stations ───────────────────────────────────────────────────────
  const STATIC_GW = [
    { id: 'GW-HAN-01', name: 'HAN', lat: 21.028, lng: 105.854, status: 'alive' },
    { id: 'GW-DAN-01', name: 'DAN', lat: 16.047, lng: 108.206, status: 'alive' },
    { id: 'GW-HCM-01', name: 'HCM', lat: 10.763, lng: 106.660, status: 'alive' },
  ];

  // ── Satellite display state ───────────────────────────────────────────────
  interface AnimSat {
    id: string;
    type: 'internet' | 'weather';
    lon: number; lat: number;
    dLon: number; dLat: number; // display (lerped)
  }

  const LERP = 0.12;
  // Seed initial positions at t=0 so template never renders empty
  let animSats: AnimSat[] = ALL_DEFS.map(d => {
    const [lon, lat] = satPos(d, 0);
    return { id: d.id, type: d.type, lon, lat, dLon: lon, dLat: lat };
  });

  // ── D3 / SVG state ────────────────────────────────────────────────────────
  let svgEl: SVGSVGElement;
  // Start with fallback dimensions; onMount will re-init with real SVG size
  let width = 1200;
  let height = 600;
  // Initialize projection immediately so template expressions never see undefined
  let projection = d3Geo.geoEquirectangular()
    .scale(width / (2 * Math.PI))
    .translate([width / 2, height / 2]);
  let pathGen = d3Geo.geoPath(projection);
  let currentTransform = d3Zoom.zoomIdentity;

  // Cached static SVG strings (world map, graticule) — rebuilt on resize only
  let landPaths: string[] = [];   // one per country feature
  let graticulePath = '';

  // Cached orbital trail paths — rebuilt every TRAIL_TTL seconds or on zoom
  const TRAIL_TTL = 2;
  let trailCache: { id: string; type: string; d: string }[] = [];
  let trailLastBuilt = -Infinity;

  // Simulation clock
  let simTime = 0;
  let animTimer: ReturnType<typeof setInterval> | null = null;
  const REFRESH_MS = 100;

  // Active tab
  let activeTab: 'all' | 'internet' | 'weather' = 'all';

  // ── Projection ────────────────────────────────────────────────────────────
  function initProjection() {
    projection = d3Geo.geoEquirectangular()
      .scale(width / (2 * Math.PI))
      .translate([width / 2, height / 2]);
    pathGen = d3Geo.geoPath(projection);
  }

  function buildStaticPaths() {
    if (!pathGen) return;
    // 1° × 1° graticule (per spec)
    graticulePath = pathGen(d3Geo.geoGraticule().step([1, 1])()) ?? '';
    // Country paths (per-feature so we can style individually; join into array)
    landPaths = WORLD_FEATURES.map(f => pathGen(f) ?? '').filter(Boolean);
  }

  // ── Project lon/lat → screen coords including current zoom transform ──────
  function proj(lon: number, lat: number): [number, number] | null {
    if (!projection) return null;   // guard: should not happen post-init
    const pt = projection([lon, lat]);
    if (!pt) return null;
    return [
      currentTransform.x + pt[0] * currentTransform.k,
      currentTransform.y + pt[1] * currentTransform.k,
    ];
  }

  // ── Trail → SVG path string (antimeridian-safe) ───────────────────────────
  function trailToPath(pts: [number, number][]): string {
    if (!projection) return '';
    const k = currentTransform.k;
    const tx = currentTransform.x, ty = currentTransform.y;
    let d = '', pen = false, prevLon = NaN;
    // Viewport clip (2× margin)
    const xMin = -width, xMax = width * 2, yMin = -height, yMax = height * 2;

    for (const [lon, lat] of pts) {
      if (!isNaN(prevLon) && Math.abs(lon - prevLon) > 150) pen = false;
      prevLon = lon;
      const xy = projection([lon, lat]);
      if (!xy) { pen = false; continue; }
      const px = tx + xy[0] * k, py = ty + xy[1] * k;
      if (px < xMin || px > xMax || py < yMin || py > yMax) { pen = false; continue; }
      d += pen ? `L${px.toFixed(1)},${py.toFixed(1)} ` : `M${px.toFixed(1)},${py.toFixed(1)} `;
      pen = true;
    }
    return d;
  }

  function rebuildTrails() {
    const defs = activeTab === 'all' ? ALL_DEFS
      : ALL_DEFS.filter(d => d.type === activeTab);
    trailCache = defs.map(def => ({
      id: def.id, type: def.type,
      d: trailToPath(buildTrail(def, simTime)),
    }));
    trailLastBuilt = simTime;
  }

  // ── D3 zoom setup ─────────────────────────────────────────────────────────
  function setupZoom() {
    if (!svgEl) return;
    const svg = d3Selection.select(svgEl);
    const zoom = d3Zoom.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.5, 20])
      .on('zoom', (e: d3Zoom.D3ZoomEvent<SVGSVGElement, unknown>) => {
        currentTransform = e.transform;
        trailLastBuilt = -Infinity; // force trail rebuild on zoom
      });
    svg.call(zoom);
    svg.call(zoom.transform, d3Zoom.zoomIdentity);
  }

  // ── Animation tick ────────────────────────────────────────────────────────
  function tick() {
    simTime += REFRESH_MS / 1000;

    // Build WebSocket position override map
    const wsMap = new Map<string, [number, number]>();
    for (const s of $satellites) {
      wsMap.set(s.satelliteId, [s.longitude, s.latitude]);
    }

    // Compute and lerp
    animSats = animSats.map((anim, i) => {
      const def = ALL_DEFS[i];
      const rawKey = def.id.replace(/^[IW]-VNU-LEO-/, 'VNU-LEO-');
      const ws = wsMap.get(rawKey);
      const [tLon, tLat] = ws ?? satPos(def, simTime);

      // Antimeridian-aware longitude lerp
      let dLon = tLon - anim.dLon;
      if (dLon >  180) dLon -= 360;
      if (dLon < -180) dLon += 360;
      const newDLon = ((anim.dLon + dLon * LERP) + 540) % 360 - 180;
      const newDLat = anim.dLat + (tLat - anim.dLat) * LERP;
      return { ...anim, lon: tLon, lat: tLat, dLon: newDLon, dLat: newDLat };
    });

    // Rebuild trails every TRAIL_TTL seconds
    if (simTime - trailLastBuilt >= TRAIL_TTL) rebuildTrails();
  }

  // ── Gateway helpers ───────────────────────────────────────────────────────
  $: gwList = $gateways.length > 0
    ? $gateways
    : STATIC_GW.map(g => ({
        ...g, currentSessions: 0, maxSessions: 0,
        minElevationDeg: 0, antennaGainDbi: 0, beamWidthDeg: 0, altitudeKm: 0,
      }));

  $: activeSatIds = new Set($sessions.map(s => s.satelliteId));

  function nearestSat(gwLat: number, gwLng: number) {
    // Pick the nearest sat from current visible list (any type)
    let best: AnimSat | null = null, bestD = Infinity;
    for (const s of animSats) {
      const d = Math.hypot(s.dLat - gwLat, s.dLon - gwLng);
      if (d < bestD) { bestD = d; best = s; }
    }
    return best;
  }

  function gwColor(status: string) {
    if (status === 'alive') return '#34d399';
    if (status === 'degraded') return '#fbbf24';
    return '#fb7185';
  }

  // ── Visible sats for current tab ──────────────────────────────────────────
  $: visibleSats = activeTab === 'all' ? animSats
    : animSats.filter(s => s.type === activeTab);

  // ── Sat colour ────────────────────────────────────────────────────────────
  const SAT_CLR: Record<string, string> = {
    internet: '#25d8f4',
    weather:  '#a78bfa',
  };

  // ── Lifecycle ─────────────────────────────────────────────────────────────
  onMount(() => {
    width  = svgEl.clientWidth  || window.innerWidth;
    height = svgEl.clientHeight || window.innerHeight - 50;
    initProjection();
    buildStaticPaths();
    setupZoom();

    // Seed positions
    animSats = ALL_DEFS.map(def => {
      const [lon, lat] = satPos(def, 0);
      return { id: def.id, type: def.type, lon, lat, dLon: lon, dLat: lat };
    });
    rebuildTrails();

    animTimer = setInterval(tick, REFRESH_MS);
  });

  onDestroy(() => {
    if (animTimer) clearInterval(animTimer);
  });

  // Re-build trails when tab changes
  $: { activeTab; trailLastBuilt = -Infinity; }

  // Keyboard close
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }

  // Counts
  $: intCount = animSats.filter(s => s.type === 'internet').length;
  $: weaCount = animSats.filter(s => s.type === 'weather').length;
  $: k = currentTransform.k;
</script>

<svelte:window on:keydown={handleKeydown} />

<!--
  position:fixed + z-index:9999 ensures this renders above the parent
  ConstellationModal (z-40) regardless of stacking context.
-->
<div class="wm-root" role="dialog" aria-label="World Constellation Map" aria-modal="true">

  <!-- ── Header ── -->
  <header class="wm-header">
    <div class="wm-header-left">
      <span class="wm-title">WORLD CONSTELLATION TRACKER</span>
      <span class="wm-sub">
        VNU-LEO &nbsp;·&nbsp; {intCount} INTERNET &nbsp;+&nbsp; {weaCount} WEATHER
      </span>
    </div>

    <div class="wm-tabs">
      {#each (['all', 'internet', 'weather'] as const) as tab}
        <button
          id="wm-tab-{tab}"
          class="wm-tab"
          class:active={activeTab === tab}
          on:click={() => { activeTab = tab; }}
        >
          {tab === 'all' ? 'ALL' : tab.toUpperCase()}
        </button>
      {/each}
    </div>

    <div class="wm-header-right">
      <span class="wm-hint">drag · scroll to zoom · ESC to close</span>
      <button
        type="button"
        id="world-map-close"
        class="wm-close"
        on:click={onClose}
        aria-label="Close world map"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <line x1="3" y1="3" x2="13" y2="13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <line x1="13" y1="3" x2="3" y2="13" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
  </header>

  <!-- ── Map ── -->
  <div class="wm-canvas">
    <svg
      bind:this={svgEl}
      id="world-map-svg"
      class="wm-svg"
      width="100%"
      height="100%"
      style="background:{BG_COLOR};"
    >
      <!-- Graticule: 1° step, scaled stroke -->
      {#if graticulePath}
        <path
          d={graticulePath}
          fill="none"
          stroke={GRAT_COLOR}
          stroke-opacity={GRAT_OPACITY}
          stroke-width={GRAT_SW / k}
        />
      {/if}

      <!-- Country fills + borders — zoom-scaled stroke -->
      {#each landPaths as d}
        <path
          {d}
          fill={COUNTRY_FILL}
          stroke={COUNTRY_STROKE}
          stroke-width={COUNTRY_SW / k}
          stroke-linejoin="round"
        />
      {/each}

      <!-- Orbital trails (from cache) -->
      {#each trailCache as tr (tr.id)}
        {#if tr.d}
          <path
            d={tr.d}
            fill="none"
            stroke={tr.type === 'internet' ? '#25d8f4' : '#a78bfa'}
            stroke-opacity={tr.type === 'internet' ? 0.22 : 0.18}
            stroke-width={0.7 / k}
            stroke-dasharray="{6 / k} {5 / k}"
          />
        {/if}
      {/each}

      <!-- Link lines: gateway → nearest sat -->
      {#each gwList as gw}
        {@const nearest = nearestSat(gw.lat, gw.lng)}
        {#if nearest}
          {@const p1 = proj(gw.lng, gw.lat)}
          {@const p2 = proj(nearest.dLon, nearest.dLat)}
          {#if p1 && p2}
            <line
              x1={p1[0]} y1={p1[1]}
              x2={p2[0]} y2={p2[1]}
              stroke={gwColor(gw.status)}
              stroke-opacity="0.55"
              stroke-width={1.1 / k}
              stroke-dasharray="{6 / k} {4 / k}"
            >
              <animate
                attributeName="stroke-dashoffset"
                values="0;{-20 / k}"
                dur="1.4s"
                repeatCount="indefinite"
              />
            </line>
          {/if}
        {/if}
      {/each}

      <!-- Satellites -->
      {#each visibleSats as sat (sat.id)}
        {@const pt = proj(sat.dLon, sat.dLat)}
        {#if pt}
          {@const c = SAT_CLR[sat.type]}
          {@const r  = 3 / k}
          {@const cr = 6 / k}
          {@const rg = 13 / k}
          <g>
            <!-- Footprint ring -->
            <circle cx={pt[0]} cy={pt[1]} r={rg}
              fill="none" stroke={c} stroke-opacity="0.10" stroke-width={0.7 / k} />
            <!-- Crosshair -->
            <line x1={pt[0]-cr} y1={pt[1]} x2={pt[0]+cr} y2={pt[1]}
              stroke={c} stroke-opacity="0.85" stroke-width={1.3 / k} />
            <line x1={pt[0]} y1={pt[1]-cr} x2={pt[0]} y2={pt[1]+cr}
              stroke={c} stroke-opacity="0.85" stroke-width={1.3 / k} />
            <!-- Core dot -->
            <circle cx={pt[0]} cy={pt[1]} r={r} fill={c} fill-opacity="0.95" />
            <!-- Pulse -->
            <circle cx={pt[0]} cy={pt[1]} r={r * 2} fill="none"
              stroke={c} stroke-opacity="0.45" stroke-width={0.8 / k}>
              <animate attributeName="r"
                values="{r*2};{r*5};{r*2}" dur="2.8s" repeatCount="indefinite"/>
              <animate attributeName="stroke-opacity"
                values="0.45;0;0.45" dur="2.8s" repeatCount="indefinite"/>
            </circle>
            <!-- Label (only when zoomed in) -->
            {#if k > 1.8}
              <text
                x={pt[0] + r + 2/k}
                y={pt[1] - r - 1/k}
                font-family="JetBrains Mono,monospace"
                font-size={8 / k}
                fill={c}
                fill-opacity="0.75"
              >{sat.id.replace(/^[IW]-VNU-LEO-/, '')}</text>
            {/if}
          </g>
        {/if}
      {/each}

      <!-- Ground stations -->
      {#each gwList as gw}
        {@const pt = proj(gw.lng, gw.lat)}
        {#if pt}
          {@const c = gwColor(gw.status)}
          {@const dr = 5 / k}
          {@const rg = 18 / k}
          <circle cx={pt[0]} cy={pt[1]} r={rg}
            fill={c} fill-opacity="0.05"
            stroke={c} stroke-opacity="0.18" stroke-width={0.9 / k}
            stroke-dasharray="{4/k} {3/k}" />
          <circle cx={pt[0]} cy={pt[1]} r={dr} fill={c} fill-opacity="0.92" />
          <circle cx={pt[0]} cy={pt[1]} r={dr*2} fill="none"
            stroke={c} stroke-opacity="0.5" stroke-width={1.4 / k}>
            <animate attributeName="r"
              values="{dr*2};{dr*4};{dr*2}" dur="3s" repeatCount="indefinite"/>
            <animate attributeName="stroke-opacity"
              values="0.5;0;0.5" dur="3s" repeatCount="indefinite"/>
          </circle>
          <text
            x={pt[0] + dr + 2/k}
            y={pt[1] + dr * 0.4}
            font-family="JetBrains Mono,monospace"
            font-size={9 / k}
            fill={c} fill-opacity="0.92"
          >{gw.name ?? gw.id}</text>
        {/if}
      {/each}
    </svg>

    <!-- HUD: top-left counts -->
    <div class="wm-hud wm-hud-tl">
      <div class="wm-badge" style="color:#25d8f4;border-color:#25d8f440;">
        <span class="wm-dot" style="background:#25d8f4;"></span>
        {intCount} INTERNET
      </div>
      <div class="wm-badge" style="color:#a78bfa;border-color:#a78bfa40;">
        <span class="wm-dot" style="background:#a78bfa;"></span>
        {weaCount} WEATHER
      </div>
      <div class="wm-badge" style="color:#34d399;border-color:#34d39940;">
        {gwList.filter(g => g.status === 'alive').length}/{gwList.length} GW ONLINE
      </div>
    </div>

    <!-- HUD: bottom-right legend -->
    <div class="wm-hud wm-hud-br">
      <div class="wm-leg-title">LEGEND</div>
      <div class="wm-leg-row"><span class="wm-leg-line" style="background:#25d8f4;"></span>Internet satellite</div>
      <div class="wm-leg-row"><span class="wm-leg-line" style="background:#a78bfa;"></span>Weather satellite</div>
      <div class="wm-leg-row"><span class="wm-leg-dash" style="border-color:#25d8f4;"></span>Orbital trail</div>
      <div class="wm-leg-row"><span class="wm-leg-dot" style="background:#34d399;"></span>Gateway alive</div>
      <div class="wm-leg-row"><span class="wm-leg-dot" style="background:#fbbf24;"></span>Gateway degraded</div>
      <div class="wm-leg-row"><span class="wm-leg-dash" style="border-color:#34d399;"></span>Uplink / downlink</div>
    </div>

    <!-- HUD: bottom-left projection label -->
    <div class="wm-hud wm-hud-bl">
      <span class="wm-proj">EQUIRECTANGULAR · 1° GRID · SGP4-LITE · worldHigh.json</span>
    </div>
  </div>
</div>

<style>
  /* Break out of any parent stacking context */
  .wm-root {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: #060d14;
    display: flex;
    flex-direction: column;
    font-family: 'JetBrains Mono', monospace;
  }

  /* Header */
  .wm-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 10px 18px;
    border-bottom: 1px solid #0e4855;
  }
  .wm-header-left { display: flex; flex-direction: column; gap: 2px; }
  .wm-title {
    font-size: 11px; font-weight: 700; letter-spacing: .12em; color: #25d8f4;
    text-shadow: 0 0 10px rgba(37,216,244,.4);
  }
  .wm-sub { font-size: 10px; color: #3a6070; letter-spacing: .06em; }

  /* Tabs */
  .wm-tabs { display: flex; gap: 4px; margin-left: 10px; }
  .wm-tab {
    font-family: 'JetBrains Mono', monospace;
    font-size: 9px; letter-spacing: .10em;
    padding: 4px 10px;
    border-radius: 4px;
    border: 1px solid #0e4855;
    background: transparent;
    color: #3a6070;
    cursor: pointer;
    transition: color .15s, border-color .15s, background .15s;
  }
  .wm-tab:hover { color: #25d8f4; border-color: #25d8f440; }
  .wm-tab.active {
    color: #25d8f4; border-color: #25d8f4;
    background: rgba(37,216,244,.08);
  }

  /* Header right */
  .wm-header-right { margin-left: auto; display: flex; align-items: center; gap: 12px; }
  .wm-hint { font-size: 9px; color: #1e3540; letter-spacing: .08em; }
  .wm-close {
    display: flex; align-items: center; justify-content: center;
    width: 32px; height: 32px; border-radius: 6px;
    background: transparent;
    border: 1px solid rgba(251,113,133,.15);
    color: rgba(251,113,133,.55);
    cursor: pointer;
    transition: color .15s, background .15s, border-color .15s;
    padding: 0;
  }
  .wm-close:hover {
    color: #fb7185; background: rgba(251,113,133,.09);
    border-color: rgba(251,113,133,.30);
  }
  .wm-close svg { display: block; pointer-events: none; }

  /* Canvas */
  .wm-canvas { flex: 1; overflow: hidden; position: relative; }
  .wm-svg {
    display: block; width: 100%; height: 100%;
    cursor: grab; user-select: none;
  }
  .wm-svg:active { cursor: grabbing; }

  /* HUDs */
  .wm-hud {
    position: absolute;
    background: rgba(6,13,20,.85);
    backdrop-filter: blur(8px);
    border: 1px solid #0e4855;
    border-radius: 8px;
    padding: 10px 14px;
    pointer-events: none;
  }
  .wm-hud-tl { top: 12px; left: 12px; display: flex; flex-direction: column; gap: 5px; }
  .wm-hud-br { bottom: 14px; right: 14px; }
  .wm-hud-bl { bottom: 14px; left: 12px; }

  /* Badges */
  .wm-badge {
    font-size: 9px; letter-spacing: .09em;
    padding: 2px 8px; border-radius: 4px; border: 1px solid;
    display: flex; align-items: center; gap: 5px;
  }
  .wm-dot { width: 6px; height: 6px; border-radius: 50%; display: inline-block; }

  /* Legend */
  .wm-leg-title { font-size: 8px; letter-spacing: .15em; color: #1e3540; margin-bottom: 8px; }
  .wm-leg-row {
    display: flex; align-items: center; gap: 8px;
    font-size: 9px; color: #4a7080; margin-bottom: 5px;
  }
  .wm-leg-line { display: inline-block; width: 20px; height: 1.5px; border-radius: 1px; flex-shrink: 0; }
  .wm-leg-dash { display: inline-block; width: 20px; height: 0; border-top: 2px dashed; flex-shrink: 0; }
  .wm-leg-dot  { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }

  /* Projection label */
  .wm-proj { font-size: 8px; letter-spacing: .10em; color: #1a2e38; }
</style>
