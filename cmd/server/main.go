package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/container"
	"github.com/clever/clever-dashboard/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.Load()

	c, err := container.New(cfg)
	if err != nil {
		log.Fatalf("init container: %v", err)
	}
	defer c.Close()

	app := fiber.New(fiber.Config{
		BodyLimit:    64 * 1024 * 1024, // выгрузки Битрикса бывают крупными
		AppName:      "CLEVER Dashboard",
		ErrorHandler: errorHandler,
	})
	app.Use(logger.New())
	app.Use(cors.New())

	handlers.New(c).Register(app)

	if cfg.StaticDir != "" {
		serveStatic(app, cfg.StaticDir)
	}

	log.Printf("CLEVER Dashboard backend на :%s (db=%s, static=%q)", cfg.Port, cfg.DBDriver, cfg.StaticDir)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// serveStatic отдаёт собранный фронтенд (Vite dist) как SPA: статические файлы
// напрямую, а любой неизвестный GET-маршрут (кроме /api) — index.html.
func serveStatic(app *fiber.App, dir string) {
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		log.Printf("STATIC_DIR=%q задан, но %s не найден — фронт не будет обслуживаться", dir, index)
		return
	}
	app.Static("/", dir, fiber.Static{Index: "index.html"})
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodGet || strings.HasPrefix(c.Path(), "/api") {
			return fiber.ErrNotFound
		}
		return c.SendFile(index)
	})
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
