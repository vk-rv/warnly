<script>
  import { timeAgo } from './format';
  let { result, user, issuePath, mutate, remove, busy, error } = $props();
  let comment = $state('');
  let preview = $state(false);
  let editor = $state();
  let mention = $derived(/@([\w.-]*)$/.exec(comment));
  let suggestions = $derived(
    mention
      ? (result.Teammates || []).filter((u) =>
          u.Username.toLowerCase().includes(mention[1].toLowerCase())
        )
      : []
  );
  function addMention(user) {
    comment = comment.slice(0, mention.index) + '@' + user.Username + ' ';
    editor?.focus();
  }
  async function post(event) {
    event.preventDefault();
    if (!comment.trim() || busy) return;
    const body = new URLSearchParams({ content: comment });
    for (const teammate of result.Teammates || []) {
      if (comment.split(/\s+/).includes('@' + teammate.Username))
        body.append('mentioned_users', teammate.ID);
    }
    await mutate(`${issuePath}/discussions`, 'POST', body);
    if (!error) {
      comment = '';
      preview = false;
    }
  }
</script>

{#snippet content(text)}
  {#each (text || '').split(/(@\w+)/g) as part}{#if /^@\w+$/.test(part)}<strong>{part}</strong
      >{:else}{part}{/if}{/each}
{/snippet}
<section class="discussion">
  <form class="discussion-composer" onsubmit={post}>
    <div class="composer-tabs" role="tablist" aria-label="Comment editor">
      <button type="button" role="tab" aria-selected={!preview} onclick={() => (preview = false)}
        >Write</button
      >
      <button type="button" role="tab" aria-selected={preview} onclick={() => (preview = true)}
        >Preview</button
      >
    </div>
    <div class="composer-body">
      {#if preview}<div class="comment-content" role="tabpanel">
          {#if comment}{@render content(comment)}{:else}Nothing to preview{/if}
        </div>
      {:else}<textarea
          bind:this={editor}
          bind:value={comment}
          aria-label="Comment"
          placeholder="Write your comment... You can tag users with @"
          rows="10"
          required></textarea>
        {#if suggestions.length}<div class="mention-menu" aria-label="Mention teammates">
            {#each suggestions as teammate}<button
                type="button"
                onclick={() => addMention(teammate)}>{teammate.Username}</button
              >{/each}
          </div>{/if}
      {/if}
    </div>
    <footer>
      <span
        >{#if result.Info?.IssueFirstSeen}First Noticed {timeAgo(result.Info.IssueFirstSeen)} ago{/if}</span
      ><button aria-label="Post comment" disabled={busy || !comment.trim()}
        >{busy ? 'Posting…' : 'Post'}</button
      >
    </footer>
  </form>
  {#each result.Messages || [] as message}<article class="discussion-message">
      <header>
        <span class="comment-avatar">{message.username?.[0]?.toUpperCase()}</span><strong
          >{message.username}</strong
        ><time datetime={message.created_at}>{timeAgo(message.created_at)} ago</time>
        {#if message.user_id === user?.ID}<details class="comment-actions">
            <summary aria-label="Comment actions">•••</summary><button
              disabled={busy}
              onclick={() => remove(`${issuePath}/discussions/${message.id}`)}>Delete</button
            >
          </details>{/if}
      </header>
      <div class="comment-content">{@render content(message.content)}</div>
    </article>{/each}
</section>
