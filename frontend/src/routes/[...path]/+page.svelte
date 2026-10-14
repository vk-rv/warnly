<script>
  import { goto, invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { api, formBody } from '$lib/api';
  import IssueTable from '$lib/IssueTable.svelte';
  import Pagination from '$lib/Pagination.svelte';
  import AlertForm from '$lib/AlertForm.svelte';
  import IssueFields from '$lib/IssueFields.svelte';
  import DeleteDialog from '$lib/DeleteDialog.svelte';
  let deletion = $state(null);
  import SystemView from '$lib/SystemView.svelte';
  import Discussion from '$lib/Discussion.svelte';
  import Icon from '$lib/Icon.svelte';
  import Logo from '$lib/Logo.svelte';
  import EventChart from '$lib/EventChart.svelte';
  import TimeRange from '$lib/TimeRange.svelte';
  import SearchFilter from '$lib/SearchFilter.svelte';
  import IssueOverview from '$lib/IssueOverview.svelte';
  import { number } from '$lib/format';
  let { data } = $props();
  let busy = $state(false);
  let actionError = $state('');
  let saved = $state('');
  let created = $state(null);
  function notify(message) {
    saved = message;
  }
  $effect(() => {
    if (saved) {
      const timer = setTimeout(() => (saved = ''), 3000);
      return () => clearTimeout(timer);
    }
  });
  let path = $derived(data.path);
  let result = $derived(data.result);
  let issue = $derived(data.issue || result);
  let previousPath = '';
  $effect(() => {
    if (path !== previousPath) {
      previousPath = path;
      actionError = '';
      saved = '';
      created = null;
    }
  });
  let parts = $derived(path.split('/').filter(Boolean));
  let issuePath = $derived(`/projects/${parts[1]}/issues/${parts[3]}`);
  let isIssue = $derived(parts[0] === 'projects' && parts[2] === 'issues');
  let period = $derived(page.url.searchParams.get('period') || result?.period || '14d');
  let title = $derived(
    path === '/'
      ? 'Issues'
      : path === '/login'
        ? 'Sign in to Warnly'
        : path === '/projects'
          ? 'Projects'
          : path === '/projects/new'
            ? 'Create project'
            : isIssue
              ? issue?.error_type || 'Issue details'
              : parts[0] === 'projects'
                ? result?.project?.name || result?.name || 'Project'
                : parts[0] === 'alerts'
                  ? parts[1] === 'new'
                    ? 'Create alert'
                    : parts[2] === 'edit'
                      ? 'Edit alert'
                      : 'Alerts'
                  : parts[0] === 'settings'
                    ? 'Settings'
                    : parts[0] === 'system'
                      ? 'System'
                      : path === '/oncall'
                        ? 'On-call'
                        : path === '/analytics'
                          ? 'Reports'
                          : 'Warnly'
  );
  async function mutate(endpoint, method, body, destination) {
    busy = true;
    actionError = '';
    saved = '';
    try {
      const response = await api(endpoint, { method, ...(body ? { body } : {}) });
      if (destination) await goto(destination, { invalidateAll: true });
      else {
        await invalidateAll();
        if (!endpoint.includes('/discussions')) notify('Saved');
      }
      return response;
    } catch (error) {
      actionError = error.message;
    } finally {
      busy = false;
    }
  }
  async function submit(event) {
    event.preventDefault();
    const body = formBody(event.currentTarget);
    if (path === '/login') await mutate('/login', 'POST', body, '/');
    else if (path === '/projects/new') {
      const response = await mutate('/projects', 'POST', body);
      if (response) created = response;
    } else if (path === '/alerts/new') await mutate('/alerts', 'POST', body, '/alerts');
    else if (parts[0] === 'alerts') await mutate(`/alerts/${parts[1]}`, 'PUT', body, '/alerts');
    else if (path === '/settings') await mutate('/settings/webhook', 'POST', body);
    else if (parts[4] === 'discussions') {
      const form = event.currentTarget;
      await mutate(`${issuePath}/discussions`, 'POST', body);
      if (!actionError) form.reset();
    }
  }
  function remove(endpoint, destination, name = '') {
    const kind = endpoint.includes('/discussions/')
      ? 'comment'
      : endpoint.startsWith('/alerts/')
        ? 'alert'
        : 'project';
    deletion = { endpoint, destination, kind, name };
  }
  async function confirmDeletion() {
    const target = deletion;
    await api(target.endpoint, { method: 'DELETE' });
    if (target.destination) await goto(target.destination, { invalidateAll: true });
    else await invalidateAll();
    notify('Deleted');
  }
  function query(event) {
    event.preventDefault();
    const params = formBody(event.currentTarget);
    if (params.get('start') && params.get('end')) params.delete('period');
    for (const [key, value] of [...params]) if (!value) params.delete(key);
    goto(`${path}?${params}`);
  }
  function changeFilter(key, value) {
    const q = new URLSearchParams(page.url.searchParams);
    if (value) q.set(key, value);
    else q.delete(key);
    q.delete('offset');
    q.delete('page');
    goto(`${path}?${q}`);
  }
  function dsn(project) {
    const url = new URL(window.location.origin);
    url.username = project.key;
    url.pathname = `/ingest/${project.id}`;
    return url.toString();
  }
</script>

<svelte:head><title>{title} · Warnly</title></svelte:head>
{#if path !== '/login' && parts[0] !== 'system'}
  {#if result?.project}<div class="breadcrumbs">
      <a href="/projects">Projects</a><span>/</span><span>Project Details</span>
    </div>{/if}
  {#if isIssue}<div class="breadcrumbs">
      <a href={`/projects/${parts[1]}`}>Issues</a><Icon name="right" size={12} /><span
        >{issue?.project_name || 'Issue Details'}</span
      >
    </div>{/if}
  <header class="page-heading" class:issue-heading={isIssue}>
    <div class="heading-title">
      {#if result?.project}<span class="project-avatar large">{result.project.name?.[0]}</span>{/if}
      <h1>{title}</h1>
      {#if isIssue && issue?.error_type}<span class="issue-view">{issue.view}</span><span
          class="badge"
          class:warning={issue.is_new}>{issue.is_new ? 'New' : 'Ongoing'}</span
        >{/if}
    </div>
    {#if path === '/projects'}<a class="button primary" href="/projects/new">Create Project</a>{/if}
    {#if result?.project}<div class="toolbar heading-actions">
        <a
          class="icon-button"
          title="Setup instructions"
          aria-label="Setup instructions"
          href={`/projects/${result.project.id}/getting-started`}><Icon name="help" /></a
        ><a
          class="icon-button"
          title="Project settings"
          aria-label="Project settings"
          href={`/settings/projects/${result.project.id}`}><Icon name="settings" /></a
        >
      </div>{/if}
    {#if isIssue && issue?.error_type}<div class="issue-header-metrics">
        <span>Errors <strong>{number(issue.times_seen)}</strong></span><span
          >Users <strong>{number(issue.user_count)}</strong></span
        >
      </div>{/if}
  </header>
  {#if path === '/' || path === '/alerts'}<nav class="tabs page-tabs">
      <a class="active" href={path}>{path === '/' ? 'All Issues' : 'All Alerts'}</a>
    </nav>{/if}
{/if}
{#if (path !== '/login' || result === null) && (data.error || actionError)}<p
    class="error"
    role="alert"
  >
    {actionError || data.error}
  </p>{/if}
{#if saved}<div class="toast-container">
    <div class="toast-message" role="status">{saved}</div>
  </div>{/if}
{#if result !== null}
  {#if path === '/login'}
    <div class="login-layout">
      <section class="login-fields">
        <div>
          {#if result.auth_url}<h1>Sign in</h1>{/if}
          {#if actionError || data.error}<p id="login-error" class="login-error" role="alert">
              {actionError || data.error}
            </p>{/if}
          {#if !result.auth_url}<form class="login-form" onsubmit={submit} aria-busy={busy}>
              <label
                >Email or username<input
                  name="identifier"
                  autocomplete="username"
                  aria-describedby={actionError ? 'login-error' : undefined}
                  aria-invalid={!!actionError}
                  required
                /></label
              >
              <label
                >Password<input
                  name="password"
                  type="password"
                  autocomplete="current-password"
                  aria-describedby={actionError ? 'login-error' : undefined}
                  aria-invalid={!!actionError}
                  required
                /></label
              >
              <label class="check"><input type="checkbox" name="remember-me" /> Remember me</label>
              <button class="primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
            </form>{:else}<a class="button oidc-button" href={result.auth_url} data-sveltekit-reload
              ><Icon name="user" />Sign in with {result.provider_name}</a
            >{/if}
          <p class="login-help">
            Running into trouble? 😊 Check out our <a
              href="https://docs.warnly.io"
              target="_blank"
              rel="noreferrer">documentation</a
            > — it has answers to most questions!
          </p>
        </div>
      </section>
      <section class="login-intro">
        <div>
          <div class="login-brand"><span>Warnly</span></div>
          <div class="login-feature">
            <h3>Self-Hosted Simplicity</h3>
            <p>
              Take control with self-hosted monitoring that eliminates enterprise complexity and
              puts you in charge.
            </p>
          </div>
          <div class="login-feature">
            <h3>Open-Source Excellence</h3>
            <p>
              Categorize errors into issues, assign to team members, and keep your apps running
              smoothly.
            </p>
          </div>
          <div class="login-feature">
            <h3>Single Binary Deployment</h3>
            <p>Enjoy essential monitoring features without the overhead.</p>
          </div>
          <div class="login-callout">
            {#if result.is_demo}<p>This demo is graciously sponsored by VPSDime ❤️</p>
              <a href="https://vpsdime.com/">Visit VPSDime</a>{:else}<p>
                Warnly is a MIT-licensed project in alpha stage. If you find it useful, consider
                contributing to the project.
              </p>
              <a href="https://github.com/vk-rv/warnly">Contribute to the project</a>{/if}
          </div>
        </div>
      </section>
    </div>
  {:else if path === '/' || (parts[0] === 'projects' && parts.length === 2 && parts[1] !== 'new')}
    <div class="filters-row">
      {#if path === '/'}<select
          aria-label="Project"
          class="filter-select"
          value={page.url.searchParams.get('project_name') || ''}
          onchange={(e) => changeFilter('project_name', e.currentTarget.value)}
          ><option value="">All Projects</option>{#each result.projects || [] as p}<option
              value={p.name}>{p.name}</option
            >{/each}</select
        >{/if}
      <TimeRange {period} />
    </div>
    {#if path === '/'}<SearchFilter tags={result.popular_tags || []} />{:else}
      <section class="chart-panel">
        <header>Number of Errors</header>
        <EventChart
          events={result.project?.events || []}
          period={result.period || period}
          start={page.url.searchParams.get('start') || ''}
          end={page.url.searchParams.get('end') || ''}
        />
        <footer>
          <strong>Total Errors:</strong>
          {number((result.project?.events || []).reduce((sum, e) => sum + e.count, 0))}
        </footer>
      </section>
      <nav class="tabs project-tabs">
        {#each [['all', 'All Issues', result.project.all_length], ['new', 'New Issues', result.project.new_length]] as [value, label, count]}<button
            class:active={(result.issues || 'all') === value}
            onclick={() => changeFilter('issues', value)}>{label} <span>{count || ''}</span></button
          >{/each}
      </nav>
    {/if}
    <IssueTable
      issues={result.project ? result.project.result_issue_list || [] : result.issues || []}
      projects={result.projects || [result.project]}
      project={result.project}
      assignments={result.assignments}
      teammates={result.teammates}
      {busy}
      assign={(id, userID) =>
        mutate(
          `/projects/${result.project.id}/issues/${id}/assignments`,
          userID ? 'POST' : 'DELETE',
          userID ? new URLSearchParams({ user_id: userID }) : null
        )}
    />
    <Pagination
      total={result.project
        ? result.issues === 'new'
          ? result.project.new_length
          : result.project.all_length
        : result.total_issues}
      size={result.project ? 5 : 50}
      pageMode={!!result.project}
    />
  {:else if path === '/projects'}
    <form class="project-filters" onsubmit={query}>
      <select
        name="team"
        aria-label="Team"
        value={page.url.searchParams.get('team') || ''}
        onchange={(e) => changeFilter('team', e.currentTarget.value)}
        ><option value="">All Teams</option>{#each result.teams || [] as team}<option
            value={team.id}>{team.name}</option
          >{/each}</select
      >
      <div class="project-search">
        <Icon name="search" /><input
          name="name"
          aria-label="Name"
          placeholder="Search for services"
          value={page.url.searchParams.get('name') || ''}
        /><button class="icon-button" aria-label="Search projects"><Icon name="right" /></button>
      </div>
    </form>
    <div class="project-grid">
      {#each result.projects || [] as project}<article class="project-card">
          <header>
            <span class="project-avatar">{project.name?.[0]}</span><a
              href={`/projects/${project.id}`}
              ><h2>{project.name}</h2>
              <p>
                Errors: {number((project.events || []).reduce((sum, e) => sum + e.count, 0))} | Last 24
                hours
              </p></a
            ><a
              href={`/settings/projects/${project.id}`}
              class="icon-button"
              aria-label={`Settings for ${project.name}`}><Icon name="settings" /></a
            >
          </header>
          <EventChart
            events={project.events || []}
            period="24h"
            label={`Errors for ${project.name}`}
          />
          <footer></footer>
        </article>{:else}<p class="muted">You need at least one project to use this page.</p>{/each}
    </div>
  {:else if path === '/projects/new'}
    {#if created}<section class="panel">
        <h2>Project created: {created.name}</h2>
        <p>Use this DSN with your Sentry SDK:</p>
        <pre>{created.dsn}</pre>
        <a href={`/projects/${created.id}`}>Open project</a>
      </section>
    {:else}<form class="form panel" onsubmit={submit}>
        <label>Project name<input name="projectName" required /></label><label
          >Platform<select name="platform"
            ><option value="golang">Go</option><option value="rust">Rust</option></select
          ></label
        ><label
          >Team<select name="team" required
            >{#each result || [] as team}<option value={team.id}>{team.name}</option>{/each}</select
          ></label
        ><button class="primary" disabled={busy || !result.length}>Create project</button>
      </form>{/if}
  {:else if (parts[0] === 'settings' && parts[1] === 'projects') || parts[2] === 'getting-started'}
    <section class="panel">
      <h2>{result.name}</h2>
      <p>Configure the Sentry SDK with this dsn:</p>
      <pre>{dsn(result)}</pre>
      <pre>{result.platform === 2
          ? `let _guard = sentry::init("${dsn(result)}");\nsentry::capture_message("It works!", sentry::Level::Info);`
          : `sentry.Init(sentry.ClientOptions{Dsn: "${dsn(result)}"})\ndefer sentry.Flush(2 * time.Second)\nsentry.CaptureMessage("It works!")`}</pre>
      <a class="button" href={`/projects/${result.id}`}>Open project</a>
      {#if parts[0] === 'settings'}<button
          class="danger"
          disabled={busy}
          onclick={() => remove(`/projects/${result.id}`, '/projects', result.name)}
          >Delete project</button
        >{/if}
    </section>
  {:else if isIssue}
    <p class="issue-error-value">{issue?.error_value || issue?.message || ''}</p>
    <nav class="tabs issue-tabs">
      <a class:active={parts.length === 4} href={issuePath}>Info</a><a
        class:active={parts[4] === 'discussions'}
        href={`${issuePath}/discussions`}>Discussion <span>{issue?.messages_count || ''}</span></a
      ><a class:active={parts[4] === 'fields'} href={`${issuePath}/fields`}>Fields</a><a
        class:active={parts[4] === 'events'}
        href={`${issuePath}/events`}>All Errors</a
      >
    </nav>
    {#if parts.length === 4}<IssueOverview {issue} {issuePath} {mutate} {busy} />
    {:else if parts[4] === 'discussions'}
      <Discussion
        {result}
        user={data.user}
        {issuePath}
        {mutate}
        {remove}
        {busy}
        error={actionError}
      />
    {:else if parts[4] === 'fields'}<IssueFields {result} />
    {:else if parts[4] === 'events'}
      <form class="filters" onsubmit={query}>
        <label>Search<input name="query" value={page.url.searchParams.get('query') || ''} /></label
        ><label
          >Period<select name="period" value={page.url.searchParams.get('period') || '90d'}
            >{#each ['1h', '24h', '7d', '14d', '30d', '90d'] as value}<option>{value}</option
              >{/each}</select
          ></label
        ><button>Filter</button>
      </form>
      <div class="table-wrap">
        <table>
          <thead><tr><th>Event</th><th>Time</th><th>Environment</th><th>User</th></tr></thead><tbody
            class="event-rows"
            >{#each result.events || [] as event}<tr
                ><td
                  ><a class="event-row-link" href={`${issuePath}?event_id=${event.event_id}`}
                    >{event.title || event.event_id}</a
                  >
                  <p>{event.message}</p></td
                ><td>{new Date(event.created_at).toLocaleString()}</td><td>{event.env}</td><td
                  >{event.user_email || event.user_username || event.user}</td
                ></tr
              >{:else}<tr><td colspan="4">No events found.</td></tr>{/each}</tbody
          >
        </table>
      </div>
      <Pagination total={result.total_events} />
    {/if}
  {:else if path === '/alerts'}
    <div class="filters-row alert-filters">
      <select
        aria-label="Team"
        class="filter-select"
        value={page.url.searchParams.get('team_name') || ''}
        onchange={(e) => changeFilter('team_name', e.currentTarget.value)}
        ><option value="">My Teams</option>{#each result.teams || [] as team}<option
            value={team.name}>{team.name}</option
          >{/each}</select
      ><select
        aria-label="Project"
        class="filter-select"
        value={page.url.searchParams.get('project_name') || ''}
        onchange={(e) => changeFilter('project_name', e.currentTarget.value)}
        ><option value="">All Projects</option>{#each result.projects || [] as p}<option
            value={p.name}>{p.name}</option
          >{/each}</select
      ><a class="button" href="/alerts/new"><Icon name="plus" />Create Alert</a>
    </div>
    <div class="table-wrap">
      <table class="alerts-table">
        <thead
          ><tr><th>Alert rule</th><th>Status</th><th>Project</th><th>Team</th><th>Actions</th></tr
          ></thead
        ><tbody
          >{#each result.alerts || [] as alert}<tr
              ><td
                ><a href={`/alerts/${alert.id}/edit`}
                  ><strong>{alert.rule_name}</strong>
                  <p>{alert.description}</p></a
                ></td
              ><td
                ><span
                  class="badge"
                  class:success-badge={alert.status === 'Active'}
                  class:danger-badge={alert.status === 'Triggered'}>{alert.status}</span
                ></td
              ><td>{result.projects?.find((p) => p.id === alert.project_id)?.name}</td><td
                >{result.teams?.find((t) => t.id === alert.team_id)?.name}</td
              ><td
                ><button
                  class="danger small"
                  disabled={busy}
                  onclick={() => remove(`/alerts/${alert.id}`, null, alert.rule_name)}
                  >Delete</button
                ></td
              ></tr
            >{:else}<tr><td colspan="5" class="empty-state">No alert rules.</td></tr>{/each}</tbody
        >
      </table>
    </div>
    <Pagination total={result.total_alerts} />
  {:else if parts[0] === 'alerts'}<AlertForm {result} {submit} {busy} />
  {:else if path === '/settings'}
    <h2>Webhook notifications</h2>
    <form class="form panel" onsubmit={submit}>
      <input type="hidden" name="team_id" value={result.webhook?.team_id || 1} /><label
        >URL<input
          name="url"
          type="url"
          value={result.webhook?.url || ''}
          placeholder="https://example.com/webhook"
        /></label
      ><label
        >Secret<input
          name="secret"
          type="password"
          value={result.webhook?.secret || ''}
          autocomplete="off"
        /></label
      ><button class="primary" disabled={busy}>Save webhook</button>
    </form>
  {:else if parts[0] === 'system'}<SystemView rows={result} {path} {notify} />
  {/if}
{:else if !data.error}
  <section class="panel">
    <p>
      {path === '/error'
        ? 'Something went wrong. Please try again.'
        : 'This feature is in development.'}
    </p>
    <a href="/">Back to issues</a>
  </section>
{/if}

{#if deletion}<DeleteDialog
    target={deletion}
    onclose={() => (deletion = null)}
    onconfirm={confirmDeletion}
  />{/if}
