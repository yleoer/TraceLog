package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	_ "time/tzdata"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"tracelog/internal/config"
	"tracelog/internal/desktop"
	"tracelog/internal/service"
)

func main() {
	cfg := config.Load()
	desktopApp := desktop.NewApp(cfg, migrations)

	err := wails.Run(&options.App{
		Title:     "TraceLog",
		Width:     1280,
		Height:    860,
		MinWidth:  960,
		MinHeight: 680,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: assetFallbackHandler(desktop.UploadHandler(filepath.Join(cfg.DataDir, "uploads")), assets),
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 251, A: 255},
		OnStartup:        desktopApp.Startup,
		OnShutdown:       desktopApp.Shutdown,
		ErrorFormatter:   desktopError,
		Bind: []interface{}{
			desktopApp,
		},
	})
	if err != nil {
		log.Fatalf("run desktop app: %v", err)
	}
}

func desktopError(err error) any {
	var appErr *service.AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return "internal server error"
}

func assetFallbackHandler(uploadHandler http.Handler, assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/uploads/") {
			uploadHandler.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet || strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		index, err := fs.ReadFile(assets, "frontend/dist/app/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
