package repository

import (
	"ic-wails/internal/models"
	"ic-wails/pkg/core/tx"
	pkgrepo "ic-wails/pkg/repository"
	"strings"
	"time"
)

func NewAiChatRecordRepository(ds *tx.DataSource) *AiChatRecordRepository {
	instance := &AiChatRecordRepository{
		BaseRepository: pkgrepo.NewBaseRepository[models.AiChatRecordModel](ds),
	}
	instance.InitializeRepository()
	return instance
}

type AiChatRecordRepository struct {
	*pkgrepo.BaseRepository[models.AiChatRecordModel]
}

type sessionAggRow struct {
	SessionId        string `json:"sessionId"`
	LastActiveTime   string `json:"lastActiveTime"`
	RoundCount       int64  `json:"roundCount"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	TotalTokens      int64  `json:"totalTokens"`
	AvgDurationMs    int64  `json:"avgDurationMs"`
	AvgFirstTokenMs  int64  `json:"avgFirstTokenMs"`
}

func (repo *AiChatRecordRepository) GetRecentModels(limit int) ([]string, error) {
	type modelRow struct {
		Model string `json:"model"`
	}
	rows := make([]modelRow, 0)
	err := repo.Db().
		Model(repo.GetEntity()).
		Select("model").
		Where("valid = ? AND model <> ''", 1).
		Group("model").
		Order("MAX(create_time) DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	modelsList := make([]string, 0, len(rows))
	for _, row := range rows {
		modelsList = append(modelsList, row.Model)
	}
	return modelsList, nil
}

func (repo *AiChatRecordRepository) GetModelStats(modelName string) (models.AiModelStats, error) {
	type statsRow struct {
		TotalCalls      int64   `json:"totalCalls"`
		AvgTokens       float64 `json:"avgTokens"`
		AvgFirstTokenMs float64 `json:"avgFirstTokenMs"`
	}
	row := statsRow{}
	err := repo.Db().
		Model(repo.GetEntity()).
		Select("COUNT(1) AS total_calls, COALESCE(AVG(total_tokens), 0) AS avg_tokens, COALESCE(AVG(first_token_latency_ms), 0) AS avg_first_token_ms").
		Where("valid = ? AND model = ? AND (error_message IS NULL OR error_message = '')", 1, modelName).
		Scan(&row).Error
	if err != nil {
		return models.AiModelStats{}, err
	}
	return models.AiModelStats{
		Model:           modelName,
		TotalCalls:      row.TotalCalls,
		AvgTokens:       row.AvgTokens,
		AvgFirstTokenMs: row.AvgFirstTokenMs,
	}, nil
}

func (repo *AiChatRecordRepository) ListRecentSingleSessions(limit int) ([]models.AiChatSessionSummary, error) {
	if limit <= 0 {
		limit = 5
	}

	rows := make([]sessionAggRow, 0)
	err := repo.Db().
		Model(repo.GetEntity()).
		Select("session_id, MAX(create_time) AS last_active_time, COUNT(1) AS round_count, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens, COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(CAST(AVG(duration_ms) AS INTEGER), 0) AS avg_duration_ms, COALESCE(CAST(AVG(first_token_latency_ms) AS INTEGER), 0) AS avg_first_token_ms").
		Where("valid = ? AND session_id <> '' AND COALESCE(chat_mode, 'single') = ?", 1, "single").
		Group("session_id").
		Order("MAX(create_time) DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]models.AiChatSessionSummary, 0, len(rows))
	for _, row := range rows {
		firstRecord, firstErr := repo.getSessionBoundaryRecord(row.SessionId, true)
		if firstErr != nil {
			return nil, firstErr
		}
		lastRecord, lastErr := repo.getSessionBoundaryRecord(row.SessionId, false)
		if lastErr != nil {
			return nil, lastErr
		}

		title := buildSessionTitle(firstRecord.UserContent, lastRecord.UserContent)
		preview := buildSessionPreview(lastRecord.AssistantContent, lastRecord.ErrorMessage, lastRecord.UserContent)
		result = append(result, models.AiChatSessionSummary{
			SessionId:        row.SessionId,
			Title:            title,
			Preview:          preview,
			Model:            strings.TrimSpace(lastRecord.Model),
			RoundCount:       row.RoundCount,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			TotalTokens:      row.TotalTokens,
			AvgDurationMs:    row.AvgDurationMs,
			AvgFirstTokenMs:  row.AvgFirstTokenMs,
			LastActiveTime:   formatRecordTime(lastRecord.CreateTime, row.LastActiveTime),
		})
	}

	return result, nil
}

func (repo *AiChatRecordRepository) ListSingleSessionRecords(sessionId string) ([]models.AiChatRecordModel, error) {
	list := make([]models.AiChatRecordModel, 0)
	err := repo.Db().
		Model(repo.GetEntity()).
		Where("valid = ? AND session_id = ? AND COALESCE(chat_mode, 'single') = ?", 1, strings.TrimSpace(sessionId), "single").
		Order("create_time ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (repo *AiChatRecordRepository) getSessionBoundaryRecord(sessionId string, asc bool) (models.AiChatRecordModel, error) {
	order := "create_time DESC, id DESC"
	if asc {
		order = "create_time ASC, id ASC"
	}
	var record models.AiChatRecordModel
	err := repo.Db().
		Model(repo.GetEntity()).
		Where("valid = ? AND session_id = ? AND COALESCE(chat_mode, 'single') = ?", 1, strings.TrimSpace(sessionId), "single").
		Order(order).
		First(&record).Error
	return record, err
}

func buildSessionTitle(firstUserContent string, lastUserContent string) string {
	title := normalizeSessionText(firstUserContent)
	if title == "" {
		title = normalizeSessionText(lastUserContent)
	}
	if title == "" {
		return "未命名会话"
	}
	return title
}

func buildSessionPreview(assistantContent string, errorMessage *string, fallbackUserContent string) string {
	preview := normalizeSessionText(assistantContent)
	if preview != "" {
		return preview
	}
	if errorMessage != nil {
		if normalized := normalizeSessionText(*errorMessage); normalized != "" {
			return "错误: " + normalized
		}
	}
	return normalizeSessionText(fallbackUserContent)
}

func normalizeSessionText(content string) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if len([]rune(trimmed)) <= 36 {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:36]) + "..."
}

func formatRecordTime(recordTime *time.Time, fallback string) string {
	if recordTime != nil && !recordTime.IsZero() {
		return recordTime.Format(time.RFC3339)
	}

	trimmed := strings.TrimSpace(fallback)
	if trimmed == "" {
		return ""
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.Format(time.RFC3339)
		}
	}

	return trimmed
}
