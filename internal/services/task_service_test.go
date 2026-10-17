package services

import "testing"

func TestTaskTypeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{TaskTypeWater, "浇水"},
		{TaskTypeFertilize, "施肥"},
		{TaskTypeRepot, "换盆"},
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := TaskTypeName(tt.input); got != tt.want {
				t.Errorf("TaskTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
