// Package orders — атомарный сервис заказов: импорт выгрузки и витрина данных.
package orders

import (
	"fmt"
	"time"

	"github.com/clever/clever-dashboard/internal/ingestion"
	"github.com/clever/clever-dashboard/internal/model"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// ImportFile парсит файл выгрузки, нормализует заказы и сохраняет их
// идемпотентно (дедуп по номеру заказа).
func (s *Service) ImportFile(filename string, data []byte) (*model.ImportResult, error) {
	records, err := ingestion.ParseFile(data)
	if err != nil {
		return nil, err
	}
	orders := ingestion.MapOrders(records)
	if len(orders) == 0 {
		return nil, fmt.Errorf("в файле не найдено ни одного заказа")
	}

	start, end := periodBounds(orders)
	importID, err := s.repo.createImport(filename, len(records), start, end)
	if err != nil {
		return nil, fmt.Errorf("создать запись импорта: %w", err)
	}
	itemsN, err := s.repo.saveOrders(orders, importID)
	if err != nil {
		return nil, fmt.Errorf("сохранить заказы: %w", err)
	}
	if err := s.repo.updateImportStats(importID, len(orders), itemsN); err != nil {
		return nil, err
	}
	return &model.ImportResult{
		ImportID:       importID,
		Filename:       filename,
		RowsTotal:      len(records),
		OrdersImported: len(orders),
		ItemsImported:  itemsN,
		PeriodStart:    start,
		PeriodEnd:      end,
	}, nil
}

func periodBounds(orders []model.Order) (*time.Time, *time.Time) {
	var min, max time.Time
	for _, o := range orders {
		if o.CreatedAt.IsZero() {
			continue
		}
		if min.IsZero() || o.CreatedAt.Before(min) {
			min = o.CreatedAt
		}
		if max.IsZero() || o.CreatedAt.After(max) {
			max = o.CreatedAt
		}
	}
	if min.IsZero() {
		return nil, nil
	}
	return &min, &max
}
