<script lang="ts">
  import GeoMap from './GeoMap.svelte';

  export let onClose: () => void;
  export let open: boolean = false;
  
  let satInView = 0;

  function handleKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') onClose();
  }

</script>

<svelte:window on:keydown={handleKeydown} />

<!-- Full-screen constellation modal (Vietnam corridor — fixed, z-40) -->
<div
  class="fixed inset-0 z-40 bg-space-950 flex flex-col"
  class:constellation-hidden={!open}
  role="dialog"
  aria-label="Constellation Map — Vietnam"
  aria-modal="true"
  aria-hidden={!open}
>
  <!-- Header bar with two buttons -->
  <div class="flex items-center justify-between px-5 py-3 border-b border-slate-700/40 flex-shrink-0">
    <div class="flex items-center gap-3">
      <div class="font-display text-xs tracking-widest text-cyan-400">CONSTELLATION MAP — VIETNAM CORRIDOR</div>
      <div class="flex items-center gap-2 px-3 py-1 bg-space-800 rounded border border-slate-700/50">
        <span class="w-1.5 h-1.5 rounded-full bg-cyan-400"></span>
        <span class="font-mono text-[10px] text-cyan-400">
          <span class="font-bold">{satInView}</span> SAT IN VIEW
        </span>
      </div>
    </div>

    <!-- Close button -->
    <div class="flex items-center gap-2">
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

  <!-- Canvas (Map) -->
  <div class="flex-1 relative overflow-hidden bg-space-950">
    <GeoMap
      initialZoom={6.5}
      initialCenter={[106.5, 16.5]}
      interactive={true}
      showSatellites={true}
      wrapWorld={true}
      showLegend={false}
      showCounts={false}
      activeTab="all"
      bind:inViewCount={satInView}
    />
  </div>
</div>

<style>
  .constellation-hidden {
    display: none;
  }
  .map-icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: transparent;
    border: 1px solid;
    cursor: pointer;
    transition: color 0.15s, background 0.15s, border-color 0.15s;
    padding: 0;
  }
  .map-icon-btn svg { display: block; pointer-events: none; }
  
  .map-icon-btn--rose {
    border-color: rgba(251,113,133,0.15);
    color: rgba(251,113,133,0.55);
  }
  .map-icon-btn--rose:hover {
    color: #fb7185;
    background: rgba(251,113,133,0.09);
    border-color: rgba(251,113,133,0.30);
  }
</style>
