// Package handlers — HTTP-слой (Fiber). Принимает запросы, валидирует вход и
// делегирует доменным сервисам.
package handlers

import (
	"bytes"
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
	api.Post("/import/uploads", h.createImportUpload)
	api.Put("/import/uploads/:id/chunks/:index", h.uploadImportChunk)
	api.Post("/import/uploads/:id/complete", h.completeImportUpload)
	api.Delete("/import/uploads/:id", h.deleteImportUpload)
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
	api.Get("/goal", h.goal)
	api.Get("/plan", h.getPlan)
	api.Put("/plan", h.putPlan)
	api.Get("/traffic", h.getTraffic)
	api.Put("/traffic", h.putTraffic)
	api.Get("/acquisition", h.acquisition)
	api.Get("/analytics/status", h.analyticsStatus)
}

func (h *Handler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// importFile принимает multipart-файл выгрузки Битрикса (поле "file")
// и накопительно добавляет новые либо обновляет совпавшие заказы.
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
	res, err := h.c.Orders.Import(fh.Filename, f)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(res)
}

// createImportUpload создаёт сессию. Клиент получает безопасный размер части,
// который проходит через ограничения внешнего reverse proxy.
func (h *Handler) createImportUpload(c *fiber.Ctx) error {
	var req struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "неверное тело запроса")
	}
	session, err := h.c.ImportUploads.Create(req.Filename, req.Size)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(session)
}

func (h *Handler) uploadImportChunk(c *fiber.Ctx) error {
	index, err := strconv.Atoi(c.Params("index"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "неверный номер части")
	}
	if err := h.c.ImportUploads.SaveChunk(c.Params("id"), index, bytes.NewReader(c.Body())); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) completeImportUpload(c *fiber.Ctx) error {
	upload, err := h.c.ImportUploads.Open(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	defer upload.Close()
	result, err := h.c.Orders.Import(upload.Filename, upload.Reader())
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(result)
}

func (h *Handler) deleteImportUpload(c *fiber.Ctx) error {
	if err := h.c.ImportUploads.Delete(c.Params("id")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
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

	file, err := os.Open(fullPath)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "не удалось открыть файл: "+err.Error())
	}
	defer file.Close()

	res, err := h.c.Orders.Import(filename, file)
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

func parseMonthQuery(raw string) int {
	if raw == "" {
		return int(time.Now().Month())
	}
	month, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return month
}

func (h *Handler) goal(c *fiber.Ctx) error {
	year := parseYearQuery(c.Query("year"))
	month := parseMonthQuery(c.Query("month"))
	if year == 0 || month == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "неверные параметры year или month")
	}
	report, err := h.c.Goal.Report(year, month)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
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

func (h *Handler) acquisition(c *fiber.Ctx) error {
	report, err := h.c.Acquisition.Report(
		c.Query("start"), c.Query("end"), c.Query("compareStart"), c.Query("compareEnd"),
	)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) analyticsStatus(c *fiber.Ctx) error {
	report, err := h.c.TrafficSync.Status()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(report)
}
