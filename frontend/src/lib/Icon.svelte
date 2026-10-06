<script>
  import { icons } from './icons';
  let { name, size = 16 } = $props();
  const paths = {
    search: 'M21 21l-4.35-4.35M19 11a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
    chevron: 'M6 9l6 6 6-6',
    left: 'M15 18l-6-6 6-6',
    right: 'M9 18l6-6-6-6',
    plus: 'M12 5v14M5 12h14',
    close: 'M6 6l12 12M6 18L18 6',
    menu: 'M4 6h16M4 12h16M4 18h16',
    logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9',
    check: 'M5 12l4 4L19 6',
    clock: 'M12 8v4l3 2M22 12a10 10 0 1 1-20 0 10 10 0 0 1 20 0',
    user: 'M20 21v-2a7 7 0 0 0-14 0v2M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0',
    copy: 'M9 9h11v12H9zM15 9V3H3v12h6'
  };
  let original = $derived(icons[name]);
</script>

<svg
  width={size}
  height={size}
  viewBox={original?.viewBox || '0 0 24 24'}
  fill="none"
  stroke={original && !original.stroke ? 'none' : 'currentColor'}
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#if original}{@html original.body}{:else}<path d={paths[name] || paths.help} />{/if}
</svg>
