<script lang="ts">
  import { afterUpdate, onDestroy, onMount } from 'svelte';
  import { Chart, type ChartConfiguration } from 'chart.js/auto';

  export let title: string;
  export let config: ChartConfiguration;

  let canvas: HTMLCanvasElement | null = null;
  let chart: Chart | null = null;

  onMount(() => {
    if (!canvas) return;
    chart = new Chart(canvas, config);
  });

  afterUpdate(() => {
    if (!chart) return;
    chart.config.type = config.type ?? chart.config.type;
    chart.config.data = config.data ?? chart.config.data;
    chart.config.options = config.options ?? chart.config.options;
    chart.update();
  });

  onDestroy(() => {
    chart?.destroy();
    chart = null;
  });
</script>

<div class="chart-container">
  <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">{title}</div>
  <canvas bind:this={canvas} height="140"></canvas>
</div>
