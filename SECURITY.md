# Security Policy

Habo Client is open source and handles a local device credential used to connect the desktop client to The Habo Hub. Security reports are taken seriously.

## Reporting a vulnerability

Please do not publish a working exploit, leaked credential, update signing secret, authentication bypass or other sensitive vulnerability in a public issue.

If GitHub shows a **Report a vulnerability** option for this repository, use that private reporting flow. Otherwise contact The Habo Hub through https://habonis.com and clearly mark the message as a security report.

Useful details include:

- affected Habo Client version or commit
- operating system
- exact reproduction steps
- expected and actual behavior
- whether credentials, update integrity or private market data may be affected

Please give us a reasonable opportunity to investigate and ship a fix before public disclosure.

## Security boundaries

The main security-sensitive areas are:

- Habo Hub device pairing and bearer-token handling
- private market ingest authentication and Premium authorization
- update download and signature verification
- packet capture permissions and local configuration/log files

The Habo Hub server, not the desktop UI, is the authority for Premium access. Modifying the open-source client does not grant Premium access because the ingest API verifies authentication and Premium status server-side.

## Signed updates

Official automatic updater payloads are required to have a detached Ed25519 signature. Habo Client contains only the public verification key. The matching private signing key must never be committed to this repository and is stored separately as a protected release secret.

If signature verification fails or a release is missing its signature, the client refuses to apply the update.

## Supported versions

Security fixes are targeted at the latest Habo Client release. Users should update to the newest signed release when one is available.
