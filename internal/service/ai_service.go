package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"ic-wails/internal/config"
	"ic-wails/internal/models"
	"ic-wails/internal/repository"
	"ic-wails/pkg/common"
	pkgmodels "ic-wails/pkg/models"
	"ic-wails/pkg/until"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewAiChatService 创建 AI 对话服务实例。
func NewAiChatService(
	configManager *config.ApplicationConfigManager,
	recordRepo *repository.AiChatRecordRepository,
	modelCacheRepo *repository.AiModelCacheRepository,
) *AiChatService {
	return &AiChatService{
		configManager:  configManager,
		recordRepo:     recordRepo,
		modelCacheRepo: modelCacheRepo,
	}
}

type AiChatService struct {
	configManager  *config.ApplicationConfigManager
	recordRepo     *repository.AiChatRecordRepository
	modelCacheRepo *repository.AiModelCacheRepository
}

const maxSingleContextMessages = 20

// getClient 根据当前配置创建 Ark 客户端，支持配置热更新后即时生效。
func (s *AiChatService) getClient() (*arkruntime.Client, string) {
	cfg := s.configManager.AppConfig.Ai
	if cfg.ApiKey == "" {
		panic(common.ServicePanic{
			Code: 400,
			Msg:  "AI API Key 未配置，请先在设置中填写",
		})
	}
	var opts []arkruntime.ConfigOption
	if cfg.BaseUrl != "" {
		opts = append(opts, arkruntime.WithBaseUrl(cfg.BaseUrl))
	}
	return arkruntime.NewClientWithApiKey(cfg.ApiKey, opts...), cfg.Model
}

func (s *AiChatService) latestUserMessage(messages []models.AiChatMessage) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		if messages[idx].Role == "user" {
			return messages[idx].Content
		}
	}
	return ""
}

func (s *AiChatService) safeGetModelStats(modelName string) *models.AiModelStats {
	if strings.TrimSpace(modelName) == "" || s.recordRepo == nil {
		return nil
	}
	stats, err := s.recordRepo.GetModelStats(modelName)
	if err != nil {
		until.Log.Errorf("读取模型统计失败: %v", err)
		return nil
	}
	return &stats
}

func (s *AiChatService) usageToDTO(usage model.Usage) models.AiUsage {
	return models.AiUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func (s *AiChatService) usagePtrToDTO(usage *model.Usage) *models.AiUsage {
	if usage == nil {
		return nil
	}
	result := s.usageToDTO(*usage)
	return &result
}

func (s *AiChatService) saveRecord(
	req models.AiChatRequest,
	modelName string,
	assistantContent string,
	reasoningContent *string,
	usage *models.AiUsage,
	firstTokenMs *int64,
	durationMs *int64,
	stream bool,
	errorMessage *string,
) {
	if s.recordRepo == nil {
		return
	}
	sessionId := strings.TrimSpace(req.SessionId)
	if sessionId == "" {
		sessionId = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	roundId := strings.TrimSpace(req.RoundId)
	if roundId == "" {
		roundId = sessionId
	}
	usageData := models.AiUsage{}
	if usage != nil {
		usageData = *usage
	}
	record := models.AiChatRecordModel{
		BaseModel:           &pkgmodels.BaseModel{},
		ChatMode:            s.normalizeChatMode(req.Mode),
		RoundId:             roundId,
		SessionId:           sessionId,
		Model:               modelName,
		UserContent:         s.latestUserMessage(req.Messages),
		AssistantContent:    assistantContent,
		ReasoningContent:    reasoningContent,
		PromptTokens:        usageData.PromptTokens,
		CompletionTokens:    usageData.CompletionTokens,
		TotalTokens:         usageData.TotalTokens,
		FirstTokenLatencyMs: firstTokenMs,
		DurationMs:          durationMs,
		Stream:              0,
		ErrorMessage:        errorMessage,
	}
	if stream {
		record.Stream = 1
	}
	if err := s.recordRepo.Create(&record); err != nil {
		until.Log.Errorf("保存 AI 对话记录失败: %v", err)
	}
}

func (s *AiChatService) normalizeChatMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "compare":
		return "compare"
	default:
		return "single"
	}
}

