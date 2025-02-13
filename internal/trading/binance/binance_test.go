package binance

import (
	"context"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2"

	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/quantaflux/internal/trading"
)

func init() {
	binance.UseTestnet = true
}

func roundToStepSize(value float64, stepSize float64) float64 {
	return math.Floor(value/stepSize) * stepSize
}

func adjustPrice(price float64, minPrice, maxPrice, tickSize float64) float64 {
	if price < minPrice {
		return minPrice
	}
	if price > maxPrice {
		return maxPrice
	}
	return math.Floor(price/tickSize) * tickSize
}

func TestBinanceExecutor_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	const (
		SYMBOL = "BTCUSDT" // 使用常量定义 symbol
	)

	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

	executor := NewBinanceExecutor(apiKey, secretKey)
	ctx := context.Background()

	// 获取交易对信息
	exchangeInfo, err := executor.client.NewExchangeInfoService().Symbol(SYMBOL).Do(ctx)
	require.NoError(t, err)

	// 获取价格和数量的限制
	var minQty, tickSize, minPrice, maxPrice float64
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Symbol == SYMBOL {
			for _, filter := range symbol.Filters {
				switch filter["filterType"].(string) {
				case "LOT_SIZE":
					minQty, _ = strconv.ParseFloat(filter["minQty"].(string), 64)
					stepSize, _ := strconv.ParseFloat(filter["stepSize"].(string), 64)
					minQty = math.Max(minQty, stepSize)
				case "PRICE_FILTER":
					minPrice, _ = strconv.ParseFloat(filter["minPrice"].(string), 64)
					maxPrice, _ = strconv.ParseFloat(filter["maxPrice"].(string), 64)
					tickSize, _ = strconv.ParseFloat(filter["tickSize"].(string), 64)
				}
			}
			break
		}
	}

	t.Run("Test Get Balance", func(t *testing.T) {
		balance, err := executor.GetBalance(ctx, "BTC")
		require.NoError(t, err)
		require.GreaterOrEqual(t, balance, 0.0)
	})

	t.Run("Test Place Market Order", func(t *testing.T) {
		amount := roundToStepSize(0.001, minQty)

		order := &trading.Order{
			Symbol:    SYMBOL,
			Side:      "buy",
			Amount:    amount,
			OrderType: "market",
		}

		err := executor.PlaceOrder(ctx, order)
		require.NoError(t, err)
		require.NotEmpty(t, order.Status)
		require.NotEmpty(t, order.OrderID)
	})

	t.Run("Test Place and Cancel Limit Order", func(t *testing.T) {
		// 获取当前市场价格
		ticker, err := executor.client.NewListPricesService().Symbol(SYMBOL).Do(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, ticker)

		currentPrice, err := strconv.ParseFloat(ticker[0].Price, 64)
		require.NoError(t, err)

		// 设置价格为当前价格的95%，并应用价格过滤器
		limitPrice := currentPrice * 0.95
		limitPrice = adjustPrice(limitPrice, minPrice, maxPrice, tickSize)

		// 设置数量
		amount := roundToStepSize(0.001, minQty)

		order := &trading.Order{
			Symbol:    SYMBOL,
			Side:      "buy",
			Amount:    amount,
			Price:     limitPrice,
			OrderType: "limit",
		}

		err = executor.PlaceOrder(ctx, order)
		require.NoError(t, err)
		require.NotEmpty(t, order.OrderID)
		require.NotEmpty(t, order.RawOrderID)

		time.Sleep(2 * time.Second)

		orderStatus, err := executor.GetOrderStatus(ctx, SYMBOL, order.OrderID)
		require.NoError(t, err)
		require.Equal(t, "NEW", orderStatus.Status)

		err = executor.CancelOrder(ctx, SYMBOL, order.OrderID)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)
		orderStatus, err = executor.GetOrderStatus(ctx, SYMBOL, order.OrderID)
		require.NoError(t, err)
		require.Equal(t, "CANCELED", orderStatus.Status)
	})

	t.Run("Test Order Status", func(t *testing.T) {
		amount := roundToStepSize(0.001, minQty)

		order := &trading.Order{
			Symbol:    SYMBOL,
			Side:      "buy",
			Amount:    amount,
			OrderType: "market",
		}

		err := executor.PlaceOrder(ctx, order)
		require.NoError(t, err)
		require.NotEmpty(t, order.OrderID)

		time.Sleep(2 * time.Second)

		orderStatus, err := executor.GetOrderStatus(ctx, SYMBOL, order.OrderID)
		require.NoError(t, err)
		require.Equal(t, "FILLED", orderStatus.Status)
	})
}
