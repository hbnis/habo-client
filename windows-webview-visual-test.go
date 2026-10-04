//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func init() {
	// Diagnostic: mirror the WebView2 environment used by the minimal
	// diagnostic builds that render correctly on the affected machine.
	// Keep this isolated to Windows and to this diagnostic branch.
	os.Setenv("COREWEBVIEW2_FORCED_HOSTING_MODE", "COREWEBVIEW2_HOSTING_MODE_WINDOW_TO_VISUAL")
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--disable-gpu --disable-gpu-compositing")
	os.Setenv("WEBVIEW2_USER_DATA_FOLDER", filepath.Join(os.TempDir(), "HaboClient-Full-KnownGood-WebView2"))
}
