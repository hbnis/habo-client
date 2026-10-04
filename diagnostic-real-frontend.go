package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/ao-data/albiondata-client/internal/dashboard"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var realFrontendAssets embed.FS

func init() {
	application.RegisterEvent[dashboard.Status]("status:changed")
	application.RegisterEvent[map[string]int64]("counters:snapshot")
	application.RegisterEvent[dashboard.LogLine]("log:line")
}

func main() {
	app := application.New(application.Options{
		Name:        "Habo Real Frontend Diagnostic",
		Description: "Runs the real Habo frontend with minimal backend startup",
		Services: []application.Service{
			application.NewService(&dashboard.DashboardService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(realFrontendAssets),
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: filepath.Join(os.TempDir(), "HaboClient-RealFrontend-Diagnostic"),
			AdditionalBrowserArgs: []string{
				"--disable-gpu",
				"--disable-gpu-compositing",
			},
			UseVisualHosting: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Habo REAL FRONTEND TEST",
		Width:  900,
		Height: 600,
		URL:    "/",
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
