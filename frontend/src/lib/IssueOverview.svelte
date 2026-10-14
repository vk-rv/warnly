<script>
  import Icon from './Icon.svelte';
  import { number, timeAgo } from './format';
  let { issue, issuePath, mutate, busy } = $props();
  let expanded = $state(false);
  let copied = $state(false);
  let event = $derived(issue.last_event);
  let fields = $derived(
    (issue.tag_count || []).map(({ tag }) => {
      const index = event?.tags_key?.indexOf(tag) ?? -1;
      const current = index < 0 ? '' : event.tags_value?.[index] || '';
      const values = (issue.tag_value_num || []).filter((value) => value.tag === tag);
      const percent = values.find((value) => value.value === current)?.percents_of_total;
      const width =
        percent == null
          ? 0
          : percent >= 100
            ? 100
            : percent >= 75
              ? 75
              : percent >= 50
                ? 50
                : percent >= 25
                  ? 25
                  : 20;
      return { tag, current, values, width };
    })
  );
  let contexts = $derived.by(() => {
    const groups = {};
    for (const [i, key] of (event?.contexts_key || []).entries()) {
      const dot = key.indexOf('.');
      const group = dot < 0 ? 'Other' : key.slice(0, dot);
      const field = dot < 0 ? key : key.slice(dot + 1);
      (groups[group] ||= []).push([field, event.contexts_value?.[i] || '']);
    }
    return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b));
  });
  async function copyID() {
    try {
      await navigator.clipboard.writeText(event?.event_id || '');
      copied = true;
    } catch {
      copied = false;
    }
  }
</script>

<div class="issue-workspace">
  <section class="issue-event">
    <div class="event-toolbar">
      <button class="event-id" onclick={copyID} title={event?.event_id}
        ><strong>id:</strong>
        {event?.event_id?.slice(0, 6) || '—'}<Icon name={copied ? 'check' : 'copy'} /></button
      >
      <div class="event-navigation">
        {#each [[issue.last_event_id, 'Oldest', 'left'], [issue.next_event_id, 'Older', 'left'], [issue.prev_event_id, 'Newer', 'right'], [issue.first_event_id, 'Newest', 'right']] as [id, label, icon]}{#if id}<a
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
        {#each [['Identifier', event.user_id], ['Name', event.user_name], ['Username', event.user_username], ['Email', event.user_email]] as [label, value]}{#if value}<div
            >
              <span>{label}</span><code>{value}</code>
            </div>{/if}{/each}{#if !event.user_id && !event.user_email && !event.user_name && !event.user_username}<span
            class="muted">No user information for this event.</span
          >{/if}
      </div>
      <h2>Tags</h2>
      <div class="tag-grid">
        {#each event.tags_key || [] as key, i}<a
            class="tag-card"
            href={`/?${new URLSearchParams({ project_name: issue.project_name, query: key + ':' + JSON.stringify(event.tags_value?.[i] || '') })}`}
            ><span>{key}</span><code>{event.tags_value?.[i]}</code></a
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
        <span class="platform-badge">{issue.platform === 2 ? 'Rust' : 'Go'}</span><span
          >Noticed in: <code>{issue.stack_details?.[0]?.filepath || issue.view || '—'}</code></span
        >
      </div>
      {#each (expanded ? issue.stack_details : issue.stack_details?.slice(0, 5)) || [] as frame}<div
          class="stack-frame"
        >
          <div>
            <code>{frame.filepath}</code> <span>in</span>
            {frame.function_name} <span>at line</span>
            {frame.line_no}
          </div>
          {#if frame.in_app}<span class="badge info">In App</span>{/if}
        </div>{:else}<p class="empty-state">
          No stack trace available.
        </p>{/each}{#if issue.stack_details?.length > 5}<button
          class="show-more"
          onclick={() => (expanded = !expanded)}>{expanded ? 'Show less' : 'Show more'}</button
        >{/if}
    </div>
  </section>
  <section class="issue-summary" aria-label="Issue summary">
    <label
      >Assigned to<select
        disabled={busy}
        value={issue.assignments?.issue_to_assigned?.[issue.issue_id]?.id || ''}
        onchange={(e) =>
          mutate(
            `${issuePath}/assignments`,
            e.currentTarget.value ? 'POST' : 'DELETE',
            e.currentTarget.value ? new URLSearchParams({ user_id: e.currentTarget.value }) : null
          )}
        ><option value="">Unassigned</option>{#each issue.teammates || [] as user}<option
            value={user.id}>{user.username || user.email}</option
          >{/each}</select
      ></label
    >
    <div class="summary-metrics">
      <div>
        <h3>Last 24 Hours</h3>
        <strong>{number(issue.total24_hours)}</strong>
      </div>
      <div>
        <h3>Last 30 Days</h3>
        <strong>{number(issue.total30_days)}</strong>
      </div>
      <div>
        <h3>First Seen</h3>
        <span title={issue.first_seen}>{timeAgo(issue.first_seen)} ago</span>
      </div>
      <div>
        <h3>Last Seen</h3>
        <span title={issue.last_seen}>{timeAgo(issue.last_seen)} ago</span>
      </div>
    </div>
    <section class="summary-fields" aria-label="Field distributions">
      <h2>Fields</h2>
      {#each fields as field}
        <details class="summary-field">
          <summary
            ><span class="field-name">{field.tag}</span><span
              class="field-current"
              title={field.current}>{field.current}</span
            ><Icon name="chevron" /></summary
          >
          <div class="field-distribution-bar" aria-hidden="true">
            <span style:width={`${field.width}%`}></span>
          </div>
          {#each field.values as value}
            <div class="field-distribution-value">
              <span class="field-dot"></span><span class="field-value" title={value.value}
                >{value.value}</span
              ><span class="field-percent">{Math.floor(value.percents_of_total)}%</span>
            </div>
          {/each}
        </details>
      {/each}
    </section>
  </section>
</div>
