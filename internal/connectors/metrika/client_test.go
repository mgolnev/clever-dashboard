package metrika

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchParsesDailyVisits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "OAuth secret" {
			t.Fatalf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("metrics") != "ym:s:visits,ym:s:users" || r.URL.Query().Get("ids") != "42" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[123.2,98]}],
			"sampled":true,"sample_share":0.5
		}`))
	}))
	defer server.Close()

	client := New("42", "secret", "Europe/Moscow")
	client.endpoint = server.URL
	items, err := client.Fetch(context.Background(), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Sessions != 123 || items[0].Users != 98 || !items[0].Sampled || items[0].SampleShare != 0.5 {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestFetchEcommerceParsesDailyUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "ym:s:productImpressionsUniq,ym:s:productBasketsUniq,ym:s:productBeginCheckoutUniq,ym:s:productPurchasedUniq"
		if r.URL.Query().Get("metrics") != want || r.URL.Query().Get("dimensions") != "ym:s:date" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"data":[{"dimensions":[{"name":"2026-08-10"}],"metrics":[6088,680,204,70]}],
			"sampled":false,"sample_share":1
		}`))
	}))
	defer server.Close()

	client := New("42", "secret", "Europe/Moscow")
	client.endpoint = server.URL
	items, err := client.FetchEcommerce(context.Background(),
		time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ProductViewUsers != 6088 || items[0].AddToCartUsers != 680 ||
		items[0].BeginCheckoutUsers != 204 || items[0].TrackedPurchaseUsers != 70 {
		t.Fatalf("unexpected items: %+v", items)
	}
}
