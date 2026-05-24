// Author: Hakan Gunay
// Date: 2026-04-04
// Shared collector builder with robots.txt compliance

package shared

import (
	"time"

	"minigaraj-scraper/internal/config"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"go.uber.org/zap"
)

// CollectorOptions configures a shared collector
type CollectorOptions struct {
	AllowedDomains  []string
	CrawlerCfg      config.CrawlerConfig
	Logger          *zap.Logger
	RespectRobotsTxt bool // true for official sites, false for wikis
	// Rate limiting overrides (0 = use CrawlerCfg defaults)
	ParallelismOverride int
	DelayOverride       int // ms
}

// BuildCollector creates a configured colly collector with retry and robots.txt support
func BuildCollector(opts CollectorOptions) *colly.Collector {
	collyOpts := []colly.CollectorOption{
		colly.AllowedDomains(opts.AllowedDomains...),
		colly.MaxDepth(opts.CrawlerCfg.MaxDepth),
		colly.Async(true),
		colly.ParseHTTPErrorResponse(),
	}

	if opts.RespectRobotsTxt {
		// No-op: colly respects robots.txt by default.
		// We explicitly do NOT add colly.IgnoreRobotsTxt() here.
	} else {
		collyOpts = append(collyOpts, colly.IgnoreRobotsTxt())
	}

	col := colly.NewCollector(collyOpts...)

	extensions.RandomUserAgent(col)
	col.SetRequestTimeout(time.Duration(opts.CrawlerCfg.TimeoutSec) * time.Second)

	parallelism := opts.CrawlerCfg.Parallelism
	if opts.ParallelismOverride > 0 {
		parallelism = opts.ParallelismOverride
	}
	delayMs := opts.CrawlerCfg.RequestDelayMs
	if opts.DelayOverride > 0 {
		delayMs = opts.DelayOverride
	}

	_ = col.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallelism,
		Delay:       time.Duration(delayMs) * time.Millisecond,
		RandomDelay: time.Duration(opts.CrawlerCfg.RandomDelayMs) * time.Millisecond,
	})

	AttachRetry(col, DefaultRetryConfig(opts.CrawlerCfg.MaxRetries), opts.Logger)

	return col
}
