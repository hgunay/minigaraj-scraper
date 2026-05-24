// Author: Hakan Gunay
// Date: 2026-04-04
// Mini GT crawler - official site (minigt.com) product pages

package minigt

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/crawler/shared"
	"minigaraj-scraper/internal/models"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"go.uber.org/zap"
)

const (
	brand      = "Mini GT"
	siteDomain = "www.minigt.com"
)

var defaultSeedURLs = []string{
	"https://www.minigt.com/collections/all",
	"https://www.minigt.com/collections/1-64",
	"https://www.minigt.com/collections/new-arrivals",
}

// MiniGTCrawler crawls Mini GT product data from the official site
type MiniGTCrawler struct {
	cfg    config.CrawlerConfig
	logger *zap.Logger
}

// New creates a new Mini GT crawler
func New(cfg config.CrawlerConfig, logger *zap.Logger) *MiniGTCrawler {
	return &MiniGTCrawler{cfg: cfg, logger: logger}
}

func (c *MiniGTCrawler) BrandName() string        { return brand }
func (c *MiniGTCrawler) DefaultSeedURLs() []string { return defaultSeedURLs }

// Crawl runs the Mini GT crawl
func (c *MiniGTCrawler) Crawl(ctx context.Context, jobID int64, seedURLs []string, out chan<- *models.RawModel) error {
	col := c.buildCollector()
	q, _ := queue.New(c.cfg.Parallelism, &queue.InMemoryQueueStorage{MaxSize: 100000})

	// Collection pages: follow product links
	col.OnHTML("a.product-card, a.product-item, .product-list a[href*='/products/']", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link != "" {
			_ = q.AddURL(e.Request.AbsoluteURL(link))
		}
	})

	// Pagination
	col.OnHTML("a.next, a[rel='next'], .pagination a[href]", func(e *colly.HTMLElement) {
		_ = q.AddURL(e.Request.AbsoluteURL(e.Attr("href")))
	})

	// Product detail page
	col.OnHTML(".product-single, .product-detail, .product__info", func(e *colly.HTMLElement) {
		model := c.parseProductPage(e, jobID)
		if model != nil {
			select {
			case <-ctx.Done():
				return
			case out <- model:
			}
		}
	})

	col.OnRequest(func(r *colly.Request) {
		c.logger.Debug("visiting", zap.String("url", r.URL.String()))
	})
	col.OnError(func(r *colly.Response, err error) {
		c.logger.Warn("request error",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Error(err),
		)
	})

	for _, u := range seedURLs {
		_ = q.AddURL(u)
	}

	c.logger.Info("Mini GT crawl started", zap.Int64("job_id", jobID))
	if err := q.Run(col); err != nil {
		return fmt.Errorf("queue run: %w", err)
	}
	c.logger.Info("Mini GT crawl completed", zap.Int64("job_id", jobID))
	return nil
}

func (c *MiniGTCrawler) buildCollector() *colly.Collector {
	return shared.BuildCollector(shared.CollectorOptions{
		AllowedDomains:      []string{siteDomain},
		CrawlerCfg:          c.cfg,
		Logger:              c.logger,
		RespectRobotsTxt:    true, // Official site — respect robots.txt
		ParallelismOverride: max(1, c.cfg.Parallelism/2),
		DelayOverride:       c.cfg.RequestDelayMs * 2,
	})
}