func appendModelOption(options []models.AiModelOption, seen map[string]struct{}, modelName string, source string, available bool, status string, hint string) []models.AiModelOption {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return options
	}
	if _, exists := seen[modelName]; exists {
		return options
	}
	seen[modelName] = struct{}{}
	return append(options, models.AiModelOption{
		Value:     modelName,
		Label:     modelName,
		Source:    source,
		Available: available,
		Status:    status,
		Hint:      hint,
	})
}

func (s *AiChatService) loadRecentModels(limit int) []string {
	if s.recordRepo == nil {
		return nil
	}
	recentModels, err := s.recordRepo.GetRecentModels(limit)
	if err != nil {
		until.Log.Errorf("读取最近模型失败: %v", err)
		return nil
	}
	return recentModels
}

func buildFallbackModelOptions(defaultModel string, recentModels []string, hint string) []models.AiModelOption {
	seen := map[string]struct{}{}
	options := make([]models.AiModelOption, 0, len(recentModels)+1)
	options = appendModelOption(options, seen, defaultModel, "default", false, "fallback", hint)
	for _, item := range recentModels {
		options = appendModelOption(options, seen, item, "recent", false, "fallback", hint)
	}
	return options
}

func appendModelOptionItem(options []models.AiModelOption, seen map[string]struct{}, item models.AiModelOption) []models.AiModelOption {
	normalized := strings.TrimSpace(item.Value)
	if normalized == "" {
		return options
	}
	if _, exists := seen[normalized]; exists {
		return options
	}
	item.Value = normalized
	if strings.TrimSpace(item.Label) == "" {
		item.Label = normalized
	}
	seen[normalized] = struct{}{}
	return append(options, item)
}

func cloneModelOptions(items []models.AiModelOption) []models.AiModelOption {
	result := make([]models.AiModelOption, len(items))
	copy(result, items)
	return result
}

