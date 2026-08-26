package ecomsync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/model"
)

type fakeSource struct{}

func (fakeSource) EcommerceName() string     { return "fake-ecommerce" }
func (fakeSource) EcommerceRevision() string { return "v1" }
func (fakeSource) Channel() string           { return "site" }
func (fakeSource) Configured() bool          { return true }
func (fakeSource) FetchEcommerce(_ context.Context, from, _ time.Time) ([]model.DailyEcommerce, error) {
	return []model.DailyEcommerce{{
		Day: from.Format("2006-01-02"), ProductViewUsers: 10, AddToCartUsers: 4,
	}}, nil
}

func TestSyncPersistsCompleteDays(t *testing.T) {
	d, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(d)
	svc := NewService(repo, []Source{fakeSource{}}, true, 2, 2, "UTC")
	if err := svc.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	var rows, views int
	if err := d.QueryRow(`SELECT COUNT(*), COALESCE(SUM(product_view_users),0)
		FROM analytics_ecommerce_daily WHERE source='fake-ecommerce'`).Scan(&rows, &views); err != nil {
		t.Fatal(err)
	}
	if rows != 2 || views != 10 {
		t.Fatalf("rows=%d views=%d", rows, views)
	}
}
