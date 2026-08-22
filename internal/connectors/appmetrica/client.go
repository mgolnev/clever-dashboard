// Package appmetrica получает обезличенные дневные агрегаты приложения через
// Reporting API AppMetrica.
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

const endpoint = "https://api.appmetrica.yandex.com/stat/v1/data"

type Client struct {
	applicationID string
	token         string
	timezone      string
	endpoint      string
	http          *http.Client
}

func New(applicationID, token, timezone string) *Client {
	return &Client{
		applicationID: strings.TrimSpace(applicationID),
		token:         strings.TrimSpace(token),
		timezone:      timezone,
		endpoint:      endpoint,
		http:          &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Name() string     { return "appmetrica" }
func (c *Client) Channel() string  { return "app" }
func (c *Client) Configured() bool { return c.applicationID != "" && c.token != "" }

type reportResponse struct {
	Data []struct {
		Dimensions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"dimensions"`
		Metrics []float64 `json:"metrics"`
	} `json:"data"`
	Sampled     bool    `json:"sampled"`
	SampleShare float64 `json:"sample_share"`
}

func (c *Client) Fetch(ctx context.Context, from, to time.Time) ([]model.DailyTraffic, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("AppMetrica не настроена")
	}
	q := url.Values{
		"ids":        {c.applicationID},
		"date1":      {from.Format("2006-01-02")},
		"date2":      {to.Format("2006-01-02")},
		"dimensions": {"ym:s:date"},
		"metrics":    {"ym:s:sessions"},
		"accuracy":   {"full"},
		"limit":      {"10000"},
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
		return nil, fmt.Errorf("запрос AppMetrica: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("AppMetrica HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload reportResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("ответ AppMetrica: %w", err)
	}
	share := payload.SampleShare
	if share == 0 && !payload.Sampled {
		share = 1
	}
	out := make([]model.DailyTraffic, 0, len(payload.Data))
	for _, row := range payload.Data {
		if len(row.Dimensions) == 0 || len(row.Metrics) == 0 {
			continue
		}
		day := row.Dimensions[0].Name
		if day == "" {
			day = row.Dimensions[0].ID
		}
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		out = append(out, model.DailyTraffic{
			Day: day, Channel: c.Channel(), Sessions: int(math.Round(row.Metrics[0])),
			Source: c.Name(), Sampled: payload.Sampled, SampleShare: share,
		})
	}
	return out, nil
}
