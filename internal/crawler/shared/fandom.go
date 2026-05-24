// Author: Hakan Gunay
// Date: 2026-04-04
// Fandom wiki shared parser — reusable across Hot Wheels, Matchbox, and other wiki-based crawlers

package shared

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/models"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"go.uber.org/zap"
)

// FieldMapper maps an infobox key-value pair to RawModel fields.
// Each brand can provide its own mapper for brand-specific fields.
type FieldMapper func(m *models.RawModel, key, val string)

// FandomCrawlerConfig holds configuration for a fandom wiki crawler
type FandomCrawlerConfig struct {
	Brand          string
	AllowedDomains []string
	CrawlerCfg     config.CrawlerConfig
	Logger         *zap.Logger
	FieldMapper    FieldMapper
}

// FandomCrawler implements common crawling logic for fandom.com wikis
type FandomCrawler struct {
	cfg FandomCrawlerConfig
}

// NewFandomCrawler creates a new fandom wiki crawler
func NewFandomCrawler(cfg FandomCrawlerConfig) *FandomCrawler {
	return &FandomCrawler{cfg: cfg}
}

// Crawl runs the wiki crawl using the provided seed URLs
func (fc *FandomCrawler) Crawl(ctx context.Context, jobID int64, seedURLs []string, out chan<- *models.RawModel) error {
	col := fc.buildCollector()
	q, _ := queue.New(fc.cfg.CrawlerCfg.Parallelism, &queue.InMemoryQueueStorage{MaxSize: 100000})

	// List pages: extract links to individual model pages
	col.OnHTML("table.wikitable tr", func(e *colly.HTMLElement) {
		e.ForEach("td a[href]", func(_ int, el *colly.HTMLElement) {
			link := el.Attr("href")
			if strings.Contains(link, "/wiki/") &&
				!strings.Contains(link, "List_of") &&
				!strings.Contains(link, "Category:") &&
				!strings.Contains(link, "File:") {
				absURL := e.Request.AbsoluteURL(link)
				_ = q.AddURL(absURL)
			}
		})

		fc.parseListRow(ctx, e, jobID, out)
	})

	// Category pages
	col.OnHTML(".category-page__member-link", func(e *colly.HTMLElement) {
		_ = q.AddURL(e.Request.AbsoluteURL(e.Attr("href")))
	})

	// Model detail pages: parse infobox
	col.OnHTML(".portable-infobox, .infobox", func(e *colly.HTMLElement) {
		model := fc.parseInfobox(e, jobID)
		if model != nil {
			select {
			case <-ctx.Done():
				return
			case out <- model:
			}
		}
	})

	col.OnRequest(func(r *colly.Request) {
		fc.cfg.Logger.Debug("visiting", zap.String("url", r.URL.String()))
	})
	col.OnError(func(r *colly.Response, err error) {
		fc.cfg.Logger.Warn("request error",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Error(err),
		)
	})

	for _, u := range seedURLs {
		_ = q.AddURL(u)
	}

	fc.cfg.Logger.Info("fandom crawl started",
		zap.String("brand", fc.cfg.Brand),
		zap.Int64("job_id", jobID),
	)
	if err := q.Run(col); err != nil {
		return fmt.Errorf("queue run: %w", err)
	}
	fc.cfg.Logger.Info("fandom crawl completed",
		zap.String("brand", fc.cfg.Brand),
		zap.Int64("job_id", jobID),
	)
	return nil
}

func (fc *FandomCrawler) buildCollector() *colly.Collector {
	return BuildCollector(CollectorOptions{
		AllowedDomains:   fc.cfg.AllowedDomains,
		CrawlerCfg:       fc.cfg.CrawlerCfg,
		Logger:           fc.cfg.Logger,
		RespectRobotsTxt: false, // Wiki sites — ignore robots.txt
	})
}

func (fc *FandomCrawler) parseInfobox(e *colly.HTMLElement, jobID int64) *models.RawModel {
	rawData := models.RawDataJSON{}
	domain := fc.cfg.AllowedDomains[0]
	m := &models.RawModel{
		JobID:        &jobID,
		Brand:        fc.cfg.Brand,
		SourceURL:    e.Request.URL.String(),
		SourceDomain: domain,
		ImageURLs:    []string{},
	}

	pageName := ExtractWikiPageName(e.Request.URL.String())
	if pageName != "" {
		m.Name = &pageName
		rawData["page_name"] = pageName
	}

	e.ForEach(".pi-item, tr", func(_ int, row *colly.HTMLElement) {
		key := strings.TrimSpace(row.ChildText(".pi-data-label, th"))
		val := strings.TrimSpace(row.ChildText(".pi-data-value, td"))
		if key == "" || val == "" {
			return
		}
		rawData[key] = val
		if fc.cfg.FieldMapper != nil {
			fc.cfg.FieldMapper(m, key, val)
		}
	})

	e.ForEach(".pi-image img, .infobox img", func(_ int, img *colly.HTMLElement) {
		src := img.Attr("src")
		if src != "" && !strings.Contains(src, "placeholder") {
			m.ImageURLs = append(m.ImageURLs, CleanImageURL(src))
			rawData["image_"+strconv.Itoa(len(m.ImageURLs))] = src
		}
	})

	m.RawData = rawData

	if m.Name == nil || *m.Name == "" {
		return nil
	}
	return m
}

func (fc *FandomCrawler) parseListRow(ctx context.Context, e *colly.HTMLElement, jobID int64, out chan<- *models.RawModel) {
	cells := []string{}
	e.ForEach("td", func(_ int, td *colly.HTMLElement) {
		cells = append(cells, strings.TrimSpace(td.Text))
	})

	if len(cells) < 2 {
		return
	}

	rawData := models.RawDataJSON{}
	domain := fc.cfg.AllowedDomains[0]
	m := &models.RawModel{
		JobID:        &jobID,
		Brand:        fc.cfg.Brand,
		SourceURL:    e.Request.URL.String(),
		SourceDomain: domain,
		ImageURLs:    []string{},
	}

	if years := YearRegex.FindStringSubmatch(e.Request.URL.String()); len(years) > 0 {
		y, _ := strconv.Atoi(years[1])
		m.Year = &y
		rawData["year_from_url"] = years[1]
	}

	for i, cell := range cells {
		rawData[fmt.Sprintf("col_%d", i)] = cell
		switch i {
		case 1:
			if cell != "" {
				v := cell
				m.Name = &v
			}
		case 2:
			if cell != "" {
				v := cell
				m.Series = &v
			}
		case 3:
			if cell != "" {
				v := cell
				m.Color = &v
			}
		case 4:
			if cell != "" {
				v := cell
				m.WheelType = &v
			}
		case 5:
			if cell != "" && RefNumRegex.MatchString(cell) {
				v := cell
				m.ReferenceNumber = &v
			}
		case 6:
			if cell != "" {
				v := cell
				m.Origin = &v
			}
		}
	}

	m.RawData = rawData

	if m.Name == nil || *m.Name == "" || *m.Name == "#" {
		return
	}

	select {
	case <-ctx.Done():
		return
	case out <- m:
	}
}
