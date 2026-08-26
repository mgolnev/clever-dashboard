package appmetrica

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestFetchParsesDailySessionsAndUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "OAuth secret" {
			t.Fatalf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		q := r.URL.Query()
		if q.Get("metrics") != "ym:s:sessions,ym:s:users" || q.Get("ids") != "84" {
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
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[77,31]}],
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
	if len(items) != 1 || items[0].Sessions != 77 || items[0].Users != 31 || items[0].Source != "appmetrica" || items[0].Channel != "app" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestFetchSplitsLongRangeIntoWeeklyRequests(t *testing.T) {
	var mu sync.Mutex
	var ranges [][2]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		date1 := r.URL.Query().Get("date1")
		date2 := r.URL.Query().Get("date2")
		mu.Lock()
		ranges = append(ranges, [2]string{date1, date2})
		mu.Unlock()
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"` + date1 + `"}],"metrics":[1]}],
			"sampled":false,"sample_share":1
		}`))
	}))
	defer server.Close()

	client := New("84", "secret")
	client.endpoint = server.URL
	items, err := client.Fetch(context.Background(),
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{
		{"2026-08-01", "2026-08-07"},
		{"2026-08-08", "2026-08-14"},
		{"2026-08-15", "2026-08-16"},
	}
	if len(items) != len(want) || len(ranges) != len(want) {
		t.Fatalf("items=%d ranges=%v, want %d ranges", len(items), ranges, len(want))
	}
	for i := range want {
		if ranges[i] != want[i] {
			t.Fatalf("range %d = %v, want %v", i, ranges[i], want[i])
		}
	}
}

func TestFetchEcommerceParsesDailyUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		want := "ym:ec2:ecomUsersProductDetailsView,ym:ec2:ecomUsersAddToCart,ym:ec2:ecomUsersBeginCheckout,ym:ec2:ecomPayingUsers"
		if q.Get("metrics") != want || q.Get("dimensions") != "ym:ec2:date" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[2340,1761,629,104]}],
			"sampled":false,"sample_share":1
		}`))
	}))
	defer server.Close()

	client := New("84", "secret")
	client.endpoint = server.URL
	items, err := client.FetchEcommerce(context.Background(),
		time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ProductViewUsers != 2340 || items[0].AddToCartUsers != 1761 ||
		items[0].BeginCheckoutUsers != 629 || items[0].TrackedPurchaseUsers != 104 {
		t.Fatalf("unexpected items: %+v", items)
	}
}
