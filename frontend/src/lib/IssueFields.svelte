<script>
  let { result } = $props();
  function search(tag, value) {
    return `/?${new URLSearchParams({ project_name: result.project_name, issues: 'all', period: '14d', query: `${tag}:${JSON.stringify(value)}` })}`;
  }
</script>

<div class="issue-fields">
  {#each result.tag_count || [] as tag}
    <section class="issue-field-card">
      <h2>{tag.tag}</h2>
      <div class="issue-field-values">
        {#each (result.field_value_num || []).filter((value) => value.tag === tag.tag) as value}
          <a href={search(tag.tag, value.value)}
            ><span>{value.value}</span><span class="field-count">{value.count}</span></a
          >
        {/each}
      </div>
    </section>
  {/each}
</div>
