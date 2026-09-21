# Habo Client

Habo Client is a lightweight Albion Online market scanner for [The Habo Hub](https://habonis.com).

The goal is intentionally narrow: help players scan marketplace data for flipping tools without adding gameplay automation or unrelated features.

## What it does

- Watches Albion Online network traffic using the proven Albion Online Data Project capture/parser foundation.
- Reads market orders and market history that the player manually loads in game.
- Shows scanner status, detected server and session counters in a Habo-branded desktop UI.
- Public mode contributes scanned market data to the Albion Online Data Project.
- Private mode routing is built into the client and becomes available when a Habo private ingest/account connection is configured.
- Only market orders and market history are forwarded by the Habo build.

Habo Client does not click, search, buy, sell, inject into the game or automate gameplay.

## Current development status

The desktop shell, market-only routing, Public/Private scan-mode foundation and Windows Habo Client packaging are in progress.

The next major piece is Habo Hub account pairing and authenticated private ingest so Private mode can associate scans with the signed-in Habo Hub account.

## Albion Online Data Project foundation

This project is forked from [ao-data/albiondata-client](https://github.com/ao-data/albiondata-client) and keeps its packet capture and Albion protocol parsing foundation.

The upstream project is licensed under the MIT License. The original copyright notice and MIT license are preserved in this repository.

Upstream developers and maintainers include the Albion Online Data Project contributors, Regner, pcdummy, Ultraporing, broderickhyman, Stanx/phendryx, Walkynn and others listed in the upstream project history.

## Windows development

Windows packet capture requires [Npcap](https://npcap.com/#download) installed in WinPcap API-compatible mode.

The app uses Go, Wails v3, Svelte and Vite.

Basic build flow:

```bash
make frontend
make build-windows
```

## Disclaimer

Habo Client is an independent third-party fan project. It is not affiliated with or endorsed by Sandbox Interactive or Albion Online.

Before a public release, current Albion Online rules and third-party software guidance should be reviewed again because those rules can change.

## License

See [LICENSE](LICENSE). The AODP foundation and modifications in this repository remain subject to the included MIT License.
