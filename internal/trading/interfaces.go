package trading

import (
	"context"
)

// TradeExecutor defines methods for executing trades
type TradeExecutor interface {
	// PlaceOrder places a new order
	PlaceOrder(ctx context.Context, order *Order) error

	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, symbol string, orderID string) error

	// GetOrderStatus retrieves the status of an order
	GetOrderStatus(ctx context.Context, symbol, orderID string) (*Order, error)

	// GetBalance retrieves account balance
	GetBalance(ctx context.Context, symbol string) (float64, error)
}

// Order 订单结构
type Order struct {
	Symbol     string  // 交易对
	Side       string  // buy 或 sell
	Amount     float64 // 数量
	Price      float64 // 价格（市价单可为0）
	OrderType  string  // market 或 limit
	Status     string  // 订单状态
	OrderID    string  // 订单ID字符串格式
	RawOrderID int64   // 订单ID数字格式
}
