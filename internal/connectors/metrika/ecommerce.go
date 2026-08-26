package metrika

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/clever/clever-dashboard/internal/model"
)

const metrikaEcommerceRevision = "ecommerce-users-v1"

func (c *Client) EcommerceName() string     { return "metrika-ecommerce" }
func (c *Client) EcommerceRevision() string { return metrikaEcommerceRevision }

// FetchEcommerce получает дневную аудиторию стандартных E-commerce событий.
// purchase сохраняется только для диагностики; заказами управляет Битрикс.
func (c *Client) FetchEcommerce(ctx context.Context, from, to time.Time) ([]model.DailyEcommerce, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("Яндекс Метрика не настроена")
	}
	q := url.Values{
		"ids":        {c.counterID},
		"date1":      {from.Format("2006-01-02")},
		"date2":      {to.Format("2006-01-02")},
		"dimensions": {"ym:s:date"},
		"metrics": {strings.Join([]string{
			"ym:s:productImpressionsUniq",
			"ym:s:productBasketsUniq",
			"ym:s:productBeginCheckoutUniq",
			"ym:s:productPurchasedUniq",
		}, ",")},
		"accuracy": {"full"},
		"limit":    {"10000"},
		"lang":     {"ru"},
	}
	if c.timezone != "" {
		q.Set("timezone", c.timezone)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос E-commerce Метрики: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("E-commerce Метрика HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload reportResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("ответ E-commerce Метрики: %w", err)
	}
	share := payload.SampleShare
	if share == 0 && !payload.Sampled {
		share = 1
	}
	out := make([]model.DailyEcommerce, 0, len(payload.Data))
	for _, row := range payload.Data {
		if len(row.Dimensions) == 0 || len(row.Metrics) < 4 {
			continue
		}
		day := row.Dimensions[0].Name
		if day == "" {
			day = row.Dimensions[0].ID
		}
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		out = append(out, model.DailyEcommerce{
			Day: day, Channel: c.Channel(),
			ProductViewUsers:     int(math.Round(row.Metrics[0])),
			AddToCartUsers:       int(math.Round(row.Metrics[1])),
			BeginCheckoutUsers:   int(math.Round(row.Metrics[2])),
			TrackedPurchaseUsers: int(math.Round(row.Metrics[3])),
			Source:               c.EcommerceName(), Sampled: payload.Sampled, SampleShare: share,
		})
	}
	return out, nil
}
