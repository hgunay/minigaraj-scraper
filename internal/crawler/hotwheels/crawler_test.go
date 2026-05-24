package hotwheels

import (
	"testing"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/models"

	"go.uber.org/zap"
)

func newTestCrawler() *HotWheelsCrawler {
	return New(config.CrawlerConfig{
		Parallelism:    2,
		RequestDelayMs: 100,
		RandomDelayMs:  50,
		MaxDepth:       3,
		TimeoutSec:     10,
		MaxRetries:     1,
	}, zap.NewNop())
}

// --- BrandName / DefaultSeedURLs ---

func TestBrandNameAndDefaultSeedURLs(t *testing.T) {
	c := newTestCrawler()
	if c.BrandName() != "Hot Wheels" {
		t.Errorf("BrandName() = %q, want %q", c.BrandName(), "Hot Wheels")
	}
	if len(c.DefaultSeedURLs()) == 0 {
		t.Error("DefaultSeedURLs() should not be empty")
	}
}

// --- MapField ---

func TestMapField(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		checkFn func(*models.RawModel) bool
	}{
		{
			name: "maps name field",
			key:  "Model Name",
			val:  "Twin Mill",
			checkFn: func(m *models.RawModel) bool {
				return m.Name != nil && *m.Name == "Twin Mill"
			},
		},
		{
			name: "maps year from debut",
			key:  "Debut Year",
			val:  "First produced in 2020",
			checkFn: func(m *models.RawModel) bool {
				return m.Year != nil && *m.Year == 2020
			},
		},
		{
			name: "maps series field",
			key:  "Series",
			val:  "Car Culture",
			checkFn: func(m *models.RawModel) bool {
				return m.Series != nil && *m.Series == "Car Culture"
			},
		},
		{
			name: "maps color field",
			key:  "Color",
			val:  "Spectraflame Red",
			checkFn: func(m *models.RawModel) bool {
				return m.Color != nil && *m.Color == "Spectraflame Red"
			},
		},
		{
			name: "maps wheel type",
			key:  "Wheel Type",
			val:  "Real Riders",
			checkFn: func(m *models.RawModel) bool {
				return m.WheelType != nil && *m.WheelType == "Real Riders"
			},
		},
		{
			name: "maps scale",
			key:  "Scale",
			val:  "1/64",
			checkFn: func(m *models.RawModel) bool {
				return m.Scale != nil && *m.Scale == "1:64"
			},
		},
		{
			name: "maps material field",
			key:  "Body Material",
			val:  "Metal/Metal",
			checkFn: func(m *models.RawModel) bool {
				return m.Material != nil && *m.Material == "Metal/Metal"
			},
		},
		{
			name: "maps country of origin",
			key:  "Country of Origin",
			val:  "Malaysia",
			checkFn: func(m *models.RawModel) bool {
				return m.Origin != nil && *m.Origin == "Malaysia"
			},
		},
		{
			name: "maps sub-series via segment key",
			key:  "Segment",
			val:  "Boulevard",
			checkFn: func(m *models.RawModel) bool {
				return m.SubSeries != nil && *m.SubSeries == "Boulevard"
			},
		},
		{
			name: "maps reference number",
			key:  "Item Number",
			val:  "GJR456",
			checkFn: func(m *models.RawModel) bool {
				return m.ReferenceNumber != nil && *m.ReferenceNumber == "GJR456"
			},
		},
		{
			name: "maps description",
			key:  "Description",
			val:  "Premium casting with opening hood",
			checkFn: func(m *models.RawModel) bool {
				return m.Description != nil && *m.Description == "Premium casting with opening hood"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &models.RawModel{}
			MapField(m, tt.key, tt.val)
			if !tt.checkFn(m) {
				t.Errorf("MapField(%q, %q) did not set expected field", tt.key, tt.val)
			}
		})
	}
}

// --- FandomCrawler is initialized ---

func TestFandomCrawlerInitialized(t *testing.T) {
	c := newTestCrawler()
	if c.fandom == nil {
		t.Fatal("fandom crawler should be initialized")
	}
}
