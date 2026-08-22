package appmetrica

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchParsesDailySessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "OAuth secret" {
			t.Fatalf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		q := r.URL.Query()
		if q.Get("metrics") != "ym:s:sessions" || q.Get("ids") != "84" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		for key, want := range map[string]string{
			"group": "Day", "dimensions": "ym:s:date", "accuracy": "medium",
			"include_undefined": "true", "currency": "RUB", "sort": "-ym:s:sessions",
			"lang": "ru", "request_domain": "ru",
		} {
			if got := q.Get(key); got != want {
				t.Fatalf("query %s = %q, want %q; raw query: %s", key, got, want, r.URL.RawQuery)
			}
		}
		if q.Has("timezone") {
			t.Fatalf("timezone must come from AppMetrica application settings: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[77]}],
			"sampled":false,"sample_share":1
		}`))
	}))
	defer server.Close()

	client := New("84", "secret")
	client.endpoint = server.URL
	items, err := client.Fetch(context.Background(), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Sessions != 77 || items[0].Source != "appmetrica" || items[0].Channel != "app" {
		t.Fatalf("unexpected items: %+v", items)
	}
}
