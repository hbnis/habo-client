<script>
  import { onDestroy } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { DashboardService } from '../../bindings/github.com/ao-data/albiondata-client/internal/dashboard/index.js';

  let status = $state({
    CaptureRunning: false,
    CaptureError: false,
    ServerID: 0,
    UploadMode: 'private',
    PrivateReady: false,
    EncryptionStatus: '',
  });
  let counts = $state({});

  DashboardService.GetStatus().then((value) => (status = value));
  DashboardService.GetUploadCounts().then((value) => (counts = value));

  const stopStatus = Events.On('status:changed', (event) => {
    status = event.data;
  });
  const stopCounters = Events.On('counters:snapshot', (event) => {
    counts = event.data;
  });

  onDestroy(() => {
    stopStatus();
    stopCounters();
  });

  const serverNames = { 0: 'Waiting', 1: 'Americas', 2: 'Asia', 3: 'Europe' };
  let serverName = $derived(serverNames[status.ServerID] ?? 'Unknown');
  let orders = $derived(counts['marketorders.ingest'] ?? 0);
  let histories = $derived(counts['markethistories.ingest'] ?? 0);
  let captureState = $derived(
    status.CaptureError ? 'Capture stopped' : status.CaptureRunning ? 'Scanning' : 'Waiting for Albion'
  );

  function openGuides() {
    // Guide videos will be wired here when the Guides section is ready.
  }
</script>

<section class="market-panel">
  <div class="hero">
    <div>
      <span class="eyebrow">MARKET SCANNER</span>
      <h1>Browse the market.<br /><em>Habo Client takes the notes.</em></h1>
      <p>
        Open the Albion marketplace and browse items normally. Habo Client records the market data
        your game loads so The Habo Hub can find useful flipping opportunities.
      </p>
    </div>
    <button class="primary guides" type="button" onclick={openGuides} disabled title="Guide videos are coming soon">Guides</button>
  </div>

  <div class="status-grid">
    <article>
      <span>Capture</span>
      <strong class:live={status.CaptureRunning && !status.CaptureError}>{captureState}</strong>
    </article>
    <article>
      <span>Server</span>
      <strong>{serverName}</strong>
    </article>
    <article>
      <span>Scan mode</span>
      <strong>{status.PrivateReady ? 'Private' : 'Unavailable'}</strong>
    </article>
    <article>
      <span>Orders scanned</span>
      <strong>{orders.toLocaleString()}</strong>
    </article>
    <article>
      <span>History rows</span>
      <strong>{histories.toLocaleString()}</strong>
    </article>
    <article>
      <span>Market data</span>
      <strong>{status.EncryptionStatus === 'clear' ? 'Readable' : status.EncryptionStatus === 'encrypted' ? 'Encrypted' : 'Waiting'}</strong>
    </article>
  </div>

  <div class="guide">
    <div class="guide-copy">
      <span class="eyebrow">HOW IT WORKS</span>
      <h2>You still do everything in Albion.</h2>
      <p>
        Habo Client does not click, search, buy or sell anything for you. You open the marketplace,
        choose the items and move through the pages. The client only records the market information
        that Albion sends to your computer.
      </p>
    </div>
    <ol>
      <li><b>1</b><span>Open Albion Online</span></li>
      <li><b>2</b><span>Open a marketplace</span></li>
      <li><b>3</b><span>Browse the items you want to scan</span></li>
      <li><b>4</b><span>Check the opportunities on The Habo Hub</span></li>
    </ol>
  </div>

  {#if status.EncryptionStatus === 'encrypted'}
    <div class="warning">
      Albion returned encrypted market data. The client cannot read that market response right now.
    </div>
  {/if}
</section>

<style>
  .market-panel {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
    padding: 2rem;
    overflow-y: auto;
  }
  .hero {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 2rem;
    padding: 1.8rem;
    border: 1px solid var(--border);
    border-radius: 18px;
    background:
      radial-gradient(circle at 90% 0%, rgba(255, 122, 26, 0.18), transparent 18rem),
      var(--bg-raised);
  }
  .hero > div { min-width: 0; }
  .eyebrow {
    color: var(--orange);
    font-size: 0.68rem;
    font-weight: 800;
    letter-spacing: 0.14em;
  }
  h1 {
    margin: 0.65rem 0 0.8rem;
    font-family: var(--font-display);
    font-size: clamp(2rem, 4vw, 3.5rem);
    line-height: 0.98;
    letter-spacing: -0.035em;
  }
  h1 em { color: var(--orange); font-weight: 500; }
  .hero p, .guide p {
    max-width: 680px;
    margin: 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .primary {
    flex: none;
    border: 1px solid var(--orange);
    border-radius: 10px;
    padding: 0.8rem 1rem;
    background: var(--orange);
    color: #111;
    font-weight: 850;
    cursor: pointer;
  }
  .primary:hover { background: var(--orange-bright); }
  .primary.guides:disabled {
    cursor: default;
    border-color: rgba(255, 122, 26, 0.35);
    background: var(--orange-soft);
    color: var(--orange-bright);
    opacity: 1;
  }
  .status-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.8rem;
  }
  .status-grid article {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
    min-width: 0;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-raised);
  }
  .status-grid span {
    color: var(--text-faint);
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }
  .status-grid strong {
    overflow: hidden;
    color: var(--text);
    font-size: 1rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .status-grid strong.live { color: var(--green); }
  .guide {
    display: grid;
    grid-template-columns: minmax(0, 1.2fr) minmax(240px, 0.8fr);
    gap: 1.5rem;
    padding: 1.4rem;
    border: 1px solid var(--border);
    border-radius: 14px;
    background: var(--bg-raised);
  }
  .guide h2 {
    margin: 0.45rem 0 0.65rem;
    font-family: var(--font-display);
    font-size: 1.6rem;
  }
  ol {
    display: grid;
    gap: 0.65rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 28px minmax(0, 1fr);
    align-items: center;
    gap: 0.7rem;
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  li b {
    display: grid;
    width: 28px;
    height: 28px;
    place-items: center;
    border: 1px solid rgba(255, 122, 26, 0.35);
    border-radius: 8px;
    background: var(--orange-soft);
    color: var(--orange-bright);
    font-size: 0.75rem;
  }
  .warning {
    padding: 0.9rem 1rem;
    border: 1px solid rgba(225, 94, 74, 0.4);
    border-radius: 10px;
    background: rgba(225, 94, 74, 0.09);
    color: #ffaaa0;
    font-size: 0.82rem;
    line-height: 1.5;
  }
  @media (max-width: 760px) {
    .market-panel { padding: 1rem; }
    .hero { align-items: stretch; flex-direction: column; }
    .primary { width: 100%; }
    .status-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .guide { grid-template-columns: 1fr; }
  }
</style>
