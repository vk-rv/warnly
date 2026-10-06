<script>
  import { goto, invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { api, formBody } from '$lib/api';
  import IssueTable from '$lib/IssueTable.svelte';
  import Pagination from '$lib/Pagination.svelte';
  import AlertForm from '$lib/AlertForm.svelte';
  import SystemTable from '$lib/SystemTable.svelte';
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
  let period = $derived(page.url.searchParams.get('period') || result?.Period || '14d');
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
              ? issue?.ErrorType || 'Issue details'
              : parts[0] === 'projects'
                ? result?.Project?.Name || result?.Name || 'Project'
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
        saved = 'Saved';
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
  function remove(endpoint, destination) {
    if (window.confirm('Delete permanently?')) return mutate(endpoint, 'DELETE', null, destination);
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
    url.username = project.Key;
    url.pathname = `/ingest/${project.ID}`;
    return url.toString();
  }
</script>

<svelte:head><title>{title} · Warnly</title></svelte:head>
{#if path !== '/login'}
  {#if result?.Project}<div class="breadcrumbs">
      <a href="/projects">Projects</a><span>/</span><span>Project Details</span>
    </div>{/if}
  {#if isIssue}<div class="breadcrumbs">
      <a href={`/projects/${parts[1]}`}>Issues</a><Icon name="right" size={12} /><span
        >{issue?.ProjectName || 'Issue Details'}</span
      >
    </div>{/if}
  <header class="page-heading" class:issue-heading={isIssue}>
    <div class="heading-title">
      {#if result?.Project}<span class="project-avatar large">{result.Project.Name?.[0]}</span>{/if}
      <h1>{title}</h1>
      {#if isIssue && issue?.ErrorType}<span class="issue-view">{issue.View}</span><span
          class="badge"
          class:warning={issue.IsNew}>{issue.IsNew ? 'New' : 'Ongoing'}</span
        >{/if}
    </div>
    {#if path === '/projects'}<a class="button primary" href="/projects/new">Create Project</a>{/if}
    {#if result?.Project}<div class="toolbar heading-actions">
        <a
          class="icon-button"
          title="Setup instructions"
          aria-label="Setup instructions"
          href={`/projects/${result.Project.ID}/getting-started`}><Icon name="help" /></a
        ><a
          class="icon-button"
          title="Project settings"
          aria-label="Project settings"
          href={`/settings/projects/${result.Project.ID}`}><Icon name="settings" /></a
        >
      </div>{/if}
    {#if isIssue && issue?.ErrorType}<div class="issue-header-metrics">
        <span>Errors <strong>{number(issue.TimesSeen)}</strong></span><span
          >Users <strong>{number(issue.UserCount)}</strong></span
        >
      </div>{/if}
  </header>
  {#if path === '/' || path === '/alerts'}<nav class="tabs page-tabs">
      <a class="active" href={path}>{path === '/' ? 'All Issues' : 'All Alerts'}</a>
    </nav>{/if}
{/if}
{#if data.error || actionError}<p class="error" role="alert">{actionError || data.error}</p>{/if}
{#if saved}<p class="success" role="status">{saved}</p>{/if}
{#if result !== null}
  {#if path === '/login'}
    <div class="login-layout">
      <section class="login-fields">
        <div>
          <h1>Sign in</h1>
          <form class="login-form" onsubmit={submit}>
            <label
              >Email or username<input name="identifier" autocomplete="username" required /></label
            >
            <label
              >Password<input
                name="password"
                type="password"
                autocomplete="current-password"
                required
              /></label
            >
            <label class="check"><input type="checkbox" name="remember-me" /> Remember me</label>
            <button class="primary" disabled={busy}>Sign in</button>
            {#if result.AuthURL}<a class="button" href={result.AuthURL} data-sveltekit-reload
                >Continue with {result.ProviderName}</a
              >{/if}
            {#if result.IsDemo}<p class="muted">Demo credentials: admin / admin</p>{/if}
          </form>
        </div>
      </section>
      <section class="login-intro">
        <div>
          <div class="login-brand"><Logo /><span>Warnly</span></div>
          <div class="login-feature"><h3>Self-Hosted Simplicity</h3><p>Take control with self-hosted monitoring that eliminates enterprise complexity and puts you in charge.</p></div>
          <div class="login-feature"><h3>Open-Source Excellence</h3><p>Categorize errors into issues, assign to team members, and keep your apps running smoothly.</p></div>
          <div class="login-feature"><h3>Single Binary Deployment</h3><p>Enjoy essential monitoring features without the overhead.</p></div>
          <div class="login-callout">{#if result.IsDemo}<p>This demo is graciously sponsored by VPSDime ❤️</p><a href="https://vpsdime.com/">Visit VPSDime</a>{:else}<p>Warnly is a MIT-licensed project in alpha stage. If you find it useful, consider contributing to the project.</p><a href="https://github.com/vk-rv/warnly">Contribute to the project</a>{/if}</div>
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
          ><option value="">All Projects</option>{#each result.Projects || [] as p}<option
              value={p.Name}>{p.Name}</option
            >{/each}</select
        >{/if}
      <TimeRange {period} />
    </div>
    {#if path === '/'}<SearchFilter tags={result.PopularTags || []} />{:else}
      <section class="chart-panel">
        <header>Number of Errors</header>
        <EventChart
          events={result.Project?.Events || []}
          period={result.Period || period}
          start={page.url.searchParams.get('start') || ''}
          end={page.url.searchParams.get('end') || ''}
        />
        <footer>
          <strong>Total Errors:</strong>
          {number((result.Project?.Events || []).reduce((sum, e) => sum + e.Count, 0))}
        </footer>
      </section>
      <nav class="tabs project-tabs">
        {#each [['all', 'All Issues', result.Project.AllLength], ['new', 'New Issues', result.Project.NewLength]] as [value, label, count]}<button
            class:active={(result.Issues || 'all') === value}
            onclick={() => changeFilter('issues', value)}>{label} <span>{count || ''}</span></button
          >{/each}
      </nav>
    {/if}
    <IssueTable
      issues={result.Project ? result.Project.ResultIssueList || [] : result.Issues || []}
      projects={result.Projects || [result.Project]}
      project={result.Project}
      assignments={result.Assignments}
      teammates={result.Teammates}
      {busy}
      assign={(id, userID) =>
        mutate(
          `/projects/${result.Project.ID}/issues/${id}/assignments`,
          userID ? 'POST' : 'DELETE',
          userID ? new URLSearchParams({ user_id: userID }) : null
        )}
    />
    <Pagination
      total={result.Project
        ? result.Issues === 'new'
          ? result.Project.NewLength
          : result.Project.AllLength
        : result.TotalIssues}
      size={result.Project ? 5 : 50}
      pageMode={!!result.Project}
    />
  {:else if path === '/projects'}
    <form class="project-filters" onsubmit={query}>
      <select
        name="team"
        aria-label="Team"
        value={page.url.searchParams.get('team') || ''}
        onchange={(e) => changeFilter('team', e.currentTarget.value)}
        ><option value="">All Teams</option>{#each result.Teams || [] as team}<option
            value={team.ID}>{team.Name}</option
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
      {#each result.Projects || [] as project}<article class="project-card">
          <header>
            <span class="project-avatar">{project.Name?.[0]}</span><a
              href={`/projects/${project.ID}`}
              ><h2>{project.Name}</h2>
              <p>
                Errors: {number((project.Events || []).reduce((sum, e) => sum + e.Count, 0))} | Last 24
                hours
              </p></a
            ><a
              href={`/settings/projects/${project.ID}`}
              class="icon-button"
              aria-label={`Settings for ${project.Name}`}><Icon name="settings" /></a
            >
          </header>
          <EventChart
            events={project.Events || []}
            period="24h"
            label={`Errors for ${project.Name}`}
          />
          <footer></footer>
        </article>{:else}<p class="muted">You need at least one project to use this page.</p>{/each}
    </div>
  {:else if path === '/projects/new'}
    {#if created}<section class="panel">
        <h2>Project created: {created.Name}</h2>
        <p>Use this DSN with your Sentry SDK:</p>
        <pre>{created.DSN}</pre>
        <a href={`/projects/${created.ID}`}>Open project</a>
      </section>
    {:else}<form class="form panel" onsubmit={submit}>
        <label>Project name<input name="projectName" required /></label><label
          >Platform<select name="platform"
            ><option value="golang">Go</option><option value="rust">Rust</option></select
          ></label
        ><label
          >Team<select name="team" required
            >{#each result || [] as team}<option value={team.ID}>{team.Name}</option>{/each}</select
          ></label
        ><button class="primary" disabled={busy || !result.length}>Create project</button>
      </form>{/if}
  {:else if (parts[0] === 'settings' && parts[1] === 'projects') || parts[2] === 'getting-started'}
    <section class="panel">
      <h2>{result.Name}</h2>
      <p>Configure the Sentry SDK with this DSN:</p>
      <pre>{dsn(result)}</pre>
      <pre>{result.Platform === 2
          ? `let _guard = sentry::init("${dsn(result)}");\nsentry::capture_message("It works!", sentry::Level::Info);`
          : `sentry.Init(sentry.ClientOptions{Dsn: "${dsn(result)}"})\ndefer sentry.Flush(2 * time.Second)\nsentry.CaptureMessage("It works!")`}</pre>
      <a class="button" href={`/projects/${result.ID}`}>Open project</a>
      {#if parts[0] === 'settings'}<button
          class="danger"
          disabled={busy}
          onclick={() => remove(`/projects/${result.ID}`, '/projects')}>Delete project</button
        >{/if}
    </section>
  {:else if isIssue}
    <p class="issue-error-value">{issue?.ErrorValue || issue?.Message || ''}</p>
    <nav class="tabs issue-tabs">
      <a class:active={parts.length === 4} href={issuePath}>Info</a><a
        class:active={parts[4] === 'discussions'}
        href={`${issuePath}/discussions`}>Discussion <span>{issue?.MessagesCount || ''}</span></a
      ><a class:active={parts[4] === 'fields'} href={`${issuePath}/fields`}>Fields</a><a
        class:active={parts[4] === 'events'}
        href={`${issuePath}/events`}>All Errors</a
      >
    </nav>
    {#if parts.length === 4}<IssueOverview {issue} {issuePath} {mutate} {busy} />
    {:else if parts[4] === 'discussions'}
      {#each result.Messages || [] as message}<article class="panel">
          <strong>{message.username}</strong>
          <small>{new Date(message.created_at).toLocaleString()}</small>
          <p>{message.content}</p>
          {#if message.user_id === data.user?.ID}<button
              disabled={busy}
              onclick={() => remove(`${issuePath}/discussions/${message.id}`)}>Delete</button
            >{/if}
        </article>{:else}<p>No comments yet.</p>{/each}
      <form class="form" onsubmit={submit}>
        <label>Comment<textarea name="content" required></textarea></label><label
          >Mention teammates<select name="mentioned_users" multiple
            >{#each result.Teammates || [] as user}<option value={user.ID}
                >{user.Username || user.Email}</option
              >{/each}</select
          ></label
        ><button disabled={busy}>Post comment</button>
      </form>
    {:else if parts[4] === 'fields'}<SystemTable rows={result.FieldValueNum || []} />
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
            >{#each result.Events || [] as event}<tr
                ><td
                  ><a href={`${issuePath}?event_id=${event.EventID}`}
                    >{event.Title || event.EventID}</a
                  >
                  <p>{event.Message}</p></td
                ><td>{new Date(event.CreatedAt).toLocaleString()}</td><td>{event.Env}</td><td
                  >{event.UserEmail || event.UserUsername || event.User}</td
                ></tr
              >{:else}<tr><td colspan="4">No events found.</td></tr>{/each}</tbody
          >
        </table>
      </div>
      <Pagination total={result.TotalEvents} />
    {/if}
  {:else if path === '/alerts'}
    <div class="filters-row alert-filters">
      <select
        aria-label="Team"
        class="filter-select"
        value={page.url.searchParams.get('team_name') || ''}
        onchange={(e) => changeFilter('team_name', e.currentTarget.value)}
        ><option value="">My Teams</option>{#each result.Teams || [] as team}<option
            value={team.Name}>{team.Name}</option
          >{/each}</select
      ><select
        aria-label="Project"
        class="filter-select"
        value={page.url.searchParams.get('project_name') || ''}
        onchange={(e) => changeFilter('project_name', e.currentTarget.value)}
        ><option value="">All Projects</option>{#each result.Projects || [] as p}<option
            value={p.Name}>{p.Name}</option
          >{/each}</select
      ><a class="button" href="/alerts/new"><Icon name="plus" />Create Alert</a>
    </div>
    <div class="table-wrap">
      <table class="alerts-table">
        <thead
          ><tr><th>Alert rule</th><th>Status</th><th>Project</th><th>Team</th><th>Actions</th></tr
          ></thead
        ><tbody
          >{#each result.Alerts || [] as alert}<tr
              ><td
                ><a href={`/alerts/${alert.ID}/edit`}
                  ><strong>{alert.RuleName}</strong>
                  <p>{alert.Description}</p></a
                ></td
              ><td
                ><span
                  class="badge"
                  class:success-badge={alert.Status === 'Active'}
                  class:danger-badge={alert.Status === 'Triggered'}>{alert.Status}</span
                ></td
              ><td>{result.Projects?.find((p) => p.ID === alert.ProjectID)?.Name}</td><td
                >{result.Teams?.find((t) => t.ID === alert.TeamID)?.Name}</td
              ><td
                ><button
                  class="danger small"
                  disabled={busy}
                  onclick={() => remove(`/alerts/${alert.ID}`)}>Delete</button
                ></td
              ></tr
            >{:else}<tr><td colspan="5" class="empty-state">No alert rules.</td></tr>{/each}</tbody
        >
      </table>
    </div>
    <Pagination total={result.TotalAlerts} />
  {:else if parts[0] === 'alerts'}<AlertForm {result} {submit} {busy} />
  {:else if path === '/settings'}
    <h2>Webhook notifications</h2>
    <form class="form panel" onsubmit={submit}>
      <input type="hidden" name="team_id" value={result.Webhook?.TeamID || 1} /><label
        >URL<input
          name="url"
          type="url"
          value={result.Webhook?.URL || ''}
          placeholder="https://example.com/webhook"
        /></label
      ><label
        >Secret<input
          name="secret"
          type="password"
          value={result.Webhook?.Secret || ''}
          autocomplete="off"
        /></label
      ><button class="primary" disabled={busy}>Save webhook</button>
    </form>
  {:else if parts[0] === 'system'}<nav class="tabs">
      <a href="/system">Slow queries</a><a href="/system/schema">Storage schema</a><a
        href="/system/errors">Store errors</a
      >
    </nav>
    <SystemTable rows={result} />
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
