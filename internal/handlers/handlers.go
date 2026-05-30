// Package handlers — HTTP-слой (Fiber). Принимает запросы, валидирует вход и
// делегирует доменным сервисам.
package handlers

import (
	"io"

	"github.com/clever/clever-dashboard/internal/container"
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
	api.Get("/bounds", h.bounds)
	api.Get("/cities", h.cities)
	api.Get("/regions", h.regions)
	api.Get("/metrics", h.metrics)
	api.Get("/funnel", h.funnel)
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

func (h *Handler) metrics(c *fiber.Ctx) error {
	report, err := h.c.Metrics.Report(c.Query("start"), c.Query("end"), c.Query("city"), c.Query("region"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}

func (h *Handler) funnel(c *fiber.Ctx) error {
	report, err := h.c.Funnel.Report(c.Query("start"), c.Query("end"), c.Query("city"), c.Query("region"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(report)
}