func (s *AiChatService) getModelConfigFingerprint() string {
	cfg := s.configManager.AppConfig.Ai
	rawKey := strings.TrimSpace(cfg.BaseUrl) + "|" + strings.TrimSpace(cfg.ApiKey)
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func (s *AiChatService) listCachedModelOptions() ([]models.AiModelOption, error) {
	if s.modelCacheRepo == nil {
		return nil, nil
	}
	cacheList, err := s.modelCacheRepo.ListByConfigFingerprint(s.getModelConfigFingerprint())
	if err != nil {
		return nil, err
	}
	result := make([]models.AiModelOption, 0, len(cacheList))
	for _, item := range cacheList {
		result = append(result, models.AiModelOption{
			Value:     strings.TrimSpace(item.Model),
			Label:     strings.TrimSpace(item.Label),
			Source:    strings.TrimSpace(item.Source),
			Available: item.Available,
			Status:    strings.TrimSpace(item.Status),
			Hint:      strings.TrimSpace(item.Hint),
		})
	}
	return result, nil
}

func (s *AiChatService) replaceCachedModelOptions(list []models.AiModelOption) error {
	if s.modelCacheRepo == nil {
		return nil
	}
	now := time.Now()
	cacheList := make([]models.AiModelCacheModel, 0, len(list))
	for idx, item := range list {
		sortValue := idx
		cacheList = append(cacheList, models.AiModelCacheModel{
			BaseModel:         &pkgmodels.BaseModel{Sort: &sortValue},
			ConfigFingerprint: s.getModelConfigFingerprint(),
			Model:             strings.TrimSpace(item.Value),
			Label:             strings.TrimSpace(item.Label),
			Source:            strings.TrimSpace(item.Source),
			Available:         item.Available,
			Status:            strings.TrimSpace(item.Status),
			Hint:              strings.TrimSpace(item.Hint),
			RefreshedAt:       &now,
		})
	}
	return s.modelCacheRepo.ReplaceByConfigFingerprint(s.getModelConfigFingerprint(), cacheList)
}

func (s *AiChatService) normalizeModelOptions(list []models.AiModelOption, defaultModel string) []models.AiModelOption {
	seen := map[string]struct{}{}
	result := make([]models.AiModelOption, 0, len(list)+1)
	defaultValue := strings.TrimSpace(defaultModel)
	if defaultValue != "" {
		for _, item := range list {
			if strings.TrimSpace(item.Value) != defaultValue {
				continue
			}
			item.Source = "default"
			if item.Status == "" {
				item.Status = "available"
			}
			result = appendModelOptionItem(result, seen, item)
			break
		}
	}
	for _, item := range list {
		if strings.TrimSpace(item.Value) == defaultValue {
			item.Source = "default"
			if item.Status == "" {
				item.Status = "available"
			}
		}
		result = appendModelOptionItem(result, seen, item)
	}
	if defaultValue != "" {
		result = appendModelOptionItem(result, seen, models.AiModelOption{
			Value:     defaultValue,
			Label:     defaultValue,
			Source:    "default",
			Available: false,
			Status:    "unavailable",
			Hint:      "当前默认模型不在可用列表中",
		})
	}
	return result
}

func (s *AiChatService) buildRemoteModelOptions(remoteModels []string, defaultModel string) []models.AiModelOption {
	seen := map[string]struct{}{}
	options := make([]models.AiModelOption, 0, len(remoteModels)+1)
	remoteSet := make(map[string]struct{}, len(remoteModels))
	defaultValue := strings.TrimSpace(defaultModel)
	for _, item := range remoteModels {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		remoteSet[trimmed] = struct{}{}
	}

	if _, ok := remoteSet[defaultValue]; ok {
		options = appendModelOption(options, seen, defaultValue, "default", true, "available", "")
	}
	for _, item := range remoteModels {
		source := "remote"
		if strings.TrimSpace(item) == defaultValue {
			source = "default"
		}
		options = appendModelOption(options, seen, item, source, true, "available", "")
	}
	if defaultValue != "" {
		if _, ok := remoteSet[defaultValue]; !ok {
			options = appendModelOption(options, seen, defaultValue, "default", false, "unavailable", "当前默认模型不在远程可用列表中")
		}
	}
	return options
}

func (s *AiChatService) fetchRemoteModels(ctx context.Context) ([]string, error) {
	cfg := s.configManager.AppConfig.Ai
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseUrl), "/")
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
	request.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New("模型列表请求失败: " + response.Status)
	}

	type modelItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type modelListResp struct {
		Data []modelItem `json:"data"`
	}
	parsed := modelListResp{}
	if err = json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if strings.TrimSpace(item.ID) != "" {
			result = append(result, item.ID)
			continue
		}
		if strings.TrimSpace(item.Name) != "" {
			result = append(result, item.Name)
		}
	}
	sort.Strings(result)
	return result, nil
}

