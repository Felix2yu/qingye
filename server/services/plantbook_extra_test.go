package services

import (
	"testing"
)

func TestCareInfo(t *testing.T) {
	d := &pbDetail{
		Light:      "明亮散射光",
		Sunlight:   "sun",
		Watering:   "见干见湿",
		SoilText:   "疏松肥沃",
		Care: &pbCare{
			Light:    "Care光",
			Watering: "Care水",
			Soil:     "Care土",
		},
	}
	c := d.careInfo()
	if c.Light != "Care光" {
		t.Errorf("light = %q, want Care光", c.Light)
	}
	if c.Watering != "Care水" {
		t.Errorf("watering = %q, want Care水", c.Watering)
	}
	if c.Soil != "Care土" {
		t.Errorf("soil = %q, want Care土", c.Soil)
	}

	// 无 care 嵌套时回退到平铺字段
	d2 := &pbDetail{Light: "平铺光", Watering: "平铺水"}
	c2 := d2.careInfo()
	if c2.Light != "平铺光" || c2.Watering != "平铺水" {
		t.Errorf("careInfo fallback = %+v", c2)
	}
}
