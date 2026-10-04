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

# Start from the real client source, then strip background startup and the
# window/tray lifecycle down to the same basic shape as the minimal frontend
# diagnostic that renders correctly on the affected Windows machine.
cp albiondata-client.go diagnostic-full-shell.go
python3 - <<'PY'
from pathlib import Path
import re

p = Path("diagnostic-full-shell.go")
s = p.read_text()

# Disable background systems first.
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

# Replace the full window/tray lifecycle with a plain visible window while
# keeping the real frontend, service bindings, and the rest of the package.
minimal_dashboard = r'''func runDashboardApp() {
	app := application.New(application.Options{
		Name:        "Habo Client Shell Diagnostic",
		Description: "Real Habo frontend with stripped window lifecycle",
		Services: []application.Service{
			application.NewService(&dashboard.DashboardService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	dashboardWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Habo Client SHELL TEST",
		Name:   "dashboard",
		Width:  900,
		Height: 600,
		Hidden: false,
		URL:    "/",
	})

	dashboardWindowMu.Lock()
	dashboardWindowRef = dashboardWindow
	dashboardWindowMu.Unlock()

	if err := app.Run(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

'''

pattern = r'func runDashboardApp\(\) \{.*?\n\}\n\n(?=func setupTray)'
s, n = re.subn(pattern, minimal_dashboard, s, count=1, flags=re.S)
if n != 1:
    raise SystemExit(f"could not replace runDashboardApp, matches={n}")

# Those packages are only used by the stripped window lifecycle code.
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
