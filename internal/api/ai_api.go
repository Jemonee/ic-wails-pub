package api

import (
	"context"
	"ic-wails/internal/models"
	"ic-wails/internal/service"
	pkgapi "ic-wails/pkg/api"
	"ic-wails/pkg/common"
	"ic-wails/pkg/until"
	"strconv"
	"strings"
	"time"
)

// NewAiChatApi 创建 AI 对话 API。
func NewAiChatApi(aiChatService *service.AiChatService) *AiChatApi {
	return &AiChatApi{
		AiChatService: aiChatService,
	}
}

type AiChatApi struct {
	AiChatService *service.AiChatService
}

// ListModels 返回配置页与聊天页可选模型列表。
func (a *AiChatApi) ListModels(ctx context.Context) (result common.R[models.AiModelListResponse]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	resp := a.AiChatService.ListModels(ctx, false)
	return common.S(&resp)
}

// ListChatSessions 返回最近活跃的历史会话。
func (a *AiChatApi) ListChatSessions(ctx context.Context, limit int) (result common.R[[]models.AiChatSessionSummary]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	resp := a.AiChatService.ListChatSessions(limit)
	return common.S(&resp)
}

// GetChatSessionDetail 返回指定会话的历史消息。
func (a *AiChatApi) GetChatSessionDetail(ctx context.Context, sessionId string) (result common.R[models.AiChatSessionDetail]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	if strings.TrimSpace(sessionId) == "" {
		panic(common.ServicePanic{Code: 400, Msg: "会话标识不能为空"})
	}
	resp := a.AiChatService.GetChatSessionDetail(sessionId)
	return common.S(&resp)
}

// Chat 执行非流式对话，并在返回前完成基础参数校验。
func (a *AiChatApi) Chat(ctx context.Context, req models.AiChatRequest) (result common.R[models.AiChatResponse]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	a.validateRequest(req)
	resp := a.AiChatService.Chat(ctx, req)
	return common.S(&resp)
}

// ChatStream 发起流式对话，返回 sessionId 给前端订阅事件流。
func (a *AiChatApi) ChatStream(ctx context.Context, req models.AiChatRequest) (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	a.validateRequest(req)
	if req.SessionId == "" {
		req.SessionId = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	sessionId := req.SessionId
	go func() {
		defer func() {
			if r := recover(); r != nil {
				until.Log.Errorf("AI 流式对话 panic: %v", r)
			}
		}()
		a.AiChatService.ChatStream(ctx, req)
	}()
	return common.S(&sessionId)
}

// validateRequest 校验前端传入的对话参数，避免无效请求进入 Service 层。
func (a *AiChatApi) validateRequest(req models.AiChatRequest) {
	if len(req.Messages) == 0 {
		panic(common.ServicePanic{
			Code: 400,
			Msg:  "消息列表不能为空",
		})
	}

	for _, message := range req.Messages {
		if strings.TrimSpace(message.Role) == "" {
			panic(common.ServicePanic{
				Code: 400,
				Msg:  "消息角色不能为空",
			})
		}
		if strings.TrimSpace(message.Content) == "" {
			panic(common.ServicePanic{
				Code: 400,
				Msg:  "消息内容不能为空",
			})
		}
	}
}
