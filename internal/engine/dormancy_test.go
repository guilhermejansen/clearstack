package engine

import (
	"context"
	"testing"
	"time"
)

func TestDormancyPolicy_DisabledAlwaysDormant(t *testing.T) {
	p := DormancyPolicy{}
	if !p.IsDormant(context.Background(), "/tmp", time.Now()) {
		t.Fatal("zero-value policy must always consider paths dormant")
	}
}

func TestDormancyPolicy_ThresholdBehaviour(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	p := DormancyPolicy{
		MinAge: 14 * 24 * time.Hour,
		Clock:  func() time.Time { return now },
	}
	old := now.Add(-30 * 24 * time.Hour)
	recent := now.Add(-2 * 24 * time.Hour)

	if !p.IsDormant(context.Background(), "/tmp", old) {
		t.Error("30-day-old entry should be dormant for a 14-day policy")
	}
	if p.IsDormant(context.Background(), "/tmp", recent) {
		t.Error("2-day-old entry should NOT be dormant for a 14-day policy")
	}
}
