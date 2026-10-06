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
    return issue.ProjectID || project?.ID || projects[0]?.ID;
  }
  function projectName(issue) {
    return (
      projects.find((p) => p?.ID === projectID(issue))?.Name || project?.Name || projectID(issue)
    );
  }
  function link(issue) {
    return `/projects/${projectID(issue)}/issues/${issue.ID}?source=issue`;
  }
</script>

<div class="table-wrap issue-table" class:project-issue-table={!!project}>
  <table>
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
              ><strong>{issue.Type || '(No error type)'}</strong>{#if project && issue.View}<span
                  class="issue-view">{issue.View}</span
                >{/if}
              <p>{issue.Message}</p></a
            >{#if project}<div class="issue-history">
                <span title={issue.FirstSeen}>First: {timeAgo(issue.FirstSeen)}</span><span
                  title={issue.LastSeen}>Last: {timeAgo(issue.LastSeen)}</span
                >{#if issue.MessagesCount}<span>{issue.MessagesCount} comments</span>{/if}
              </div>{/if}</td
          >
          {#if !project}<td>{projectName(issue)}</td>{/if}<td class="metric-number"
            >{number(issue.TimesSeen)}</td
          ><td>{number(issue.UserCount)}</td>
          {#if project}<td
              ><select
                class="assignment-select"
                aria-label={`Responsible for ${issue.Type}`}
                value={assignments?.IssueToAssigned?.[issue.ID]?.ID || ''}
                disabled={busy}
                onchange={(e) => assign?.(issue.ID, e.currentTarget.value)}
                ><option value="">Unassigned</option>{#each teammates || [] as user}<option
                    value={user.ID}>{user.Username || user.Email}</option
                  >{/each}</select
              ></td
            >
          {:else}<td class="muted" title={issue.FirstSeen}>{timeAgo(issue.FirstSeen)}</td><td
              class="muted"
              title={issue.LastSeen}>{timeAgo(issue.LastSeen)}</td
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
        ><strong>{issue.Type || '(No error type)'}</strong>
        <p>{issue.Message}</p></a
      ><span class="badge info">{projectName(issue)}</span>
      <dl>
        <div>
          <dt>Events</dt>
          <dd>{number(issue.TimesSeen)}</dd>
        </div>
        <div>
          <dt>Users</dt>
          <dd>{number(issue.UserCount)}</dd>
        </div>
        <div>
          <dt>First seen</dt>
          <dd>{timeAgo(issue.FirstSeen)}</dd>
        </div>
        <div>
          <dt>Last seen</dt>
          <dd>{timeAgo(issue.LastSeen)}</dd>
        </div>
      </dl>
    </article>{:else}<p class="empty-state">No issues for these filters.</p>{/each}
</div>
