<script lang="ts">
  import Topbar from '$lib/components/Topbar.svelte';
  import { alerts } from '$lib/stores';
  import { devices } from '$lib/api/mock';
  import type { Device, DeviceStatus } from '$lib/types';

  let deviceSearch = '';
  let statusFilter: DeviceStatus | 'all' = 'all';
  let activeTab: 'devices' | 'alerts' | 'rbac' = 'devices';
  let confirmRevoke: string | null = null;
  let exportMsg = '';

  $: filteredDevices = devices.filter(d => {
    const matchStatus = statusFilter === 'all' || d.status === statusFilter;
    const matchSearch = !deviceSearch ||
      d.mac.toLowerCase().includes(deviceSearch.toLowerCase()) ||
      d.owner.toLowerCase().includes(deviceSearch.toLowerCase()) ||
      d.hardwareId.toLowerCase().includes(deviceSearch.toLowerCase());
    return matchStatus && matchSearch;
  });

  let deviceList = [...devices];

  function revokeDevice(mac: string) {
    deviceList = deviceList.map(d => d.mac === mac ? { ...d, status: 'revoked' as DeviceStatus } : d);
    confirmRevoke = null;
    alerts.update(list => [{
      id: `a-${Date.now()}`,
      severity: 'info',
      message: `Device ${mac} revoked by admin`,
      timestamp: new Date(),
      resolved: false,
      category: 'security',
    }, ...list]);
  }

  function suspendDevice(mac: string) {
    deviceList = deviceList.map(d => d.mac === mac ? { ...d, status: 'suspended' as DeviceStatus } : d);
  }

  function exportCSV() {
    const rows = [
      ['MAC', 'Hardware ID', 'Model', 'Owner', 'Status', 'Tier', 'Data Used (GB)', 'Registered'],
      ...deviceList.map(d => [d.mac, d.hardwareId, d.model, d.owner, d.status, d.tier, d.dataUsedGb, new Date(d.registeredAt).toLocaleDateString()])
    ];
    const csv = rows.map(r => r.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a'); a.href = url; a.download = 'devices.csv'; a.click();
    URL.revokeObjectURL(url);
    exportMsg = 'Exported!';
    setTimeout(() => exportMsg = '', 2000);
  }

  const tierCounts = {
    Fixed: devices.filter(d => d.tier === 'Fixed').length,
    Mobile: devices.filter(d => d.tier === 'Mobile').length,
    Trial: devices.filter(d => d.tier === 'Trial').length,
  };

  function statusBadge(s: DeviceStatus) {
    if (s === 'active') return 'badge-online';
    if (s === 'suspended') return 'badge-warning';
    return 'badge-offline';
  }

  function relTime(d: Date): string {
    const mins = Math.floor((Date.now() - new Date(d).getTime()) / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    if (mins < 1440) return `${Math.floor(mins/60)}h ago`;
    return `${Math.floor(mins/1440)}d ago`;
  }

  const adminUsers = [
    { id: 'u-001', username: 'admin', email: 'admin@vnu.edu.vn', role: 'Super Admin', lastLogin: new Date(Date.now() - 300000), active: true },
    { id: 'u-002', username: 'operator1', email: 'op1@vnu.edu.vn', role: 'Network Operator', lastLogin: new Date(Date.now() - 3600000), active: true },
    { id: 'u-003', username: 'oldop', email: 'oldop@vnu.edu.vn', role: 'Network Operator', lastLogin: new Date(Date.now() - 7 * 86400000), active: false },
  ];
</script>

<Topbar title="SECURITY" subtitle="Device registry · provisioning · RBAC" />

<main class="flex-1 overflow-y-auto p-6 space-y-5">

  <!-- Summary Cards -->
  <div class="grid grid-cols-2 xl:grid-cols-4 gap-3">
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-500 tracking-widest mb-2">FIXED SUBSCRIBERS</div>
      <div class="font-display text-2xl text-amber-400">{tierCounts.Fixed}</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-500 tracking-widest mb-2">MOBILE SUBSCRIBERS</div>
      <div class="font-display text-2xl text-cyan-400">{tierCounts.Mobile}</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-500 tracking-widest mb-2">TRIAL ACCOUNTS</div>
      <div class="font-display text-2xl text-emerald-400">{tierCounts.Trial}</div>
    </div>
    <div class="stat-card">
      <div class="font-mono text-xs text-slate-500 tracking-widest mb-2">ACTIVE ALERTS</div>
      <div class="font-display text-2xl text-rose-400">{$alerts.filter(a => !a.resolved).length}</div>
    </div>
  </div>

  <!-- Tabs -->
  <div class="flex gap-1 border-b border-slate-800/60 pb-0">
    {#each [['devices','Device Registry'],['alerts','Security Alerts'],['rbac','Admin RBAC']] as [tab, label]}
      <button
        on:click={() => activeTab = tab as typeof activeTab}
        class="px-4 py-2 font-mono text-xs tracking-wide transition-colors border-b-2
          {activeTab === tab
            ? 'text-cyan-400 border-cyan-400'
            : 'text-slate-500 border-transparent hover:text-slate-300'}"
      >
        {label}
      </button>
    {/each}
  </div>

  <!-- ── Device Registry ── -->
  {#if activeTab === 'devices'}
    <div class="chart-container">
      <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
        <div class="font-mono text-xs text-slate-500 tracking-widest">REGISTERED DEVICES ({filteredDevices.length})</div>
        <div class="flex gap-2 flex-wrap">
          <input
            bind:value={deviceSearch}
            placeholder="Search MAC / owner / HW ID..."
            class="bg-space-900 border border-slate-700/50 rounded px-3 py-1 text-xs font-mono text-slate-300 placeholder-slate-600 focus:outline-none focus:border-cyan-400/40 w-52"
          />
          <select
            bind:value={statusFilter}
            class="bg-space-900 border border-slate-700/50 rounded px-2 py-1 text-xs font-mono text-slate-300 focus:outline-none focus:border-cyan-400/40"
          >
            <option value="all">All status</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="revoked">Revoked</option>
          </select>
          <button on:click={exportCSV} class="px-3 py-1 font-mono text-xs bg-cyan-400/10 border border-cyan-400/20 text-cyan-400 rounded hover:bg-cyan-400/20 transition-colors">
            {exportMsg || 'EXPORT CSV'}
          </button>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="data-table">
          <thead>
            <tr>
              <th>MAC</th>
              <th>Hardware ID</th>
              <th>Model</th>
              <th>Owner</th>
              <th>Tier</th>
              <th>Registered</th>
              <th>Last Seen</th>
              <th>Data (GB)</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredDevices as d (d.id)}
              <tr class="animate-fade-in">
                <td class="text-cyan-400 font-mono">{d.mac}</td>
                <td class="text-slate-500">{d.hardwareId}</td>
                <td class="text-slate-400">{d.model}</td>
                <td class="text-slate-300">{d.owner}</td>
                <td>
                  <span class="font-mono text-xs {d.tier === 'Fixed' ? 'text-amber-400' : d.tier === 'Mobile' ? 'text-cyan-400' : 'text-emerald-400'}">{d.tier}</span>
                </td>
                <td class="text-slate-500">{new Date(d.registeredAt).toLocaleDateString('vi-VN')}</td>
                <td class="text-slate-500">{relTime(d.lastSeen)}</td>
                <td class="text-slate-400">{d.dataUsedGb.toFixed(1)}</td>
                <td><span class="{statusBadge(deviceList.find(x => x.id === d.id)?.status || d.status)}">{deviceList.find(x => x.id === d.id)?.status || d.status}</span></td>
                <td>
                  <div class="flex gap-1">
                    {#if (deviceList.find(x => x.id === d.id)?.status || d.status) === 'active'}
                      <button
                        on:click={() => suspendDevice(d.mac)}
                        class="px-1.5 py-0.5 font-mono text-xs text-amber-400 border border-amber-400/20 rounded hover:bg-amber-400/10 transition-colors"
                      >SUSPEND</button>
                    {/if}
                    {#if (deviceList.find(x => x.id === d.id)?.status || d.status) !== 'revoked'}
                      <button
                        on:click={() => confirmRevoke = d.mac}
                        class="px-1.5 py-0.5 font-mono text-xs text-rose-400 border border-rose-400/20 rounded hover:bg-rose-400/10 transition-colors"
                      >REVOKE</button>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Confirm Revoke Modal -->
    {#if confirmRevoke}
      <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 animate-fade-in">
        <div class="bg-space-800 border border-rose-500/30 rounded-lg p-6 max-w-sm w-full mx-4">
          <div class="font-display text-sm text-rose-400 mb-2">CONFIRM REVOCATION</div>
          <p class="font-sans text-xs text-slate-400 mb-1">Device MAC: <span class="text-slate-200 font-mono">{confirmRevoke}</span></p>
          <p class="font-sans text-xs text-slate-500 mb-5">This action will immediately deny all connections from this device. It cannot be undone without re-registering the device.</p>
          <div class="flex gap-3">
            <button on:click={() => confirmRevoke && revokeDevice(confirmRevoke)} class="flex-1 py-2 font-mono text-xs bg-rose-500/20 border border-rose-500/40 text-rose-400 rounded hover:bg-rose-500/30 transition-colors">
              CONFIRM REVOKE
            </button>
            <button on:click={() => confirmRevoke = null} class="flex-1 py-2 font-mono text-xs bg-space-900 border border-slate-700 text-slate-400 rounded hover:border-slate-500 transition-colors">
              CANCEL
            </button>
          </div>
        </div>
      </div>
    {/if}

  <!-- ── Security Alerts ── -->
  {:else if activeTab === 'alerts'}
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">ALL ALERTS (RESOLVED + ACTIVE)</div>
      <div class="space-y-2">
        {#each $alerts as alert (alert.id)}
          {@const cfg = { critical: { c: 'text-rose-400', bg: 'bg-rose-400/8', b: 'border-rose-400/20', i: '✕' }, warning: { c: 'text-amber-400', bg: 'bg-amber-400/8', b: 'border-amber-400/20', i: '△' }, info: { c: 'text-cyan-400', bg: 'bg-cyan-400/8', b: 'border-cyan-400/20', i: '◉' } }[alert.severity]}
          <div class="flex items-start gap-3 p-3 rounded {cfg.bg} border {cfg.b} {alert.resolved ? 'opacity-40' : ''}">
            <span class="font-mono {cfg.c} shrink-0">{cfg.i}</span>
            <div class="flex-1">
              <p class="font-sans text-xs text-slate-300">{alert.message}</p>
              <div class="flex gap-3 mt-1">
                <span class="font-mono text-xs text-slate-600">{relTime(alert.timestamp)}</span>
                <span class="font-mono text-xs text-slate-600 uppercase">{alert.category}</span>
                {#if alert.resolved}
                  <span class="font-mono text-xs text-emerald-400">RESOLVED</span>
                {/if}
              </div>
            </div>
            {#if !alert.resolved}
              <button
                on:click={() => alerts.update(list => list.map(a => a.id === alert.id ? { ...a, resolved: true } : a))}
                class="px-2 py-0.5 font-mono text-xs text-slate-500 border border-slate-700 rounded hover:border-slate-500 transition-colors shrink-0"
              >RESOLVE</button>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Suspicious activity summary -->
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
      <div class="chart-container">
        <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">SUSPICIOUS ACTIVITY FLAGS</div>
        <div class="space-y-2">
          <div class="p-2.5 rounded bg-rose-400/5 border border-rose-400/15">
            <div class="font-mono text-xs text-rose-400 mb-1">GEO-FENCE BREACH</div>
            <div class="font-sans text-xs text-slate-400">Device FE:DC:BA:98:76:54 (Bùi Thị H) moved outside Fixed zone boundary by 12km</div>
          </div>
          <div class="p-2.5 rounded bg-amber-400/5 border border-amber-400/15">
            <div class="font-mono text-xs text-amber-400 mb-1">UNREGISTERED DEVICE</div>
            <div class="font-sans text-xs text-slate-400">Connection attempt from 99:88:77:66:55:44 on HCMC GW — denied</div>
          </div>
        </div>
      </div>
      <div class="chart-container">
        <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">ADMIN AUDIT LOG</div>
        <div class="space-y-1.5">
          {#each [
            { action: 'Device BA:DC:AF:E0:01:23 revoked', user: 'admin', time: '2h ago' },
            { action: 'Device export generated (May 2025)', user: 'admin', time: '3h ago' },
            { action: 'Device 12:34:56:78:9A:BC suspended', user: 'admin', time: '5h ago' },
            { action: 'Admin login from 192.168.1.100', user: 'operator1', time: '8h ago' },
          ] as entry}
            <div class="flex gap-3 py-1.5 border-b border-slate-800/50 last:border-0">
              <span class="font-mono text-xs text-slate-600 shrink-0 w-14">{entry.time}</span>
              <span class="font-sans text-xs text-slate-400 flex-1">{entry.action}</span>
              <span class="font-mono text-xs text-cyan-400 shrink-0">{entry.user}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>

  <!-- ── RBAC ── -->
  {:else if activeTab === 'rbac'}
    <div class="chart-container">
      <div class="flex items-center justify-between mb-3">
        <div class="font-mono text-xs text-slate-500 tracking-widest">ADMIN USERS & ROLES</div>
        <button class="px-3 py-1 font-mono text-xs bg-cyan-400/10 border border-cyan-400/20 text-cyan-400 rounded hover:bg-cyan-400/20 transition-colors">
          + ADD USER
        </button>
      </div>
      <table class="data-table">
        <thead>
          <tr><th>Username</th><th>Email</th><th>Role</th><th>Last Login</th><th>Status</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {#each adminUsers as u (u.id)}
            <tr>
              <td class="text-cyan-400">{u.username}</td>
              <td class="text-slate-500">{u.email}</td>
              <td>
                <span class="font-mono text-xs {u.role === 'Super Admin' ? 'text-rose-400' : u.role === 'Network Operator' ? 'text-cyan-400' : 'text-amber-400'}">
                  {u.role}
                </span>
              </td>
              <td class="text-slate-500">{relTime(u.lastLogin)}</td>
              <td>
                <span class="{u.active ? 'badge-online' : 'badge-offline'}">{u.active ? 'active' : 'disabled'}</span>
              </td>
              <td>
                <div class="flex gap-1">
                  <button class="px-1.5 py-0.5 font-mono text-xs text-slate-500 border border-slate-700 rounded hover:text-slate-300 transition-colors">EDIT</button>
                  {#if u.active}
                    <button class="px-1.5 py-0.5 font-mono text-xs text-amber-400 border border-amber-400/20 rounded hover:bg-amber-400/10 transition-colors">DISABLE</button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Role permissions -->
    <div class="chart-container">
      <div class="font-mono text-xs text-slate-500 mb-3 tracking-widest">ROLE PERMISSIONS MATRIX</div>
      <table class="data-table text-xs">
        <thead>
          <tr>
            <th>Permission</th>
            <th class="text-rose-400">Super Admin</th>
            <th class="text-cyan-400">Net. Operator</th>
          </tr>
        </thead>
        <tbody>
          {#each [
            ['View Overview Dashboard', true, true],
            ['View Monitoring', true, true],
            ['View Security', true, false],
            ['Revoke Devices', true, false],
            ['Manage Admin Users', true, false],
            ['Modify Gateway Config', true, true],
          ] as [perm, sa, op]}
            <tr>
              <td class="text-slate-400">{perm}</td>
              <td class="text-center">{sa ? '✓' : '—'}</td>
              <td class="text-center {op ? 'text-emerald-400' : 'text-slate-700'}">{op ? '✓' : '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

</main>
