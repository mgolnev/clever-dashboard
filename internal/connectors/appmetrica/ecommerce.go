package appmetrica

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

const appMetricaEcommerceRevision = "ecommerce-users-weekly-v1"

func (c *Client) EcommerceName() string     { return "appmetrica-ecommerce" }
func (c *Client) EcommerceRevision() string { return appMetricaEcommerceRevision }

func (c *Client) FetchEcommerce(ctx context.Context, from, to time.Time) ([]model.DailyEcommerce, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("AppMetrica не настроена")
	}
	var out []model.DailyEcommerce
	for chunkFrom := from; !chunkFrom.After(to); chunkFrom = chunkFrom.AddDate(0, 0, reportChunkDays) {
		chunkTo := chunkFrom.AddDate(0, 0, reportChunkDays-1)
		if chunkTo.After(to) {
			chunkTo = to
		}
		items, err := c.fetchEcommerceChunk(ctx, chunkFrom, chunkTo)
		if err != nil {
			return nil, fmt.Errorf("AppMetrica E-commerce %s—%s: %w",
				chunkFrom.Format("2006-01-02"), chunkTo.Format("2006-01-02"), err)
		}
		out = append(out, items...)
	}
	return out, nil
}

func (c *Client) fetchEcommerceChunk(ctx context.Context, from, to time.Time) ([]model.DailyEcommerce, error) {
	q := url.Values{
		"ids":        {c.applicationID},
		"date1":      {from.Format("2006-01-02")},
		"date2":      {to.Format("2006-01-02")},
		"group":      {"Day"},
		"dimensions": {"ym:ec2:date"},
		"metrics": {strings.Join([]string{
			"ym:ec2:ecomUsersProductDetailsView",
			"ym:ec2:ecomUsersAddToCart",
			"ym:ec2:ecomUsersBeginCheckout",
			"ym:ec2:ecomPayingUsers",
		}, ",")},
		"accuracy":          {"medium"},
		"include_undefined": {"true"},
		"currency":          {"RUB"},
		"sort":              {"-ym:ec2:ecomUsersProductDetailsView"},
		"lang":              {"ru"},
		"request_domain":    {"ru"},
		"limit":             {"10000"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос E-commerce AppMetrica: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("E-commerce AppMetrica HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload reportResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("ответ E-commerce AppMetrica: %w", err)
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
