// Package orders — атомарный сервис заказов: импорт выгрузки и витрина данных.
package orders

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/clever/clever-dashboard/internal/ingestion"
	"github.com/clever/clever-dashboard/internal/model"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

const importBatchSize = 200

// ImportFile — совместимый адаптер для вызывающего кода с готовым []byte.
func (s *Service) ImportFile(filename string, data []byte) (*model.ImportResult, error) {
	return s.Import(filename, bytes.NewReader(data))
}

// Import потоково парсит выгрузку и полностью заменяет витрину заказов. Заказы
// сохраняются батчами, но одна транзакция гарантирует, что при любой ошибке в БД
// останется предыдущая целая витрина.
func (s *Service) Import(filename string, reader io.Reader) (*model.ImportResult, error) {
	replacement, err := s.repo.beginReplacement(filename)
	if err != nil {
		return nil, fmt.Errorf("начать импорт: %w", err)
	}
	defer replacement.Rollback()

	batch := make([]model.Order, 0, importBatchSize)
	ordersTotal := 0
	var minCreated, maxCreated time.Time
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := replacement.SaveBatch(batch); err != nil {
			return fmt.Errorf("сохранить батч заказов: %w", err)
		}
		batch = batch[:0]
		return nil
	}

	rowsTotal, err := ingestion.Stream(reader, func(record ingestion.Record) error {
		order, ok := ingestion.MapOrder(record)
		if !ok {
			return nil
		}
		ordersTotal++
		if !order.CreatedAt.IsZero() {
			if minCreated.IsZero() || order.CreatedAt.Before(minCreated) {
				minCreated = order.CreatedAt
			}
			if maxCreated.IsZero() || order.CreatedAt.After(maxCreated) {
				maxCreated = order.CreatedAt
			}
		}
		batch = append(batch, order)
		if len(batch) == importBatchSize {
			return flush()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ordersTotal == 0 {
		return nil, fmt.Errorf("в файле не найдено ни одного заказа")
	}
	if err := flush(); err != nil {
		return nil, err
	}
	start, end := timePointers(minCreated, maxCreated)
	if err := replacement.Complete(rowsTotal, ordersTotal, start, end); err != nil {
		return nil, fmt.Errorf("завершить импорт: %w", err)
	}
	return &model.ImportResult{
		ImportID:       replacement.ImportID(),
		Filename:       filename,
		RowsTotal:      rowsTotal,
		OrdersImported: ordersTotal,
		ItemsImported:  replacement.Items(),
		PeriodStart:    start,
		PeriodEnd:      end,
		OrdersCleared:  replacement.Cleared(),
	}, nil
}

func timePointers(min, max time.Time) (*time.Time, *time.Time) {
	if min.IsZero() {
		return nil, nil
	}
	return &min, &max
}
