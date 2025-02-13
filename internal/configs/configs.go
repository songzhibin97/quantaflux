package configs

import (
	"github.com/songzhibin97/quantaflux/internal/risk"
)

type Config struct {
	// 基础配置
	Symbols         []string ` json:"symbols" yaml:"symbols"`                  // 交易对列表
	RefreshInterval string   `json:"refresh_interval" yaml:"refresh_interval"` // 数据刷新间隔

	Database Database `json:"database" yaml:"database"`

	// 风险控制参数
	RiskParams risk.RiskParameters `json:"risk_parameters" yaml:"risk_params"`

	// AI 模型参数
	AIConfig AIConfig `json:"ai_config" yaml:"ai_config"`

	// 交易参数
	TradingConfig TradingConfig `json:"trading_config" yaml:"trading_config"`

	// 交易所配置
	ExchangeConfig ExchangeConfig `json:"exchange_config" yaml:"exchange_config"`
}

type AIConfig struct {
	MinConfidence    float64 ` json:"min_confidence" yaml:"min_confidence"`        // AI预测最小置信度
	PredictTimeFrame string  `json:"predict_time_frame" yaml:"predict_time_frame"` // 预测时间范围
	ScamThreshold    float64 `json:"scam_threshold" yaml:"scam_threshold"`         // 诈骗判定阈值
	APIKey           string  `json:"api_key" yaml:"api_key"`                       // AI服务API密钥
	ModelType        string  `json:"model_type" yaml:"model_type"`                 // AI模型类型
}

type TradingConfig struct {
	MaxOrderAmount float64 `json:"max_order_amount" yaml:"max_order_amount"` // 单笔最大交易量
	MinOrderAmount float64 `json:"min_order_amount" yaml:"min_order_amount"` // 单笔最小交易量
	PriceTolerance float64 `json:"price_tolerance" yaml:"price_tolerance"`   // 价格容差
	OrderType      string  `json:"order_type" yaml:"order_type"`             // 订单类型(market/limit)
}

type Database struct {
	ConnStr string `json:"conn_str" yaml:"conn_str"` // 数据库连接字符串
}

type ExchangeConfig struct {
	Debug     bool   `json:"debug" yaml:"debug"`
	APIKey    string `json:"api_key" yaml:"api_key"`       // 交易所API密钥
	SecretKey string `json:"secret_key" yaml:"secret_key"` // 交易所密钥
}
