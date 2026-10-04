package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var diagnosticAssets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Habo WebView Diagnostic",
		Description: "Compare direct HTML rendering with the Wails asset server",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(diagnosticAssets),
		},
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
		Title:  "DIRECT HTML TEST (RED)",
		Width:  700,
		Height: 500,
		X:      40,
		Y:      80,
		InitialPosition: application.WindowXY,
		HTML: `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Direct HTML</title></head>
<body style="margin:0;background:#d40000;color:#fff;font-family:Arial,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;">
  <div style="text-align:center;">
    <h1 style="font-size:42px;margin:0 0 16px;">DIRECT HTML TEST</h1>
    <p style="font-size:22px;">This window should be RED.</p>
  </div>
</body>
</html>`,
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ASSET SERVER TEST (GREEN)",
		Width:  700,
		Height: 500,
		X:      780,
		Y:      80,
		InitialPosition: application.WindowXY,
		URL:    "/",
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
