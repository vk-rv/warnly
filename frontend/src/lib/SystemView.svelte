<script>
  import Icon from './Icon.svelte';
  let { rows = [], path, notify } = $props();
  let sort = $state('avg_duration');
  let descending = $state(true);
  const queryColumns = [
    ['normalized_query', 'Query'],
    ['avg_duration', 'Avg duration (ms)'],
    ['avg_result_rows', 'Avg result rows'],
    ['calls_per_minute', 'Calls/minute'],
    ['total_calls', 'Total calls'],
    ['total_read_bytes_numeric', 'Total read bytes'],
    ['percentage_iops', '% IOPS'],
    ['percentage_runtime', '% Runtime']
  ];
  let columns = $derived(
    path === '/system/schema'
      ? [
          ['name', 'Name'],
          ['readable_bytes', 'Size'],
          ['total_rows', 'Rows'],
          ['engine', 'Engine'],
          ['partition_key', 'Partition key']
        ]
      : path === '/system/errors'
        ? [
            ['Name', 'Errors'],
            ['MaxLastErrorTime', 'Most recent (UTC)'],
            ['Count', 'Count']
          ]
        : queryColumns
  );
  let sorted = $derived(
    path === '/system'
      ? [...(rows || [])].sort((a, b) => ((a[sort] || 0) - (b[sort] || 0)) * (descending ? -1 : 1))
      : rows || []
  );
  function sortBy(key) {
    if (sort === key) descending = !descending;
    else {
      sort = key;
      descending = true;
    }
  }
  function display(row, key) {
    if (key === 'total_read_bytes_numeric') return row.total_read_bytes;
    if (key === 'MaxLastErrorTime')
      return new Date(row[key]).toLocaleString('en-US', {
        timeZone: 'UTC',
        month: 'long',
        day: 'numeric',
        year: 'numeric',
        hour: 'numeric',
        minute: '2-digit'
      });
    if (typeof row[key] === 'number')
      return row[key].toLocaleString('en-US', { maximumFractionDigits: 2 });
    return row[key] ?? '—';
  }
  async function copy(sql) {
    try {
      await navigator.clipboard.writeText(sql);
      notify('Copied to clipboard');
    } catch {
      notify('Could not copy to clipboard');
    }
  }
</script>

<section class="system-view">
  <nav class="tabs" aria-label="System">
    <a class:active={path === '/system'} href="/system">Queries</a><a
      class:active={path === '/system/schema'}
      href="/system/schema">Schema</a
    ><a class:active={path === '/system/errors'} href="/system/errors">Errors</a>
  </nav>
  <div class="table-wrap system-table">
    <table>
      <thead
        ><tr
          >{#each columns as [key, label]}<th
              aria-sort={path === '/system' && key === sort
                ? descending
                  ? 'descending'
                  : 'ascending'
                : undefined}
              >{#if path === '/system' && key !== 'normalized_query'}<button
                  onclick={() => sortBy(key)}
                  >{label} {sort === key ? (descending ? '↓' : '↑') : '↕'}</button
                >{:else}{label}{/if}</th
            >{/each}</tr
        ></thead
      >
      <tbody
        >{#each sorted as row}<tr
            >{#each columns as [key]}<td
                >{#if key === 'normalized_query'}<details class="query-detail">
                    <summary
                      ><span class="query-chevron"><Icon name="right" size={14} /></span><code
                        >{row[key]}</code
                      ></summary
                    >
                    <div class="query-sql">
                      <pre>{row[key]}</pre>
                      <button
                        class="icon-button query-copy"
                        aria-label="Copy SQL"
                        title="Copy SQL"
                        onclick={() => copy(row[key])}><Icon name="copy" /></button
                      >
                    </div>
                  </details>{:else}{display(row, key)}{#if key === 'total_read_bytes_numeric'}<div
                      class="read-progress"
                    >
                      <span style:width={`${Math.min(100, Math.max(0, row.percent || 0))}%`}></span>
                    </div>{/if}{/if}</td
              >{/each}</tr
          >{:else}<tr><td colspan={columns.length} class="empty-state">No data available.</td></tr
          >{/each}</tbody
      >
    </table>
  </div>
</section>
