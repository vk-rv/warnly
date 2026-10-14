<script>
  import { page } from '$app/state';
  import Icon from './Icon.svelte';
  let { total = 0, size = 50, pageMode = false } = $props();
  let key = $derived(pageMode ? 'page' : 'offset');
  let current = $derived(Number(page.url.searchParams.get(key) || (pageMode ? 1 : 0)));
  let offset = $derived(pageMode ? (current - 1) * size : current);
  function link(delta) {
    const q = new URLSearchParams(page.url.searchParams);
    q.set(key, String(current + delta * (pageMode ? 1 : size)));
    return `${page.url.pathname}?${q}`;
  }
</script>

<div class="pagination">
  <span
    >Showing {total ? offset + 1 : 0} to {Math.min(offset + size, total)} of {total} results</span
  >
  <div>
    {#if offset > 0}<a class="button" href={link(-1)}><Icon name="left" />Previous</a>{:else}<button
        disabled><Icon name="left" />Previous</button
      >{/if}
    {#if offset + size < total}<a class="button" href={link(1)}>Next<Icon name="right" /></a
      >{:else}<button disabled>Next<Icon name="right" /></button>{/if}
  </div>
</div>
