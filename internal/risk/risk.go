package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/songzhibin97/quantaflux/internal/trading"
)

type BasicRiskManager struct {
	params     RiskParameters
	paramsMu   sync.RWMutex
	dailyStats struct {
		totalLoss     float64
		tradingVolume float64
		tradeCount    int
	}
	statsReset time.Time
}

func NewBasicRiskManager(initialParams RiskParameters) *BasicRiskManager {
	return &BasicRiskManager{
		params:     initialParams,
		statsReset: time.Now(),
	}
}

func (rm *BasicRiskManager) CheckTradeRisk(ctx context.Context, order *trading.Order) (*RiskAssessment, error) {
	rm.paramsMu.RLock()
	params := rm.params
	rm.paramsMu.RUnlock()

	assessment := &RiskAssessment{
		IsAcceptable:    true,
		RiskLevel:       0,
		RiskFactors:     make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// 计算订单总值
	orderValue := order.Amount * order.Price

	// 检查仓位大小 - 这是最主要的风险检查
	if orderValue > params.MaxPositionSize {
		assessment.IsAcceptable = false
		assessment.RiskLevel += 0.3
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Position size exceeds maximum allowed")
		assessment.Recommendations = append(assessment.Recommendations,
			fmt.Sprintf("Reduce position size below %.2f", params.MaxPositionSize))
	} else {
		// 只有在仓位没有超过限制的情况下，才检查潜在亏损
		potentialLoss := orderValue * 0.1
		if order.Side == "buy" && potentialLoss > params.MaxLossPerTrade {
			assessment.IsAcceptable = false
			assessment.RiskLevel += 0.25
			assessment.RiskFactors = append(assessment.RiskFactors,
				"Potential loss exceeds maximum allowed per trade")
			assessment.Recommendations = append(assessment.Recommendations,
				fmt.Sprintf("Reduce position size to limit potential loss below %.2f", params.MaxLossPerTrade))
		}
	}

	// 检查当日总亏损限制
	if rm.dailyStats.totalLoss+orderValue*0.1 > params.MaxDailyLoss {
		assessment.IsAcceptable = false
		assessment.RiskLevel += 0.25
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Trade could exceed maximum daily loss limit")
		assessment.Recommendations = append(assessment.Recommendations,
			"Wait for daily loss limit to reset or reduce position size")
	}

	// 检查市价单风险
	if order.OrderType == "market" {
		assessment.RiskLevel += 0.1
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Market order may result in slippage")
		assessment.Recommendations = append(assessment.Recommendations,
			"Consider using limit order for better price control")
	}

	// 检查交易量限制
	if rm.dailyStats.tradingVolume+orderValue > params.MaxPositionSize*5 {
		assessment.IsAcceptable = false
		assessment.RiskLevel += 0.2
		assessment.RiskFactors = append(assessment.RiskFactors,
			"Daily trading volume would exceed safe limits")
		assessment.Recommendations = append(assessment.Recommendations,
			"Reduce trading volume or wait for daily reset")
	}

	// 检查交易频率
	if rm.dailyStats.tradeCount > 100 {
		assessment.RiskLevel += 0.15
		assessment.RiskFactors = append(assessment.RiskFactors,
			"High trading frequency detected")
		assessment.Recommendations = append(assessment.Recommendations,
			"Consider reducing trading frequency")
	}

	return assessment, nil
}

func (rm *BasicRiskManager) SetRiskParameters(ctx context.Context, params *RiskParameters) error {
	if params.MaxPositionSize <= 0 || params.MaxLossPerTrade <= 0 ||
		params.MaxDailyLoss <= 0 || params.MaxLeverage <= 0 || params.MinLiquidity <= 0 {
		return fmt.Errorf("invalid risk parameters: all values must be positive")
	}

	rm.paramsMu.Lock()
	rm.params = *params
	rm.paramsMu.Unlock()

	return nil
}

func (rm *BasicRiskManager) MonitorPositions(ctx context.Context) (<-chan RiskAlert, error) {
	alerts := make(chan RiskAlert, 100)

	go func() {
		defer close(alerts)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		dayReset := time.NewTicker(24 * time.Hour)
		defer dayReset.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-dayReset.C:
				rm.paramsMu.Lock()
				rm.dailyStats.totalLoss = 0
				rm.dailyStats.tradingVolume = 0
				rm.dailyStats.tradeCount = 0
				rm.statsReset = time.Now()
				rm.paramsMu.Unlock()

			case <-ticker.C:
				positions := rm.getCurrentPositions()
				for _, pos := range positions {
					if pos.UnrealizedPnL < -rm.params.MaxLossPerTrade {
						alert := RiskAlert{
							Symbol:      pos.Symbol,
							AlertType:   "Position Loss",
							Severity:    getSeverityLevel(pos.UnrealizedPnL),
							Description: fmt.Sprintf("Position loss exceeded threshold for %s", pos.Symbol),
							Timestamp:   time.Now(),
						}

						select {
						case alerts <- alert:
						default:
							// Channel full, could log this situation
						}
					}
				}
			}
		}
	}()

	return alerts, nil
}

// Position represents a current trading position
type Position struct {
	Symbol        string
	UnrealizedPnL float64
}

func (rm *BasicRiskManager) getCurrentPositions() []Position {
	// 这个方法需要实际实现，连接到交易系统
	return []Position{}
}

func getSeverityLevel(pnl float64) string {
	switch {
	case pnl < -10000:
		return "HIGH"
	case pnl < -5000:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