// ListModels 返回配置页与对话页可选模型列表。
func (s *AiChatService) ListModels(ctx context.Context, forceRefresh bool) models.AiModelListResponse {
	cfg := s.configManager.AppConfig.Ai
	recentModels := s.loadRecentModels(20)
	defaultModel := strings.TrimSpace(cfg.Model)

	cachedOptions, cacheErr := s.listCachedModelOptions()
	if cacheErr != nil {
		until.Log.Errorf("读取本地模型缓存失败: %v", cacheErr)
	}
	if !forceRefresh && len(cachedOptions) > 0 {
		return models.AiModelListResponse{
			Models:   s.normalizeModelOptions(cloneModelOptions(cachedOptions), defaultModel),
			Fallback: false,
			Message:  "已使用本地缓存模型列表",
		}
	}

	if strings.TrimSpace(cfg.ApiKey) == "" {
		if len(cachedOptions) > 0 {
			return models.AiModelListResponse{
				Models:   s.normalizeModelOptions(cloneModelOptions(cachedOptions), defaultModel),
				Fallback: false,
				Message:  "未配置 API Key，已使用本地缓存模型列表",
			}
		}
		return models.AiModelListResponse{
			Models:   buildFallbackModelOptions(cfg.Model, recentModels, "当前模型列表未经过远程校验"),
			Fallback: true,
			Message:  "未配置 API Key，当前展示默认模型和最近使用模型",
		}
	}

	remoteModels, err := s.fetchRemoteModels(ctx)
	if err != nil {
		until.Log.Errorf("获取远程模型列表失败: %v", err)
		if len(cachedOptions) > 0 {
			return models.AiModelListResponse{
				Models:   s.normalizeModelOptions(cloneModelOptions(cachedOptions), defaultModel),
				Fallback: false,
				Message:  "远程模型列表刷新失败，已回退到本地缓存",
			}
		}
		return models.AiModelListResponse{
			Models:   buildFallbackModelOptions(cfg.Model, recentModels, "远程模型列表获取失败，当前仅展示本地记录模型"),
			Fallback: true,
			Message:  "远程模型列表获取失败，已回退到本地可用模型",
		}
	}

	if len(remoteModels) == 0 {
		if len(cachedOptions) > 0 {
			return models.AiModelListResponse{
				Models:   s.normalizeModelOptions(cloneModelOptions(cachedOptions), defaultModel),
				Fallback: false,
				Message:  "远程模型列表为空，已回退到本地缓存",
			}
		}
		return models.AiModelListResponse{
			Models:   buildFallbackModelOptions(cfg.Model, recentModels, "远程模型列表为空，当前仅展示本地记录模型"),
			Fallback: true,
			Message:  "远程模型列表为空，已回退到本地可用模型",
		}
	}

	options := s.buildRemoteModelOptions(remoteModels, defaultModel)
	if err = s.replaceCachedModelOptions(options); err != nil {
		until.Log.Errorf("写入本地模型缓存失败: %v", err)
	}
	if forceRefresh {
		return models.AiModelListResponse{
			Models:   options,
			Fallback: false,
			Message:  "模型列表已刷新并写入本地缓存",
		}
	}

	return models.AiModelListResponse{
		Models:   options,
		Fallback: false,
	}
}

// buildMessages 将前端消息转换为火山引擎 SDK 所需的消息结构。
func (s *AiChatService) buildMessages(msgs []models.AiChatMessage) []*model.ChatCompletionMessage {
	trimmedMessages := s.trimContextMessages(msgs)
	result := make([]*model.ChatCompletionMessage, 0, len(trimmedMessages))
	for _, m := range trimmedMessages {
		result = append(result, &model.ChatCompletionMessage{
			Role:    m.Role,
			Content: &model.ChatCompletionMessageContent{StringValue: &m.Content},
		})
	}
	return result
}

func (s *AiChatService) trimContextMessages(msgs []models.AiChatMessage) []models.AiChatMessage {
	if len(msgs) <= maxSingleContextMessages+1 {
		return msgs
	}
	lastIdx := len(msgs) - 1
	if lastIdx < 0 {
		return msgs
	}
	previous := msgs[:lastIdx]
	if len(previous) > maxSingleContextMessages {
		previous = previous[len(previous)-maxSingleContextMessages:]
	}
	result := make([]models.AiChatMessage, 0, len(previous)+1)
	result = append(result, previous...)
	result = append(result, msgs[lastIdx])
	return result
}

func (s *AiChatService) ListChatSessions(limit int) []models.AiChatSessionSummary {
	if s.recordRepo == nil {
		return []models.AiChatSessionSummary{}
	}
	result, err := s.recordRepo.ListRecentSingleSessions(limit)
	if err != nil {
		until.Log.Errorf("读取会话列表失败: %v", err)
		panic(err)
	}
	return result
}

