<script>
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from './api';
  import Icon from './Icon.svelte';
  let { tags = [] } = $props();
  let input;
  let root;
  let draft = $state('');
  let tokens = $state([]);
  let open = $state(false);
  let values = $state([]);
  let selectedTag = $state('');
  let loading = $state(false);
  let requestID = 0;
  $effect(() => {
    const parts =
      (page.url.searchParams.get('query') || '').match(/(?:[^\s"']+|"[^"]*"|'[^']*')+/g) || [];
    tokens = parts.filter((p) => p.includes(':'));
    draft = parts.filter((p) => !p.includes(':')).join(' ');
  });
  function apply(next = tokens, text = draft) {
    const q = new URLSearchParams(page.url.searchParams);
    const query = [...next, text.trim()].filter(Boolean).join(' ');
    if (query) q.set('query', query);
    else q.delete('query');
    q.delete('offset');
    q.delete('page');
    open = false;
    selectedTag = '';
    goto(`${page.url.pathname}?${q}`);
  }
  async function chooseTag(tag) {
    const id = ++requestID;
    selectedTag = tag;
    loading = true;
    values = [];
    const q = new URLSearchParams(page.url.searchParams);
    q.set('tag', tag);
    try {
      const result = await api(`/search/tag-values?${q}`);
      if (id === requestID) values = result || [];
    } catch {
      if (id === requestID) values = [];
    } finally {
      if (id === requestID) loading = false;
    }
  }
  function chooseValue(value) {
    apply([...tokens, `${selectedTag}:${JSON.stringify(value)}`], '');
  }
  function toggle(index) {
    const next = [...tokens];
    const i = next[index].indexOf(':');
    const key = next[index].slice(0, i);
    const value = next[index].slice(i + 1);
    next[index] = `${key}:${value.startsWith('!') ? value.slice(1) : '!' + value}`;
    apply(next);
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
<div class="search-container" bind:this={root}>
  <form
    class="search-input-wrapper"
    onsubmit={(e) => {
      e.preventDefault();
      if (selectedTag && draft) {
        chooseValue(draft);
        draft = '';
      } else apply();
    }}
  >
    <Icon name="search" />
    {#each tokens as token, i}{@const separator = token.indexOf(':')}{@const value = token.slice(
        separator + 1
      )}<span class="tag-pill"
        ><span>{token.slice(0, separator)}</span><button
          class="tag-pill-operator"
          type="button"
          aria-label={`Toggle operator for ${token.slice(0, separator)}`}
          onclick={() => toggle(i)}>{value.startsWith('!') ? 'is not' : 'is'}</button
        ><span>{value.replace(/^!/, '').replace(/^"|"$/g, '')}</span><button
          type="button"
          class="token-remove"
          aria-label={`Remove ${token}`}
          onclick={() => apply(tokens.filter((_, index) => index !== i))}
          ><Icon name="close" size={12} /></button
        ></span
      >{/each}
    <input
      bind:this={input}
      aria-label="Search issues"
      placeholder={selectedTag ? `Value for ${selectedTag}…` : 'Search for issues, add filters…'}
      bind:value={draft}
      onfocus={() => (open = true)}
    />
    <button class="icon-button" aria-label="Search"><Icon name="right" /></button>
  </form>
  {#if open && tags.length}<div class="search-suggestions">
      {#if selectedTag}<button
          class="suggestion-heading"
          onclick={() => {
            selectedTag = '';
            requestID++;
          }}><Icon name="left" />{selectedTag}</button
        >{#if loading}<p>Loading values…</p>{:else}{#each values.filter((v) => !draft || v.value
                .toLowerCase()
                .includes(draft.toLowerCase())) as value}<button
              onclick={() => {
                chooseValue(value.value);
                draft = '';
              }}><span>{value.value}</span><small>{value.count}</small></button
            >{:else}<p>Type a value and press Enter.</p>{/each}{/if}
      {:else}<p class="suggestion-heading">Tags</p>
        {#each tags.filter((t) => !draft || t.Tag.toLowerCase().includes(draft.toLowerCase())) as tag}<button
            onclick={() => chooseTag(tag.Tag)}
            ><span>{tag.Tag}</span><Icon name="right" size={14} /></button
          >{/each}{/if}
    </div>{/if}
</div>
