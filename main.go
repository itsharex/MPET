package main

import (
	"context"
	"embed"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	rt "runtime"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "Multi-Protocol Exploitation Toolkit",
		Width:             1200,
		Height:            820,
		MinWidth:          1024,
		MinHeight:         768,
		// MaxWidth:          1280,
		// MaxHeight:         800,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         rt.GOOS != "darwin",
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Handler:    nil,
			Middleware: nil,
		},
		DragAndDrop:   DragAndDropOptions(),
		OnDomReady: func(ctx context.Context) {
			runtime.OnFileDrop(ctx, func(x, y int, paths []string) {

			})
		},
		Menu:             nil,
		Logger:           nil,
		LogLevel:         logger.WARNING,
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		WindowStartState: options.Normal,
		Bind: []interface{}{
			app,
			app.backend,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableWindowIcon:                 false,
			// DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			BackdropType:                      windows.Acrylic,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			Appearance:           mac.NSAppearanceNameVibrantLight,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Multi-Protocol Exploitation Toolkit",
				Message: "",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
func DragAndDropOptions() *options.DragAndDrop {
	if rt.GOOS == "windows" {
		return &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		}
	} else {
		return &options.DragAndDrop{
			EnableFileDrop: true,
		}
	}
}