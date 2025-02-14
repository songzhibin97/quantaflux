package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/songzhibin97/quantaflux/internal/utils/request"

	"github.com/songzhibin97/quantaflux/internal/models"
)

type BinanceDataSource struct {
	baseURL    string
	httpClient *resty.Client
}

func NewBinanceDataSource() *BinanceDataSource {
	return &BinanceDataSource{
		baseURL:    "https://api.binance.com",
		httpClient: request.Request,
	}
}

func (b *BinanceDataSource) Name() string {
	return "binance"
}

func (b *BinanceDataSource) CollectTokenInfo(ctx context.Context, symbol string) (*models.TokenInfo, error) {
	// Binance API doesn't provide comprehensive token info
	// We'll only get what's available from the symbol info endpoint
	url := fmt.Sprintf("%s/api/v3/exchangeInfo?symbol=%s", b.baseURL, symbol)

	resp, err := b.httpClient.R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	var result struct {
		Symbols []struct {
			Symbol     string `json:"symbol"`
			BaseAsset  string `json:"baseAsset"`
			QuoteAsset string `json:"quoteAsset"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Symbols) == 0 {
		return nil, fmt.Errorf("symbol not found")
	}

	return &models.TokenInfo{
		Symbol: result.Symbols[0].BaseAsset,
		Name:   result.Symbols[0].BaseAsset,
	}, nil
}

func (b *BinanceDataSource) CollectMarketData(ctx context.Context, symbol string) (*models.MarketData, error) {
	// Use 24hr ticker price change statistics endpoint
	url := fmt.Sprintf("%s/api/v3/ticker/24hr?symbol=%s", b.baseURL, symbol)

	resp, err := b.httpClient.R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	var ticker struct {
		LastPrice          string `json:"lastPrice"`
		Volume             string `json:"volume"`
		PriceChangePercent string `json:"priceChangePercent"`
		QuoteVolume        string `json:"quoteVolume"`
	}

	if err := json.Unmarshal(resp.Body(), &ticker); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	price, err := strconv.ParseFloat(ticker.LastPrice, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	volume, err := strconv.ParseFloat(ticker.Volume, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume: %w", err)
	}

	priceChange, err := strconv.ParseFloat(ticker.PriceChangePercent, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price change: %w", err)
	}

	return &models.MarketData{
		Symbol:         symbol,
		Price:          price,
		Volume24h:      volume,
		PriceChange24h: priceChange,
		Timestamp:      time.Now(),
	}, nil
}

func (b *BinanceDataSource) CollectSocialMetrics(ctx context.Context, symbol string) (map[string]float64, error) {
	// Binance doesn't provide social metrics directly
	// This is a placeholder that could be implemented by combining with other APIs
	return map[string]float64{}, nil
}
