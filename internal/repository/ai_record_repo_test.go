package repository

import (
	"testing"
	"time"
)

func TestBuildSessionTitlePrefersFirstUserThenFallback(t *testing.T) {
	if got := buildSessionTitle("first question", "last question"); got != "first question" {
		t.Fatalf("expected first user content as title, got %q", got)
	}

	if got := buildSessionTitle("", "last question"); got != "last question" {
		t.Fatalf("expected fallback user content as title, got %q", got)
	}

	if got := buildSessionTitle("", ""); got != "未命名会话" {
		t.Fatalf("expected unnamed session fallback, got %q", got)
	}
}

func TestBuildSessionPreviewUsesAssistantThenErrorThenFallback(t *testing.T) {
	errMsg := "request timeout"

	if got := buildSessionPreview("assistant preview", &errMsg, "user fallback"); got != "assistant preview" {
		t.Fatalf("expected assistant content preview, got %q", got)
	}

	if got := buildSessionPreview("", &errMsg, "user fallback"); got != "错误: request timeout" {
		t.Fatalf("expected error preview, got %q", got)
	}

	if got := buildSessionPreview("", nil, "user fallback"); got != "user fallback" {
		t.Fatalf("expected fallback user preview, got %q", got)
	}
}

func TestNormalizeSessionTextTrimsWhitespaceAndEllipsizes(t *testing.T) {
	if got := normalizeSessionText("  hello   world  "); got != "hello world" {
		t.Fatalf("expected normalized whitespace, got %q", got)
	}

	longText := "abcdefghijklmnopqrstuvwxyz1234567890-extra"
	got := normalizeSessionText(longText)
	if got != "abcdefghijklmnopqrstuvwxyz1234567890..." {
		t.Fatalf("expected ellipsized content, got %q", got)
	}
}

func TestFormatRecordTimePrefersRecordTimeAndFallsBackToParsedString(t *testing.T) {
	recordTime := time.Date(2026, 3, 23, 17, 30, 0, 0, time.UTC)
	if got := formatRecordTime(&recordTime, ""); got != "2026-03-23T17:30:00Z" {
		t.Fatalf("expected RFC3339 formatted record time, got %q", got)
	}

	if got := formatRecordTime(nil, "2026-03-23 17:30:00"); got != "2026-03-23T17:30:00Z" {
		t.Fatalf("expected parsed fallback time, got %q", got)
	}

	if got := formatRecordTime(nil, " raw-time "); got != "raw-time" {
		t.Fatalf("expected trimmed raw fallback, got %q", got)
	}
}
