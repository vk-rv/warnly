<script>
  let { result, submit, busy } = $props();
  let alert = $derived(result.alert || {});
</script>

<form class="form panel" onsubmit={submit}>
  <label>Rule name<input name="rule_name" value={alert.rule_name || ''} required /></label>
  {#if !alert.id}<label
      >Project<select name="project_id" required
        >{#each result.projects || [] as p}<option value={p.id}>{p.name}</option>{/each}</select
      ></label
    >{/if}
  <label
    >Trigger when<select name="condition" value={alert.condition || 1}
      ><option value="1">Number of occurrences</option><option value="2"
        >Number of affected users</option
      ></select
    ></label
  >
  <label
    >Threshold<input
      name="threshold"
      type="number"
      min="1"
      value={alert.threshold || 1}
      required
    /></label
  >
  <label
    >Time window<select name="timeframe" value={alert.timeframe || 4}
      >{#each ['1 minute', '5 minutes', '15 minutes', '1 hour', '1 day', '1 week', '30 days'] as name, i}<option
          value={i + 1}>{name}</option
        >{/each}</select
    ></label
  >
  <label class="check"
    ><input
      type="checkbox"
      name="high_priority"
      value="true"
      checked={alert.high_priority || false}
    /> High priority</label
  >
  {#if alert.id}<label
      >Status<select name="status" value={alert.status}
        ><option>Active</option><option>Inactive</option><option>Triggered</option></select
      ></label
    >{/if}
  <div class="alert-form-actions">
    <button class="primary" disabled={busy}>Save rule</button>
    <a class="button" href="/alerts">Cancel</a>
  </div>
</form>
