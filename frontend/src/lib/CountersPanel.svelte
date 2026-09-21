<script>
  import { onDestroy } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { DashboardService } from '../../bindings/github.com/ao-data/albiondata-client/internal/dashboard/index.js';

  let counts = $state({});
  DashboardService.GetUploadCounts().then((value) => (counts = value));
  const stop = Events.On('counters:snapshot', (event) => {
    counts = event.data;
  });
  onDestroy(stop);

  const labels = [
    ['marketorders.ingest', 'Market orders'],
    ['markethistories.ingest', 'History rows'],
  ];
</script>

<section class="counters">
  <span class="heading">THIS SESSION</span>
  {#each labels as [topic, label]}
    <div class="counter">
      <span>{label}</span>
      <b>{(counts[topic] ?? 0).toLocaleString()}</b>
    </div>
  {/each}
</section>

<style>
  .counters {
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
    padding: 0.9rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-raised);
  }
  .heading {
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 800;
    letter-spacing: 0.1em;
  }
  .counter {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.75rem;
  }
  .counter span {
    color: var(--text-muted);
    font-size: 0.74rem;
  }
  .counter b {
    color: var(--orange-bright);
    font-family: var(--font-mono);
    font-size: 0.95rem;
    font-variant-numeric: tabular-nums;
  }
</style>
