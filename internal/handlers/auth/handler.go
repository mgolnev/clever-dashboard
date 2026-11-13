package auth

import (
	"time"

	"github.com/clever/clever-dashboard/internal/services/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

const cookieName = "clever_session"

type Handler struct {
	service *auth.Service
	secure  bool
}

func Register(app *fiber.App, service *auth.Service, secure bool) {
	h := &Handler{service, secure}
	app.Use("/api", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		// Custom header + no CORS prevents browser cross-origin mutations, including login CSRF.
		if c.Method() != "GET" && c.Method() != "HEAD" && c.Method() != "OPTIONS" && c.Get("X-Requested-With") != "CleverDashboard" {
			return fiber.NewError(403, "Недопустимый запрос")
		}
		return c.Next()
	})
	app.Post("/api/auth/login", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute, LimitReached: func(c *fiber.Ctx) error {
		return fiber.NewError(429, "Слишком много попыток. Повторите через минуту")
	}}), h.login)
	app.Get("/api/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Use("/api", func(c *fiber.Ctx) error {
		if !service.Valid(c.Cookies(cookieName)) {
			return fiber.NewError(401, "Необходим вход в сервис")
		}
		return c.Next()
	})
	app.Get("/api/auth/session", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"username": service.Username()}) })
	app.Post("/api/auth/logout", h.logout)
}

func (h *Handler) cookie(value string, maxAge int, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{Name: cookieName, Value: value, Path: "/", HTTPOnly: true, Secure: h.secure, SameSite: "Strict", MaxAge: maxAge, Expires: expires}
}
func (h *Handler) login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if len(c.Body()) > 4096 || c.BodyParser(&req) != nil {
		return fiber.NewError(400, "Некорректный запрос")
	}
	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		return fiber.NewError(401, "Неверный логин или пароль")
	}
	h.service.Logout(c.Cookies(cookieName))
	c.Cookie(h.cookie(token, int(auth.Lifetime.Seconds()), time.Now().Add(auth.Lifetime)))
	return c.JSON(fiber.Map{"username": h.service.Username()})
}
func (h *Handler) logout(c *fiber.Ctx) error {
	h.service.Logout(c.Cookies(cookieName))
	c.Cookie(h.cookie("", -1, time.Unix(1, 0)))
	return c.SendStatus(204)
}
