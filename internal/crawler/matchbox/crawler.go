// Author: Hakan Gunay
// Date: 2026-04-04
// Matchbox crawler - primary source: Matchbox Wiki (fandom.com)

package matchbox

import (
	"context"
	"strconv"
	"strings"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/crawler/shared"
	"minigaraj-scraper/internal/models"

	"go.uber.org/zap"
)

const (
	brand      = "Matchbox"
	wikiDomain = "matchbox.fandom.com"
)

var defaultSeedURLs = []string{
	"https://matchbox.fandom.com/wiki/List_of_2024_Matchbox",
	"https://matchbox.fandom.com/wiki/List_of_2023_Matchbox",
	"https://matchbox.fandom.com/wiki/List_of_2022_Matchbox",
	"https://matchbox.fandom.com/wiki/List_of_2021_Matchbox",
	"https://matchbox.fandom.com/wiki/List_of_2020_Matchbox",
	"https://matchbox.fandom.com/wiki/Category:Matchbox_Collectors",
	"https://matchbox.fandom.com/wiki/Matchbox_Collectors",
}

// MatchboxCrawler crawls Matchbox data from fandom wiki
type MatchboxCrawler struct {
	cfg    config.CrawlerConfig
	logger *zap.Logger
	fandom *shared.FandomCrawler
}

// New creates a new Matchbox crawler
func New(cfg config.CrawlerConfig, logger *zap.Logger) *MatchboxCrawler {
	c := &MatchboxCrawler{cfg: cfg, logger: logger}
	c.fandom = shared.NewFandomCrawler(shared.FandomCrawlerConfig{
		Brand:          brand,
		AllowedDomains: []string{wikiDomain},
		CrawlerCfg:     cfg,
		Logger:         logger,
		FieldMapper:    MapField,
	})
	return c
}

func (c *MatchboxCrawler) BrandName() string        { return brand }
func (c *MatchboxCrawler) DefaultSeedURLs() []string { return defaultSeedURLs }

// Crawl delegates to the shared fandom crawler
func (c *MatchboxCrawler) Crawl(ctx context.Context, jobID int64, seedURLs []string, out chan<- *models.RawModel) error {
	return c.fandom.Crawl(ctx, jobID, seedURLs, out)
}

// MapField maps a parsed Matchbox infobox key-value to the appropriate RawModel field.
// Matchbox wiki uses similar field names to Hot Wheels with some differences.
func MapField(m *models.RawModel, key, val string) {
	k := strings.ToLower(strings.TrimSpace(key))

	switch {
	case shared.Contains(k, "name", "model"):
		m.Name = shared.StrPtr(val)
	case shared.Contains(k, "year", "debut", "introduced"):
		if years := shared.YearRegex.FindStringSubmatch(val); len(years) > 0 {
			y, _ := strconv.Atoi(years[1])
			m.Year = &y
		}
	case shared.Contains(k, "series", "collection", "line", "range"):
		m.Series = shared.StrPtr(val)
	case shared.Contains(k, "sub-series", "subseries", "segment", "theme"):
		m.SubSeries = shared.StrPtr(val)
	case shared.Contains(k, "color", "colour"):
		m.Color = shared.StrPtr(val)
	case shared.Contains(k, "wheel"):
		m.WheelType = shared.StrPtr(val)
	case shared.Contains(k, "scale"):
		m.Scale = shared.StrPtr(shared.NormalizeScale(val))
	case shared.Contains(k, "material", "body", "base"):
		m.Material = shared.StrPtr(val)
	case shared.Contains(k, "country", "origin", "made in"):
		m.Origin = shared.StrPtr(val)
	case shared.Contains(k, "number", "ref", "item", "sku", "mb"):
		if shared.RefNumRegex.MatchString(val) {
			m.ReferenceNumber = shared.StrPtr(val)
		}
	case shared.Contains(k, "description", "note", "detail"):
		m.Description = shared.StrPtr(val)
	}
}