func (s *AiChatService) GetChatSessionDetail(sessionId string) models.AiChatSessionDetail {
	if s.recordRepo == nil {
		return models.AiChatSessionDetail{
			SessionId: sessionId,
			Messages:  []models.AiChatHistoryMessage{},
		}
	}
	records, err := s.recordRepo.ListSingleSessionRecords(sessionId)
	if err != nil {
		until.Log.Errorf("读取会话详情失败: %v", err)
		panic(err)
	}
	messages := make([]models.AiChatHistoryMessage, 0, len(records)*2)
	for _, record := range records {
		userContent := strings.TrimSpace(record.UserContent)
		if userContent != "" {
			messages = append(messages, models.AiChatHistoryMessage{
				Role:    "user",
				Content: userContent,
			})
		}

		assistantContent := strings.TrimSpace(record.AssistantContent)
		errorText := ""
		if record.ErrorMessage != nil {
			errorText = strings.TrimSpace(*record.ErrorMessage)
		}
		if assistantContent == "" && errorText != "" {
			assistantContent = "错误: " + errorText
		}
		if assistantContent == "" && errorText == "" {
			continue
		}

		var usage *models.AiUsage
		if record.PromptTokens > 0 || record.CompletionTokens > 0 || record.TotalTokens > 0 {
			usage = &models.AiUsage{
				PromptTokens:     record.PromptTokens,
				CompletionTokens: record.CompletionTokens,
				TotalTokens:      record.TotalTokens,
			}
		}

		messages = append(messages, models.AiChatHistoryMessage{
			Role:             "assistant",
			Content:          assistantContent,
			ReasoningContent: record.ReasoningContent,
			Model:            strings.TrimSpace(record.Model),
			Usage:            usage,
			DurationMs:       record.DurationMs,
			FirstTokenMs:     record.FirstTokenLatencyMs,
			Error:            errorText,
		})
	}

	return models.AiChatSessionDetail{
		SessionId: strings.TrimSpace(sessionId),
		Messages:  messages,
	}
}

// Chat 发起一次非流式对话，并在模型完成生成后返回完整结果。
func (s *AiChatService) Chat(ctx context.Context, req models.AiChatRequest) models.AiChatResponse {
	client, defaultModel := s.getClient()
	modelName := req.Model
	if modelName == "" {
		modelName = defaultModel
	}
	startTime := time.Now()

	chatReq := model.ChatCompletionRequest{
		Model:    modelName,
		Messages: s.buildMessages(req.Messages),
	}
	if req.Temperature != nil {
		chatReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	}

	resp, err := client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		until.Log.Errorf("AI 对话失败: %v", err)
		panic(err)
	}

	var content string
	var reasoning *string
	if len(resp.Choices) > 0 {
		if resp.Choices[0].Message.Content != nil && resp.Choices[0].Message.Content.StringValue != nil {
			content = *resp.Choices[0].Message.Content.StringValue
		}
		reasoning = resp.Choices[0].Message.ReasoningContent
	}
	durationMs := time.Since(startTime).Milliseconds()
	usage := s.usageToDTO(resp.Usage)
	s.saveRecord(req, modelName, content, reasoning, &usage, nil, &durationMs, false, nil)
	stats := s.safeGetModelStats(modelName)

	return models.AiChatResponse{
		Content:          content,
		ReasoningContent: reasoning,
		Model:            resp.Model,
		Usage:            usage,
		DurationMs:       durationMs,
		ModelStats:       stats,
	}
}

// ChatStream 发起流式对话，并通过 Wails 事件持续向前端推送增量内容。
func (s *AiChatService) ChatStream(ctx context.Context, req models.AiChatRequest) {
	_ = s.StreamChat(ctx, req, func(chunk models.AiChatStreamChunk) error {
		s.emitChunk(req.SessionId, chunk)
		return nil
	})
}

