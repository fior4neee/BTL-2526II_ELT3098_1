<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import { onMount } from 'svelte';
  import { devices, lastError, refreshDevices, revokeDeviceById, suspendDeviceByMac } from '$lib/stores';
  import type { Device, DeviceStatus } from '$lib/types';

  let deviceSearch = '';
  let statusFilter: DeviceStatus | 'all' = 'all';
  let confirmRevoke: Device | null = null;
  let actionMessage = '';
  let actionError = '';

  $: filteredDevices = $devices.filter((device) => {
    const statusMatch = statusFilter === 'all' || device.status === statusFilter;
    const search = deviceSearch.trim().toLowerCase();
    const searchMatch = !search ||
      device.deviceId.toLowerCase().includes(search) ||
      device.mac.toLowerCase().includes(search) ||
      device.hardwareId.toLowerCase().includes(search);
    return statusMatch && searchMatch;
  });

  onMount(() => {
    void refreshDevices();
  });

  function statusBadge(status: DeviceStatus) {
    if (status === 'active') return 'badge-online';
    if (status === 'registered' || status === 'suspended') return 'badge-warning';
    return 'badge-offline';
  }

  function relTime(date: Date): string {
    const mins = Math.floor((Date.now() - date.getTime()) / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    if (mins < 1440) return `${Math.floor(mins / 60)}h ago`;
    return `${Math.floor(mins / 1440)}d ago`;
  }

  async function suspendDevice(device: Device) {
    actionMessage = '';
    actionError = '';
    try {
      await suspendDeviceByMac(device.mac);
      actionMessage = `Device ${device.mac} suspended`;
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Device suspend failed';
    }
  }

  async function revokeDevice(device: Device) {
    actionMessage = '';
    actionError = '';
    try {
      await revokeDeviceById(device.deviceId);
      actionMessage = `Device ${device.deviceId} revoked`;
      confirmRevoke = null;
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Device revoke failed';
    }
  }
</script>

<Topbar title="SECURITY" subtitle="Device registry and provisioning controls" />

<main class="flex-1 overflow-y-auto p-6 space-y-5">
  {#if $lastError}
    <div class="rounded border border-rose-400/30 bg-rose-400/8 px-4 py-3 font-mono text-xs text-rose-300">
      API refresh failed: {$lastError}
    </div>
  {/if}
  {#if actionMessage}
    <div class="rounded border border-emerald-400/30 bg-emerald-400/8 px-4 py-3 font-mono text-xs text-emerald-300">
      {actionMessage}
    </div>
  {/if}
  {#if actionError}
    <div class="rounded border border-rose-400/30 bg-rose-400/8 px-4 py-3 font-mono text-xs text-rose-300">
      {actionError}
    </div>
  {/if}

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="font-mono text-xs text-slate-500 tracking-widest">REGISTERED DEVICES ({filteredDevices.length})</div>
      <div class="flex gap-2 flex-wrap">
        <input
          bind:value={deviceSearch}
          placeholder="Search MAC / HW ID / device..."
          class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-60"
        />
        <select
          bind:value={statusFilter}
          class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-300 focus:outline-none focus:border-cyan-400/40"
        >
          <option value="all">All status</option>
          <option value="registered">Registered</option>
          <option value="active">Active</option>
          <option value="suspended">Suspended</option>
          <option value="revoked">Revoked</option>
        </select>
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Device ID</th>
            <th>MAC</th>
            <th>Hardware ID</th>
            <th>Model</th>
            <th>Registered</th>
            <th>Revoked</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredDevices as device (device.deviceId)}
            <tr class="animate-fade-in">
              <td class="text-cyan-400">{device.deviceId}</td>
              <td class="text-slate-300">{device.mac}</td>
              <td class="text-slate-500">{device.hardwareId}</td>
              <td class="text-slate-500">{device.model ?? '-'}</td>
              <td class="text-slate-500">{relTime(device.registeredAt)}</td>
              <td class="text-slate-500">{device.revokedAt ? relTime(device.revokedAt) : '-'}</td>
              <td><span class="{statusBadge(device.status)}">{device.status}</span></td>
              <td>
                <div class="flex gap-1">
                  {#if device.status === 'active' || device.status === 'registered'}
                    <button
                      type="button"
                      on:click={() => suspendDevice(device)}
                      class="px-2 py-1 font-mono text-xs text-amber-400 border border-amber-400/20 rounded hover:bg-amber-400/10 transition-colors"
                    >
                      SUSPEND
                    </button>
                  {/if}
                  {#if device.status !== 'revoked'}
                    <button
                      type="button"
                      on:click={() => confirmRevoke = device}
                      class="px-2 py-1 font-mono text-xs text-rose-400 border border-rose-400/20 rounded hover:bg-rose-400/10 transition-colors"
                    >
                      REVOKE
                    </button>
                  {/if}
                </div>
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="8" class="text-center text-slate-600 py-8">No devices returned from API</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

  {#if confirmRevoke}
    <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 animate-fade-in">
      <div class="bg-space-800 border border-rose-500/30 rounded-lg p-6 max-w-sm w-full mx-4">
        <div class="font-display text-sm text-rose-400 mb-2">CONFIRM REVOCATION</div>
        <p class="font-sans text-xs text-slate-400 mb-1">
          Device: <span class="text-slate-200 font-mono">{confirmRevoke.deviceId}</span>
        </p>
        <p class="font-sans text-xs text-slate-500 mb-5">
          The backend will mark this device as revoked and deny future verification.
        </p>
        <div class="flex gap-3">
          <button
            type="button"
            on:click={() => confirmRevoke && revokeDevice(confirmRevoke)}
            class="flex-1 py-2 font-mono text-xs bg-rose-500/20 border border-rose-500/40 text-rose-400 rounded hover:bg-rose-500/30 transition-colors"
          >
            CONFIRM
          </button>
          <button
            type="button"
            on:click={() => confirmRevoke = null}
            class="flex-1 py-2 font-mono text-xs bg-space-900 border border-slate-700 text-slate-400 rounded hover:border-slate-500 transition-colors"
          >
            CANCEL
          </button>
        </div>
      </div>
    </div>
  {/if}
</main>
