package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/songzhibin97/quantaflux/internal/trading"

	"github.com/adshao/go-binance/v2"
)

// BinanceExecutor implements TradeExecutor interface for Binance
type BinanceExecutor struct {
	client    *binance.Client
	apiKey    string
	secretKey string
	mu        sync.RWMutex
}

// NewBinanceExecutor creates a new BinanceExecutor instance
func NewBinanceExecutor(apiKey, secretKey string, debug ...bool) *BinanceExecutor {
	debug = append(debug, false)
	if debug[0] {
		binance.UseTestnet = true
	}

	client := binance.NewClient(apiKey, secretKey)

	return &BinanceExecutor{
		client:    client,
		apiKey:    apiKey,
		secretKey: secretKey,
	}
}

// PlaceOrder implements order placement for Binance
func (b *BinanceExecutor) PlaceOrder(ctx context.Context, order *trading.Order) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Convert order type to Binance format
	var orderType binance.OrderType
	switch order.OrderType {
	case "market":
		orderType = binance.OrderTypeMarket
	case "limit":
		orderType = binance.OrderTypeLimit
	default:
		return fmt.Errorf("unsupported order type: %s", order.OrderType)
	}

	// Convert side to Binance format
	var side binance.SideType
	switch order.Side {
	case "buy":
		side = binance.SideTypeBuy
	case "sell":
		side = binance.SideTypeSell
	default:
		return fmt.Errorf("invalid side: %s", order.Side)
	}

	// Create order request
	orderService := b.client.NewCreateOrderService().
		Symbol(order.Symbol).
		Side(side).
		Type(orderType)

	// Set quantity
	quantity := strconv.FormatFloat(order.Amount, 'f', -1, 64)
	orderService.Quantity(quantity)

	// Set price for limit orders
	if orderType == binance.OrderTypeLimit {
		price := strconv.FormatFloat(order.Price, 'f', -1, 64)
		orderService.TimeInForce(binance.TimeInForceTypeGTC)
		orderService.Price(price)
	}

	// Execute order
	result, err := orderService.Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	// Update order with response data
	order.Status = string(result.Status)
	order.RawOrderID = result.OrderID
	order.OrderID = strconv.FormatInt(result.OrderID, 10)
	return nil
}

// CancelOrder implements order cancellation for Binance
func (b *BinanceExecutor) CancelOrder(ctx context.Context, symbol string, orderID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	_, err = b.client.NewCancelOrderService().
		Symbol(symbol).
		OrderID(id).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// GetOrderStatus implements order status retrieval for Binance
func (b *BinanceExecutor) GetOrderStatus(ctx context.Context, symbol, orderID string) (*trading.Order, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	result, err := b.client.NewGetOrderService().
		Symbol(symbol).
		OrderID(id).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	price, _ := strconv.ParseFloat(result.Price, 64)
	amount, _ := strconv.ParseFloat(result.OrigQuantity, 64)

	return &trading.Order{
		Symbol:     result.Symbol,
		Side:       string(result.Side),
		Amount:     amount,
		Price:      price,
		OrderType:  string(result.Type),
		Status:     string(result.Status),
		OrderID:    strconv.FormatInt(result.OrderID, 10),
		RawOrderID: result.OrderID,
	}, nil
}

// GetBalance implements balance retrieval for Binance
func (b *BinanceExecutor) GetBalance(ctx context.Context, symbol string) (float64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Get account information
	account, err := b.client.NewGetAccountService().Do(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get account info: %w", err)
	}

	// Find balance for specified symbol
	for _, balance := range account.Balances {
		if balance.Asset == symbol {
			free, err := strconv.ParseFloat(balance.Free, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse balance: %w", err)
			}
			return free, nil
		}
	}

	return 0, fmt.Errorf("balance not found for symbol: %s", symbol)
}