// StreamChat 发起流式对话，并将增量消息块交给调用方消费。
func (s *AiChatService) StreamChat(ctx context.Context, req models.AiChatRequest, emit func(models.AiChatStreamChunk) error) error {
	client, defaultModel := s.getClient()
	modelName := req.Model
	if modelName == "" {
		modelName = defaultModel
	}
	sessionId := req.SessionId
	startTime := time.Now()
	var firstTokenMs *int64
	fullContent := ""
	fullReasoning := ""
	var usage *models.AiUsage

	chatReq := model.ChatCompletionRequest{
		Model:         modelName,
		Messages:      s.buildMessages(req.Messages),
		StreamOptions: &model.StreamOptions{IncludeUsage: true},
	}
	if req.Temperature != nil {
		chatReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	}

	stream, err := client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		until.Log.Errorf("AI 流式对话创建失败: %v", err)
		errText := err.Error()
		durationMs := time.Since(startTime).Milliseconds()
		s.saveRecord(req, modelName, "", nil, nil, nil, &durationMs, true, &errText)
		stats := s.safeGetModelStats(modelName)
		return emit(models.AiChatStreamChunk{
			SessionId:  sessionId,
			RoundId:    req.RoundId,
			Model:      modelName,
			DurationMs: &durationMs,
			ModelStats: stats,
			Done:       true,
			Error:      err.Error(),
		})
	}
	defer stream.Close()

	for {
		recv, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			durationMs := time.Since(startTime).Milliseconds()
			var reasoning *string
			if fullReasoning != "" {
				reasoning = &fullReasoning
			}
			s.saveRecord(req, modelName, fullContent, reasoning, usage, firstTokenMs, &durationMs, true, nil)
			stats := s.safeGetModelStats(modelName)
			return emit(models.AiChatStreamChunk{
				SessionId:    sessionId,
				RoundId:      req.RoundId,
				Model:        modelName,
				Usage:        usage,
				DurationMs:   &durationMs,
				FirstTokenMs: firstTokenMs,
				ModelStats:   stats,
				Done:         true,
			})
		}
		if recvErr != nil {
			until.Log.Errorf("AI 流式读取失败: %v", recvErr)
			errText := recvErr.Error()
			durationMs := time.Since(startTime).Milliseconds()
			var reasoning *string
			if fullReasoning != "" {
				reasoning = &fullReasoning
			}
			s.saveRecord(req, modelName, fullContent, reasoning, usage, firstTokenMs, &durationMs, true, &errText)
			stats := s.safeGetModelStats(modelName)
			return emit(models.AiChatStreamChunk{
				SessionId:    sessionId,
				RoundId:      req.RoundId,
				Model:        modelName,
				Usage:        usage,
				DurationMs:   &durationMs,
				FirstTokenMs: firstTokenMs,
				ModelStats:   stats,
				Done:         true,
				Error:        recvErr.Error(),
			})
		}
		if recv.Model != "" {
			modelName = recv.Model
		}
		if recv.Usage != nil {
			usage = s.usagePtrToDTO(recv.Usage)
		}
		if len(recv.Choices) > 0 {
			delta := recv.Choices[0].Delta
			if firstTokenMs == nil && (delta.Content != "" || (delta.ReasoningContent != nil && *delta.ReasoningContent != "")) {
				current := time.Since(startTime).Milliseconds()
				firstTokenMs = &current
			}
			if delta.Content != "" {
				fullContent += delta.Content
			}
			if delta.ReasoningContent != nil {
				fullReasoning += *delta.ReasoningContent
			}
			chunk := models.AiChatStreamChunk{
				SessionId:        sessionId,
				RoundId:          req.RoundId,
				Model:            modelName,
				Content:          delta.Content,
				ReasoningContent: delta.ReasoningContent,
			}
			if err = emit(chunk); err != nil {
				return err
			}
		}
	}
}

// emitChunk 将单个流式消息块投递给主窗口，供前端实时消费。
func (s *AiChatService) emitChunk(sessionId string, chunk models.AiChatStreamChunk) {
	app := application.Get()
	if app == nil {
		return
	}
	w, ok := app.Window.Get("Main window")
	if ok {
		w.EmitEvent("ai:chat:stream:"+sessionId, chunk)
	}
}
