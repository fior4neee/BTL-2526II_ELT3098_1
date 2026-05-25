<script lang="ts">
  // ─────────────────────────────────────────────────────────────────────────────
  // WorldMapModal.svelte  —  Real-time LEO constellation tracker
  //
  // Projection:  Equirectangular (geoEquirectangular)
  // Map data:    worldHigh.json (TopoJSON, country outlines only)
  // Satellites:  Two constellations parsed from TLE constants:
  //              • Internet  (24 sats, ~500 km, Walker 24/6/1, inc 53°)
  //              • Weather   (18 sats, ~1000 km, Walker 18/6/1, inc 53°)
  // Live telemetry: merged from WebSocket store ($satellites / $sessions / $gateways)
  //                 Falls back to local SGP4-lite animation when no WS data.
  // Zoom UX:     D3 zoom; stroke-widths and radii divided by k so they stay
  //              pixel-sharp at any zoom level.
  // ─────────────────────────────────────────────────────────────────────────────

  import { onDestroy, onMount } from 'svelte';
  import { gateways, sessions, satellites } from '$lib/stores';
  import * as d3Geo from 'd3-geo';
  import * as d3Zoom from 'd3-zoom';
  import * as d3Selection from 'd3-selection';
  import * as topojson from 'topojson-client';
  import worldTopoRaw from '$lib/data/worldHigh.json';

  // ── TopoJSON decode (module-level, once) ─────────────────────────────────
  const worldTopo = worldTopoRaw as any;
  const _worldGeo = topojson.feature(worldTopo, worldTopo.objects.countries) as any;
  const WORLD_FEATURES: any[] = _worldGeo.type === 'FeatureCollection'
    ? _worldGeo.features
    : [{ type: 'Feature', geometry: _worldGeo, properties: {} }];

  // ── Props ─────────────────────────────────────────────────────────────────
  export let onClose: () => void;

  // ── Styling constants (per spec) ──────────────────────────────────────────
  const BG_COLOR         = '#060d14';
  const WATER_COLOR      = 'none';          // transparent — bg shows through
  const COUNTRY_FILL     = '#081b23';
  const COUNTRY_STROKE   = '#0e4855';
  const COUNTRY_SW       = 1.50;            // base px (divided by k at render)
  const GRAT_COLOR       = '#25d8f4';
  const GRAT_OPACITY     = 0.18;
  const GRAT_SW          = 0.4;             // base px

  // ── TLE-derived constellation definitions ─────────────────────────────────
  // Parsed from orbit_calc/outputs/simulation/internet/vnu_leo.tle  (24 sats)
  //            orbit_calc/outputs/simulation/weather/vnu_leo.tle   (18 sats)
  //
  // SGP4-lite for circular orbits (e≈0):
  //   mean motion n (rev/day) → orbital period T_s
  //   RAAN Ω, inclination i, mean anomaly M₀ from TLE line 2
  //   Earth rotation rate  ω_E = 2π / 86400 rad/s

  interface SatDef {
    id: string;
    inc: number;   // rad
    raan: number;  // rad
    m0: number;    // rad  mean anomaly at epoch
    n: number;     // rad/s  mean motion
    type: 'internet' | 'weather';
  }

  // Helper: degrees → radians
  const D2R = Math.PI / 180;

  // Internet constellation  — n = 15.24308387 rev/day
  const N_INTERNET = (15.24308387 * 2 * Math.PI) / 86400; // rad/s
  const INTERNET_PARAMS: [string, number, number, number][] = [
    // [id, RAAN°, M0°, plane-seat]
    ['VNU-LEO-0101',   0,   0, 0],
    ['VNU-LEO-0102',   0,  90, 0],
    ['VNU-LEO-0103',   0, 180, 0],
    ['VNU-LEO-0104',   0, 270, 0],
    ['VNU-LEO-0201',  60,  15, 1],
    ['VNU-LEO-0202',  60, 105, 1],
    ['VNU-LEO-0203',  60, 195, 1],
    ['VNU-LEO-0204',  60, 285, 1],
    ['VNU-LEO-0301', 120,  30, 2],
    ['VNU-LEO-0302', 120, 120, 2],
    ['VNU-LEO-0303', 120, 210, 2],
    ['VNU-LEO-0304', 120, 300, 2],
    ['VNU-LEO-0401', 180,  45, 3],
    ['VNU-LEO-0402', 180, 135, 3],
    ['VNU-LEO-0403', 180, 225, 3],
    ['VNU-LEO-0404', 180, 315, 3],
    ['VNU-LEO-0501', 240,  60, 4],
    ['VNU-LEO-0502', 240, 150, 4],
    ['VNU-LEO-0503', 240, 240, 4],
    ['VNU-LEO-0504', 240, 330, 4],
    ['VNU-LEO-0601', 300,  75, 5],
    ['VNU-LEO-0602', 300, 165, 5],
    ['VNU-LEO-0603', 300, 255, 5],
    ['VNU-LEO-0604', 300, 345, 5],
  ];

  // Weather constellation  — n = 13.71870588 rev/day
  const N_WEATHER = (13.71870588 * 2 * Math.PI) / 86400;
  const WEATHER_PARAMS: [string, number, number, number][] = [
    ['VNU-LEO-0101',   0,   0, 0],
    ['VNU-LEO-0102',   0, 120, 0],
    ['VNU-LEO-0103',   0, 240, 0],
    ['VNU-LEO-0201',  60,  20, 1],
    ['VNU-LEO-0202',  60, 140, 1],
    ['VNU-LEO-0203',  60, 260, 1],
    ['VNU-LEO-0301', 120,  40, 2],
    ['VNU-LEO-0302', 120, 160, 2],
    ['VNU-LEO-0303', 120, 280, 2],
    ['VNU-LEO-0401', 180,  60, 3],
    ['VNU-LEO-0402', 180, 180, 3],
    ['VNU-LEO-0403', 180, 300, 3],
    ['VNU-LEO-0501', 240,  80, 4],
    ['VNU-LEO-0502', 240, 200, 4],
    ['VNU-LEO-0503', 240, 320, 4],
    ['VNU-LEO-0601', 300, 100, 5],
    ['VNU-LEO-0602', 300, 220, 5],
    ['VNU-LEO-0603', 300, 340, 5],
  ];

  const INC_DEG = 53;

  function buildSatDefs(
    params: [string, number, number, number][],
    n: number,
    type: 'internet' | 'weather'
  ): SatDef[] {
    return params.map(([rawId, raanDeg, m0Deg]) => ({
      id: `${type === 'internet' ? 'I' : 'W'}-${rawId}`,
      inc: INC_DEG * D2R,
      raan: raanDeg * D2R,
      m0: m0Deg * D2R,
      n,
      type,
    }));
  }

  const ALL_SAT_DEFS: SatDef[] = [
    ...buildSatDefs(INTERNET_PARAMS, N_INTERNET, 'internet'),
    ...buildSatDefs(WEATHER_PARAMS,  N_WEATHER,  'weather'),
  ];

  // ── Ground stations ───────────────────────────────────────────────────────
  const GROUND_STATIONS = [
    { id: 'GW-HAN-01', name: 'HAN', lat: 21.028, lng: 105.854 },
    { id: 'GW-DAN-01', name: 'DAN', lat: 16.047, lng: 108.206 },
    { id: 'GW-HCM-01', name: 'HCM', lat: 10.763, lng: 106.660 },
  ];

  // ── SGP4-lite: compute geodetic lat/lon from SatDef at time t_s ───────────
  // Uses circular orbit approximation (e=0, argument of perigee=0).
  // Earth rotation: ω_E ≈ 7.2921150e-5 rad/s  (sidereal)
  const EARTH_ROT_RATE = 7.2921150e-5; // rad/s

  function satPosition(def: SatDef, t: number): [number, number] {
    // Mean anomaly at time t
    const M = def.m0 + def.n * t;
    // Position in orbital plane (circular → E = M)
    const xOrb = Math.cos(M);
    const yOrb = Math.sin(M);
    // Rotate by inclination (x stays, y splits into y·cos(i) and z=y·sin(i))
    const x3 = xOrb;
    const y3 = yOrb * Math.cos(def.inc);
    const z3 = yOrb * Math.sin(def.inc);
    // Rotate by RAAN around Z axis to get ECI
    const xEci = x3 * Math.cos(def.raan) - y3 * Math.sin(def.raan);
    const yEci = x3 * Math.sin(def.raan) + y3 * Math.cos(def.raan);
    const zEci = z3;
    // Subtract Earth's rotation (GMST approximation: 0 at t=0)
    const theta = EARTH_ROT_RATE * t;
    const xEcef = xEci * Math.cos(theta) + yEci * Math.sin(theta);
    const yEcef = -xEci * Math.sin(theta) + yEci * Math.cos(theta);
    const zEcef = zEci;
    // Geodetic coordinates
    let lon = Math.atan2(yEcef, xEcef) * (180 / Math.PI);
    lon = ((lon + 540) % 360) - 180; // wrap to [-180, 180]
    const lat = Math.asin(Math.max(-1, Math.min(1, zEcef))) * (180 / Math.PI);
    return [lon, lat];
  }

  // ── Orbital trail: one full period, TRAIL_STEPS segments ─────────────────
  const TRAIL_STEPS = 90;

  function buildTrail(def: SatDef, t0: number): [number, number][] {
    const T = (2 * Math.PI) / def.n; // orbital period in seconds
    const pts: [number, number][] = [];
    for (let i = 0; i <= TRAIL_STEPS; i++) {
      const t = t0 + (i / TRAIL_STEPS) * T;
      pts.push(satPosition(def, t));
    }
    return pts;
  }

  // ── State ─────────────────────────────────────────────────────────────────
  let svgEl: SVGSVGElement;
  let width = 0;
  let height = 0;

  // Simulation time (seconds since mount)
  let simTime = 0;
  let animTimer: ReturnType<typeof setInterval> | null = null;
  const REFRESH_MS = 100; // 10 fps

  // Zoom transform
  let currentTransform = d3Zoom.zoomIdentity;

  // D3 projection + path
  let projection: d3Geo.GeoProjection;
  let pathGen: d3Geo.GeoPath;

  // Pre-built static SVG path strings (rebuilt on resize)
  let landPath = '';
  let graticulePath = '';

  // Satellite display state (interpolated positions)
  interface AnimSat {
    id: string;
    type: 'internet' | 'weather';
    lon: number;
    lat: number;
    displayLon: number;
    displayLat: number;
  }

  const LERP = 0.12;
  let animSats: AnimSat[] = ALL_SAT_DEFS.map(d => ({
    id: d.id, type: d.type,
    lon: 0, lat: 0, displayLon: 0, displayLat: 0,
  }));

  // Active tab: 'all' | 'internet' | 'weather'
  let activeTab: 'all' | 'internet' | 'weather' = 'all';

  // ── Reactive: which sats are "in session" from Core Network store ─────────
  $: activeSatIds = new Set($sessions.map(s => s.satelliteId));

  // Gateway list from store (fallback to static)
  $: gwList = $gateways.length > 0
    ? $gateways
    : GROUND_STATIONS.map(g => ({
        id: g.id, name: g.name, lat: g.lat, lng: g.lng, status: 'alive' as const,
        currentSessions: 0, maxSessions: 0, minElevationDeg: 0,
        antennaGainDbi: 0, beamWidthDeg: 0, altitudeKm: 0,
      }));

  // ── Filtered sat list for current tab ────────────────────────────────────
  $: visibleSats = activeTab === 'all'
    ? animSats
    : animSats.filter(s => s.type === activeTab);

  $: visibleDefs = activeTab === 'all'
    ? ALL_SAT_DEFS
    : ALL_SAT_DEFS.filter(d => d.type === activeTab);

  // ── Projection setup ─────────────────────────────────────────────────────
  function initProjection() {
    projection = d3Geo.geoEquirectangular()
      .scale(width / (2 * Math.PI))
      .translate([width / 2, height / 2]);
    pathGen = d3Geo.geoPath(projection);
  }

  // ── Build static paths (map + graticule) ──────────────────────────────────
  function renderStaticPaths() {
    if (!pathGen) return;
    // 1-degree step graticule (per spec)
    const grat = d3Geo.geoGraticule().step([1, 1]);
    graticulePath = pathGen(grat()) ?? '';
    // Country fills
    landPath = WORLD_FEATURES.map(f => pathGen(f) ?? '').join(' ');
  }

  // ── Project a [lon, lat] point through projection + current zoom ──────────
  function project(lon: number, lat: number): [number, number] | null {
    const pt = projection([lon, lat]);
    if (!pt) return null;
    return [
      currentTransform.x + pt[0] * currentTransform.k,
      currentTransform.y + pt[1] * currentTransform.k,
    ];
  }

  // ── Convert trail [lon,lat][] → SVG path string, breaking at dateline ─────
  // Anti-meridian rule: if consecutive lon jump > 180°, lift the pen.
  // Also clips to the extended viewport so path commands stay finite.
  function trailToPath(pts: [number, number][]): string {
    if (!projection) return '';
    let d = '';
    let penDown = false;
    let prevLon = NaN;
    const k = currentTransform.k;
    const tx = currentTransform.x;
    const ty = currentTransform.y;
    // Extended viewport clip bounds (2× margin)
    const xMin = -width * 0.5;
    const xMax = width * 1.5;
    const yMin = -height * 0.5;
    const yMax = height * 1.5;

    for (const [lon, lat] of pts) {
      // Break at antimeridian crossing
      if (!isNaN(prevLon) && Math.abs(lon - prevLon) > 150) {
        penDown = false;
      }
      prevLon = lon;

      const xy = projection([lon, lat]);
      if (!xy) { penDown = false; continue; }
      const px = tx + xy[0] * k;
      const py = ty + xy[1] * k;

      // Basic viewport cull
      if (px < xMin || px > xMax || py < yMin || py > yMax) {
        penDown = false;
        continue;
      }

      if (!penDown) {
        d += `M ${px.toFixed(1)} ${py.toFixed(1)} `;
        penDown = true;
      } else {
        d += `L ${px.toFixed(1)} ${py.toFixed(1)} `;
      }
    }
    return d;
  }

  // ── Memoised trail cache (rebuilt every TRAIL_REBUILD_INTERVAL ms) ─────────
  const TRAIL_REBUILD_INTERVAL_MS = 2000; // trails recomputed every 2 s
  let lastTrailRebuild = -Infinity;
  let cachedTrails: { id: string; type: string; path: string }[] = [];

  function rebuildTrails() {
    cachedTrails = visibleDefs.map(def => ({
      id: def.id,
      type: def.type,
      path: trailToPath(buildTrail(def, simTime)),
    }));
    lastTrailRebuild = simTime;
  }

  // ── D3 zoom setup ─────────────────────────────────────────────────────────
  function setupZoom() {
    if (!svgEl) return;
    const svg = d3Selection.select(svgEl);
    const zoom = d3Zoom.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.6, 16])
      .on('zoom', (event: d3Zoom.D3ZoomEvent<SVGSVGElement, unknown>) => {
        currentTransform = event.transform;
        // Mark trails as stale so they rebuild with new transform
        lastTrailRebuild = -Infinity;
      });
    svg.call(zoom);
    svg.call(zoom.transform, d3Zoom.zoomIdentity);
  }

  // ── Animation tick ────────────────────────────────────────────────────────
  function tick() {
    simTime += REFRESH_MS / 1000;

    // Compute raw positions for all sats
    const rawPositions = ALL_SAT_DEFS.map(def => satPosition(def, simTime));

    // Merge with WebSocket data if available
    const wsMap = new Map<string, { lat: number; lon: number }>();
    for (const s of $satellites) {
      wsMap.set(s.satelliteId, { lat: s.latitude, lon: s.longitude });
    }

    animSats = animSats.map((anim, i) => {
      const raw = rawPositions[i];
      // Use WS position if available and recent
      const ws = wsMap.get(ALL_SAT_DEFS[i].id.replace(/^[IW]-/, ''));
      const targetLon = ws ? ws.lon : raw[0];
      const targetLat = ws ? ws.lat : raw[1];

      // Lerp — handle antimeridian wrap in lon
      let dLon = targetLon - anim.displayLon;
      if (dLon > 180) dLon -= 360;
      if (dLon < -180) dLon += 360;
      const newDisplayLon = ((anim.displayLon + dLon * LERP) + 540) % 360 - 180;
      const newDisplayLat = anim.displayLat + (targetLat - anim.displayLat) * LERP;

      return {
        ...anim,
        lon: targetLon,
        lat: targetLat,
        displayLon: newDisplayLon,
        displayLat: newDisplayLat,
      };
    });

    // Rebuild trails if stale
    if (simTime - lastTrailRebuild > TRAIL_REBUILD_INTERVAL_MS / 1000) {
      rebuildTrails();
    }
  }

  // ── Nearest active satellite for a gateway ────────────────────────────────
  function nearestActiveSat(gwLat: number, gwLng: number) {
    let best: AnimSat | null = null;
    let bestD = Infinity;
    for (const s of animSats) {
      // Only match with WS-active satellites (fallback: show all)
      const active = activeSatIds.size === 0 || activeSatIds.has(s.id) || activeSatIds.has(s.id.replace(/^[IW]-/, ''));
      if (!active) continue;
      // Great-circle approximation (good enough for rendering)
      const d = Math.hypot(s.displayLat - gwLat, s.displayLon - gwLng);
      if (d < bestD) { bestD = d; best = s; }
    }
    return best;
  }

  // ── Gateway status → color ────────────────────────────────────────────────
  function gwColor(status: string) {
    if (status === 'alive')    return '#34d399';
    if (status === 'degraded') return '#fbbf24';
    return '#fb7185';
  }

  // ── Sat colour by type ────────────────────────────────────────────────────
  const SAT_COLOR: Record<string, string> = {
    internet: '#25d8f4',  // cyan
    weather:  '#a78bfa',  // violet
  };

  // ── Keyboard close ────────────────────────────────────────────────────────
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }

  // ── Lifecycle ─────────────────────────────────────────────────────────────
  onMount(() => {
    width  = svgEl.clientWidth  || 1200;
    height = svgEl.clientHeight || 680;
    initProjection();
    renderStaticPaths();
    setupZoom();

    // Seed initial positions
    animSats = ALL_SAT_DEFS.map((def, i) => {
      const [lon, lat] = satPosition(def, 0);
      return { id: def.id, type: def.type, lon, lat, displayLon: lon, displayLat: lat };
    });
    rebuildTrails();

    animTimer = setInterval(tick, REFRESH_MS);
  });

  onDestroy(() => {
    if (animTimer) clearInterval(animTimer);
  });

  // ── Derived counts ────────────────────────────────────────────────────────
  $: internetCount = animSats.filter(s => s.type === 'internet').length;
  $: weatherCount  = animSats.filter(s => s.type === 'weather').length;

  // ── Zoom-scale for pass-through into template ─────────────────────────────
  $: k = currentTransform.k;
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- ────────────────────────────────────────────────────────────────────────── -->
<!-- Full-screen overlay                                                        -->
<!-- ────────────────────────────────────────────────────────────────────────── -->
<div
  class="wm-overlay"
  role="dialog"
  aria-label="World Constellation Map"
  aria-modal="true"
