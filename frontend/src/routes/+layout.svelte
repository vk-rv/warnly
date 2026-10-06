<script>
  import { page } from '$app/state';
  import { afterNavigate } from '$app/navigation';
  import { api } from '$lib/api';
  import Logo from '$lib/Logo.svelte';
  import Icon from '$lib/Icon.svelte';
  import './app.css';
  let { children } = $props();
  let error = $state('');
  let menuOpen = $state(false);
  let user = $derived(page.data.user);
  let fullName = $derived(
    [user?.Name, user?.Surname].filter(Boolean).join(' ') || user?.Username || 'Account'
  );
  let initials = $derived(
    (user?.Name?.[0] || user?.Username?.[0] || '') + (user?.Surname?.[0] || '')
  );
  let login = $derived(page.url.pathname === '/login');
  afterNavigate(() => {
    menuOpen = false;
  });
  async function logout() {
    try {
      await api('/session', { method: 'DELETE' });
      window.location.assign('/login');
    } catch (e) {
      error = e.message;
    }
  }
</script>

<svelte:head><title>Warnly — Exception monitoring</title></svelte:head>
<svelte:window
  onkeydown={(e) => {
    if (e.key === 'Escape') menuOpen = false;
  }}
/>
{#if !login}
  <header class="mobile-header">
    <button
      class="icon-button"
      aria-label="Toggle navigation"
      aria-expanded={menuOpen}
      aria-controls="sidebar"
      onclick={() => (menuOpen = !menuOpen)}><Icon name="menu" size={24} /></button
    >
    <a class="brand" href="/"><Logo /><span>Warnly</span></a><span class="avatar"
      >{initials.toUpperCase()}</span
    >
  </header>
  {#if menuOpen}<button
      class="mobile-overlay"
      aria-label="Close navigation"
      onclick={() => (menuOpen = false)}
    ></button>{/if}
  <aside id="sidebar" class="sidebar" class:mobile-open={menuOpen}>
    <a class="brand" href="/"><Logo /><span>Warnly</span></a>
    <div class="sidebar-sections">
      <p class="nav-label">Navigation</p>
      <nav aria-label="Main navigation">
        {#each [['/', 'Dashboard', 'home'], ['/projects', 'Services', 'services'], ['/alerts', 'Alerts', 'bell'], ['/analytics', 'Reports', 'chart'], ['/system', 'System', 'system']] as [url, label, icon]}
          {@const active =
            url === '/' ? page.url.pathname === '/' : page.url.pathname.startsWith(url)}
          <a href={url} class:active aria-current={active ? 'page' : undefined}
            ><Icon name={icon} /><span>{label}</span></a
          >
        {/each}
      </nav>
      <p class="nav-label">Settings</p>
      <nav aria-label="Settings and support">
        <a href="/settings" class:active={page.url.pathname.startsWith('/settings')}
          ><Icon name="settings" /><span>Settings</span></a
        >
        <a href="https://github.com/vk-rv/warnly" target="_blank" rel="noreferrer"
          ><Icon name="help" /><span>Help &amp; Support</span></a
        >
      </nav>
    </div>
    <div class="sidebar-profile">
      <span class="avatar">{initials.toUpperCase()}</span>
      <div class="profile-name">
        <strong>{fullName}</strong><small>{user?.Username || ''}</small>
      </div>
      <button class="icon-button" aria-label="Sign out" title="Sign out" onclick={logout}
        ><Icon name="logout" /></button
      >
    </div>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  </aside>
{/if}
<main
  class="main-content"
  class:login
  class:issues-page={page.url.pathname === '/'}
  class:issue-page={/\/issues\/\d+/.test(page.url.pathname)}
>
  {@render children()}
</main>
