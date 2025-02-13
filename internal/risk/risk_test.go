package risk

import (
	"context"
	"testing"
	"time"

	"github.com/songzhibin97/quantaflux/internal/trading"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicRiskManager_CheckTradeRisk(t *testing.T) {
	// 设置基础风险参数
	params := RiskParameters{
		MaxPositionSize: 10000.0, // 最大仓位
		MaxLossPerTrade: 1000.0,  // 单笔最大亏损
		MaxDailyLoss:    3000.0,  // 每日最大亏损
		MaxLeverage:     3.0,     // 最大杠杆
		MinLiquidity:    5000.0,  // 最小流动性
	}

	tests := []struct {
		name           string
		order          trading.Order
		wantAcceptable bool
		wantRiskLevel  float64
		wantFactors    int
	}{
		{
			name: "safe order",
			order: trading.Order{
				Symbol:    "BTC-USDT",
				Side:      "buy",
				Amount:    1.0,
				Price:     1000.0,
				OrderType: "limit",
				Status:    "new",
			},
			wantAcceptable: true,
			wantRiskLevel:  0,
			wantFactors:    0,
		},
		{
			name: "excessive position size",
			order: trading.Order{
				Symbol:    "BTC-USDT",
				Side:      "buy",
				Amount:    20.0,
				Price:     1000.0,
				OrderType: "limit",
				Status:    "new",
			},
			wantAcceptable: false,
			wantRiskLevel:  0.3,
			wantFactors:    1,
		},
		{
			name: "market order risk",
			order: trading.Order{
				Symbol:    "BTC-USDT",
				Side:      "buy",
				Amount:    1.0,
				Price:     1000.0,
				OrderType: "market",
				Status:    "new",
			},
			wantAcceptable: true,
			wantRiskLevel:  0.1,
			wantFactors:    1,
		},
		{
			name: "multiple risk factors",
			order: trading.Order{
				Symbol:    "BTC-USDT",
				Side:      "buy",
				Amount:    15.0,
				Price:     1000.0,
				OrderType: "market",
				Status:    "new",
			},
			wantAcceptable: false,
			wantRiskLevel:  0.4, // 0.3 (size) + 0.1 (market)
			wantFactors:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewBasicRiskManager(params)
			ctx := context.Background()

			assessment, err := rm.CheckTradeRisk(ctx, &tt.order)
			require.NoError(t, err)
			require.NotNil(t, assessment)

			assert.Equal(t, tt.wantAcceptable, assessment.IsAcceptable)
			assert.Equal(t, tt.wantRiskLevel, assessment.RiskLevel)
			assert.Len(t, assessment.RiskFactors, tt.wantFactors)

			if len(assessment.RiskFactors) > 0 {
				t.Logf("Risk Factors: %v", assessment.RiskFactors)
				t.Logf("Recommendations: %v", assessment.Recommendations)
			}
		})
	}
}

func TestBasicRiskManager_SetRiskParameters(t *testing.T) {
	rm := NewBasicRiskManager(RiskParameters{})
	ctx := context.Background()

	tests := []struct {
		name    string
		params  RiskParameters
		wantErr bool
	}{
		{
			name: "valid parameters",
			params: RiskParameters{
				MaxPositionSize: 10000.0,
				MaxLossPerTrade: 1000.0,
				MaxDailyLoss:    3000.0,
				MaxLeverage:     3.0,
				MinLiquidity:    5000.0,
			},
			wantErr: false,
		},
		{
			name: "invalid position size",
			params: RiskParameters{
				MaxPositionSize: -1000.0,
				MaxLossPerTrade: 1000.0,
				MaxDailyLoss:    3000.0,
				MaxLeverage:     3.0,
				MinLiquidity:    5000.0,
			},
			wantErr: true,
		},
		{
			name: "zero values",
			params: RiskParameters{
				MaxPositionSize: 0,
				MaxLossPerTrade: 0,
				MaxDailyLoss:    0,
				MaxLeverage:     0,
				MinLiquidity:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.SetRiskParameters(ctx, &tt.params)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBasicRiskManager_MonitorPositions(t *testing.T) {
	params := RiskParameters{
		MaxPositionSize: 10000.0,
		MaxLossPerTrade: 1000.0,
		MaxDailyLoss:    3000.0,
		MaxLeverage:     3.0,
		MinLiquidity:    5000.0,
	}

	rm := NewBasicRiskManager(params)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	alerts, err := rm.MonitorPositions(ctx)
	require.NoError(t, err)
	require.NotNil(t, alerts)

	// 等待 context 取消
	<-ctx.Done()

	// 确保 channel 被关闭
	_, ok := <-alerts
	assert.False(t, ok, "alerts channel should be closed")
}

func TestGetSeverityLevel(t *testing.T) {
	tests := []struct {
		name string
		pnl  float64
		want string
	}{
		{
			name: "high severity",
			pnl:  -15000,
			want: "HIGH",
		},
		{
			name: "medium severity",
			pnl:  -7500,
			want: "MEDIUM",
		},
		{
			name: "low severity",
			pnl:  -1000,
			want: "LOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSeverityLevel(tt.pnl)
			assert.Equal(t, tt.want, got)
		})
	}
}
