# Habo Client Safety & Privacy

Habo Client is intentionally narrow. It is a market scanner built around the Albion Online Data Project packet-capture/parser foundation and The Habo Hub's private market tools.

## What Habo Client reads

Habo Client listens to Albion Online network traffic through the system packet-capture driver. The Habo build uses that traffic to identify supported market-order and market-history messages that the player manually loads in game.

It does not need or request your Albion Online password.

## What Habo Client sends to The Habo Hub

When you connect Habo Client to your Habo Hub account, the client may send:

- a randomly generated device token used for authentication
- the device name reported by the operating system
- the operating-system platform
- the Habo Client version
- supported Albion market orders and market-history observations when private scanning is available

The Habo Hub server stores a hash of the device token rather than the raw bearer token. Premium access is checked by the server before private market ingest is accepted.

## Local credential storage

The bearer token is stored only so the device can reconnect without pairing every launch.

- On Windows, the token is encrypted with Windows DPAPI and bound to the current Windows user before it is written to disk.
- On other supported platforms, the token is stored in the user's configuration directory with restrictive file permissions.
- Disconnecting the Habo account removes the saved token from the client configuration.

An older plaintext Windows token file is automatically migrated to DPAPI-protected storage after a successful read.

## Network destination protections

Production Habo account requests are restricted to HTTPS on `habonis.com` or its subdomains. Private-ingest and browser-pairing URLs returned by the server must keep the same trusted origin and expected path before the client will use them.

Plain HTTP is accepted only for localhost development overrides.

## Automatic updates

Official automatic updater payloads are signed with Ed25519. The client contains the public verification key and verifies the detached signature before decompressing or applying an update.

The private signing key is not part of the application or public repository. A release without a valid signature is rejected.

Update metadata and release assets are fetched only through HTTPS GitHub endpoints and trusted GitHub asset hosts.

## What Habo Client does not do

Habo Client does not:

- click, buy, sell or search for you
- inject input into Albion Online
- automate combat, movement or marketplace actions
- read keyboard input or mouse input
- capture screenshots or video
- request your Albion Online account password
- remotely control the game client

The user still performs every in-game action manually.

## Logs

Habo Client keeps local diagnostic logs to help troubleshoot capture and connection problems. Users should review logs before posting them publicly because diagnostic output can contain local environment details.

## Open source

The source code is public so users and contributors can inspect how packet capture, account pairing, private ingest and updates work. Public source improves transparency, but it does not replace security controls, which is why authentication, server-side Premium checks, strict URL validation and signed updates are enforced independently.
