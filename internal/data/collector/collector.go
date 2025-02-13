package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/songzhibin97/quantaflux/internal/models"
)

// MultiSourceCollector implements DataCollector interface by aggregating multiple data sources
type MultiSourceCollector struct {
	sources []DataSource
	logger  Logger
}

type Logger interface {
	Error(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

type DataSource interface {
	Name() string
	CollectTokenInfo(ctx context.Context, symbol string) (*models.TokenInfo, error)
	CollectMarketData(ctx context.Context, symbol string) (*models.MarketData, error)
	CollectSocialMetrics(ctx context.Context, symbol string) (map[string]float64, error)
}

func NewMultiSourceCollector(sources []DataSource, logger Logger) *MultiSourceCollector {
	return &MultiSourceCollector{
		sources: sources,
		logger:  logger,
	}
}

// CollectTokenInfo implements DataCollector interface
func (c *MultiSourceCollector) CollectTokenInfo(ctx context.Context, symbol string) (*models.TokenInfo, error) {
	var result *models.TokenInfo
	var err error

	for _, source := range c.sources {
		result, err = source.CollectTokenInfo(ctx, symbol)
		if err == nil && result != nil {
			c.logger.Info("collected token info", "source", source.Name(), "symbol", symbol)
			return result, nil
		}
		c.logger.Error("failed to collect token info", "source", source.Name(), "error", err)
	}

	return nil, fmt.Errorf("failed to collect token info from all sources")
}

// CollectMarketData implements DataCollector interface
func (c *MultiSourceCollector) CollectMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	var result *models.MarketData
	var err error

	for _, source := range c.sources {
		result, err = source.CollectMarketData(ctx, symbol)
		if err == nil && result != nil {
			c.logger.Info("collected market data", "source", source.Name(), "symbol", symbol)
			return result, nil
		}
		c.logger.Error("failed to collect market data", "source", source.Name(), "error", err)
	}

	return nil, fmt.Errorf("failed to collect market data from all sources")
}

// CollectSocialMetrics implements DataCollector interface
func (c *MultiSourceCollector) CollectSocialMetrics(ctx context.Context, symbol string) (map[string]float64, error) {
	results := make(map[string]float64)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, source := range c.sources {
		wg.Add(1)
		go func(src DataSource) {
			defer wg.Done()

			metrics, err := src.CollectSocialMetrics(ctx, symbol)
			if err != nil {
				c.logger.Error("failed to collect social metrics", "source", src.Name(), "error", err)
				return
			}

			mu.Lock()
			for k, v := range metrics {
				results[k] = v
			}
			mu.Unlock()

			c.logger.Info("collected social metrics", "source", src.Name(), "symbol", symbol)
		}(source)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to collect social metrics from all sources")
	}

	return results, nil
}

// SubscribeToMarketData implements DataCollector interface
func (c *MultiSourceCollector) SubscribeToMarketData(ctx context.Context, symbols []string, refreshInterval time.Duration) (<-chan models.MarketData, error) {
	out := make(chan models.MarketData, 100)
	var wg sync.WaitGroup

	// 启动所有数据源的订阅
	for _, source := range c.sources {
		wg.Add(1)
		go func(src DataSource) {
			defer wg.Done()

			ticker := time.NewTicker(refreshInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					for _, symbol := range symbols {
						data, err := src.CollectMarketData(ctx, symbol)
						if err != nil {
							c.logger.Error("failed to collect market data", "source", src.Name(), "symbol", symbol, "error", err)
							continue
						}

						select {
						case out <- *data:
						default:
							c.logger.Error("channel full, dropping market data", "source", src.Name(), "symbol", symbol)
						}
					}
				}
			}
		}(source)
	}

	// 等待所有goroutine结束后关闭channel
	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
