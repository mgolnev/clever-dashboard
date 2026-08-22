// Package trafficsync отвечает только за плановую загрузку и сохранение
// обезличенных дневных агрегатов внешней аналитики.
package trafficsync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/clever/clever-dashboard/internal/model"
)

type Source interface {
	Name() string
	Channel() string
	Configured() bool
	Fetch(ctx context.Context, from, to time.Time) ([]model.DailyTraffic, error)
}

type revisionedSource interface {
	Revision() string
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

// Sync обновляет закрытые дни до вчера включительно. Если источник ещё пуст,
// загружается backfill; в дальнейшем повторно запрашивается только lookback.
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
		latest, err := s.repo.latestDataDay(source.Name())
		if err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", source.Name(), err))
			continue
		}
		revision := ""
		revisionChanged := false
		if versioned, ok := source.(revisionedSource); ok {
			revision = versioned.Revision()
			if revision != "" {
				storedRevision, revisionErr := s.repo.sourceRevision(source.Name())
				if revisionErr != nil {
					syncErrors = append(syncErrors, fmt.Errorf("%s: ревизия: %w", source.Name(), revisionErr))
					continue
				}
				revisionChanged = storedRevision != revision
			}
		}
		from := to.AddDate(0, 0, -(s.backfillDays - 1))
		if latest != "" && !revisionChanged {
			from = to.AddDate(0, 0, -(s.lookbackDays - 1))
		}
		if from.After(to) {
			continue
		}
		runID, err := s.repo.startRun(source.Name(), from.Format("2006-01-02"), to.Format("2006-01-02"))
		if err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: журнал: %w", source.Name(), err))
			continue
		}
		items, fetchErr := source.Fetch(ctx, from, to)
		if fetchErr == nil {
			items = completeDays(source, from, to, items)
			fetchErr = s.repo.upsert(items)
		}
		if fetchErr == nil && revision != "" {
			fetchErr = s.repo.setSourceRevision(source.Name(), revision)
		}
		status := "success"
		if fetchErr != nil {
			status = "error"
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", source.Name(), fetchErr))
		}
		if err := s.repo.finishRun(runID, status, len(items), fetchErr); err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: завершение журнала: %w", source.Name(), err))
		}
	}
	return errors.Join(syncErrors...)
}

// completeDays материализует нулевые дни и одновременно закрепляет канал и
// источник за адаптером. Поэтому удалённая корректировка значения до нуля не
// оставит в локальной БД старый ненулевой агрегат.
func completeDays(source Source, from, to time.Time, items []model.DailyTraffic) []model.DailyTraffic {
	byDay := make(map[string]model.DailyTraffic, len(items))
	for _, item := range items {
		if _, err := time.Parse("2006-01-02", item.Day); err != nil {
			continue
		}
		item.Channel = source.Channel()
		item.Source = source.Name()
		if item.SampleShare == 0 && !item.Sampled {
			item.SampleShare = 1
		}
		byDay[item.Day] = item
	}
	out := make([]model.DailyTraffic, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		item, ok := byDay[key]
		if !ok {
			item = model.DailyTraffic{
				Day: key, Channel: source.Channel(), Source: source.Name(), SampleShare: 1,
			}
		}
		out = append(out, item)
	}
	return out
}

// Run выполняет синхронизацию сразу после старта и затем с заданным интервалом.
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

func (s *Service) Status() (*StatusReport, error) {
	latest, err := s.repo.latestRuns()
	if err != nil {
		return nil, err
	}
	report := &StatusReport{Enabled: s.enabled, Sources: make([]SourceStatus, 0, len(s.sources))}
	for _, source := range s.sources {
		st := latest[source.Name()]
		st.Source = source.Name()
		st.Channel = source.Channel()
		st.Configured = source.Configured()
		if st.Status == "" {
			st.Status = "never"
		}
		st.LastDataDay, err = s.repo.latestDataDay(source.Name())
		if err != nil {
			return nil, err
		}
		report.Sources = append(report.Sources, st)
	}
	return report, nil
}
