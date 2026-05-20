<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import { onMount } from 'svelte';
  import {
    devices,
    lastError,
    refreshDevices,
    revokeDeviceByMac,
    securityAlerts,
    suspendDeviceByMac,
  } from '$lib/stores';
  import type { Device, DeviceStatus, SecurityAlert } from '$lib/types';

  const devicesEnabled = import.meta.env.VITE_ENABLE_DEVICES === 'true';

  let deviceSearch = '';
  let statusFilter: DeviceStatus | 'all' = 'all';
  let confirmRevoke: Device | null = null;
  let actionMessage = '';
  let actionError = '';
  let page = 0;
  const PAGE_SIZE = 15;
  let sortKey: keyof Device = 'registeredAt';
  let sortDir: 'asc' | 'desc' = 'desc';

  $: filteredDevices = $devices.filter((device) => {
    const statusMatch = statusFilter === 'all' || device.status === statusFilter;
    const search = deviceSearch.trim().toLowerCase();
    const searchMatch =
      !search ||
      device.deviceId.toLowerCase().includes(search) ||
      device.mac.toLowerCase().includes(search) ||
      device.hardwareId.toLowerCase().includes(search);
    return statusMatch && searchMatch;
  });

  $: sortedDevices = [...filteredDevices].sort((a, b) => {
    const dir = sortDir === 'asc' ? 1 : -1;
    const av = a[sortKey];
    const bv = b[sortKey];
    if (av instanceof Date && bv instanceof Date) return (av.getTime() - bv.getTime()) * dir;
    return String(av ?? '').localeCompare(String(bv ?? '')) * dir;
  });

  $: pagedDevices = sortedDevices.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);
  $: totalPages = Math.max(1, Math.ceil(sortedDevices.length / PAGE_SIZE));

  onMount(() => {
    if (devicesEnabled) {
      void refreshDevices();
    }
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

  function toggleSort(key: keyof Device) {
    if (sortKey === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortKey = key;
      sortDir = 'asc';
    }
  }

  async function suspendDevice(device: Device) {
    if (!devicesEnabled) {
      actionError = 'Devices endpoint is disabled';
      return;
    }
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
    if (!devicesEnabled) {
      actionError = 'Devices endpoint is disabled';
      return;
    }
    actionMessage = '';
    actionError = '';
    try {
      await revokeDeviceByMac(device.mac);
      actionMessage = `Device ${device.mac} revoked`;
      confirmRevoke = null;
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Device revoke failed';
    }
  }

  function severityBadge(alert: SecurityAlert) {
    if (alert.severity === 'high') return 'badge-offline';
    if (alert.severity === 'medium') return 'badge-warning';
    return 'badge-online';
  }
</script>

<Topbar title="SECURITY" subtitle="Device registry and security controls" />

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
  {#if !devicesEnabled}
    <div class="rounded border border-amber-400/30 bg-amber-400/8 px-4 py-3 font-mono text-xs text-amber-300">
      Devices endpoint is disabled. Set VITE_ENABLE_DEVICES=true to enable device registry calls.
    </div>
  {/if}

  <section class="chart-container">
    <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
      <div class="font-mono text-xs text-slate-500 tracking-widest">REGISTERED DEVICES ({sortedDevices.length})</div>
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
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSort('deviceId')}>Device ID</button></th>
            <th>MAC</th>
            <th>Hardware ID</th>
            <th>Model</th>
            <th>Owner</th>
            <th><button type="button" class="font-mono text-xs" on:click={() => toggleSort('registeredAt')}>Registered</button></th>
            <th>Revoked</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each pagedDevices as device (device.deviceId)}
            <tr class="animate-fade-in">
              <td class="text-cyan-400">{device.deviceId}</td>
              <td class="text-slate-300">{device.mac}</td>
              <td class="text-slate-500">{device.hardwareId}</td>
              <td class="text-slate-500">{device.model ?? '-'}</td>
              <td class="text-slate-500">{device.owner ?? '-'}</td>
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
              <td colspan="9" class="text-center text-slate-600 py-8">No devices returned from API</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <div class="flex items-center justify-between mt-3">
      <span class="font-mono text-xs text-slate-600">
        {sortedDevices.length === 0 ? 0 : page * PAGE_SIZE + 1}-{Math.min((page + 1) * PAGE_SIZE, sortedDevices.length)} of {sortedDevices.length}
      </span>
      <div class="flex gap-1">
        <button type="button" on:click={() => page = Math.max(0, page - 1)} disabled={page === 0} class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors">PREV</button>
        <button type="button" on:click={() => page = Math.min(totalPages - 1, page + 1)} disabled={page >= totalPages - 1} class="px-2 py-1 font-mono text-xs bg-space-900 border border-slate-700/50 rounded disabled:opacity-30 hover:border-cyan-400/30 transition-colors">NEXT</button>
      </div>
    </div>
  </section>

  <section class="chart-container">
    <div class="font-mono text-xs text-slate-500 tracking-widest mb-3">SUSPICIOUS ACTIVITY ALERTS</div>
    <div class="overflow-x-auto max-h-72 overflow-y-auto">
      <table class="data-table">
        <thead>
          <tr>
            <th>Timestamp</th>
            <th>Type</th>
            <th>Device/MAC</th>
            <th>Message</th>
            <th>Severity</th>
          </tr>
        </thead>
        <tbody>
          {#each $securityAlerts as alert (alert.id)}
            <tr class="animate-fade-in">
              <td class="text-slate-500 whitespace-nowrap">{alert.timestamp.toLocaleString('en-GB', { hour12: false })}</td>
              <td class="text-slate-300">{alert.type.replace('_', ' ')}</td>
              <td class="text-slate-400">{alert.deviceId ?? alert.mac ?? '-'}</td>
              <td class="text-slate-400">{alert.message}</td>
              <td><span class="{severityBadge(alert)}">{alert.severity.toUpperCase()}</span></td>
            </tr>
          {:else}
            <tr>
              <td colspan="5" class="text-center text-slate-600 py-8">No security alerts</td>
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
