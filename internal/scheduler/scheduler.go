// Author: Hakan Gunay
// Date: 2026-04-04
// Cron scheduler for automated crawl jobs

package scheduler

import (
	"context"

	"minigaraj-scraper/internal/crawler/manager"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Schedule defines a brand crawl schedule
type Schedule struct {
	Brand    string
	CronExpr string // cron expression (e.g. "0 2 * * 0" = every Sunday 2am)
}

// Scheduler manages periodic crawl jobs
type Scheduler struct {
	cron    *cron.Cron
	manager *manager.Manager
	logger  *zap.Logger
}

// New creates a new Scheduler
func New(mgr *manager.Manager, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		manager: mgr,
		logger:  logger,
	}
}

// AddSchedule registers a periodic crawl for a brand
func (s *Scheduler) AddSchedule(schedule Schedule) error {
	_, err := s.cron.AddFunc(schedule.CronExpr, func() {
		s.logger.Info("scheduled crawl triggered",
			zap.String("brand", schedule.Brand),
			zap.String("cron", schedule.CronExpr),
		)

		jobID, err := s.manager.StartJob(context.Background(), schedule.Brand)
		if err != nil {
			s.logger.Error("scheduled crawl failed to start",
				zap.String("brand", schedule.Brand),
				zap.Error(err),
			)
			return
		}

		s.logger.Info("scheduled crawl job created",
			zap.String("brand", schedule.Brand),
			zap.Int64("job_id", jobID),
		)
	})

	if err != nil {
		return err
	}

	s.logger.Info("schedule registered",
		zap.String("brand", schedule.Brand),
		zap.String("cron", schedule.CronExpr),
	)
	return nil
}

// Start begins the cron scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Info("scheduler started", zap.Int("jobs", len(s.cron.Entries())))
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}
