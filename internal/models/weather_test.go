package models

import "testing"

func TestDefaultWeatherConfig(t *testing.T) {
	cfg := DefaultWeatherConfig()
	if cfg.ColdTemp != 5 {
		t.Errorf("ColdTemp = %v, want 5", cfg.ColdTemp)
	}
	if cfg.HotTemp != 32 {
		t.Errorf("HotTemp = %v, want 32", cfg.HotTemp)
	}
	if cfg.WaterAdj != 30 {
		t.Errorf("WaterAdj = %v, want 30", cfg.WaterAdj)
	}
	if cfg.FertAdj != 30 {
		t.Errorf("FertAdj = %v, want 30", cfg.FertAdj)
	}
	if cfg.RainDelayH != 24 {
		t.Errorf("RainDelayH = %v, want 24", cfg.RainDelayH)
	}
	if cfg.Enabled {
		t.Error("Enabled should be false")
	}
	if cfg.PollMinutes != 60 {
		t.Errorf("PollMinutes = %v, want 60", cfg.PollMinutes)
	}
}
