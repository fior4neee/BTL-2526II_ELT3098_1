<script lang="ts">
  import { page } from '$app/stores';

  const nav = [
    { href: '/', label: 'Overview', icon: 'OV' },
    { href: '/monitoring', label: 'Monitoring', icon: 'MO' },
    { href: '/security', label: 'Security', icon: 'SE' },
  ];

  function normalizePath(path: string) {
    if (path === '/') return '/';
    return path.endsWith('/') ? path.slice(0, -1) : path;
  }

  $: currentPath = normalizePath($page.url.pathname);
</script>

<aside class="w-56 min-h-screen bg-space-900 border-r border-slate-800/60 flex flex-col">
  <div class="p-5 border-b border-slate-800/60">
    <div class="flex items-center gap-2">
      <div class="relative w-8 h-8">
        <svg viewBox="0 0 32 32" class="w-8 h-8" aria-hidden="true">
          <circle cx="16" cy="16" r="14" fill="none" stroke="#22d3ee" stroke-width="1.5" opacity="0.6"/>
          <circle cx="16" cy="16" r="8" fill="none" stroke="#22d3ee" stroke-width="1" opacity="0.4"/>
          <ellipse cx="16" cy="16" rx="14" ry="5" fill="none" stroke="#22d3ee" stroke-width="1" opacity="0.3"/>
          <circle cx="16" cy="2" r="2" fill="#22d3ee"/>
        </svg>
      </div>
      <div>
        <div class="font-display text-sm text-cyan-400 leading-none">VNU-LEO</div>
        <div class="font-mono text-xs text-slate-500 leading-none mt-0.5">ADMIN</div>
      </div>
    </div>
  </div>

  <nav class="flex-1 p-3 space-y-1">
    {#each nav as item}
      <a
        href={item.href}
        data-sveltekit-preload-data="hover"
        class="flex items-center gap-3 px-3 py-2.5 rounded text-sm transition-all duration-150
          {currentPath === normalizePath(item.href)
            ? 'bg-cyan-400/10 text-cyan-400 border border-cyan-400/20'
            : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50 border border-transparent'}"
      >
        <span class="font-mono text-xs w-5 text-center">{item.icon}</span>
        <span class="font-sans">{item.label}</span>
      </a>
    {/each}
  </nav>

  <div class="p-4 border-t border-slate-800/60">
    <div class="flex items-center gap-2">
      <span class="pulse-dot online"></span>
      <span class="font-mono text-xs text-slate-500">CORE API</span>
    </div>
    <div class="font-mono text-xs text-slate-600 mt-2">
      v1.0.0-alpha<br />
      VNU-ELT3098
    </div>
  </div>
</aside>
