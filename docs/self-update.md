# Self-update mechanics

`startUpdater()` in `albiondata-client.go` only runs at all when
`version != "" && !strings.Contains(version, "dev")` - a dev build
(the default `main.version=dev` ldflags used by `scripts/run.sh` and
`buildall_main.sh`) never checks for updates. It polls hourly via
`github.com/ao-data/go-githubupdate`, which wraps `minio/selfupdate` to
replace the running binary in place (see gui-dashboard.md for why
that's always a single-file replace, never a bundle).

## Exact filename matching - no fuzzy match, no fallback

The updater builds the expected release-asset filename as:

```
"update-" + runtime.GOOS + "-" + runtime.GOARCH + ".gz"          // .exe.gz on windows
```

and does a literal `asset.GetName() == reqFilename` string comparison
against the target repo's latest release assets. No fallback, no
partial match - if the exact filename isn't there, it's
`ErrorNoBinary` ("No binary for the update found"), regardless of
whether *some* compatible binary exists in the release.

Habo Client is shipped for Windows only. Release automation therefore
publishes the Windows updater payload for users; there is no macOS
updater asset or macOS build target.

## Target repo is compile-time hardcoded

`client.ConfigGlobal.UpdateGithubOwner`/`UpdateGithubRepo` default to
`"ao-data"`/`"albiondata-client"` (`client/config.go`). There *is* a
viper-based override for these via `config.yaml`, but it's commented
out in `setupWebsocketFlags()` ("Keeping for local development, but
commenting out so it's not live") - so in a normal build there is no
config-file or flag way to point the updater at a fork. Testing against
a personal fork (e.g. `phendryx/albiondata-client`) currently means
building a local binary with `UpdateGithubOwner`/`UpdateGithubRepo`
edited directly in `client/config.go` before building - not committed,
reverted before any real commit.
