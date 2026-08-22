// Package container собирает зависимости приложения (DI). Сервисы получают
// порты/репозитории, а не внутренности соседних доменов.
package container

import (
	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/connectors/appmetrica"
	"github.com/clever/clever-dashboard/internal/connectors/metrika"
	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/services/acquisition"
	"github.com/clever/clever-dashboard/internal/services/funnel"
	"github.com/clever/clever-dashboard/internal/services/logistics"
	"github.com/clever/clever-dashboard/internal/services/metrics"
	"github.com/clever/clever-dashboard/internal/services/orders"
	"github.com/clever/clever-dashboard/internal/services/plan"
	"github.com/clever/clever-dashboard/internal/services/traffic"
	"github.com/clever/clever-dashboard/internal/services/trafficsync"
)

type Container struct {
	Cfg         config.Config
	DB          *db.DB
	Orders      *orders.Service
	Metrics     *metrics.Service
	Funnel      *funnel.Service
	Logistics   *logistics.Service
	Plan        *plan.Service
	Traffic     *traffic.Service
	Acquisition *acquisition.Service
	TrafficSync *trafficsync.Service
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
	logisticsSvc := logistics.NewService(
		logistics.NewRepository(database),
		cfg.LogisticsPilotCities,
		cfg.LogisticsPilotStart,
	)
	planSvc := plan.NewService(plan.NewRepository(database))
	trafficSvc := traffic.NewService(traffic.NewRepository(database))
	acquisitionSvc := acquisition.NewService(acquisition.NewRepository(database))
	metrikaClient := metrika.New(cfg.MetrikaCounterID, cfg.MetrikaOAuthToken, cfg.AnalyticsTimezone)
	appMetricaClient := appmetrica.New(cfg.AppMetricaAppID, cfg.AppMetricaOAuthToken, cfg.AnalyticsTimezone)
	trafficSyncSvc := trafficsync.NewService(
		trafficsync.NewRepository(database),
		[]trafficsync.Source{metrikaClient, appMetricaClient},
		cfg.AnalyticsSyncEnabled,
		cfg.AnalyticsLookbackDays,
		cfg.AnalyticsBackfillDays,
		cfg.AnalyticsTimezone,
	)

	return &Container{
		Cfg:         cfg,
		DB:          database,
		Orders:      ordersSvc,
		Metrics:     metricsSvc,
		Funnel:      funnelSvc,
		Logistics:   logisticsSvc,
		Plan:        planSvc,
		Traffic:     trafficSvc,
		Acquisition: acquisitionSvc,
		TrafficSync: trafficSyncSvc,
	}, nil
}

func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
