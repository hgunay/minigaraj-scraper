package minigt

import (
	"testing"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/models"

	"go.uber.org/zap"
)

func newTestCrawler() *MiniGTCrawler {
	return New(config.CrawlerConfig{
		Parallelism:    2,
		RequestDelayMs: 100,
		RandomDelayMs:  50,
		MaxDepth:       3,
		TimeoutSec:     10,
	}, zap.NewNop())
}

func TestBrandNameAndDefaultSeedURLs(t *testing.T) {
	c := newTestCrawler()
	if c.BrandName() != "Mini GT" {
		t.Errorf("BrandName() = %q, want %q", c.BrandName(), "Mini GT")
	}
	if len(c.DefaultSeedURLs()) == 0 {
		t.Error("DefaultSeedURLs() should not be empty")
	}
}

func TestParseProductTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		ref   string
		scale string
	}{
		{
			name:  "standard title with MGT number",
			title: "Mini GT 1:64 MGT00456 Honda Civic Type R Flame Red",
			ref:   "MGT00456",
			scale: "1:64",
		},
		{
			name:  "title with 1:43 scale",
			title: "MGT00789 Porsche 911 GT3 1:43",
			ref:   "MGT00789",
			scale: "1:43",
		},
		{
			name:  "title without MGT number",
			title: "Toyota GR Supra Racing Concept",
			ref:   "",
			scale: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &models.RawModel{}
			parseProductTitle(m, tt.title)

			if tt.ref == "" {
				if m.ReferenceNumber != nil {
					t.Errorf("expected nil ReferenceNumber, got %q", *m.ReferenceNumber)
				}
			} else {
				if m.ReferenceNumber == nil || *m.ReferenceNumber != tt.ref {
					got := "<nil>"
					if m.ReferenceNumber != nil {
						got = *m.ReferenceNumber
					}
					t.Errorf("ReferenceNumber = %s, want %q", got, tt.ref)
				}
			}

			if tt.scale == "" {
				if m.Scale != nil {
					t.Errorf("expected nil Scale, got %q", *m.Scale)
				}
			} else {
				if m.Scale == nil || *m.Scale != tt.scale {
					got := "<nil>"
					if m.Scale != nil {
						got = *m.Scale
					}
					t.Errorf("Scale = %s, want %q", got, tt.scale)
				}
			}
		})
	}
}

func TestMapField(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		checkFn func(*models.RawModel) bool
	}{
		{
			name:    "maps model name",
			key:     "Vehicle",
			val:     "Honda NSX",
			checkFn: func(m *models.RawModel) bool { return m.Name != nil && *m.Name == "Honda NSX" },
		},
		{
			name:    "maps color via exterior",
			key:     "Exterior Color",
			val:     "Midnight Blue",
			checkFn: func(m *models.RawModel) bool { return m.Color != nil && *m.Color == "Midnight Blue" },
		},
		{
			name:    "maps scale",
			key:     "Scale",
			val:     "1/64",
			checkFn: func(m *models.RawModel) bool { return m.Scale != nil && *m.Scale == "1:64" },
		},
		{
			name:    "maps SKU",
			key:     "SKU",
			val:     "MGT00123",
			checkFn: func(m *models.RawModel) bool { return m.ReferenceNumber != nil && *m.ReferenceNumber == "MGT00123" },
		},
		{
			name:    "maps origin",
			key:     "Made In",
			val:     "China",
			checkFn: func(m *models.RawModel) bool { return m.Origin != nil && *m.Origin == "China" },
		},
		{
			name: "maps year",
			key:  "Year",
			val:  "2024",
			checkFn: func(m *models.RawModel) bool {
				return m.Year != nil && *m.Year == 2024
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

func TestBuildCollector(t *testing.T) {
	c := newTestCrawler()
	col := c.buildCollector()
	if col == nil {
		t.Fatal("buildCollector() returned nil")
	}
}
