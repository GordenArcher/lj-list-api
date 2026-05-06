package models

// DashboardSeriesPoint is a single bucket in the admin analytics chart.
type DashboardSeriesPoint struct {
	Date          string `json:"date"`
	Users         int    `json:"users"`
	Products      int    `json:"products"`
	Applications  int    `json:"applications"`
	Conversations int    `json:"conversations"`
	Messages      int    `json:"messages"`
}

// DashboardStats is the admin overview payload.
type DashboardStats struct {
	Range         string                 `json:"range"`
	From          string                 `json:"from"`
	To            string                 `json:"to"`
	TotalUsers    int                    `json:"total_users"`
	TotalProducts int                    `json:"total_products"`
	TotalApps     int                    `json:"total_applications"`
	TotalConvs    int                    `json:"total_conversations"`
	TotalMessages int                    `json:"total_messages"`
	Series        []DashboardSeriesPoint `json:"series"`
}
