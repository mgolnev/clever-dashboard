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
		if r.URL.Query().Get("metrics") != "ym:s:sessions" || r.URL.Query().Get("ids") != "84" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[77]}],
			"sampled":false,"sample_share":1
		}`))
	}))
	defer server.Close()

	client := New("84", "secret", "Europe/Moscow")
	client.endpoint = server.URL
	items, err := client.Fetch(context.Background(), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Sessions != 77 || items[0].Source != "appmetrica" || items[0].Channel != "app" {
		t.Fatalf("unexpected items: %+v", items)
	}
}
