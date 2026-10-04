#!/usr/bin/env bash

set -eo pipefail

rm -f rsrc_windows_*
rm -f habo-client.exe
rm -f habo-client.*.bak
rm -f .habo-client.*.old
rm -f diagnostic-full-shell.go

rm -f habo-client-amd64-installer.exe

sudo apt-get update && sudo apt-get install -y nsis

go install github.com/tc-hib/go-winres@v0.3.1

export PATH="$PATH:$(go env GOPATH)/bin"

go-winres make

(cd frontend && npm ci && npm run build)

# Build the real Habo window/tray shell, but temporarily remove background
# startup work. This isolates whether packet capture, account refresh,
# the driver check, or the updater interferes with WebView startup.
cp albiondata-client.go diagnostic-full-shell.go
python3 - <<'PY'
from pathlib import Path

p = Path("diagnostic-full-shell.go")
s = p.read_text()
replacements = [
    ("\tgo applyHaboAccountState(client.RefreshHaboAccount())\n", "\t// diagnostic: account refresh disabled\n"),
    ("\tgo func() {\n\t\ttime.Sleep(3 * time.Second)\n\t\tcheckCaptureDriver()\n\t}()\n", "\t// diagnostic: capture-driver startup check disabled\n"),
    ("\tstartUpdater()\n", "\t// diagnostic: updater disabled\n"),
    ("\tgo runClient()\n", "\t// diagnostic: packet capture disabled\n"),
]
for old, new in replacements:
    if old not in s:
        raise SystemExit(f"expected startup block not found: {old!r}")
    s = s.replace(old, new, 1)
p.write_text(s)
PY

env GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui -X main.version=$GITHUB_REF_NAME" -o habo-client.exe diagnostic-full-shell.go windows-webview-visual-test.go

go-winres patch habo-client.exe

cd pkg/nsis
make nsis

cd ../..
ls -la habo-client*

cp habo-client.exe habo-client.exe.copy
gzip -9 habo-client.exe
mv habo-client.exe.gz update-windows-amd64.exe.gz
mv habo-client.exe.copy habo-client.exe
