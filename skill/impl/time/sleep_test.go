package time

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSleep_Do_Success(t *testing.T) {
	s := &Sleep{}
	err := s.Do(context.Background(), map[string]any{"duration": "1ms"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSleep_Do_ContextCanceled(t *testing.T) {
	s := &Sleep{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.Do(ctx, map[string]any{"duration": "10ms"}, nil)
	if err == nil {
		t.Fatalf("expected context canceled error")
	}
}

func TestSleep_Do_InvalidDuration(t *testing.T) {
	s := &Sleep{}
	if err := s.Do(context.Background(), map[string]any{"duration": "not-a-duration"}, nil); err == nil {
		t.Fatalf("expected invalid duration error")
	}
}

func TestSleep_Do_InvalidParams(t *testing.T) {
	s := &Sleep{}
	if err := s.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

func TestSleep_Do_MissingDuration(t *testing.T) {
	s := &Sleep{}
	if err := s.Do(context.Background(), map[string]any{}, nil); err == nil {
		t.Fatalf("expected missing duration error")
	}
}

func TestSleep_Do_DurationTypeError(t *testing.T) {
	s := &Sleep{}
	if err := s.Do(context.Background(), map[string]any{"duration": 1}, nil); err == nil {
		t.Fatalf("expected duration type error")
	}
}

func TestSleep_Do_StopsAroundDuration(t *testing.T) {
	s := &Sleep{}
	start := time.Now()
	if err := s.Do(context.Background(), map[string]any{"duration": "2ms"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) < time.Millisecond {
		t.Fatalf("sleep returned too quickly")
	}
}

func TestSleep_Descriptions(t *testing.T) {
	s := &Sleep{}
	desc, err := s.GetDescription()
	if err != nil || desc == "" || s.ShortDescription() == "" {
		t.Fatalf("descriptions should not be empty")
	}
	if !strings.Contains(desc, "duration") {
		t.Fatalf("expected duration mention in description")
	}
}
