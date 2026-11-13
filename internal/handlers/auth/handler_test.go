package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clever/clever-dashboard/internal/services/auth"
	"github.com/gofiber/fiber/v2"
)

func TestHTTPAccess(t *testing.T) {
	s, _ := auth.New("admin", "test-password-123")
	app := fiber.New()
	Register(app, s, true)
	app.All("/api/private", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	request := func(method, path, body string, cookie *http.Cookie, csrf bool) *http.Response {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if csrf {
			r.Header.Set("X-Requested-With", "CleverDashboard")
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		res, err := app.Test(r)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		return res
	}
	check := func(res *http.Response, status int) {
		t.Helper()
		if res.StatusCode != status {
			t.Fatalf("status %d, want %d", res.StatusCode, status)
		}
	}
	check(request("GET", "/api/health", "", nil, false), 200)
	for _, path := range []string{"/api/private", "/api/auth/session", "/api/import/local", "/api/metrics"} {
		check(request("GET", path, "", nil, false), 401)
	}
	credentials := `{"username":"admin","password":"test-password-123"}`
	check(request("POST", "/api/auth/login", credentials, nil, false), 403)
	check(request("POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, true), 401)
	login := request("POST", "/api/auth/login", credentials, nil, true)
	check(login, 200)
	if len(login.Cookies()) != 1 {
		t.Fatal("missing session cookie")
	}
	cookie := login.Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 43200 {
		t.Fatalf("unsafe cookie: %+v", cookie)
	}
	check(request("GET", "/api/private", "", cookie, false), 200)
	check(request("POST", "/api/private", "", cookie, false), 403)
	check(request("POST", "/api/private", "", cookie, true), 200)
	check(request("POST", "/api/auth/logout", "", cookie, true), 204)
	check(request("GET", "/api/private", "", cookie, false), 401)
	for i := 0; i < 10; i++ {
		request("POST", "/api/auth/login", `{}`, nil, true)
	}
	check(request("POST", "/api/auth/login", credentials, nil, true), 429)
}
