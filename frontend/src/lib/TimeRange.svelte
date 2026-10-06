<script>
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { periods } from './format';
  import Icon from './Icon.svelte';
  let { period = '14d' } = $props();
  let root;
  let open = $state(false);
  let custom = $state('');
  let error = $state('');
  let absolute = $state(false);
  let start = $state('');
  let end = $state('');
  let rangeLabel = $derived(
    page.url.searchParams.get('start') && page.url.searchParams.get('end')
      ? 'Custom date range'
      : periods.find((p) => p[0] === period)?.[1] || period
  );
  $effect(() => {
    start = page.url.searchParams.get('start') || '';
    end = page.url.searchParams.get('end') || '';
  });
  function apply(value) {
    const q = new URLSearchParams(page.url.searchParams);
    q.delete('offset');
    q.delete('page');
    q.delete('start');
    q.delete('end');
    if (value) q.set('period', value);
    else {
      q.delete('period');
      q.set('start', start.length === 16 ? start + ':00' : start);
      q.set('end', end.length === 16 ? end + ':00' : end);
    }
    open = false;
    error = '';
    goto(`${page.url.pathname}?${q}`);
  }
  function submit(event) {
    event.preventDefault();
    if (absolute) {
      if (!start || !end || new Date(start) >= new Date(end)) {
        error = 'Choose a start date before the end date.';
        return;
      }
      apply('');
    } else {
      if (!/^[1-9]\d*[smhdw]$/.test(custom)) {
        error = 'Use a range such as 2h, 4d or 8w.';
        return;
      }
      apply(custom);
    }
  }
</script>

<svelte:window
  onpointerdown={(e) => {
    if (!root?.contains(e.target)) open = false;
  }}
  onkeydown={(e) => {
    if (e.key === 'Escape') open = false;
  }}
/>
<div class="range-picker" bind:this={root}>
  <button
    class="filter-button"
    aria-label={`Period: ${rangeLabel}`}
    aria-expanded={open}
    onclick={() => (open = !open)}>{rangeLabel}<Icon name="chevron" /></button
  >
  {#if open}<div class="range-menu">
      <h3>Filter Time Range</h3>
      {#if !absolute}
        <form onsubmit={submit} class="custom-duration">
          <input
            aria-label="Custom duration"
            placeholder="Custom range: 2h, 4d, 8w..."
            bind:value={custom}
          /><button class="icon-button" aria-label="Apply custom duration"
            ><Icon name="right" /></button
          >
        </form>
        <div class="range-presets">
          {#each periods as [value, label]}<button
              class:chosen={period === value}
              onclick={() => apply(value)}
              ><span class="check-slot"
                >{#if period === value}<Icon name="check" />{/if}</span
              >{label}</button
            >{/each}<button onclick={() => (absolute = true)}
            ><span class="check-slot"><Icon name="clock" /></span>Absolute date<Icon
              name="right"
            /></button
          >
        </div>
      {:else}<form class="absolute-range" onsubmit={submit}>
          <label
            >From (UTC)<input type="datetime-local" step="1" bind:value={start} required /></label
          ><label>To (UTC)<input type="datetime-local" step="1" bind:value={end} required /></label>
          <div class="toolbar">
            <button type="button" onclick={() => (absolute = false)}>Back</button><button
              class="primary">Apply range</button
            >
          </div>
        </form>{/if}
      {#if error}<p class="error" role="alert">{error}</p>{/if}
    </div>{/if}
</div>
