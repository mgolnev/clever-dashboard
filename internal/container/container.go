// Package container собирает зависимости приложения (DI). Сервисы получают
// порты/репозитории, а не внутренности соседних доменов.
package container

import (
	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/connectors/appmetrica"
	"github.com/clever/clever-dashboard/internal/connectors/metrika"
	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/services/acquisition"
	"github.com/clever/clever-dashboard/internal/services/customeranalytics"
	"github.com/clever/clever-dashboard/internal/services/ecomsync"
	"github.com/clever/clever-dashboard/internal/services/funnel"
	"github.com/clever/clever-dashboard/internal/services/goal"
	"github.com/clever/clever-dashboard/internal/services/importupload"
	"github.com/clever/clever-dashboard/internal/services/logistics"
	"github.com/clever/clever-dashboard/internal/services/metrics"
	"github.com/clever/clever-dashboard/internal/services/orders"
	"github.com/clever/clever-dashboard/internal/services/plan"
	"github.com/clever/clever-dashboard/internal/services/traffic"
	"github.com/clever/clever-dashboard/internal/services/trafficsync"
)

type Container struct {
	Cfg               config.Config
	DB                *db.DB
	Orders            *orders.Service
	ImportUploads     *importupload.Service
	Metrics           *metrics.Service
	Funnel            *funnel.Service
	Goal              *goal.Service
	Logistics         *logistics.Service
	Plan              *plan.Service
	Traffic           *traffic.Service
	Acquisition       *acquisition.Service
	CustomerAnalytics *customeranalytics.Service
	TrafficSync       *trafficsync.Service
	EcommerceSync     *ecomsync.Service
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
	importUploadSvc := importupload.New(cfg.ImportTempDir)
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
	customerAnalyticsSvc := customeranalytics.NewService(customeranalytics.NewRepository(database))
	metrikaClient := metrika.New(cfg.MetrikaCounterID, cfg.MetrikaOAuthToken, cfg.AnalyticsTimezone)
	appMetricaClient := appmetrica.New(cfg.AppMetricaAppID, cfg.AppMetricaOAuthToken)
	goalSvc := goal.NewService(goal.NewRepository(database), goal.Options{
		AnalyticsEnabled: cfg.AnalyticsSyncEnabled,
		Sources: []goal.SourceDefinition{
			{Source: metrikaClient.Name(), Channel: metrikaClient.Channel(), Configured: metrikaClient.Configured()},
			{Source: appMetricaClient.Name(), Channel: appMetricaClient.Channel(), Configured: appMetricaClient.Configured()},
		},
	})
	trafficSyncSvc := trafficsync.NewService(
		trafficsync.NewRepository(database),
		[]trafficsync.Source{metrikaClient, appMetricaClient},
		cfg.AnalyticsSyncEnabled,
		cfg.AnalyticsLookbackDays,
		cfg.AnalyticsBackfillDays,
		cfg.AnalyticsTimezone,
	)
	ecommerceSyncSvc := ecomsync.NewService(
		ecomsync.NewRepository(database),
		[]ecomsync.Source{metrikaClient, appMetricaClient},
		cfg.AnalyticsSyncEnabled,
		cfg.AnalyticsLookbackDays,
		cfg.AnalyticsBackfillDays,
		cfg.AnalyticsTimezone,
	)

	return &Container{
		Cfg:               cfg,
		DB:                database,
		Orders:            ordersSvc,
		ImportUploads:     importUploadSvc,
		Metrics:           metricsSvc,
		Funnel:            funnelSvc,
		Goal:              goalSvc,
		Logistics:         logisticsSvc,
		Plan:              planSvc,
		Traffic:           trafficSvc,
		Acquisition:       acquisitionSvc,
		CustomerAnalytics: customerAnalyticsSvc,
		TrafficSync:       trafficSyncSvc,
		EcommerceSync:     ecommerceSyncSvc,
	}, nil
}

func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
