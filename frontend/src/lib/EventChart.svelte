<script>
  import { number } from './format';
  let { events = [], period = '24h', start = '', end = '', label = 'Number of Errors' } = $props();
  let selected = $state(-1);
  let width = $state(600);
  let chart = $derived.by(() => {
    const units = { s: 1000, m: 60000, h: 3600000, d: 86400000, w: 604800000 };
    const match = /^(\d+)([smhdw])$/.exec(period);
    const duration = match ? Number(match[1]) * units[match[2]] : 86400000;
    const to = end ? new Date(end + 'Z').getTime() : Math.ceil(Date.now() / 3600000) * 3600000;
    const from = start ? new Date(start + 'Z').getTime() : to - duration;
    const span = Math.max(to - from, 3600000);
    const interval = Math.max(
      3600000,
      span <= 86400000 ? 3600000 : span <= 604800000 ? 21600000 : 86400000,
      Math.ceil(span / 100)
    );
    const count = Math.max(1, Math.ceil(span / interval));
    const buckets = Array.from({ length: count }, (_, i) => ({
      time: from + i * interval,
      count: 0
    }));
    for (const event of events || []) {
      const i = Math.floor((new Date(event.TS).getTime() - from) / interval);
      if (i >= 0 && i < buckets.length) buckets[i].count += event.Count;
    }
    return { buckets, max: Math.max(1, ...buckets.map((b) => b.count)), span };
  });
  let ticks = $derived(
    Array.from({ length: width < 400 ? 3 : 5 }, (_, i) =>
      Math.round((i * (chart.buckets.length - 1)) / (width < 400 ? 2 : 4))
    )
  );
  function tickLabel(time) {
    return new Date(time).toLocaleString(
      'en',
      chart.span <= 86400000
        ? { hour: '2-digit', minute: '2-digit', hour12: false }
        : { month: 'short', day: 'numeric' }
    );
  }
</script>

<div class="event-chart" bind:clientWidth={width}>
  <svg viewBox={`0 0 ${Math.max(width, 240)} 188`} role="img" aria-label={label}>
    <title>{label}: {(events || []).reduce((sum, e) => sum + e.Count, 0)} total events</title>
    {#each [0, 0.5, 1] as fraction}<text
        x="36"
        y={145 - fraction * 124}
        text-anchor="end"
        class="axis-label">{number(Math.round(chart.max * fraction))}</text
      >{/each}
    <line x1="46" x2={Math.max(width, 240) - 16} y1="143" y2="143" class="axis-line" />
    {#each chart.buckets as bucket, i}
      {@const step = (Math.max(width, 240) - 64) / chart.buckets.length}
      <rect
        x={48 + i * step}
        y={143 - (bucket.count / chart.max) * 124}
        width={Math.max(1, step * 0.8)}
        height={(bucket.count / chart.max) * 124}
        class="chart-bar"
        class:selected={selected === i}
        ><title>{new Date(bucket.time).toLocaleString()}: {bucket.count} errors</title></rect
      >
    {/each}
    {#each [...new Set(ticks)] as index}<text
        x={48 + (index * (Math.max(width, 240) - 64)) / chart.buckets.length}
        y="167"
        text-anchor="middle"
        class="axis-label">{tickLabel(chart.buckets[index].time)}</text
      >{/each}
  </svg>
  <div class="chart-hitarea" onpointerleave={() => (selected = -1)} aria-hidden="true">
    {#each chart.buckets as bucket, i}<div role="presentation" onpointerenter={() => (selected = i)}></div>{/each}
  </div>
  <div class="chart-legend">
    <span class="chart-swatch"></span> Errors
    <strong>{selected >= 0 ? number(chart.buckets[selected]?.count || 0) : '—'}</strong
    >{#if selected >= 0}<span>{new Date(chart.buckets[selected].time).toLocaleString()}</span>{/if}
  </div>
</div>
