// Package handlers — HTTP-слой (Fiber). Принимает запросы, валидирует вход и
// делегирует доменным сервисам.
package handlers

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/clever/clever-dashboard/internal/container"
	"github.com/clever/clever-dashboard/internal/services/funnel"
	"github.com/clever/clever-dashboard/internal/services/logistics"
	"github.com/clever/clever-dashboard/internal/services/metrics"
	"github.com/clever/clever-dashboard/internal/services/plan"
	"github.com/clever/clever-dashboard/internal/services/traffic"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	c *container.Container
}

func New(c *container.Container) *Handler { return &Handler{c: c} }

func (h *Handler) Register(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", h.health)
	api.Post("/import", h.importFile)
	api.Get("/import/local", h.localFiles)
	api.Post("/import/local", h.importLocalFile)
	api.Get("/bounds", h.bounds)
	api.Get("/cities", h.cities)
	api.Get("/regions", h.regions)
	api.Get("/channels", h.channels)
	api.Get("/payments", h.payments)
	api.Get("/deliveries", h.deliveries)
	api.Get("/coupons", h.coupons)
	api.Get("/metrics", h.metrics)
	api.Get("/funnel", h.funnel)
	api.Get("/logistics", h.logistics)
	api.Get("/dynamics", h.dynamics)
	api.Get("/plan", h.getPlan)
	api.Put("/plan", h.putPlan)
	api.Get("/traffic", h.getTraffic)
	api.Put("/traffic", h.putTraffic)
}

func (h *Handler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// importFile принимает multipart-файл выгрузки Битрикса (поле "file").
func (h *Handler) importFile(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ожидается файл в поле 'file'")
	}
	f, err := fh.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "не удалось открыть файл")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "ошибка чтения файла")
	}
	res, err := h.c.Orders.ImportFile(fh.Filename, data)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(res)
}

// localFiles возвращает список доступных для импорта файлов в папке данных (data/ или /data).
func (h *Handler) localFiles(c *fiber.Ctx) error {
	dir := filepath.Dir(h.c.Cfg.DBDSN)
	files, err := os.ReadDir(dir)
	if err != nil {
		return c.JSON([]string{})
	}
	var list []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		ext := strings.ToLower(filepath.Ext(name))
		// Игнорируем файлы SQLite и скрытые файлы
		if ext == ".db" || ext == "-wal" || ext == "-shm" || strings.HasPrefix(name, ".") || strings.Contains(name, "clever.db") {
			continue
		}
		list = append(list, name)
	}
	return c.JSON(list)
}

// importLocalFile выполняет импорт файла, находящегося локально на сервере в папке данных.
func (h *Handler) importLocalFile(c *fiber.Ctx) error {
	type Req struct {
		Filename string `json:"filename"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "неверное тело запроса")
	}
	if req.Filename == "" {
		return fiber.NewError(fiber.StatusBadRequest, "требуется имя файла")
	}

	filename := filepath.Base(req.Filename)
	dir := filepath.Dir(h.c.Cfg.DBDSN)
	fullPath := filepath.Join(dir, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "не удалось открыть файл: "+err.Error())
	}

	res, err := h.c.Orders.ImportFile(filename, data)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(res)
}

func (h *Handler) bounds(c *fiber.Ctx) error {
	min, max, err := h.c.Metrics.DataBounds()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"min": min, "max": max})
}

func (h *Handler) cities(c *fiber.Ctx) error {
	cities, err := h.c.Metrics.Cities()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(cities)
}

func (h *Handler) regions(c *fiber.Ctx) error {
	regions, err := h.c.Metrics.Regions()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(regions)
}

func (h *Handler) channels(c *fiber.Ctx) error {
	channels, err := h.c.Metrics.Channels()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(channels)
}

func (h *Handler) payments(c *fiber.Ctx) error {
	payments, err := h.c.Metrics.Payments()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(payments)
}

func (h *Handler) deliveries(c *fiber.Ctx) error {
	deliveries, err := h.c.Metrics.Deliveries()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(deliveries)
}

func (h *Handler) coupons(c *fiber.Ctx) error {
	coupons, err := h.c.Metrics.Coupons()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(coupons)
}

func (h *Handler) metrics(c *fiber.Ctx) error {
	report, err := h.c.Metrics.Report(c.Query("start"), c.Query("end"), c.Query("compareStart"), c.Query("compareEnd"), metrics.Filters{
		City:     c.Query("city"),
		Region:   c.Query("region"),
		Channel:  c.Query("channel"),
		Payment:  c.Query("payment"),
		Delivery: c.Query("delivery"),
		Coupon:   c.Query("coupon"),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) funnel(c *fiber.Ctx) error {
	report, err := h.c.Funnel.Report(c.Query("start"), c.Query("end"), c.Query("compareStart"), c.Query("compareEnd"), funnel.Filters{
		City:     c.Query("city"),
		Region:   c.Query("region"),
		Channel:  c.Query("channel"),
		Payment:  c.Query("payment"),
		Delivery: c.Query("delivery"),
		Coupon:   c.Query("coupon"),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) logistics(c *fiber.Ctx) error {
	report, err := h.c.Logistics.Report(c.Query("start"), c.Query("end"), c.Query("compareStart"), c.Query("compareEnd"), logistics.Filters{
		City:     c.Query("city"),
		Region:   c.Query("region"),
		Channel:  c.Query("channel"),
		Payment:  c.Query("payment"),
		Delivery: c.Query("delivery"),
		Coupon:   c.Query("coupon"),
	}, c.Query("granularity"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) dynamics(c *fiber.Ctx) error {
	report, err := h.c.Logistics.SeriesBreakdown(c.Query("start"), c.Query("end"), logistics.Filters{
		City:     c.Query("city"),
		Region:   c.Query("region"),
		Channel:  c.Query("channel"),
		Payment:  c.Query("payment"),
		Delivery: c.Query("delivery"),
		Coupon:   c.Query("coupon"),
	}, c.Query("groupBy"), c.Query("granularity"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func parseYearQuery(raw string) int {
	if raw == "" {
		return time.Now().Year()
	}
	y, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return y
}

func (h *Handler) getPlan(c *fiber.Ctx) error {
	year := parseYearQuery(c.Query("year"))
	if year == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "неверный параметр year")
	}
	report, err := h.c.Plan.Get(year)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) putPlan(c *fiber.Ctx) error {
	var req plan.SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "неверное тело запроса")
	}
	report, err := h.c.Plan.Save(req.Year, req.Items)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) getTraffic(c *fiber.Ctx) error {
	year := parseYearQuery(c.Query("year"))
	if year == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "неверный параметр year")
	}
	report, err := h.c.Traffic.Get(year)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) putTraffic(c *fiber.Ctx) error {
	var req traffic.SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "неверное тело запроса")
	}
	report, err := h.c.Traffic.Save(req.Year, req.Items)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}
