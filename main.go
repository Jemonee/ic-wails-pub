package main

import (
	"context"
	"embed"
	"ic-wails/cmd"
	"ic-wails/internal/instancechecker"
	"log"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	applicationHolder := cmd.InitializeApp()
	instanceChecker := instancechecker.GetInstanceChecker()

	config := applicationHolder.ConfigManager.AppConfig
	// 将内置前端资源配置到前端资源处理器上
	applicationHolder.AddAssets(assets)

	if instanceChecker.IsAlreadyRunning() {
		return
	}

	defer instanceChecker.Close()
	app := application.New(application.Options{
		Name:        "ic-wails",
		Description: "shadow",
		Services:    applicationHolder.GetApplicationServices(),
		Assets: application.AssetOptions{
			Handler: applicationHolder.ServerMux,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// 获取窗口ID
					clientId := r.Header.Get("x-wails-client-id")
					println("获取到的客户端 ID：", clientId)
					if clientId != "" {
						// 将窗口ID存入上下文
						ctx := context.WithValue(r.Context(), "clientId", clientId)
						r = r.WithContext(ctx)
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	mainWindows := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "ic",
		Name:      "Main window",
		Frameless: false,
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHidden,
			InvisibleTitleBarHeight: 44,
		},
		// BackgroundColour: application.NewRGB(27, 38, 54),
		Width:     config.Window.GetWidth(),
		Height:    config.Window.GetHeight(),
		MinWidth:  800,
		MinHeight: 600,
		URL:       "/static/",
	})

	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			mainWindows.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
