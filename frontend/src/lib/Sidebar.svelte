<script>
  import { onDestroy } from 'svelte';
  import { Browser, Events } from '@wailsio/runtime';
  import { DashboardService } from '../../bindings/github.com/ao-data/albiondata-client/internal/dashboard/index.js';
  import CountersPanel from './CountersPanel.svelte';

  let status = $state({
    Version: '',
    UpdateAvailable: '',
    CaptureRunning: false,
    CaptureError: false,
    ServerID: 0,
    DriverWarning: '',
    DriverHelpURL: '',
    EncryptionStatus: '',
    UploadMode: 'public',
    PrivateReady: false,
    HaboConnected: false,
    HaboPairing: false,
    HaboPremium: false,
    HaboDisplayName: '',
    HaboPairCode: '',
    HaboConnectURL: '',
    HaboError: '',
  });

  DashboardService.GetStatus().then((value) => (status = value));
  const stopStatus = Events.On('status:changed', (event) => {
    status = event.data;
  });
  const stopOpenConnect = Events.On('habo:open-connect-url', (event) => {
    if (typeof event.data === 'string' && event.data) Browser.OpenURL(event.data);
  });
  onDestroy(() => {
    stopStatus();
    stopOpenConnect();
  });

  const serverNames = { 0: 'Waiting for Albion', 1: 'Americas', 2: 'Asia', 3: 'Europe' };
  let serverLabel = $derived(serverNames[status.ServerID] ?? 'Unknown');
  let captureLabel = $derived(
    status.CaptureError ? 'Capture error' : status.CaptureRunning ? 'Scanner running' : 'Waiting for Albion'
  );

  async function setMode(mode) {
    if (mode === 'private' && !status.PrivateReady) return;
    await Events.Emit('habo:scan-mode', mode);
  }

  function openDriverHelp(event) {
    event.preventDefault();
    Browser.OpenURL(status.DriverHelpURL);
  }

  async function connectHabo() {
    if (status.HaboConnectURL) {
      Browser.OpenURL(status.HaboConnectURL);
      return;
    }
    await Events.Emit('habo:connect');
  }

  function finishPairing() {
    if (status.HaboConnectURL) Browser.OpenURL(status.HaboConnectURL);
  }

  async function disconnectHabo() {
    await Events.Emit('habo:disconnect');
  }

  function openHub() {
    Browser.OpenURL('https://habonis.com');
  }
</script>

