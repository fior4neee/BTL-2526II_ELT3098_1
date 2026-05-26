<script lang="ts">
  import { gateways } from '$lib/stores';
  import ConstellationModal from './ConstellationModal.svelte';
  import GeoMap from './GeoMap.svelte';

  // ── Hover + modal state ───────────────────────────────────────────────────
  let isHovered = false;
  let showConstellation = false;

  function openConstellation() { showConstellation = true; }
  function closeConstellation() { showConstellation = false; }
</script>

{#if showConstellation}
  <ConstellationModal onClose={closeConstellation} />
{/if}

<div class="chart-container flex flex-col h-full">
  <div class="flex items-center justify-between mb-3 flex-shrink-0">
    <div class="font-mono text-xs text-slate-500 tracking-widest">CONSTELLATION VIEW — VIETNAM CORRIDOR</div>
    <div class="font-mono text-xs text-slate-600">{$gateways.length} GW</div>
  </div>

  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="mini-map-wrapper flex-1"
    on:mouseenter={() => isHovered = true}
    on:mouseleave={() => isHovered = false}
    on:click={openConstellation}
    role="button"
    tabindex="0"
    aria-label="Click to open Constellation Map"
    on:keydown={(e) => e.key === 'Enter' && openConstellation()}
  >
    <!-- SVG map -->
    <div class="geomap-holder" style="filter: {isHovered ? 'blur(3px) brightness(0.5)' : 'none'};">
      <GeoMap
        initialZoom={4.5}
        initialCenter={[106.5, 16.5]}
        interactive={false}
        showSatellites={false}
        showLegend={false}
        showCounts={false}
        activeTab="all"
      />
    </div>

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
    <div class="absolute bottom-2 right-2 space-y-1 transition-opacity duration-300 z-10"
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
    flex-direction: column;
    cursor: pointer;
    border-radius: 6px;
    overflow: hidden;
    min-height: 250px; /* give it some vertical space */
  }

  .geomap-holder {
    flex: 1;
    position: relative;
    overflow: hidden;
    transition: filter 0.3s ease;
    border-radius: 6px;
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
