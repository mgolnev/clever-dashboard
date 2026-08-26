// Package ecomsync отвечает только за загрузку и сохранение обезличенных
// дневных агрегатов этапов E-commerce из внешней аналитики.
package ecomsync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/clever/clever-dashboard/internal/model"
)

type Source interface {
	EcommerceName() string
	Channel() string
	Configured() bool
	FetchEcommerce(ctx context.Context, from, to time.Time) ([]model.DailyEcommerce, error)
}

type revisionedSource interface {
	EcommerceRevision() string
}

type Service struct {
	repo         *Repository
	sources      []Source
	enabled      bool
	lookbackDays int
	backfillDays int
	location     *time.Location
	mu           sync.Mutex
}

func NewService(repo *Repository, sources []Source, enabled bool, lookbackDays, backfillDays int, timezone string) *Service {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.Local
	}
	return &Service{
		repo: repo, sources: sources, enabled: enabled,
		lookbackDays: max(1, lookbackDays), backfillDays: max(1, backfillDays), location: loc,
	}
}

func (s *Service) Sync(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().In(s.location)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location).AddDate(0, 0, -1)
	var syncErrors []error
	for _, source := range s.sources {
		if !source.Configured() {
			continue
		}
		name := source.EcommerceName()
		latest, err := s.repo.latestDataDay(name)
		if err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", name, err))
			continue
		}
		revision := ""
		revisionChanged := false
		if versioned, ok := source.(revisionedSource); ok {
			revision = versioned.EcommerceRevision()
			if revision != "" {
				stored, revisionErr := s.repo.sourceRevision(name)
				if revisionErr != nil {
					syncErrors = append(syncErrors, fmt.Errorf("%s: ревизия: %w", name, revisionErr))
					continue
				}
				revisionChanged = stored != revision
			}
		}
		from := to.AddDate(0, 0, -(s.backfillDays - 1))
		if latest != "" && !revisionChanged {
			from = to.AddDate(0, 0, -(s.lookbackDays - 1))
		}
		if from.After(to) {
			continue
		}
		runID, err := s.repo.startRun(name, from.Format("2006-01-02"), to.Format("2006-01-02"))
		if err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: журнал: %w", name, err))
			continue
		}
		items, fetchErr := source.FetchEcommerce(ctx, from, to)
		if fetchErr == nil {
			items = completeDays(source, from, to, items)
			fetchErr = s.repo.upsert(items)
		}
		if fetchErr == nil && revision != "" {
			fetchErr = s.repo.setSourceRevision(name, revision)
		}
		status := "success"
		if fetchErr != nil {
			status = "error"
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", name, fetchErr))
		}
		if err := s.repo.finishRun(runID, status, len(items), fetchErr); err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: завершение журнала: %w", name, err))
		}
	}
	return errors.Join(syncErrors...)
}

func completeDays(source Source, from, to time.Time, items []model.DailyEcommerce) []model.DailyEcommerce {
	byDay := make(map[string]model.DailyEcommerce, len(items))
	for _, item := range items {
		if _, err := time.Parse("2006-01-02", item.Day); err != nil {
			continue
		}
		item.Channel = source.Channel()
		item.Source = source.EcommerceName()
		if item.SampleShare == 0 && !item.Sampled {
			item.SampleShare = 1
		}
		byDay[item.Day] = item
	}
	out := make([]model.DailyEcommerce, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		item, ok := byDay[key]
		if !ok {
			item = model.DailyEcommerce{
				Day: key, Channel: source.Channel(), Source: source.EcommerceName(), SampleShare: 1,
			}
		}
		out = append(out, item)
	}
	return out
}

func (s *Service) Run(ctx context.Context, interval time.Duration, onError func(error)) {
	if !s.enabled {
		return
	}
	run := func() {
		if err := s.Sync(ctx); err != nil && onError != nil {
			onError(err)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
