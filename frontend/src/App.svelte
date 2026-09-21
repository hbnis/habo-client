<script>
  import Sidebar from './lib/Sidebar.svelte';
  import MarketPanel from './lib/MarketPanel.svelte';
  import LogTail from './lib/LogTail.svelte';
  import Footer from './lib/Footer.svelte';
</script>

<main>
  <div class="body">
    <Sidebar />
    <div class="content">
      <MarketPanel />
      <details class="technical">
        <summary>Technical logs</summary>
        <div class="log-wrap"><LogTail /></div>
      </details>
    </div>
  </div>
  <Footer />
</main>

<style>
  :global(:root) {
    --bg: #0b0f14;
    --bg-raised: #111820;
    --bg-soft: #151e27;
    --bg-sunken: #080b0f;
    --border: rgba(225, 232, 238, 0.10);
    --border-strong: rgba(225, 232, 238, 0.18);
    --text: #f3f5f6;
    --text-muted: #9aa6b0;
    --text-faint: #66727d;
    --orange: #ff7a1a;
    --orange-bright: #ff9b52;
    --orange-soft: rgba(255, 122, 26, 0.12);
    --green: #86c66f;
    --red: #e15e4a;
    --font-display: Georgia, 'Times New Roman', serif;
    --font-body: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    --font-mono: ui-monospace, 'SF Mono', SFMono-Regular, Menlo, monospace;
  }
  :global(*) { box-sizing: border-box; }
  :global(html, body, #app) {
    margin: 0;
    width: 100%;
    height: 100%;
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-body);
  }
  :global(button) { font: inherit; }
  main {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }
  .body {
    display: flex;
    flex: 1;
    min-height: 0;
  }
  .content {
    display: flex;
    flex: 1;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    overflow-y: auto;
    background:
      radial-gradient(circle at 75% 0%, rgba(82, 43, 19, 0.22), transparent 24rem),
      var(--bg);
  }
  .technical {
    margin: 0 2rem 2rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-raised);
    overflow: hidden;
  }
  .technical summary {
    padding: 0.8rem 1rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 700;
    cursor: pointer;
  }
  .log-wrap {
    height: 220px;
    border-top: 1px solid var(--border);
  }
  @media (max-width: 760px) {
    main { overflow: auto; height: auto; min-height: 100vh; }
    .body { flex-direction: column; }
    .content { overflow: visible; }
    .technical { margin: 0 1rem 1rem; }
  }
</style>