func (c *MiniGTCrawler) parseProductPage(e *colly.HTMLElement, jobID int64) *models.RawModel {
	rawData := models.RawDataJSON{}
	m := &models.RawModel{
		JobID:        &jobID,
		Brand:        brand,
		SourceURL:    e.Request.URL.String(),
		SourceDomain: siteDomain,
		ImageURLs:    []string{},
	}

	// Product title — typically contains model name + number
	title := strings.TrimSpace(e.ChildText("h1, .product-title, .product__title"))
	if title != "" {
		m.Name = &title
		rawData["title"] = title
		parseProductTitle(m, title)
	}

	// Price (informational)
	price := strings.TrimSpace(e.ChildText(".product-price, .price, .product__price"))
	if price != "" {
		rawData["price"] = price
	}

	// Description
	desc := strings.TrimSpace(e.ChildText(".product-description, .product__description, .rte"))
	if desc != "" {
		m.Description = shared.StrPtr(desc)
		rawData["description"] = desc
		parseDescription(m, desc, rawData)
	}

	// Product meta/specs (key-value pairs in description or spec table)
	e.ForEach(".product-specs tr, .product-meta li, dl dt", func(_ int, row *colly.HTMLElement) {
		key := strings.TrimSpace(row.ChildText("td:first-child, .meta-label, dt"))
		val := strings.TrimSpace(row.ChildText("td:last-child, .meta-value, dd"))
		if key != "" && val != "" {
			rawData[key] = val
			MapField(m, key, val)
		}
	})

	// Images
	e.ForEach(".product-images img, .product__media img, .product-photo img", func(_ int, img *colly.HTMLElement) {
		src := img.Attr("src")
		if src == "" {
			src = img.Attr("data-src")
		}
		if src != "" && !strings.Contains(src, "placeholder") {
			if strings.HasPrefix(src, "//") {
				src = "https:" + src
			}
			m.ImageURLs = append(m.ImageURLs, src)
			rawData["image_"+strconv.Itoa(len(m.ImageURLs))] = src
		}
	})

	// Mini GT default scale
	if m.Scale == nil {
		scale := "1:64"
		m.Scale = &scale
	}

	m.RawData = rawData

	if m.Name == nil || *m.Name == "" {
		return nil
	}
	return m
}

// parseProductTitle extracts info from Mini GT product titles.
// Typical format: "Mini GT 1:64 MGT00123 Honda Civic Type R Blue"
func parseProductTitle(m *models.RawModel, title string) {
	// Extract reference number (MGTxxxxx pattern)
	if idx := strings.Index(strings.ToUpper(title), "MGT"); idx != -1 {
		rest := title[idx:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			ref := parts[0]
			m.ReferenceNumber = shared.StrPtr(ref)
		}
	}

	// Extract scale if present
	if strings.Contains(title, "1:64") {
		scale := "1:64"
		m.Scale = &scale
	} else if strings.Contains(title, "1:43") {
		scale := "1:43"
		m.Scale = &scale
	}
}

// parseDescription extracts structured data from product descriptions
func parseDescription(m *models.RawModel, desc string, rawData models.RawDataJSON) {
	lines := strings.Split(desc, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key != "" && val != "" {
					rawData["desc_"+key] = val
					MapField(m, key, val)
				}
			}
		}
	}
}

// MapField maps a product page key-value to RawModel fields
func MapField(m *models.RawModel, key, val string) {
	k := strings.ToLower(strings.TrimSpace(key))

	switch {
	case shared.Contains(k, "name", "model", "vehicle"):
		m.Name = shared.StrPtr(val)
	case shared.Contains(k, "year"):
		if years := shared.YearRegex.FindStringSubmatch(val); len(years) > 0 {
			y, _ := strconv.Atoi(years[1])
			m.Year = &y
		}
	case shared.Contains(k, "series", "collection", "line"):
		m.Series = shared.StrPtr(val)
	case shared.Contains(k, "color", "colour", "exterior"):
		m.Color = shared.StrPtr(val)
	case shared.Contains(k, "scale"):
		m.Scale = shared.StrPtr(shared.NormalizeScale(val))
	case shared.Contains(k, "material"):
		m.Material = shared.StrPtr(val)
	case shared.Contains(k, "country", "origin", "made in"):
		m.Origin = shared.StrPtr(val)
	case shared.Contains(k, "sku", "item", "ref", "number", "mgt"):
		m.ReferenceNumber = shared.StrPtr(val)
	}
}
