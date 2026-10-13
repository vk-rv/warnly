<script>
  let { result } = $props();
  function search(tag, value) {
    return `/?${new URLSearchParams({ project_name: result.ProjectName, issues: 'all', period: '14d', query: `${tag}:${JSON.stringify(value)}` })}`;
  }
</script>

<div class="issue-fields">
  {#each result.TagCount || [] as tag}
    <section class="issue-field-card">
      <h2>{tag.Tag}</h2>
      <div class="issue-field-values">
        {#each (result.FieldValueNum || []).filter((value) => value.Tag === tag.Tag) as value}
          <a href={search(tag.Tag, value.Value)}
            ><span>{value.Value}</span><span class="field-count">{value.Count}</span></a
          >
        {/each}
      </div>
    </section>
  {/each}
</div>