<aside class="sidebar">
  <div class="brand">
    <div class="brand-mark">H</div>
    <div>
      <strong>Habo Client</strong>
      <span>by The Habo Hub</span>
    </div>
  </div>

  <section class="status-card">
    <span class="status-dot" class:live={status.CaptureRunning && !status.CaptureError} class:error={status.CaptureError}></span>
    <div>
      <small>CAPTURE</small>
      <b>{captureLabel}</b>
      <span>{serverLabel}</span>
    </div>
  </section>

  <section class="account-card">
    <div class="section-head">
      <span>HABO HUB ACCOUNT</span>
      <small class:connected={status.HaboConnected}>{status.HaboConnected ? 'CONNECTED' : status.HaboPairing ? 'PAIRING' : 'NOT CONNECTED'}</small>
    </div>

    {#if status.HaboConnected}
      <div class="account-name">
        <span class="account-dot"></span>
        <div><b>{status.HaboDisplayName || 'Habo Hub user'}</b><small>{status.HaboPremium ? 'Premium · Private scans available' : 'Free · Public scans available'}</small></div>
      </div>
      <button class="account-secondary" type="button" onclick={disconnectHabo}>Disconnect</button>
    {:else if status.HaboPairing && status.HaboConnectURL}
      <p class="pair-copy">Finish connecting in your browser.</p>
      {#if status.HaboPairCode}
        <div class="pair-code"><small>PAIRING CODE</small><b>{status.HaboPairCode}</b></div>
      {/if}
      <button class="account-primary" type="button" onclick={finishPairing}>Continue in browser</button>
    {:else}
      <p class="pair-copy">Connect your Habo Hub account to use Habo Hub market tools. Public scanning is available to every connected user.</p>
      <button class="account-primary" type="button" onclick={connectHabo}>Connect Habo Hub</button>
    {/if}

    {#if status.HaboError}
      <p class="account-error">{status.HaboError}</p>
    {/if}
  </section>

  <section class="mode-card">
    <div class="section-head">
      <span>SCAN MODE</span>
      <small>{status.UploadMode === 'private' ? 'PRIVATE' : 'PUBLIC'}</small>
    </div>
    <div class="mode-switch" role="group" aria-label="Scan mode">
      <button
        type="button"
        class:active={status.UploadMode !== 'private'}
        onclick={() => setMode('public')}
      >Public</button>
      <button
        type="button"
        class:active={status.UploadMode === 'private'}
        disabled={!status.PrivateReady}
        title={status.PrivateReady ? 'Keep scans private to your Habo Hub account' : status.HaboConnected ? 'Private mode requires Habo Hub Premium' : 'Connect your Habo Hub account first'}
        onclick={() => setMode('private')}
      >Private</button>
    </div>
    {#if status.UploadMode === 'private'}
      <p>Scanned market data goes only to your Habo Hub private ingest.</p>
    {:else}
      <p>New market scans contribute to the public Albion Online Data Project. If you just switched from Private, reopen or refresh the market category in Albion to send a new scan.</p>
    {/if}
    {#if !status.PrivateReady}
      <small class="private-note">{status.HaboConnected ? 'Private mode is a Habo Hub Premium feature. Public mode still contributes your scans to AODP.' : 'Connect Habo Hub to see your account access. Public scans contribute to AODP.'}</small>
    {/if}
  </section>

  <CountersPanel />

  {#if status.DriverWarning}
    <section class="warning">
      <b>Capture setup needed</b>
      <span>{status.DriverWarning}</span>
      {#if status.DriverHelpURL}
        <a href={status.DriverHelpURL} onclick={openDriverHelp}>Get Npcap</a>
      {/if}
    </section>
  {/if}

  {#if status.UpdateAvailable}
    <section class="update">
      <span>UPDATE AVAILABLE</span>
      <b>{status.UpdateAvailable}</b>
    </section>
  {/if}

  <button class="hub-button" type="button" onclick={openHub}>Open The Habo Hub</button>
</aside>

<style>
  .sidebar {
    display: flex;
    width: 260px;
    flex: 0 0 260px;
    min-height: 0;
    flex-direction: column;
    gap: 1rem;
    padding: 1.25rem;
    overflow-y: auto;
    border-right: 1px solid var(--border);
    background: #0d1319;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding-bottom: 0.35rem;
  }
  .brand-mark {
    display: grid;
    width: 38px;
    height: 38px;
    place-items: center;
    flex: none;
    border: 1px solid rgba(255, 122, 26, 0.4);
    border-radius: 11px;
    background: linear-gradient(145deg, #ff963f, #c84a0e);
    color: #160b03;
    font-family: var(--font-display);
    font-size: 1.45rem;
    font-weight: 900;
  }
  .brand > div:last-child {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0.1rem;
  }
  .brand strong { font-size: 1rem; }
  .brand span { color: var(--orange-bright); font-size: 0.7rem; font-weight: 700; }

  .status-card, .account-card, .mode-card, .warning, .update {
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-raised);
  }
  .status-card {
    display: grid;
    grid-template-columns: 10px minmax(0, 1fr);
    gap: 0.7rem;
    padding: 0.9rem;
  }
  .status-card > div {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0.18rem;
  }
  .status-card small, .section-head span, .update span {
    color: var(--text-faint);
    font-size: 0.62rem;
    font-weight: 800;
    letter-spacing: 0.1em;
  }
  .status-card b { font-size: 0.86rem; }
  .status-card span:not(.status-dot) {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.74rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .status-dot {
    width: 9px;
    height: 9px;
    margin-top: 0.15rem;
    border-radius: 50%;
    background: #737c84;
  }
  .status-dot.live {
    background: var(--green);
    box-shadow: 0 0 0 5px rgba(134, 198, 111, 0.10);
  }
  .status-dot.error { background: var(--red); }

  .account-card, .mode-card { padding: 0.9rem; }
  .section-head small.connected { color: var(--green); }
  .account-name {
    display: grid;
    grid-template-columns: 9px minmax(0, 1fr);
    align-items: center;
    gap: 0.65rem;
    margin: 0.2rem 0 0.75rem;
  }
  .account-dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: var(--green);
    box-shadow: 0 0 0 5px rgba(134, 198, 111, 0.10);
  }
  .account-name > div { display: flex; min-width: 0; flex-direction: column; gap: 0.15rem; }
  .account-name b { overflow: hidden; font-size: 0.8rem; text-overflow: ellipsis; white-space: nowrap; }
  .account-name small, .pair-copy {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.68rem;
    line-height: 1.45;
  }
  .pair-code {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.65rem;
    margin: 0.7rem 0;
    padding: 0.65rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-sunken);
  }
  .pair-code small { color: var(--text-faint); font-size: 0.58rem; font-weight: 800; letter-spacing: 0.08em; }
  .pair-code b { color: var(--orange-bright); font-family: var(--font-mono); font-size: 0.9rem; letter-spacing: 0.04em; }
  .account-primary, .account-secondary {
    width: 100%;
    margin-top: 0.7rem;
    border-radius: 8px;
    padding: 0.6rem 0.7rem;
    font-size: 0.7rem;
    font-weight: 850;
    cursor: pointer;
  }
  .account-primary {
    border: 1px solid var(--orange);
    background: var(--orange);
    color: #15100c;
  }
  .account-secondary {
    border: 1px solid var(--border-strong);
    background: var(--bg-sunken);
    color: var(--text-muted);
  }
  .account-error {
    margin: 0.65rem 0 0;
    color: #ffaaa0;
    font-size: 0.65rem;
    line-height: 1.4;
  }
  .section-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    margin-bottom: 0.7rem;
  }
  .section-head small {
    color: var(--orange-bright);
    font-size: 0.61rem;
    font-weight: 900;
  }
  .mode-switch {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.35rem;
    padding: 0.25rem;
    border: 1px solid var(--border);
    border-radius: 9px;
    background: var(--bg-sunken);
  }
  .mode-switch button {
    border: 0;
    border-radius: 7px;
    padding: 0.55rem;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.75rem;
    font-weight: 800;
    cursor: pointer;
  }
  .mode-switch button.active {
    background: var(--orange-soft);
    color: var(--orange-bright);
  }
  .mode-switch button:disabled {
    cursor: not-allowed;
    opacity: 0.4;
  }
  .mode-card p, .private-note {
    display: block;
    margin: 0.7rem 0 0;
    color: var(--text-muted);
    font-size: 0.7rem;
    line-height: 1.5;
  }
  .private-note {
    color: #7f8992;
    font-size: 0.65rem;
  }
  .warning, .update {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.85rem;
  }
  .warning { border-color: rgba(225, 94, 74, 0.28); }
  .warning b { color: #ffaaa0; font-size: 0.76rem; }
  .warning span { color: var(--text-muted); font-size: 0.68rem; line-height: 1.45; }
  .warning a { color: var(--orange-bright); font-size: 0.7rem; font-weight: 800; text-decoration: none; }
  .update b { font-size: 0.78rem; }
  .hub-button {
    margin-top: auto;
    border: 1px solid rgba(255, 122, 26, 0.4);
    border-radius: 9px;
    padding: 0.7rem 0.8rem;
    background: var(--orange-soft);
    color: var(--orange-bright);
    font-size: 0.75rem;
    font-weight: 850;
    cursor: pointer;
  }
  .hub-button:hover { border-color: var(--orange); }
  @media (max-width: 760px) {
    .sidebar {
      width: 100%;
      flex: none;
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .hub-button { margin-top: 0; }
  }
</style>
