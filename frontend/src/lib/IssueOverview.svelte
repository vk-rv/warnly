<script>
  import Icon from './Icon.svelte';
  import { number, timeAgo } from './format';
  let { issue, issuePath, mutate, busy } = $props();
  let expanded = $state(false);
  let copied = $state(false);
  let event = $derived(issue.LastEvent);
  let contexts = $derived.by(() => {
    const groups = {};
    for (const [i, key] of (event?.ContextsKey || []).entries()) {
      const dot = key.indexOf('.');
      const group = dot < 0 ? 'Other' : key.slice(0, dot);
      const field = dot < 0 ? key : key.slice(dot + 1);
      (groups[group] ||= []).push([field, event.ContextsValue?.[i] || '']);
    }
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b));
  });
  async function copyID() {
    try {
      await navigator.clipboard.writeText(event?.EventID || '');
      copied = true;
    } catch {
      copied = false;
    }
  }
</script>

<div class="issue-workspace">
  <section class="issue-event">
    <div class="event-toolbar">
      <button class="event-id" onclick={copyID} title={event?.EventID}
        ><strong>ID:</strong>
        {event?.EventID?.slice(0, 6) || '—'}<Icon name={copied ? 'check' : 'copy'} /></button
      >
      <div class="event-navigation">
        {#each [[issue.LastEventID, 'Oldest', 'left'], [issue.NextEventID, 'Older', 'left'], [issue.PrevEventID, 'Newer', 'right'], [issue.FirstEventID, 'Newest', 'right']] as [id, label, icon]}{#if id}<a
              class="button"
              href={`${issuePath}?event_id=${id}`}
              aria-label={label}
              title={label}
              ><Icon name={icon} />{#if label === 'Oldest' || label === 'Newest'}<Icon
                  name={icon}
                />{/if}</a
            >{:else}<button disabled aria-label={label}><Icon name={icon} /></button>{/if}{/each}
      </div>
    </div>
    {#if event}
      <h2>User Info</h2>
      <div class="user-info">
        {#each [['Identifier', event.UserID], ['Name', event.UserName], ['Username', event.UserUsername], ['Email', event.UserEmail]] as [label, value]}{#if value}<div
            >
              <span>{label}</span><code>{value}</code>
            </div>{/if}{/each}{#if !event.UserID && !event.UserEmail && !event.UserName && !event.UserUsername}<span
            class="muted">No user information for this event.</span
          >{/if}
      </div>
      <h2>Tags</h2>
      <div class="tag-grid">
        {#each event.TagsKey || [] as key, i}<a
            class="tag-card"
            href={`/?${new URLSearchParams({ project_name: issue.ProjectName, query: key + ':' + JSON.stringify(event.TagsValue?.[i] || '') })}`}
            ><span>{key}</span><code>{event.TagsValue?.[i]}</code></a
          >{:else}<p class="muted">No tags.</p>{/each}
      </div>
      <h2>Contexts</h2>
      <div class="context-grid">
        {#each contexts as [group, fields]}<section class="context-card">
            <h3>{group}<Icon name={group === 'user' ? 'user' : 'system'} /></h3>
            <dl>
              {#each fields as [key, value]}<dt>{key}</dt>
                <dd>{value}</dd>{/each}
            </dl>
          </section>{:else}<p class="muted">No context available.</p>{/each}
      </div>
    {/if}
    <h2>Stack trace</h2>
    <div class="stack-panel">
      <div class="stack-header">
        <span class="platform-badge">{issue.Platform === 2 ? 'Rust' : 'Go'}</span><span
          >Noticed in: <code>{issue.StackDetails?.[0]?.Filepath || issue.View || '—'}</code></span
        >
      </div>
      {#each (expanded ? issue.StackDetails : issue.StackDetails?.slice(0, 5)) || [] as frame}<div
          class="stack-frame"
        >
          <div>
            <code>{frame.Filepath}</code> <span>in</span>
            {frame.FunctionName} <span>at line</span>
            {frame.LineNo}
          </div>
          {#if frame.InApp}<span class="badge info">In App</span>{/if}
        </div>{:else}<p class="empty-state">
          No stack trace available.
        </p>{/each}{#if issue.StackDetails?.length > 5}<button
          class="show-more"
          onclick={() => (expanded = !expanded)}>{expanded ? 'Show less' : 'Show more'}</button
        >{/if}
    </div>
  </section>
  <section class="issue-summary" aria-label="Issue summary">
    <label
      >Assigned to<select
        disabled={busy}
        value={issue.Assignments?.IssueToAssigned?.[issue.IssueID]?.ID || ''}
        onchange={(e) =>
          mutate(
            `${issuePath}/assignments`,
            e.currentTarget.value ? 'POST' : 'DELETE',
            e.currentTarget.value ? new URLSearchParams({ user_id: e.currentTarget.value }) : null
          )}
        ><option value="">Unassigned</option>{#each issue.Teammates || [] as user}<option
            value={user.ID}>{user.Username || user.Email}</option
          >{/each}</select
      ></label
    >
    <div class="summary-metrics">
      <div>
        <h3>Last 24 Hours</h3>
        <strong>{number(issue.Total24Hours)}</strong>
      </div>
      <div>
        <h3>Last 30 Days</h3>
        <strong>{number(issue.Total30Days)}</strong>
      </div>
      <div>
        <h3>First Seen</h3>
        <span title={issue.FirstSeen}>{timeAgo(issue.FirstSeen)} ago</span>
      </div>
      <div>
        <h3>Last Seen</h3>
        <span title={issue.LastSeen}>{timeAgo(issue.LastSeen)} ago</span>
      </div>
    </div>
    {#if event?.Release}<h3>Release</h3>
      <code>{event.Release}</code>{/if}{#if event?.Env}<h3>Environment</h3>
      <span class="badge">{event.Env}</span>{/if}
  </section>
</div>