>
  <!-- ── Header ──────────────────────────────────────────────────────────── -->
  <header class="wm-header">
    <div class="wm-header-left">
      <div class="wm-title">WORLD CONSTELLATION TRACKER</div>
      <div class="wm-subtitle">
        VNU-LEO &nbsp;·&nbsp; {internetCount} INTERNET &nbsp;+&nbsp; {weatherCount} WEATHER SATS
      </div>
    </div>

    <!-- Tab switcher -->
    <div class="wm-tabs">
      {#each (['all', 'internet', 'weather'] as const) as tab}
        <button
          id="wm-tab-{tab}"
          class="wm-tab"
          class:wm-tab--active={activeTab === tab}
          on:click={() => { activeTab = tab; lastTrailRebuild = -Infinity; }}
        >
          {tab === 'all' ? 'ALL' : tab === 'internet' ? 'INTERNET' : 'WEATHER'}
        </button>
      {/each}
    </div>

    <div class="wm-header-right">
      <span class="wm-hint">drag · scroll to zoom</span>
      <button
        type="button"
        id="world-map-close"
        class="wm-close-btn"
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

  <!-- ── Map canvas ──────────────────────────────────────────────────────── -->
  <div class="wm-canvas-wrapper">
    <svg
      bind:this={svgEl}
      id="world-map-svg"
      class="wm-svg"
      width="100%"
      height="100%"
      style="background: {BG_COLOR};"
    >
      <!-- Water: transparent (bg colour shows through) -->
      <!-- Graticule — 1° step, scaled stroke -->
      {#if graticulePath}
        <path
          d={graticulePath}
          fill="none"
          stroke={GRAT_COLOR}
          stroke-opacity={GRAT_OPACITY}
          stroke-width={GRAT_SW / k}
        />
      {/if}

      <!-- Country fills + borders (zoom-scaled stroke) -->
      {#each WORLD_FEATURES as feature}
        {@const d = pathGen ? (pathGen(feature) ?? '') : ''}
        {#if d}
          <path
            {d}
            fill={COUNTRY_FILL}
            stroke={COUNTRY_STROKE}
            stroke-width={COUNTRY_SW / k}
            stroke-linejoin="round"
          />
        {/if}
      {/each}

      <!-- Orbital trails (rebuilt every 2 s or on zoom change) -->
      {#each cachedTrails as trail (trail.id)}
        {#if trail.path}
          {@const isInternet = trail.type === 'internet'}
          <path
            d={trail.path}
            fill="none"
            stroke={isInternet ? '#25d8f4' : '#a78bfa'}
            stroke-opacity={isInternet ? 0.20 : 0.16}
            stroke-width={0.8 / k}
            stroke-dasharray="{6 / k} {5 / k}"
          />
        {/if}
      {/each}

      <!-- Link lines: nearest active sat → each gateway -->
      {#each gwList as gw}
        {@const nearest = nearestActiveSat(gw.lat, gw.lng)}
        {#if nearest}
          {@const p1 = project(gw.lng, gw.lat)}
          {@const p2 = project(nearest.displayLon, nearest.displayLat)}
          {#if p1 && p2}
            <line
              x1={p1[0]} y1={p1[1]}
              x2={p2[0]} y2={p2[1]}
              stroke={gwColor(gw.status)}
              stroke-opacity="0.60"
              stroke-width={1.2 / k}
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
        {@const pt = project(sat.displayLon, sat.displayLat)}
        {#if pt}
          {@const color = SAT_COLOR[sat.type]}
          {@const r = 3 / k}
          {@const cross = 6 / k}
          {@const ring = 14 / k}
          <g class="sat-marker">
            <!-- Footprint ring -->
            <circle
              cx={pt[0]} cy={pt[1]}
              r={ring}
              fill="none"
              stroke={color}
              stroke-opacity="0.10"
              stroke-width={0.8 / k}
            />
            <!-- Crosshair -->
            <line
              x1={pt[0] - cross} y1={pt[1]}
              x2={pt[0] + cross} y2={pt[1]}
              stroke={color} stroke-opacity="0.85" stroke-width={1.4 / k}
            />
            <line
              x1={pt[0]} y1={pt[1] - cross}
              x2={pt[0]} y2={pt[1] + cross}
              stroke={color} stroke-opacity="0.85" stroke-width={1.4 / k}
            />
            <!-- Core dot -->
            <circle cx={pt[0]} cy={pt[1]} r={r} fill={color} fill-opacity="0.95"/>
            <!-- Pulse ring (SVG animate — no JS overhead) -->
            <circle cx={pt[0]} cy={pt[1]} r={r * 2} fill="none" stroke={color} stroke-opacity="0.45" stroke-width={0.9 / k}>
              <animate attributeName="r"              values="{r * 2};{r * 5};{r * 2}" dur="2.8s" repeatCount="indefinite"/>
              <animate attributeName="stroke-opacity" values="0.45;0;0.45"             dur="2.8s" repeatCount="indefinite"/>
            </circle>
            <!-- Label (only readable when zoomed in enough) -->
            {#if k > 1.5}
              <text
                x={pt[0] + (r + 2)}
                y={pt[1] - (r + 1)}
                font-family="JetBrains Mono, monospace"
                font-size={8 / k}
                fill={color}
                fill-opacity="0.75"
              >
                {sat.id.replace(/^[IW]-VNU-LEO-/, '')}
              </text>
            {/if}
          </g>
        {/if}
      {/each}

      <!-- Ground stations -->
      {#each gwList as gw}
        {@const pt = project(gw.lng, gw.lat)}
        {#if pt}
          {@const color = gwColor(gw.status)}
          {@const dotR = 5 / k}
          {@const ringR = 20 / k}
          <!-- Coverage ring -->
          <circle
            cx={pt[0]} cy={pt[1]}
            r={ringR}
            fill={color} fill-opacity="0.05"
            stroke={color} stroke-opacity="0.20"
            stroke-width={1 / k}
            stroke-dasharray="{4 / k} {3 / k}"
          />
          <!-- Station dot -->
          <circle cx={pt[0]} cy={pt[1]} r={dotR} fill={color} fill-opacity="0.92"/>
          <!-- Pulse -->
          <circle cx={pt[0]} cy={pt[1]} r={dotR * 2} fill="none" stroke={color} stroke-opacity="0.5" stroke-width={1.5 / k}>
            <animate attributeName="r"              values="{dotR * 2};{dotR * 4};{dotR * 2}" dur="3s" repeatCount="indefinite"/>
            <animate attributeName="stroke-opacity" values="0.5;0;0.5"                         dur="3s" repeatCount="indefinite"/>
          </circle>
          <!-- Label -->
          <text
            x={pt[0] + dotR + 2 / k}
            y={pt[1] + dotR * 0.4}
            font-family="JetBrains Mono, monospace"
            font-size={9 / k}
            fill={color}
            fill-opacity="0.92"
          >
            {gw.name ?? gw.id}
          </text>
        {/if}
      {/each}
    </svg>

    <!-- ── HUD overlays ────────────────────────────────────────────────────── -->

    <!-- Top-left: sat counts -->
    <div class="wm-hud wm-hud-tl">
      <div class="wm-badge-row">
        <span class="wm-badge" style="color:#25d8f4;border-color:#25d8f460;">
          <span class="wm-badge-dot" style="background:#25d8f4;"></span>
          {internetCount} INTERNET
        </span>
        <span class="wm-badge" style="color:#a78bfa;border-color:#a78bfa60;">
          <span class="wm-badge-dot" style="background:#a78bfa;"></span>
          {weatherCount} WEATHER
        </span>
      </div>
      <div class="wm-badge-row" style="margin-top:4px;">
        <span class="wm-badge" style="color:#34d399;border-color:#34d39960;">
          {gwList.filter(g => g.status === 'alive').length}/{gwList.length} GW ONLINE
        </span>
      </div>
    </div>

    <!-- Bottom-right: legend -->
    <div class="wm-hud wm-hud-br">
      <div class="wm-legend-title">LEGEND</div>
      <div class="wm-legend-row">
        <span class="wm-legend-line" style="background:#25d8f4;"></span>
        <span>Internet sat</span>
      </div>
      <div class="wm-legend-row">
        <span class="wm-legend-line" style="background:#a78bfa;"></span>
        <span>Weather sat</span>
      </div>
      <div class="wm-legend-row">
        <span class="wm-legend-dash" style="border-color:#25d8f4;"></span>
        <span>Orbital trail</span>
      </div>
      <div class="wm-legend-row">
        <span class="wm-legend-dot" style="background:#34d399;"></span>
        <span>Gateway alive</span>
      </div>
      <div class="wm-legend-row">
        <span class="wm-legend-dot" style="background:#fbbf24;"></span>
        <span>Gateway degraded</span>
      </div>
      <div class="wm-legend-row">
        <span class="wm-legend-dash" style="border-color:#34d399;"></span>
        <span>Uplink / downlink</span>
      </div>
    </div>

    <!-- Bottom-left: projection label -->
    <div class="wm-hud wm-hud-bl">
      <span class="wm-proj-label">EQUIRECTANGULAR · 1° GRID · SGP4-LITE</span>
    </div>
  </div>
</div>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<style>
  /* ── Layout ──────────────────────────────────────────────────────────────── */
  .wm-overlay {
    position: fixed;
    inset: 0;
    z-index: 50;
    background: #060d14;
    display: flex;
    flex-direction: column;
    font-family: 'JetBrains Mono', monospace;
  }

  /* ── Header ───────────────────────────────────────────────────────────────── */
  .wm-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 10px 18px;
    border-bottom: 1px solid #0e4855;
    background: #060d14;
  }
  .wm-header-left {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .wm-title {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.12em;
    color: #25d8f4;
    text-shadow: 0 0 10px rgba(37, 216, 244, 0.45);
  }
  .wm-subtitle {
    font-size: 10px;
    color: #4a7a88;
    letter-spacing: 0.06em;
  }

  /* Tabs */
  .wm-tabs {
    display: flex;
    gap: 4px;
    margin-left: 12px;
  }
  .wm-tab {
    font-family: 'JetBrains Mono', monospace;
    font-size: 9px;
    letter-spacing: 0.1em;
    padding: 4px 10px;
    border-radius: 4px;
    border: 1px solid #0e4855;
    background: transparent;
    color: #4a7a88;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s, background 0.15s;
  }
  .wm-tab:hover {
    color: #25d8f4;
    border-color: #25d8f440;
  }
  .wm-tab--active {
    color: #25d8f4;
    border-color: #25d8f4;
    background: rgba(37, 216, 244, 0.08);
  }

  .wm-header-right {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .wm-hint {
    font-size: 9px;
    color: #2a4550;
    letter-spacing: 0.08em;
  }

  /* Close button */
  .wm-close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px; height: 32px;
    border-radius: 6px;
    background: transparent;
    border: 1px solid rgba(251,113,133,0.15);
    color: rgba(251,113,133,0.55);
    cursor: pointer;
    transition: color 0.15s, background 0.15s, border-color 0.15s;
    padding: 0;
  }
  .wm-close-btn:hover {
    color: #fb7185;
    background: rgba(251,113,133,0.09);
    border-color: rgba(251,113,133,0.30);
  }
  .wm-close-btn svg { display: block; pointer-events: none; }

  /* ── Canvas ───────────────────────────────────────────────────────────────── */
  .wm-canvas-wrapper {
    flex: 1;
    overflow: hidden;
    position: relative;
  }
  .wm-svg {
    display: block;
    width: 100%;
    height: 100%;
    cursor: grab;
    user-select: none;
  }
  .wm-svg:active {
    cursor: grabbing;
  }

  /* ── HUD overlays ─────────────────────────────────────────────────────────── */
  .wm-hud {
    position: absolute;
    background: rgba(6, 13, 20, 0.82);
    backdrop-filter: blur(8px);
    border: 1px solid #0e4855;
    border-radius: 8px;
    padding: 10px 14px;
    pointer-events: none;
  }
  .wm-hud-tl { top: 12px; left: 12px; }
  .wm-hud-br { bottom: 14px; right: 14px; }
  .wm-hud-bl { bottom: 14px; left: 12px; }

  /* Badges */
  .wm-badge-row {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .wm-badge {
    font-size: 9px;
    letter-spacing: 0.09em;
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid;
    display: flex;
    align-items: center;
    gap: 5px;
  }
  .wm-badge-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    display: inline-block;
  }

  /* Legend */
  .wm-legend-title {
    font-size: 8px;
    letter-spacing: 0.15em;
    color: #2a4550;
    margin-bottom: 8px;
  }
  .wm-legend-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 9px;
    color: #5a8a99;
    margin-bottom: 5px;
  }
  .wm-legend-line {
    display: inline-block;
    width: 20px; height: 1.5px;
    border-radius: 1px;
    flex-shrink: 0;
  }
  .wm-legend-dash {
    display: inline-block;
    width: 20px; height: 0;
    border-top: 2px dashed;
    flex-shrink: 0;
  }
  .wm-legend-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  /* Projection label */
  .wm-proj-label {
    font-size: 8px;
    letter-spacing: 0.10em;
    color: #1e3a44;
  }
</style>
