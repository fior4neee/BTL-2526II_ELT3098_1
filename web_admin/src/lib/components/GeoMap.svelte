<script lang="ts">
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
  export let initialZoom: number = 1;
  export let initialCenter: [number, number] = [0, 0]; // [lon, lat]
  export let interactive: boolean = true;
  export let showSatellites: boolean = true;
  export let showTrails: boolean = true;
  export let showLegend: boolean = true;
  export let showCounts: boolean = true;
  export let activeTab: 'all' | 'internet' | 'weather' = 'all';
  export let satelliteMode: 'all' | 'connected' = 'all';
  export let fitToVietnam: boolean = false;
  export let wrapWorld: boolean = false;

  export let inViewCount: number = 0;

  // ── Map styling constants ─────────────────────────────────────
  const BG_COLOR       = '#060d14';
  const COUNTRY_FILL   = '#081b23';
  const COUNTRY_STROKE = '#0e4855';
  const COUNTRY_SW     = 1.5;
  const GRAT_COLOR     = '#25d8f4';
  const GRAT_OPACITY   = 0.14;
  const GRAT_SW        = 0.35;

  // ── TLE-derived constellation definitions ─────────────────────────────────
  interface SatDef {
    id: string;
    type: 'internet' | 'weather';
    inc: number;   // radians
    raan: number;  // radians
    m0: number;    // radians
    n: number;     // rad/s
  }

  const D2R = Math.PI / 180;
  const EARTH_ROT = 7.2921150e-5;
  const N_INT = (15.24308387 * 2 * Math.PI) / 86400;
  const N_WEA = (13.71870588 * 2 * Math.PI) / 86400;

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

  function makeDefs(params: [string, number, number][], n: number, type: 'internet' | 'weather'): SatDef[] {
    return params.map(([suffix, raanDeg, m0Deg]) => ({
      id: `${type === 'internet' ? 'I' : 'W'}-VNU-LEO-${suffix}`,
      type, inc: INC, raan: raanDeg * D2R, m0: m0Deg * D2R, n,
    }));
  }

  const ALL_DEFS: SatDef[] = [
    ...makeDefs(INT_PARAMS, N_INT, 'internet'),
    ...makeDefs(WEA_PARAMS, N_WEA, 'weather'),
  ];

  // ── SGP4-lite ─────────────────────────────────────────────────────────────
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

  const TRAIL_STEPS = 90;
  function buildTrail(def: SatDef, t0: number): [number, number][] {
    const T = (2 * Math.PI) / def.n;
    const pts: [number, number][] = [];
    for (let i = 0; i <= TRAIL_STEPS; i++) {
      pts.push(satPos(def, t0 + (i / TRAIL_STEPS) * T));
    }
    return pts;
  }

  const STATIC_GW = [
    { id: 'GW-HAN-01', name: 'HAN', lat: 21.028, lng: 105.854, status: 'alive' },
    { id: 'GW-DAN-01', name: 'DAN', lat: 16.047, lng: 108.206, status: 'alive' },
    { id: 'GW-HCM-01', name: 'HCM', lat: 10.763, lng: 106.660, status: 'alive' },
  ];

  interface AnimSat {
    id: string; type: 'internet' | 'weather';
    lon: number; lat: number;
    dLon: number; dLat: number;
  }

  const _t0 = Date.now() / 1000;
  let animSats: AnimSat[] = ALL_DEFS.map(d => {
    const [lon, lat] = satPos(d, _t0);
    return { id: d.id, type: d.type, lon, lat, dLon: lon, dLat: lat };
  });

  // ── D3 / SVG state ────────────────────────────────────────────────────────
  let svgEl: SVGSVGElement;
  let width = 1200;
  let height = 600;
  let projection = d3Geo.geoEquirectangular().scale(width / (2 * Math.PI)).translate([width / 2, height / 2]);
  let pathGen = d3Geo.geoPath(projection);
  let currentTransform = d3Zoom.zoomIdentity;

  let landPaths: string[] = [];
  let graticulePath = '';
  const TRAIL_TTL = 2;
  let trailCache: { id: string; type: string; d: string }[] = [];
  let trailLastBuilt = -Infinity;
  let trailBuildPending = false;
  let baseReady = false;
  let connectionsReady = false;

  let staticBuildPending = false;
  const PATH_CACHE = new Map<string, { land: string[]; grat: string }>();

  let simTime = Date.now() / 1000;
  let animTimer: ReturnType<typeof setInterval> | null = null;
  const REFRESH_MS = 100;

  function getVietnamFeature() {
    return WORLD_FEATURES.find((f: any) => {
      const props = f?.properties ?? {};
      return props['ISO3166-1-Alpha-3'] === 'VNM' || props['ISO3166-1-Alpha-2'] === 'VN' || props.name === 'Vietnam';
    }) ?? null;
  }

  function fitProjectionToFeature(feature: any) {
    if (!feature) return;
    const base = d3Geo.geoEquirectangular().scale(1).translate([0, 0]);
    const basePath = d3Geo.geoPath(base);
    const b = basePath.bounds(feature);
    const dx = b[1][0] - b[0][0];
    const dy = b[1][1] - b[0][1];
    if (!dx || !dy) return;
    const padding = 0.92;
    const s = padding / Math.max(dx / width, dy / height);
    const t: [number, number] = [
      (width - s * (b[1][0] + b[0][0])) / 2,
      (height - s * (b[1][1] + b[0][1])) / 2,
    ];
    projection = d3Geo.geoEquirectangular().scale(s).translate(t);
  }

  function initProjection() {
    projection = d3Geo.geoEquirectangular()
      .scale(width / (2 * Math.PI))
      .translate([width / 2, height / 2]);
    if (fitToVietnam) {
      fitProjectionToFeature(getVietnamFeature());
    }
    pathGen = d3Geo.geoPath(projection);
  }

  function getPathCacheKey() {
    if (!projection) return '';
    const sc = projection.scale();
    const [tx, ty] = projection.translate();
    return `${width}x${height}:${fitToVietnam ? 'VN' : 'WORLD'}:${sc.toFixed(3)}:${tx.toFixed(2)},${ty.toFixed(2)}`;
  }

  function buildStaticPaths() {
    if (!pathGen) return;
    const key = getPathCacheKey();
    const cached = PATH_CACHE.get(key);
    if (cached) {
      landPaths = cached.land;
      graticulePath = cached.grat;
      baseReady = true;
      return;
    }
    const grat = pathGen(d3Geo.geoGraticule().step([1, 1])()) ?? '';
    const land = WORLD_FEATURES.map(f => pathGen(f) ?? '').filter(Boolean);
    landPaths = land;
    graticulePath = grat;
    PATH_CACHE.set(key, { land, grat });
    baseReady = true;
  }

  function scheduleTask(fn: () => void) {
    const ric = (globalThis as any)?.requestIdleCallback as ((cb: () => void, opts?: { timeout?: number }) => number) | undefined;
    if (typeof ric === 'function') {
      ric(fn, { timeout: 1200 });
      return;
    }
    const raf = (globalThis as any)?.requestAnimationFrame as ((cb: () => void) => number) | undefined;
    if (typeof raf === 'function') {
      raf(() => setTimeout(fn, 0));
      return;
    }
    setTimeout(fn, 0);
  }

  function scheduleStaticBuild() {
    if (staticBuildPending) return;
    baseReady = false;
    staticBuildPending = true;
    const run = () => {
      staticBuildPending = false;
      buildStaticPaths();
    };
    scheduleTask(run);
  }

  function proj(lon: number, lat: number): [number, number] | null {
    if (!projection) return null;
    return projection([lon, lat]) as [number, number] | null;
  }

  function trailToPath(pts: [number, number][]): string {
    if (!projection) return '';
    let d = '', pen = false, prevLon = NaN;
    for (const [lon, lat] of pts) {
      if (!isNaN(prevLon) && Math.abs(lon - prevLon) > 150) pen = false;
      prevLon = lon;
      const xy = projection([lon, lat]);
      if (!xy) { pen = false; continue; }
      d += pen
        ? `L${xy[0].toFixed(1)},${xy[1].toFixed(1)} `
        : `M${xy[0].toFixed(1)},${xy[1].toFixed(1)} `;
      pen = true;
    }
    return d;
  }

  function rebuildTrails() {
    if (!showSatellites || !showTrails) return;
    const defs = activeTab === 'all' ? ALL_DEFS : ALL_DEFS.filter(d => d.type === activeTab);
    trailCache = defs.map(def => ({
      id: def.id, type: def.type,
      d: trailToPath(buildTrail(def, simTime)),
    }));
    trailLastBuilt = simTime;
  }

  function scheduleTrailRebuild() {
    if (!showSatellites || !showTrails || trailBuildPending) return;
    trailBuildPending = true;
    const run = () => {
      trailBuildPending = false;
      rebuildTrails();
    };
    scheduleTask(run);
  }

  let suppressWrap = false;

  function setupZoom() {
    if (!svgEl) return;
    const svg = d3Selection.select(svgEl);
    let initialTransform = d3Zoom.zoomIdentity;

    if (!fitToVietnam) {
      // Convert initial Center [lon, lat] into SVG translation
      const centerPx = projection(initialCenter);
      let xOffset = width / 2;
      let yOffset = height / 2;
      if (centerPx) {
        xOffset -= centerPx[0] * initialZoom;
        yOffset -= centerPx[1] * initialZoom;
      }
      initialTransform = d3Zoom.zoomIdentity.translate(xOffset, yOffset).scale(initialZoom);
    }

    currentTransform = initialTransform;

    const zoom = d3Zoom.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.5, 30])
      .on('zoom', (e: d3Zoom.D3ZoomEvent<SVGSVGElement, unknown>) => {
        if (!wrapWorld || suppressWrap) {
          currentTransform = e.transform;
          return;
        }
        const k = e.transform.k;
        const worldHeight = (Math.PI * projection.scale()) * k;
        let y = e.transform.y;

        const maxY = 0;
        const minY = Math.min(0, height - worldHeight);
        y = Math.max(minY, Math.min(maxY, y));

        currentTransform = d3Zoom.zoomIdentity.translate(e.transform.x, y).scale(k);
      });
    
    if (interactive) {
      suppressWrap = true;
      svg.call(zoom);
      svg.call(zoom.transform, initialTransform);
      setTimeout(() => { suppressWrap = false; }, 0);
    } else {
      svg.on(".zoom", null); // disable zoom interactions
    }
  }

  let connectionLinks: { gwId: string; x1: number; y1: number; x2: number; y2: number; color: string }[] = [];
  let connectedSatIds: Set<string> = new Set();

  function rebuildConnections() {
    if (!showSatellites) {
      connectionLinks = [];
      connectedSatIds = new Set();
      connectionsReady = true;
      return;
    }
    const links: { gwId: string; x1: number; y1: number; x2: number; y2: number; color: string }[] = [];
    const ids = new Set<string>();

    for (const gw of gwList) {
      const nearest = nearestSat(gw.lat, gw.lng);
      if (!nearest) continue;
      const p1 = proj(gw.lng, gw.lat);
      const p2 = proj(nearest.dLon, nearest.dLat);
      if (!p1 || !p2) continue;
      ids.add(nearest.id);
      links.push({
        gwId: gw.id ?? gw.name ?? 'gw',
        x1: p1[0], y1: p1[1], x2: p2[0], y2: p2[1],
        color: gwColor(gw.status),
      });
    }
    connectionLinks = links;
    connectedSatIds = new Set(ids);
    connectionsReady = true;
  }

  function tick() {
    simTime = Date.now() / 1000;

    if (showSatellites) {
      // NOTE: We intentionally ignore WebSocket overrides here to ensure 
      // the visual dots stay EXACTLY on the visual orbital trails.
      animSats = animSats.map((anim, i) => {
        const def = ALL_DEFS[i];
        const [lon, lat] = satPos(def, simTime);
        return { ...anim, lon, lat, dLon: lon, dLat: lat };
      });
      if (simTime - trailLastBuilt >= TRAIL_TTL) scheduleTrailRebuild();
    }

    rebuildConnections();
  }

  $: gwList = $gateways.length > 0
    ? $gateways
    : STATIC_GW.map(g => ({
        ...g, currentSessions: 0, maxSessions: 0,
        minElevationDeg: 0, antennaGainDbi: 0, beamWidthDeg: 0, altitudeKm: 0,
      }));

  function nearestSat(gwLat: number, gwLng: number) {
    if (!showSatellites) return null;
    const MIN_ELEV = 15;
    let best: AnimSat | null = null, bestEl = -Infinity;
    for (const s of animSats) {
      if (activeTab !== 'all' && s.type !== activeTab) continue;
      const altKm = s.type === 'internet' ? 500 : 1000;
      const d2r = Math.PI / 180;
      const cosEta = Math.sin(gwLat*d2r) * Math.sin(s.dLat*d2r)
                   + Math.cos(gwLat*d2r) * Math.cos(s.dLat*d2r)
                   * Math.cos((s.dLon - gwLng)*d2r);
      const c = Math.max(-1, Math.min(1, cosEta));
      const a = 6371 + altKm;
      const dist = Math.sqrt(a*a - 2*a*6371*c + 6371*6371);
      if (dist < 1e-6) { best = s; break; }
      const sinEl = (a * c - 6371) / dist;
      const el = Math.asin(Math.max(-1, Math.min(1, sinEl))) * (180 / Math.PI);
      if (el >= MIN_ELEV && el > bestEl) { bestEl = el; best = s; }
    }
    return best;
  }

  function gwColor(status: string) {
    if (status === 'alive') return '#34d399';
    if (status === 'degraded') return '#fbbf24';
    return '#fb7185';
  }

  $: visibleSats = (() => {
    const base = activeTab === 'all' ? animSats : animSats.filter(s => s.type === activeTab);
    return satelliteMode === 'connected'
      ? base.filter(s => connectedSatIds.has(s.id))
      : base;
  })();
  const SAT_CLR: Record<string, string> = { internet: '#25d8f4', weather:  '#a78bfa' };

  function ensureSizedInit() {
    const nextWidth = svgEl.clientWidth || window.innerWidth;
    const nextHeight = svgEl.clientHeight || window.innerHeight - 50;
    if (!nextWidth || !nextHeight) {
      scheduleTask(ensureSizedInit);
      return;
    }
    width = nextWidth;
    height = nextHeight;
    initProjection();
    setupZoom();
    scheduleStaticBuild();
    rebuildConnections();
  }

  onMount(() => {
    ensureSizedInit();

    const t0 = Date.now() / 1000;
    animSats = ALL_DEFS.map(def => {
      const [lon, lat] = satPos(def, t0);
      return { id: def.id, type: def.type, lon, lat, dLon: lon, dLat: lat };
    });
    rebuildConnections();
    
    if (showSatellites && showTrails) scheduleTrailRebuild();
    animTimer = setInterval(tick, REFRESH_MS);
  });

  onDestroy(() => {
    if (animTimer) clearInterval(animTimer);
  });

  $: { activeTab; trailLastBuilt = -Infinity; scheduleTrailRebuild(); }
  $: {
    fitToVietnam; wrapWorld; width; height; initialZoom; initialCenter;
    if (svgEl) {
      initProjection();
      setupZoom();
      scheduleStaticBuild();
      rebuildConnections();
    }
  }
  
  // Compute visible bounds in projection space
  $: viewMinX = -currentTransform.x / currentTransform.k;
  $: viewMaxX = (width - currentTransform.x) / currentTransform.k;
  $: viewMinY = -currentTransform.y / currentTransform.k;
  $: viewMaxY = (height - currentTransform.y) / currentTransform.k;

  function isInView(lon: number, lat: number) {
    const pt = proj(lon, lat);
    if (!pt) return false;
    return pt[0] >= viewMinX && pt[0] <= viewMaxX && pt[1] >= viewMinY && pt[1] <= viewMaxY;
  }

  $: intCount = animSats.filter(s => s.type === 'internet' && isInView(s.dLon, s.dLat)).length;
  $: weaCount = animSats.filter(s => s.type === 'weather' && isInView(s.dLon, s.dLat)).length;
  $: inViewCount = animSats.filter(s => isInView(s.dLon, s.dLat)).length;
  $: k = currentTransform.k;
  $: worldWidthPx = projection ? 2 * Math.PI * projection.scale() : 0;
  $: wrapOffsets = wrapWorld && worldWidthPx > 0 ? [-worldWidthPx, 0, worldWidthPx] : [0];
  $: mapReady = baseReady && connectionsReady;
