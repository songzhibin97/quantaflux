package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, path string, response interface{}) (*httptest.Server, *BinanceDataSource) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		require.NoError(t, err)
	}))

	binanceDS := NewBinanceDataSource()
	binanceDS.baseURL = server.URL
	binanceDS.httpClient = resty.NewWithClient(server.Client())

	return server, binanceDS
}

func TestBinanceDataSource_Name(t *testing.T) {
	ds := NewBinanceDataSource()
	assert.Equal(t, "binance", ds.Name())
}

func TestBinanceDataSource_CollectTokenInfo(t *testing.T) {
	tests := []struct {
		name        string
		symbol      string
		response    interface{}
		expectError bool
		expected    struct {
			symbol string
			name   string
		}
	}{
		{
			name:   "valid response",
			symbol: "BTCUSDT",
			response: struct {
				Symbols []struct {
					Symbol     string `json:"symbol"`
					BaseAsset  string `json:"baseAsset"`
					QuoteAsset string `json:"quoteAsset"`
				} `json:"symbols"`
			}{
				Symbols: []struct {
					Symbol     string `json:"symbol"`
					BaseAsset  string `json:"baseAsset"`
					QuoteAsset string `json:"quoteAsset"`
				}{
					{
						Symbol:     "BTCUSDT",
						BaseAsset:  "BTC",
						QuoteAsset: "USDT",
					},
				},
			},
			expectError: false,
			expected: struct {
				symbol string
				name   string
			}{
				symbol: "BTC",
				name:   "BTC",
			},
		},
		{
			name:        "empty response",
			symbol:      "INVALID",
			response:    struct{ Symbols []struct{} }{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, ds := setupTestServer(t, "/api/v3/exchangeInfo", tt.response)
			defer server.Close()

			ctx := context.Background()
			info, err := ds.CollectTokenInfo(ctx, tt.symbol)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.symbol, info.Symbol)
			assert.Equal(t, tt.expected.name, info.Name)
		})
	}
}

func TestBinanceDataSource_CollectMarketData(t *testing.T) {
	tests := []struct {
		name        string
		symbol      string
		response    interface{}
		expectError bool
		expected    struct {
			price          float64
			volume         float64
			priceChange24h float64
		}
	}{
		{
			name:   "valid response",
			symbol: "BTCUSDT",
			response: struct {
				LastPrice          string `json:"lastPrice"`
				Volume             string `json:"volume"`
				PriceChangePercent string `json:"priceChangePercent"`
				QuoteVolume        string `json:"quoteVolume"`
			}{
				LastPrice:          "50000.00",
				Volume:             "1000.50",
				PriceChangePercent: "2.5",
				QuoteVolume:        "50000000.00",
			},
			expectError: false,
			expected: struct {
				price          float64
				volume         float64
				priceChange24h float64
			}{
				price:          50000.00,
				volume:         1000.50,
				priceChange24h: 2.5,
			},
		},
		{
			name:   "invalid number format",
			symbol: "BTCUSDT",
			response: struct {
				LastPrice          string `json:"lastPrice"`
				Volume             string `json:"volume"`
				PriceChangePercent string `json:"priceChangePercent"`
			}{
				LastPrice:          "invalid",
				Volume:             "1000.50",
				PriceChangePercent: "2.5",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, ds := setupTestServer(t, "/api/v3/ticker/24hr", tt.response)
			defer server.Close()

			ctx := context.Background()
			data, err := ds.CollectMarketData(ctx, tt.symbol)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, data)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.symbol, data.Symbol)
			assert.Equal(t, tt.expected.price, data.Price)
			assert.Equal(t, tt.expected.volume, data.Volume24h)
			assert.Equal(t, tt.expected.priceChange24h, data.PriceChange24h)
			assert.WithinDuration(t, time.Now(), data.Timestamp, 2*time.Second)
		})
	}
}

func TestBinanceDataSource_CollectSocialMetrics(t *testing.T) {
	ds := NewBinanceDataSource()
	ctx := context.Background()

	metrics, err := ds.CollectSocialMetrics(ctx, "BTCUSDT")
	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "social metrics not available")
}

func TestBinanceDataSource_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		statusCode  int
		response    interface{}
		method      string
		expectError bool
	}{
		{
			name:        "http 404 error",
			path:        "/api/v3/ticker/24hr",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "http 429 rate limit",
			path:        "/api/v3/ticker/24hr",
			statusCode:  http.StatusTooManyRequests,
			expectError: true,
		},
		{
			name:        "invalid json response",
			path:        "/api/v3/ticker/24hr",
			statusCode:  http.StatusOK,
			response:    "invalid json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.path, r.URL.Path)
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					_, err := w.Write([]byte(tt.response.(string)))
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			ds := NewBinanceDataSource()
			ds.baseURL = server.URL
			ds.httpClient = resty.NewWithClient(server.Client())

			ctx := context.Background()
			_, err := ds.CollectMarketData(ctx, "BTCUSDT")
			assert.Error(t, err)
		})
	}
}

