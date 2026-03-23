package core

import (
	"ic-wails/internal/api"
	"ic-wails/internal/config"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ApplicationHolder struct {
	ServerMux      *http.ServeMux // 资源多路复用器
	FrontendApi    *api.FrontendApi
	HandlerRequest *api.FrontendNetApi
	ConfigManager  *config.ApplicationConfigManager
}

func NewApplicationHolder(handlerRequest *api.FrontendNetApi, configManager *config.ApplicationConfigManager, frontendApi *api.FrontendApi) *ApplicationHolder {
	// 绑定一下自定义后台接口地址
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", handlerRequest.HandlerRequest)
	return &ApplicationHolder{
		HandlerRequest: handlerRequest,
		ConfigManager:  configManager,
		FrontendApi:    frontendApi,
		ServerMux:      mux,
	}
}

// GetApplicationServices 获取暴露给前端的业务接口
func (ah *ApplicationHolder) GetApplicationServices() []application.Service {
	return ah.FrontendApi.GetApplicationServices()
}

// AddAssets 注册前端静态资源与 SPA 路由回退处理器。
func (ah *ApplicationHolder) AddAssets(assets fs.FS) {
	assetsHandler := application.AssetFileServerFS(assets)
	devServerURL := os.Getenv("FRONTEND_DEVSERVER_URL")
	ah.ServerMux.Handle("/static/", http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if devServerURL != "" {
			assetsHandler.ServeHTTP(response, request)
			return
		}

		resourcePath := strings.TrimPrefix(request.URL.Path, "/static")
		if resourcePath == "" {
			resourcePath = "/index.html"
		}

		spaRequest := request.Clone(request.Context())
		spaRequest.URL.Path = resourcePath
		if shouldServeIndex(resourcePath) {
			spaRequest.URL.Path = "/index.html"
		}

		assetsHandler.ServeHTTP(response, spaRequest)
	}))
	ah.ServerMux.Handle("/", http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			http.Redirect(response, request, "/static/", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(response, request)
	}))
}

// shouldServeIndex 判断当前 /static 下的请求是否应回退到前端入口页。
func shouldServeIndex(resourcePath string) bool {
	if resourcePath == "/" || resourcePath == "/index.html" {
		return true
	}
	if strings.HasPrefix(resourcePath, "/assets/") {
		return false
	}
	return path.Ext(resourcePath) == ""
}
