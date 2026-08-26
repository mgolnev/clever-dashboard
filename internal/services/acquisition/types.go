package acquisition

type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

type ChannelMetrics struct {
	Channel              string           `json:"channel"`
	Label                string           `json:"label"`
	Sessions             int              `json:"sessions"`
	Users                int              `json:"users"`
	Orders               int              `json:"orders"`
	PaidOrders           int              `json:"paidOrders"`
	NetOrders            int              `json:"netOrders"`
	OrderCR              float64          `json:"orderCr"`
	PaidCR               float64          `json:"paidCr"`
	NetCR                float64          `json:"netCr"`
	EcommerceAvailable   bool             `json:"ecommerceAvailable"`
	TrackedPurchaseUsers int              `json:"trackedPurchaseUsers"`
	EcommerceFunnel      []EcommerceStage `json:"ecommerceFunnel"`
}

type EcommerceStage struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Count        int      `json:"count"`
	Unit         string   `json:"unit"`
	FromPrevious float64  `json:"fromPrevious"`
	FromCreated  *float64 `json:"fromCreated,omitempty"`
}

type DailyPoint struct {
	Day            string `json:"day"`
	SiteSessions   int    `json:"siteSessions"`
	AppSessions    int    `json:"appSessions"`
	SiteUsers      int    `json:"siteUsers"`
	AppUsers       int    `json:"appUsers"`
	SiteOrders     int    `json:"siteOrders"`
	AppOrders      int    `json:"appOrders"`
	SitePaidOrders int    `json:"sitePaidOrders"`
	AppPaidOrders  int    `json:"appPaidOrders"`
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
