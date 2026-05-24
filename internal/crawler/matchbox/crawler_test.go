package matchbox

import (
	"testing"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/models"

	"go.uber.org/zap"
)

func newTestCrawler() *MatchboxCrawler {
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
	if c.BrandName() != "Matchbox" {
		t.Errorf("BrandName() = %q, want %q", c.BrandName(), "Matchbox")
	}
	if len(c.DefaultSeedURLs()) == 0 {
		t.Error("DefaultSeedURLs() should not be empty")
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
			key:     "Model Name",
			val:     "Tesla Model S",
			checkFn: func(m *models.RawModel) bool { return m.Name != nil && *m.Name == "Tesla Model S" },
		},
		{
			name:    "maps year",
			key:     "Year Introduced",
			val:     "2023",
			checkFn: func(m *models.RawModel) bool { return m.Year != nil && *m.Year == 2023 },
		},
		{
			name:    "maps series with range keyword",
			key:     "Range",
			val:     "Moving Parts",
			checkFn: func(m *models.RawModel) bool { return m.Series != nil && *m.Series == "Moving Parts" },
		},
		{
			name:    "maps color",
			key:     "Color",
			val:     "Pearl White",
			checkFn: func(m *models.RawModel) bool { return m.Color != nil && *m.Color == "Pearl White" },
		},
		{
			name:    "maps scale",
			key:     "Scale",
			val:     "1/64",
			checkFn: func(m *models.RawModel) bool { return m.Scale != nil && *m.Scale == "1:64" },
		},
		{
			name:    "maps sub-series via theme",
			key:     "Theme",
			val:     "MBX Electric Drivers",
			checkFn: func(m *models.RawModel) bool { return m.SubSeries != nil && *m.SubSeries == "MBX Electric Drivers" },
		},
		{
			name:    "maps origin",
			key:     "Made In",
			val:     "Thailand",
			checkFn: func(m *models.RawModel) bool { return m.Origin != nil && *m.Origin == "Thailand" },
		},
		{
			name:    "maps material",
			key:     "Body Material",
			val:     "Zamac",
			checkFn: func(m *models.RawModel) bool { return m.Material != nil && *m.Material == "Zamac" },
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

func TestFandomCrawlerInitialized(t *testing.T) {
	c := newTestCrawler()
	if c.fandom == nil {
		t.Fatal("fandom crawler should be initialized")
	}
}
