<script lang="ts">
  import '../app.css';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import { onDestroy, onMount } from 'svelte';
  import { startLiveUpdates, stopLiveUpdates } from '$lib/stores';

  onMount(() => {
    startLiveUpdates(5000);
  });
  onDestroy(() => stopLiveUpdates());
</script>

<div class="flex h-screen bg-grid scanlines overflow-hidden" style="background-color: #060d14;">
  <Sidebar />
  <div class="flex-1 flex flex-col overflow-hidden">
    <slot />
  </div>
</div>
