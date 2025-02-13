package data

import (
	"context"
	"time"

	"github.com/songzhibin97/quantaflux/internal/models"
)

// DataCollector 负责从各种源收集数据
type DataCollector interface {
	// CollectTokenInfo retrieves basic token information
	CollectTokenInfo(ctx context.Context, symbol string) (*models.TokenInfo, error)

	// CollectMarketData retrieves real-time market data
	CollectMarketData(ctx context.Context, symbol string) (*models.MarketData, error)

	// CollectSocialMetrics retrieves social media metrics
	CollectSocialMetrics(ctx context.Context, symbol string) (map[string]float64, error)

	// SubscribeToMarketData returns a channel for real-time market updates
	SubscribeToMarketData(ctx context.Context, symbols []string, refreshInterval time.Duration) (<-chan models.MarketData, error)
}

// DataStorage 处理数据的持久化
type DataStorage interface {
	// SaveTokenInfo stores token information
	SaveTokenInfo(ctx context.Context, info *models.TokenInfo) error

	// SaveMarketData stores market data
	SaveMarketData(ctx context.Context, data *models.MarketData) error

	// GetHistoricalData retrieves historical market data
	GetHistoricalData(ctx context.Context, symbol string, start, end time.Time) ([]models.MarketData, error)

	// GetProjectMetrics retrieves project metrics
	GetProjectMetrics(ctx context.Context, symbol string) (*models.ProjectMetrics, error)
}
