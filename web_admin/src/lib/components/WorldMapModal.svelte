<script lang="ts">
  import GeoMap from './GeoMap.svelte';
  
  export let onClose: () => void;
  let activeTab: 'all' | 'internet' | 'weather' = 'all';

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="wm-root" role="dialog" aria-label="World Constellation Map" aria-modal="true">
  <!-- ── Header ── -->
  <header class="wm-header">
    <div class="wm-header-left">
      <span class="wm-title">WORLD CONSTELLATION TRACKER</span>
      <span class="wm-sub">VNU-LEO &nbsp;·&nbsp; LIVE TELEMETRY</span>
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
    <GeoMap
      initialZoom={1}
      initialCenter={[0, 0]}
      interactive={true}
      showSatellites={true}
      showLegend={true}
      showCounts={true}
      {activeTab}
    />
  </div>
</div>

<style>
  .wm-root {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: #060d14;
    display: flex;
    flex-direction: column;
    font-family: 'JetBrains Mono', monospace;
  }
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
  .wm-canvas { flex: 1; overflow: hidden; position: relative; }
</style>