</script>

<div class="geomap-wrapper">
  <svg
    bind:this={svgEl}
    class="wm-svg {interactive ? 'interactive' : ''}"
    width="100%"
    height="100%"
    style="background:{BG_COLOR};"
  >
    <g transform="translate({currentTransform.x},{currentTransform.y}) scale({currentTransform.k})">
      {#each wrapOffsets as dx}
        <g transform="translate({dx},0)">
          {#if graticulePath}
            <path d={graticulePath} fill="none" stroke={GRAT_COLOR} stroke-opacity={GRAT_OPACITY} stroke-width={GRAT_SW / k} />
          {/if}

          {#each landPaths as d}
            <path {d} fill={COUNTRY_FILL} stroke={COUNTRY_STROKE} stroke-width={COUNTRY_SW / k} stroke-linejoin="round" />
          {/each}

          {#if showSatellites && showTrails}
            {#each trailCache as tr (tr.id)}
              {#if tr.d}
                <path d={tr.d} fill="none" stroke={tr.type === 'internet' ? '#25d8f4' : '#a78bfa'}
                      stroke-opacity={tr.type === 'internet' ? 0.22 : 0.18} stroke-width={0.7 / k} stroke-dasharray="{6 / k} {5 / k}" />
              {/if}
            {/each}
          {/if}

          {#each connectionLinks as link (link.gwId)}
            <line x1={link.x1} y1={link.y1} x2={link.x2} y2={link.y2}
                  stroke={link.color} stroke-opacity="0.55" stroke-width={1.1 / k} stroke-dasharray="{6 / k} {4 / k}">
              <animate attributeName="stroke-dashoffset" values="0;{-20 / k}" dur="1.4s" repeatCount="indefinite" />
            </line>
          {/each}

          {#if showSatellites}
            {#each visibleSats as sat (sat.id)}
              {@const pt = proj(sat.dLon, sat.dLat)}
              {#if pt}
                {@const c = SAT_CLR[sat.type]}
                {@const r  = 3 / k}
                {@const cr = 6 / k}
                {@const rg = 13 / k}
                <g>
                  <circle cx={pt[0]} cy={pt[1]} r={rg} fill="none" stroke={c} stroke-opacity="0.10" stroke-width={0.7 / k} />
                  <line x1={pt[0]-cr} y1={pt[1]} x2={pt[0]+cr} y2={pt[1]} stroke={c} stroke-opacity="0.85" stroke-width={1.3 / k} />
                  <line x1={pt[0]} y1={pt[1]-cr} x2={pt[0]} y2={pt[1]+cr} stroke={c} stroke-opacity="0.85" stroke-width={1.3 / k} />
                  <circle cx={pt[0]} cy={pt[1]} r={r} fill={c} fill-opacity="0.95" />
                  <circle cx={pt[0]} cy={pt[1]} r={r * 2} fill="none" stroke={c} stroke-opacity="0.45" stroke-width={0.8 / k}>
                    <animate attributeName="r" values="{r*2};{r*5};{r*2}" dur="2.8s" repeatCount="indefinite"/>
                    <animate attributeName="stroke-opacity" values="0.45;0;0.45" dur="2.8s" repeatCount="indefinite"/>
                  </circle>
                  {#if k > 1.8}
                    <text x={pt[0] + r + 2/k} y={pt[1] - r - 1/k} font-family="JetBrains Mono,monospace" font-size={8 / k} fill={c} fill-opacity="0.75">
                      {sat.id.replace(/^[IW]-VNU-LEO-/, '')}
                    </text>
                  {/if}
                </g>
              {/if}
            {/each}
          {/if}

          {#each gwList as gw}
            {@const pt = proj(gw.lng, gw.lat)}
            {#if pt}
              {@const c = gwColor(gw.status)}
              {@const dr = 5 / k}
              {@const rg = 18 / k}
              <circle cx={pt[0]} cy={pt[1]} r={rg} fill={c} fill-opacity="0.05" stroke={c} stroke-opacity="0.18" stroke-width={0.9 / k} stroke-dasharray="{4/k} {3/k}" />
              <circle cx={pt[0]} cy={pt[1]} r={dr} fill={c} fill-opacity="0.92" />
              <circle cx={pt[0]} cy={pt[1]} r={dr*2} fill="none" stroke={c} stroke-opacity="0.5" stroke-width={1.4 / k}>
                <animate attributeName="r" values="{dr*2};{dr*4};{dr*2}" dur="3s" repeatCount="indefinite"/>
                <animate attributeName="stroke-opacity" values="0.5;0;0.5" dur="3s" repeatCount="indefinite"/>
              </circle>
              <text x={pt[0] + dr + 2/k} y={pt[1] + dr * 0.4} font-family="JetBrains Mono,monospace" font-size={9 / k} fill={c} fill-opacity="0.92">
                {gw.name ?? gw.id}
              </text>
            {/if}
          {/each}
        </g>
      {/each}
    </g>
  </svg>

  {#if !mapReady}
    <div class="wm-loading">
      <div class="wm-loading-text">Map loading...</div>
      <div class="wm-loading-ring"></div>
    </div>
  {/if}

  {#if showCounts}
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
  {/if}

  {#if showLegend}
    <div class="wm-hud wm-hud-br">
      <div class="wm-leg-title">LEGEND</div>
      <div class="wm-leg-row"><span class="wm-leg-line" style="background:#25d8f4;"></span>Internet satellite</div>
      <div class="wm-leg-row"><span class="wm-leg-line" style="background:#a78bfa;"></span>Weather satellite</div>
      <div class="wm-leg-row"><span class="wm-leg-dash" style="border-color:#25d8f4;"></span>Orbital trail</div>
      <div class="wm-leg-row"><span class="wm-leg-dot" style="background:#34d399;"></span>Gateway alive</div>
      <div class="wm-leg-row"><span class="wm-leg-dot" style="background:#fbbf24;"></span>Gateway degraded</div>
      <div class="wm-leg-row"><span class="wm-leg-dash" style="border-color:#34d399;"></span>Uplink / downlink</div>
    </div>
    <div class="wm-hud wm-hud-bl">
      <span class="wm-proj">EQUIRECTANGULAR · 1° GRID · SGP4-LITE · worldHigh.json</span>
    </div>
  {/if}
</div>

<style>
  .geomap-wrapper {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
  }
  .wm-svg {
    display: block; width: 100%; height: 100%;
    user-select: none;
  }
  .wm-svg.interactive { cursor: grab; }
  .wm-svg.interactive:active { cursor: grabbing; }

  .wm-loading {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    background: #060d14;
  }
  .wm-loading-text {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    letter-spacing: 0.18em;
    color: #2f5666;
    text-transform: uppercase;
  }
  .wm-loading-ring {
    width: 28px;
    height: 28px;
    border-radius: 999px;
    border: 2px solid rgba(34, 211, 238, 0.15);
    border-top-color: rgba(34, 211, 238, 0.85);
    animation: wm-spin 1s linear infinite;
  }
  @keyframes wm-spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

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
  .wm-proj { font-size: 8px; letter-spacing: .10em; color: #1a2e38; }
</style>
