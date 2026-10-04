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

# Start from the real client source. Keep background startup disabled, use the
# stripped visible window that rendered correctly, but re-enable ONLY the real
# Habo system tray. This isolates whether setupTray() itself causes the blank
# WebView on the affected Windows machine.
cp albiondata-client.go diagnostic-full-shell.go
python3 - <<'PY'
from pathlib import Path
import re

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

tray_dashboard = r'''func runDashboardApp() {
\tapp := application.New(application.Options{
\t\tName:        "Habo Client Tray Diagnostic",
\t\tDescription: "Real Habo frontend with only tray re-enabled",
\t\tServices: []application.Service{
\t\t\tapplication.NewService(&dashboard.DashboardService{}),
\t\t},
\t\tAssets: application.AssetOptions{
\t\t\tHandler: application.AssetFileServerFS(assets),
\t\t},
\t})

\tdashboardWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
\t\tTitle:  "Habo Client TRAY TEST",
\t\tName:   "dashboard",
\t\tWidth:  900,
\t\tHeight: 600,
\t\tHidden: false,
\t\tURL:    "/",
\t})

\tdashboardWindowMu.Lock()
\tdashboardWindowRef = dashboardWindow
\tdashboardWindowMu.Unlock()

\t// Only difference from the successful stripped-shell build:
\t// restore the real Habo system tray.
\tsetupTray(app, dashboardWindow)

\tif err := app.Run(); err != nil {
\t\tlog.Error(err)
\t\tos.Exit(1)
\t}
}

'''

pattern = r'func runDashboardApp\(\) \{.*?\n\}\n\n(?=func setupTray)'
s, n = re.subn(pattern, tray_dashboard, s, count=1, flags=re.S)
if n != 1:
    raise SystemExit(f"could not replace runDashboardApp, matches={n}")

# These packages are only used by the lifecycle code that remains stripped.
for import_line in [
    '\t"github.com/ao-data/albiondata-client/internal/dockicon"\n',
    '\t"github.com/ao-data/albiondata-client/internal/winstate"\n',
    '\t"github.com/wailsapp/wails/v3/pkg/events"\n',
]:
    s = s.replace(import_line, "", 1)

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
