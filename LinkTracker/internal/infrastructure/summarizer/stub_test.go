package summarizer

import (
	"context"
	"testing"
)

func TestSummarizer_Summarize_LongText_ReturnsShortenedText(t *testing.T) {
	//arrange
	summarizer := NewStubSummarizer(10)
	text := "abcdefghijklmnopqrstuvwxyz"

	//act
	result, err := summarizer.Summarize(context.Background(), text)

	//assert
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	expected := string([]rune(text)[:10]) + "..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSummarizer_Summarize_ShortText_ReturnsOriginalText(t *testing.T) {
	//arrange
	summarizer := NewStubSummarizer(100)
	text := "short text"

	//act
	result, err := summarizer.Summarize(context.Background(), text)

	//assert
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}
