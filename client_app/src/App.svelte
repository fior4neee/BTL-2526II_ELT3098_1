<script>
  import SignalDashboard from './SignalDashboard.svelte';
  import DataUsage from './DataUsage.svelte';

  let activeView = 'signal';
</script>

<main class="app-shell">
  <header class="topbar">
    <div>
      <p class="eyebrow">VNU-LEO End-user Router</p>
      <h1>Satellite Link Console</h1>
    </div>
    <nav class="segmented" aria-label="Client views">
      <button class:active={activeView === 'signal'} on:click={() => (activeView = 'signal')}>
        Signal
      </button>
      <button class:active={activeView === 'usage'} on:click={() => (activeView = 'usage')}>
        Usage
      </button>
    </nav>
  </header>

  {#if activeView === 'signal'}
    <SignalDashboard />
  {:else}
    <DataUsage />
  {/if}
</main>
