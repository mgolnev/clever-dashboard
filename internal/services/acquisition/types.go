package acquisition

type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

type ChannelMetrics struct {
	Channel    string  `json:"channel"`
	Label      string  `json:"label"`
	Sessions   int     `json:"sessions"`
	Orders     int     `json:"orders"`
	PaidOrders int     `json:"paidOrders"`
	NetOrders  int     `json:"netOrders"`
	OrderCR    float64 `json:"orderCr"`
	PaidCR     float64 `json:"paidCr"`
	NetCR      float64 `json:"netCr"`
}

type DailyPoint struct {
	Day          string `json:"day"`
	SiteSessions int    `json:"siteSessions"`
	AppSessions  int    `json:"appSessions"`
	SiteOrders   int    `json:"siteOrders"`
	AppOrders    int    `json:"appOrders"`
}

type PeriodData struct {
	Channels   []ChannelMetrics `json:"channels"`
	Daily      []DailyPoint     `json:"daily"`
	HasTraffic bool             `json:"hasTraffic"`
	Sampled    bool             `json:"sampled"`
}

type Report struct {
	Period   Range      `json:"period"`
	Previous Range      `json:"previous"`
	Current  PeriodData `json:"current"`
	Prev     PeriodData `json:"prev"`
}
