package models

import "time"

// TokenInfo 代币基本信息
type TokenInfo struct {
	Symbol            string    `json:"symbol"`
	Name              string    `json:"name"`
	ContractAddress   string    `json:"contract_address"`
	Network           string    `json:"network"`     // eth, bsc, etc
	LaunchType        string    `json:"launch_type"` // IDO, IEO
	LaunchDate        time.Time `json:"launch_date"`
	InitialPrice      float64   `json:"initial_price"`
	TotalSupply       float64   `json:"total_supply"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TeamAllocation    float64   `json:"team_allocation"`
	VestingSchedule   string    `json:"vesting_schedule"`
}

// ProjectMetrics 项目指标
type ProjectMetrics struct {
	TokenInfo        TokenInfo `json:"token_info"`
	SocialScore      float64   `json:"social_score"`
	DevelopmentScore float64   `json:"development_score"`
	CommunityGrowth  float64   `json:"community_growth"`
	MarketSentiment  float64   `json:"market_sentiment"`
	RiskScore        float64   `json:"risk_score"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// MarketData 市场数据
type MarketData struct {
	Symbol         string    `json:"symbol"`
	Price          float64   `json:"price"`
	Volume24h      float64   `json:"volume_24h"`
	MarketCap      float64   `json:"market_cap"`
	PriceChange1h  float64   `json:"price_change_1h"`
	PriceChange24h float64   `json:"price_change_24h"`
	Timestamp      time.Time `json:"timestamp"`
}
