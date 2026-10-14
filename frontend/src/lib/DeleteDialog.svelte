<script>
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  let { target, onclose, onconfirm } = $props();
  let dialog = $state();
  let cancelButton = $state();
  let busy = $state(false);
  let error = $state('');
  onMount(() => {
    const previous = document.activeElement;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    dialog.showModal();
    cancelButton.focus();
    return () => {
      document.body.style.overflow = overflow;
      if (previous instanceof HTMLElement) previous.focus();
    };
  });
  async function confirm() {
    if (busy) return;
    busy = true;
    error = '';
    try {
      await onconfirm();
      onclose();
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<dialog
  bind:this={dialog}
  class="delete-dialog"
  aria-labelledby="delete-title"
  aria-describedby="delete-description"
  oncancel={(e) => {
    e.preventDefault();
    if (!busy) onclose();
  }}
>
  <div class="delete-dialog-body">
    <div class="delete-icon">
      <svg
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"><path d="M3 6h18M9 6V4h6v2M5 6l1 14h12l1-14M10 10v6M14 10v6" /></svg
      >
    </div>
    <button
      class="icon-button delete-close"
      aria-label="Close dialog"
      disabled={busy}
      onclick={onclose}><Icon name="close" size={18} /></button
    >
    <h2 id="delete-title">Delete {target.kind}?</h2>
    <p id="delete-description">
      {#if target.name}<strong>{target.name}</strong> will be permanently deleted.{:else}This {target.kind}
        will be permanently deleted.{/if} This action cannot be undone.
    </p>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  </div>
  <footer>
    <button bind:this={cancelButton} disabled={busy} onclick={onclose}>Cancel</button><button
      class="delete-confirm"
      disabled={busy}
      onclick={confirm}
      >{#if busy}<span class="delete-spinner"></span>Deleting…{:else}Delete {target.kind}{/if}</button
    >
  </footer>
</dialog>