func TestBinanceIntegration(t *testing.T) {
	// 如果设置了 -short 标志,跳过集成测试
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ds := NewBinanceDataSource()
	ctx := context.Background()

	// 测试一些常见的交易对
	testSymbols := []string{
		"BTCUSDT", // Bitcoin
		"ETHUSDT", // Ethereum
		"BNBUSDT", // Binance Coin
	}

	t.Run("collect token info for multiple symbols", func(t *testing.T) {
		for _, symbol := range testSymbols {
			t.Run(symbol, func(t *testing.T) {
				info, err := ds.CollectTokenInfo(ctx, symbol)
				require.NoError(t, err)
				require.NotNil(t, info)

				assert.NotEmpty(t, info.Symbol)
				assert.NotEmpty(t, info.Name)

				// 记录获取到的信息,方便调试
				t.Logf("Token Info for %s: %+v", symbol, info)
			})

			// 避免触发 API 限制
			time.Sleep(time.Second)
		}
	})

	t.Run("collect market data for multiple symbols", func(t *testing.T) {
		for _, symbol := range testSymbols {
			t.Run(symbol, func(t *testing.T) {
				data, err := ds.CollectMarketData(ctx, symbol)
				require.NoError(t, err)
				require.NotNil(t, data)

				assert.Equal(t, symbol, data.Symbol)
				assert.Greater(t, data.Price, 0.0)
				assert.Greater(t, data.Volume24h, 0.0)
				assert.NotZero(t, data.Timestamp)

				// 记录市场数据,方便调试
				t.Logf("Market Data for %s: %+v", symbol, data)
			})

			// 避免触发 API 限制
			time.Sleep(time.Second)
		}
	})

	t.Run("verify rate limits", func(t *testing.T) {
		// 测试快速连续请求是否会触发限制
		symbol := "BTCUSDT"
		for i := 0; i < 5; i++ {
			data, err := ds.CollectMarketData(ctx, symbol)
			if err != nil {
				t.Logf("Request %d failed: %v", i+1, err)
				// 不将此视为测试失败,因为这可能是预期的限制
				continue
			}
			require.NotNil(t, data)
			// 记录每次请求的时间戳
			t.Logf("Request %d successful at %v", i+1, time.Now())
		}
	})

	t.Run("test invalid symbols", func(t *testing.T) {
		invalidSymbols := []string{
			"INVALIDTOKEN",
			"XXXXYYY",
			"BTC-USDT", // 使用了无效的分隔符
		}

		for _, symbol := range invalidSymbols {
			t.Run(symbol, func(t *testing.T) {
				// 测试 CollectTokenInfo
				info, err := ds.CollectTokenInfo(ctx, symbol)
				assert.Error(t, err)
				assert.Nil(t, info)

				// 测试 CollectMarketData
				data, err := ds.CollectMarketData(ctx, symbol)
				assert.Error(t, err)
				assert.Nil(t, data)

				time.Sleep(time.Second)
			})
		}
	})

	t.Run("test timeout handling", func(t *testing.T) {
		// 创建一个带超时的 context
		ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		symbol := "BTCUSDT"
		_, err := ds.CollectMarketData(ctxTimeout, symbol)

		// 这个测试可能通过也可能失败,取决于网络条件
		if err != nil {
			assert.Contains(t, err.Error(), "context")
			t.Log("Timeout occurred as expected")
		} else {
			t.Log("Request completed before timeout")
		}
	})

	t.Run("test social metrics unavailability", func(t *testing.T) {
		metrics, err := ds.CollectSocialMetrics(ctx, "BTCUSDT")
		assert.Error(t, err)
		assert.Nil(t, metrics)
		assert.Contains(t, err.Error(), "not available")
	})
}

// BenchmarkBinanceDataSource 包含性能测试
func BenchmarkBinanceDataSource(b *testing.B) {
	ds := NewBinanceDataSource()
	ctx := context.Background()
	symbol := "BTCUSDT"

	b.Run("benchmark market data collection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := ds.CollectMarketData(ctx, symbol)
			if err != nil {
				b.Fatal(err)
			}
			// 避免触发 API 限制
			time.Sleep(time.Second)
		}
	})
}
