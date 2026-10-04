package main

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	app := application.New(application.Options{
		Name:        "Habo WebView Diagnostic",
		Description: "Minimal WebView2 diagnostic",
		Windows: application.WindowsOptions{
			WebviewUserDataPath: filepath.Join(os.TempDir(), "HaboClient-WebView2-Diagnostic"),
			AdditionalBrowserArgs: []string{
				"--disable-gpu",
				"--disable-gpu-compositing",
			},
			UseVisualHosting: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Habo WebView Diagnostic",
		Width:  900,
		Height: 600,
		HTML: `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Habo WebView Diagnostic</title></head>
<body style="margin:0;background:#d40000;color:#fff;font-family:Arial,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;">
  <div style="text-align:center;">
    <h1 style="font-size:48px;margin:0 0 16px;">HABO WEBVIEW DIRECT HTML TEST</h1>
    <p style="font-size:24px;">If you can read this, WebView2 is rendering.</p>
  </div>
</body>
</html>`,
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
