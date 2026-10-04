//go:build windows

package main

import "os"

func init() {
	// Diagnostic: force WebView2 visual hosting. The minimal diagnostic
	// rendered correctly with this mode on the affected Windows machine.
	os.Setenv("COREWEBVIEW2_FORCED_HOSTING_MODE", "COREWEBVIEW2_HOSTING_MODE_WINDOW_TO_VISUAL")
}
