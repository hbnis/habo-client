# Habo Client

Habo Client is a lightweight Albion Online market scanner for [The Habo Hub](https://habonis.com).

The goal is intentionally narrow: help players scan marketplace data for flipping tools without adding gameplay automation or unrelated features.

## What it does

- Watches Albion Online network traffic using the proven Albion Online Data Project capture/parser foundation.
- Reads market orders and market history that the player manually loads in game.
- Shows scanner status, detected server and session counters in a Habo-branded desktop UI.
- Habo Client scanning is private-only: supported market observations are sent to the connected Habo Hub account when Private scanning is available.
- Habo Hub Premium is required for private market ingest and Private Flips.
- Only market orders and market history are forwarded by the Habo build.

Habo Client does not click, search, buy, sell, inject into the game or automate gameplay.

## Current development status

The desktop shell, market-only routing, Habo Hub account pairing, authenticated private ingest and Windows packaging are implemented.

The client no longer exposes a Public scan mode. The AODP project remains the packet capture/parser foundation, but Habo Client market uploads are routed only to the connected Habo Hub account when private ingest is available.

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
