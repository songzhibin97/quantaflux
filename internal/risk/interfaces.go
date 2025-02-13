package risk

import (
	"context"
	"time"

	"github.com/songzhibin97/quantaflux/internal/trading"
)

// RiskManager defines methods for risk management
type RiskManager interface {
	// CheckTradeRisk evaluates the risk of a potential trade
	CheckTradeRisk(ctx context.Context, order *trading.Order) (*RiskAssessment, error)

	// SetRiskParameters sets risk management parameters
	SetRiskParameters(ctx context.Context, params *RiskParameters) error

	// MonitorPositions monitors open positions for risk
	MonitorPositions(ctx context.Context) (<-chan RiskAlert, error)
}

// RiskParameters 风险参数配置
type RiskParameters struct {
	MaxPositionSize float64 `json:"max_position_size"`
	MaxLossPerTrade float64 `json:"max_loss_per_trade"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxLeverage     float64 `json:"max_leverage"`
	MinLiquidity    float64 `json:"min_liquidity"`
}

// RiskAssessment 风险评估结果
type RiskAssessment struct {
	IsAcceptable    bool     `json:"is_acceptable"`
	RiskLevel       float64  `json:"risk_level"`
	RiskFactors     []string `json:"risk_factors"`
	Recommendations []string `json:"recommendations"`
}

// RiskAlert 风险预警信息
type RiskAlert struct {
	Symbol      string    `json:"symbol"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}
