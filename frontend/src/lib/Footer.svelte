<script>
  import { onDestroy } from 'svelte';
  import { Browser, Events } from '@wailsio/runtime';
  import { DashboardService } from '../../bindings/github.com/ao-data/albiondata-client/internal/dashboard/index.js';

  let version = $state('');
  DashboardService.GetStatus().then((value) => (version = value.Version));
  const stop = Events.On('status:changed', (event) => {
    version = event.data.Version;
  });
  onDestroy(stop);

  function open(event, url) {
    event.preventDefault();
    Browser.OpenURL(url);
  }
</script>

<footer class="footer">
  <div class="links">
    <a href="https://habonis.com" onclick={(event) => open(event, 'https://habonis.com')}>habonis.com</a>
    <a href="https://discord.gg/habo" onclick={(event) => open(event, 'https://discord.gg/habo')}>Discord</a>
    <a href="https://github.com/ao-data/albiondata-client" onclick={(event) => open(event, 'https://github.com/ao-data/albiondata-client')}>AODP foundation</a>
  </div>
  <span>Habo Client v{version || 'dev'}</span>
</footer>

<style>
  .footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex: none;
    padding: 0.55rem 1.1rem;
    border-top: 1px solid var(--border);
    background: #0d1319;
  }
  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
  a, .footer > span {
    color: var(--text-faint);
    font-size: 0.68rem;
    text-decoration: none;
  }
  a:hover { color: var(--orange-bright); }
  @media (max-width: 620px) {
    .footer { align-items: flex-start; flex-direction: column; }
  }
</style>
