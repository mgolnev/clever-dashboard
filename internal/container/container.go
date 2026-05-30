// Package container собирает зависимости приложения (DI). Сервисы получают
// порты/репозитории, а не внутренности соседних доменов.
package container

import (
	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/services/funnel"
	"github.com/clever/clever-dashboard/internal/services/metrics"
	"github.com/clever/clever-dashboard/internal/services/orders"
)

type Container struct {
	Cfg     config.Config
	DB      *db.DB
	Orders  *orders.Service
	Metrics *metrics.Service
	Funnel  *funnel.Service
}

func New(cfg config.Config) (*Container, error) {
	database, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(); err != nil {
		return nil, err
	}

	ordersSvc := orders.NewService(orders.NewRepository(database))
	metricsSvc := metrics.NewService(metrics.NewRepository(database))
	funnelSvc := funnel.NewService(funnel.NewRepository(database))

	return &Container{
		Cfg:     cfg,
		DB:      database,
		Orders:  ordersSvc,
		Metrics: metricsSvc,
		Funnel:  funnelSvc,
	}, nil
}

func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
