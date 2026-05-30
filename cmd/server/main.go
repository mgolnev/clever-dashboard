package main

import (
	"log"

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

	log.Printf("CLEVER Dashboard backend на :%s (db=%s)", cfg.Port, cfg.DBDriver)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
