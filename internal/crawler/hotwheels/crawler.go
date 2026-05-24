// Author: Hakan Gunay
// Date: 2026-04-04
// Hot Wheels crawler - primary source: Hot Wheels Wiki (fandom.com)

package hotwheels

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
	brand      = "Hot Wheels"
	wikiDomain = "hotwheels.fandom.com"
)

// defaultSeedURLs are fallback URLs used when the DB has no seeds for this brand
var defaultSeedURLs = []string{
	"https://hotwheels.fandom.com/wiki/List_of_2024_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2023_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2022_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2021_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2020_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2019_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2018_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2017_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2016_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2015_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2014_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2013_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2012_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2011_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/List_of_2010_Hot_Wheels",
	"https://hotwheels.fandom.com/wiki/Car_Culture",
	"https://hotwheels.fandom.com/wiki/Hot_Wheels_Premium",
	"https://hotwheels.fandom.com/wiki/RLC_(Red_Line_Club)",
}

// HotWheelsCrawler crawls Hot Wheels data from fandom wiki
type HotWheelsCrawler struct {
	cfg    config.CrawlerConfig
	logger *zap.Logger
	fandom *shared.FandomCrawler
}

// New creates a new Hot Wheels crawler
func New(cfg config.CrawlerConfig, logger *zap.Logger) *HotWheelsCrawler {
	c := &HotWheelsCrawler{cfg: cfg, logger: logger}
	c.fandom = shared.NewFandomCrawler(shared.FandomCrawlerConfig{
		Brand:          brand,
		AllowedDomains: []string{wikiDomain},
		CrawlerCfg:     cfg,
		Logger:         logger,
		FieldMapper:    MapField,
	})
	return c
}

func (c *HotWheelsCrawler) BrandName() string        { return brand }
func (c *HotWheelsCrawler) DefaultSeedURLs() []string { return defaultSeedURLs }

// Crawl delegates to the shared fandom crawler
func (c *HotWheelsCrawler) Crawl(ctx context.Context, jobID int64, seedURLs []string, out chan<- *models.RawModel) error {
	return c.fandom.Crawl(ctx, jobID, seedURLs, out)
}

// MapField maps a parsed infobox key-value to the appropriate RawModel field.
// Exported so it can be tested and reused.
func MapField(m *models.RawModel, key, val string) {
	k := strings.ToLower(strings.TrimSpace(key))

	switch {
	case shared.Contains(k, "name"):
		m.Name = shared.StrPtr(val)
	case shared.Contains(k, "year", "debut"):
		if years := shared.YearRegex.FindStringSubmatch(val); len(years) > 0 {
			y, _ := strconv.Atoi(years[1])
			m.Year = &y
		}
	case shared.Contains(k, "series", "collection", "line"):
		m.Series = shared.StrPtr(val)
	case shared.Contains(k, "sub-series", "subseries", "segment"):
		m.SubSeries = shared.StrPtr(val)
	case shared.Contains(k, "color", "colour", "tampo"):
		m.Color = shared.StrPtr(val)
	case shared.Contains(k, "wheel", "rim"):
		m.WheelType = shared.StrPtr(val)
	case shared.Contains(k, "scale"):
		m.Scale = shared.StrPtr(shared.NormalizeScale(val))
	case shared.Contains(k, "material", "body", "base"):
		m.Material = shared.StrPtr(val)
	case shared.Contains(k, "country", "origin", "made in", "manufacture"):
		m.Origin = shared.StrPtr(val)
	case shared.Contains(k, "number", "ref", "item", "sku", "col"):
		if shared.RefNumRegex.MatchString(val) {
			m.ReferenceNumber = shared.StrPtr(val)
		}
	case shared.Contains(k, "description", "note", "detail"):
		m.Description = shared.StrPtr(val)
	}
}
