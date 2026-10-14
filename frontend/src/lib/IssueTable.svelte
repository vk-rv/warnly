<script>
  import Icon from './Icon.svelte';
  import { number, timeAgo } from './format';
  let {
    issues = [],
    projects = [],
    project = null,
    assignments = null,
    teammates = [],
    assign,
    busy = false
  } = $props();
  function projectID(issue) {
    return issue.project_id || project?.id || projects[0]?.id;
  }
  function projectName(issue) {
    return (
      projects.find((p) => p?.id === projectID(issue))?.name || project?.name || projectID(issue)
    );
  }
  function link(issue) {
    return `/projects/${projectID(issue)}/issues/${issue.id}?source=issue`;
  }
</script>

<div class="table-wrap issue-table" class:project-issue-table={!!project}>
  <table>
    {#if !project}
      <colgroup>
        <col class="issues-col-description" />
        <col class="issues-col-project" />
        <col class="issues-col-count" />
        <col class="issues-col-count" />
        <col class="issues-col-time" />
        <col class="issues-col-time" />
      </colgroup>
    {/if}
    <thead
      ><tr
        ><th>Issue</th>{#if !project}<th>Project</th>{/if}<th>{project ? 'Errors' : 'Events'}</th
        ><th>Users</th>{#if project}<th>Responsible</th>{:else}<th>First seen</th><th>Last seen</th
          >{/if}</tr
      ></thead
    >
    <tbody
      >{#each issues || [] as issue}<tr>
          <td class="issue-description"
            ><a href={link(issue)}
              ><strong>{issue.type || '(No error type)'}</strong>{#if project && issue.view}<span
                  class="issue-view">{issue.view}</span
                >{/if}
              <p>{issue.message}</p></a
            >{#if project}<div class="issue-history">
                <span title={issue.first_seen}>First: {timeAgo(issue.first_seen)}</span><span
                  title={issue.last_seen}>Last: {timeAgo(issue.last_seen)}</span
                >{#if issue.messages_count}<span>{issue.messages_count} comments</span>{/if}
              </div>{/if}</td
          >
          {#if !project}<td>{projectName(issue)}</td>{/if}<td class="metric-number"
            >{number(issue.times_seen)}</td
          ><td>{number(issue.user_count)}</td>
          {#if project}<td class="issue-responsible"
              ><select
                class="assignment-select"
                aria-label={`Responsible for ${issue.type}`}
                value={assignments?.issue_to_assigned?.[issue.id]?.id || ''}
                disabled={busy}
                onchange={(e) => assign?.(issue.id, e.currentTarget.value)}
                ><option value="">Unassigned</option>{#each teammates || [] as user}<option
                    value={user.id}>{user.username || user.email}</option
                  >{/each}</select
              ></td
            >
          {:else}<td class="muted" title={issue.first_seen}>{timeAgo(issue.first_seen)}</td><td
              class="muted"
              title={issue.last_seen}>{timeAgo(issue.last_seen)}</td
            >{/if}
        </tr>{:else}<tr
          ><td colspan={project ? 4 : 6} class="empty-state"
            ><Icon name="check" size={24} />
            <p>No issues for these filters.</p></td
          ></tr
        >{/each}</tbody
    >
  </table>
</div>
<div class="mobile-issues">
  {#each issues || [] as issue}<article class="mobile-issue">
      <a href={link(issue)}
        ><strong>{issue.type || '(No error type)'}</strong>
        <p>{issue.message}</p></a
      ><span class="badge info">{projectName(issue)}</span>
      <dl>
        <div>
          <dt>Events</dt>
          <dd>{number(issue.times_seen)}</dd>
        </div>
        <div>
          <dt>Users</dt>
          <dd>{number(issue.user_count)}</dd>
        </div>
        <div>
          <dt>First seen</dt>
          <dd>{timeAgo(issue.first_seen)}</dd>
        </div>
        <div>
          <dt>Last seen</dt>
          <dd>{timeAgo(issue.last_seen)}</dd>
        </div>
      </dl>
    </article>{:else}<p class="empty-state">No issues for these filters.</p>{/each}
</div>
