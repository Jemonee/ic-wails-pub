package api

import (
	"encoding/json"
	"fmt"
	"ic-wails/internal/config"
	"ic-wails/internal/models"
	pkgapi "ic-wails/pkg/api"
	"ic-wails/pkg/common"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewFrontendApi 创建前端统一服务聚合入口。
func NewFrontendApi(taskCategoryApi *TaskCategoryApi,
	windowManagerApi *WindowManagerApi,
	localResourceApi *LocalResourceApi,
	aiChatApi *AiChatApi,
	configManager *config.ApplicationConfigManager) *FrontendApi {
	return &FrontendApi{
		TaskCategoryApi:          taskCategoryApi,
		WindowManagerApi:         windowManagerApi,
		LocalResourceApi:         localResourceApi,
		AiChatApi:                aiChatApi,
		ApplicationConfigManager: configManager,
	}
}

// FrontendApi 暴露给前端接口
type FrontendApi struct {
	TaskCategoryApi          *TaskCategoryApi
	WindowManagerApi         *WindowManagerApi
	LocalResourceApi         *LocalResourceApi
	AiChatApi                *AiChatApi
	ApplicationConfigManager *config.ApplicationConfigManager
}

// GetApplicationServices 获取当前app暴露给前端的接口，全局的接口统一在此处管理
func (fa *FrontendApi) GetApplicationServices() []application.Service {
	return []application.Service{
		application.NewServiceWithOptions(fa.TaskCategoryApi, application.ServiceOptions{
			Name: "TaskCategory",
		}),
		application.NewService(fa.WindowManagerApi),
		application.NewService(fa.LocalResourceApi),
		application.NewService(fa.AiChatApi),
		application.NewService(fa.ApplicationConfigManager),
	}
}

///  自定义网络接口

// NewFrontendNetApi 创建自定义 HTTP 接口处理器。
func NewFrontendNetApi(aiChatApi *AiChatApi) *FrontendNetApi {
	// 创建Gin引擎（禁用控制台颜色，适用于服务端）
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery()) // 添加恢复中间件

	res := &FrontendNetApi{
		NetRequestHandler: router,
		AiChatApi:         aiChatApi,
		BaseApi:           &pkgapi.BaseApi{},
	}
	res.registerRoutes()
	return res
}

// HandlerRequest 将外部 HTTP 请求转发给 Gin 路由处理。
func (fna *FrontendNetApi) HandlerRequest(response http.ResponseWriter, request *http.Request) {
	fna.NetRequestHandler.ServeHTTP(response, request)
}

// registerRoutes 注册自定义网络接口路由。
func (fna *FrontendNetApi) registerRoutes() {
	group := fna.NetRequestHandler.Group("/api")
	group.GET("/ai/models", fna.listModels)
	group.GET("/ai/chat-sessions", fna.listChatSessions)
	group.GET("/ai/chat-sessions/:sessionId", fna.getChatSessionDetail)
	group.GET("/ai/sessions", fna.listChatSessions)
	group.GET("/ai/sessions/:sessionId", fna.getChatSessionDetail)
	group.POST("/ai/chat", fna.chat)
	group.POST("/ai/chat/stream", fna.chatStream)
}

func (fna *FrontendNetApi) listModels(c *gin.Context) {
	defer fna.BaseApi.DeferPanicHandler(c)
	refresh := strings.TrimSpace(c.Query("refresh"))
	forceRefresh := refresh == "1" || strings.EqualFold(refresh, "true")
	resp := fna.AiChatApi.AiChatService.ListModels(c.Request.Context(), forceRefresh)
	c.JSON(http.StatusOK, common.S(&resp))
}

func (fna *FrontendNetApi) listChatSessions(c *gin.Context) {
	defer fna.BaseApi.DeferPanicHandler(c)
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	resp := fna.AiChatApi.AiChatService.ListChatSessions(limit)
	c.JSON(http.StatusOK, common.S(&resp))
}

func (fna *FrontendNetApi) getChatSessionDetail(c *gin.Context) {
	defer fna.BaseApi.DeferPanicHandler(c)
	sessionId := strings.TrimSpace(c.Param("sessionId"))
	if sessionId == "" {
		panic(common.ServicePanic{Code: 400, Msg: "会话标识不能为空"})
	}
	resp := fna.AiChatApi.AiChatService.GetChatSessionDetail(sessionId)
	c.JSON(http.StatusOK, common.S(&resp))
}

func (fna *FrontendNetApi) chat(c *gin.Context) {
	defer fna.BaseApi.DeferPanicHandler(c)

	var req models.AiChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		panic(common.ServicePanic{Code: 400, Msg: "请求参数解析失败: " + err.Error()})
	}

	fna.AiChatApi.validateRequest(req)
	resp := fna.AiChatApi.AiChatService.Chat(c.Request.Context(), req)
	c.JSON(http.StatusOK, common.S(&resp))
}

func (fna *FrontendNetApi) chatStream(c *gin.Context) {
	var req models.AiChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fna.writeStreamError(c, models.AiChatStreamChunk{Done: true, Error: "请求参数解析失败: " + err.Error()})
		return
	}
	if req.SessionId == "" {
		req.SessionId = strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	writer, flusher, ok := fna.prepareStreamResponse(c)
	if !ok {
		c.JSON(http.StatusOK, common.F[any](500, "当前环境不支持流式响应"))
		return
	}

	defer func() {
		if r := recover(); r != nil {
			_ = writeStreamChunk(writer, flusher, models.AiChatStreamChunk{
				SessionId: req.SessionId,
				RoundId:   req.RoundId,
				Model:     req.Model,
				Done:      true,
				Error:     panicToMessage(r),
			})
		}
	}()

	fna.AiChatApi.validateRequest(req)
	_ = fna.AiChatApi.AiChatService.StreamChat(c.Request.Context(), req, func(chunk models.AiChatStreamChunk) error {
		return writeStreamChunk(writer, flusher, chunk)
	})
}

func (fna *FrontendNetApi) prepareStreamResponse(c *gin.Context) (http.ResponseWriter, http.Flusher, bool) {
	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, nil, false
	}
	writer.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	return writer, flusher, true
}

func (fna *FrontendNetApi) writeStreamError(c *gin.Context, chunk models.AiChatStreamChunk) {
	writer, flusher, ok := fna.prepareStreamResponse(c)
	if !ok {
		c.JSON(http.StatusOK, common.F[any](500, "当前环境不支持流式响应"))
		return
	}
	_ = writeStreamChunk(writer, flusher, chunk)
}

func writeStreamChunk(writer http.ResponseWriter, flusher http.Flusher, chunk models.AiChatStreamChunk) error {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err = writer.Write(payload); err != nil {
		return err
	}
	if _, err = writer.Write([]byte("\n")); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func panicToMessage(r any) string {
	switch err := r.(type) {
	case common.ServicePanic:
		return err.Msg
	case error:
		return "操作产生异常错误：" + err.Error()
	default:
		return "发生未知异常：" + fmt.Sprintf("%v", r)
	}
}

type FrontendNetApi struct {
	NetRequestHandler *gin.Engine
	AiChatApi         *AiChatApi
	BaseApi           *pkgapi.BaseApi
}
