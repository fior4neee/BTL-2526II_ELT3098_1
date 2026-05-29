<script>
  import { onMount } from 'svelte';
  import SignalDashboard from './SignalDashboard.svelte';
  import DataUsage from './DataUsage.svelte';
  import { initializeNetwork, connectionState, activeDevice, DEVICES, forceBreach, deviceStatus } from './network.js';

  let activeView = 'signal';

  onMount(() => {
    initializeNetwork();
  });
</script>

<main class="app-shell">
  <header class="topbar">
    <div>
      <p class="eyebrow">VNU-LEO End-user Router</p>
      <h1>Satellite Link Console</h1>
    </div>

    <div class="header-controls">
      {#if $connectionState === 'connected'}
        <div class="network-badge connected">
          <span class="badge-dot pulsing"></span>
          Core Network Live
        </div>
      {:else}
        <div class="network-badge simulation">
          <span class="badge-dot"></span>
          Demo Mode (Offline)
        </div>
      {/if}

      <div class="network-badge {$deviceStatus === 'active' ? 'connected' : $deviceStatus === 'suspended' || $deviceStatus === 'revoked' ? 'simulation' : 'connecting'}">
        <span class="badge-dot"></span>
        Device {$deviceStatus}
      </div>

      <div class="device-select-container">
        <span>Router:</span>
        <select class="device-select" bind:value={$activeDevice}>
          {#each DEVICES as dev}
            <option value={dev.id}>{dev.name}</option>
          {/each}
        </select>
      </div>

      <label class="breach-control">
        <input type="checkbox" bind:checked={$forceBreach} />
        Force Geofence Breach
      </label>

      <nav class="segmented" aria-label="Client views">
        <button class:active={activeView === 'signal'} on:click={() => (activeView = 'signal')}>
          Signal
        </button>
        <button class:active={activeView === 'usage'} on:click={() => (activeView = 'usage')}>
          Usage
        </button>
      </nav>
    </div>
  </header>

  {#if activeView === 'signal'}
    <SignalDashboard />
  {:else}
    <DataUsage />
  {/if}
</main>
