<script>
  let { rows = [] } = $props();
  let columns = $derived(Object.keys(rows?.[0] || {}));
</script>

<div class="table-wrap">
  <table>
    <thead
      ><tr
        >{#each columns as column}<th>{column.replaceAll('_', ' ')}</th>{/each}</tr
      ></thead
    >
    <tbody
      >{#each rows || [] as row}<tr
          >{#each columns as column}<td>{row[column]}</td>{/each}</tr
        >{:else}<tr><td>No data available.</td></tr>{/each}</tbody
    >
  </table>
</div>
