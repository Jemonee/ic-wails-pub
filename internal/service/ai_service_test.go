package service

import (
	"testing"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"ic-wails/internal/models"
)

func TestLatestUserMessageReturnsMostRecentUserMessage(t *testing.T) {
	svc := &AiChatService{}
	messages := []models.AiChatMessage{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "latest question"},
	}

	got := svc.latestUserMessage(messages)
	if got != "latest question" {
		t.Fatalf("expected latest user message, got %q", got)
	}
}

func TestLatestUserMessageReturnsEmptyWhenMissingUser(t *testing.T) {
	svc := &AiChatService{}
	messages := []models.AiChatMessage{
		{Role: "system", Content: "system prompt"},
		{Role: "assistant", Content: "answer only"},
	}

	got := svc.latestUserMessage(messages)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNormalizeChatMode(t *testing.T) {
	svc := &AiChatService{}

	if got := svc.normalizeChatMode("compare"); got != "compare" {
		t.Fatalf("expected compare, got %q", got)
	}
	if got := svc.normalizeChatMode(" single "); got != "single" {
		t.Fatalf("expected single fallback, got %q", got)
	}
	if got := svc.normalizeChatMode(""); got != "single" {
		t.Fatalf("expected empty mode to normalize to single, got %q", got)
	}
}

func TestUsageConversions(t *testing.T) {
	svc := &AiChatService{}
	usage := arkmodel.Usage{PromptTokens: 11, CompletionTokens: 22, TotalTokens: 33}

	converted := svc.usageToDTO(usage)
	if converted.PromptTokens != 11 || converted.CompletionTokens != 22 || converted.TotalTokens != 33 {
		t.Fatalf("unexpected usage conversion result: %+v", converted)
	}

	ptrConverted := svc.usagePtrToDTO(&usage)
	if ptrConverted == nil {
		t.Fatal("expected non-nil usage pointer conversion")
	}
	if *ptrConverted != converted {
		t.Fatalf("expected pointer conversion to match value conversion, got %+v", *ptrConverted)
	}

	if svc.usagePtrToDTO(nil) != nil {
		t.Fatal("expected nil input to return nil")
	}
}

func TestBuildFallbackModelOptionsDeduplicatesAndPreservesHint(t *testing.T) {
	hint := "fallback to cached models"
	options := buildFallbackModelOptions("gpt-4", []string{"gpt-4", "deepseek-r1", "deepseek-r1", "glm-4"}, hint)

	if len(options) != 3 {
		t.Fatalf("expected 3 deduplicated options, got %d", len(options))
	}

	if options[0].Value != "gpt-4" || options[0].Source != "default" || options[0].Status != "fallback" || options[0].Hint != hint {
		t.Fatalf("unexpected default option: %+v", options[0])
	}

	if options[1].Value != "deepseek-r1" || options[1].Source != "recent" {
		t.Fatalf("unexpected recent option: %+v", options[1])
	}

	if options[2].Value != "glm-4" || options[2].Source != "recent" {
		t.Fatalf("unexpected last option: %+v", options[2])
	}
	for _, item := range options {
		if item.Available {
			t.Fatalf("expected fallback options to be unavailable by default, got %+v", item)
		}
	}
}
