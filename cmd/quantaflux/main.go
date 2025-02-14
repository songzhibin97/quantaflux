package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/songzhibin97/quantaflux/internal/data/collector/binance"

	collectorData "github.com/songzhibin97/quantaflux/internal/data/collector"

	"github.com/songzhibin97/quantaflux/internal/ai/deepseek"

	"github.com/songzhibin97/quantaflux/internal/data/storage"
	binanceTrading "github.com/songzhibin97/quantaflux/internal/trading/binance"

	"github.com/songzhibin97/quantaflux/internal/ai"
	"github.com/songzhibin97/quantaflux/internal/configs"
	"github.com/songzhibin97/quantaflux/internal/data"
	"github.com/songzhibin97/quantaflux/internal/models"
	"github.com/songzhibin97/quantaflux/internal/risk"
	"github.com/songzhibin97/quantaflux/internal/trading"
)

type QuantSystem struct {
	config        *configs.Config
	dataCollector data.DataCollector
	dataStorage   data.DataStorage
	aiAnalyzer    ai.Analyzer
	riskManager   risk.RiskManager
	tradeExecutor trading.TradeExecutor
}

func NewQuantSystem(
	config *configs.Config,
	collector data.DataCollector,
	storage data.DataStorage,
	analyzer ai.Analyzer,
	riskMgr risk.RiskManager,
	executor trading.TradeExecutor,
) *QuantSystem {
	return &QuantSystem{
		config:        config,
		dataCollector: collector,
		dataStorage:   storage,
		aiAnalyzer:    analyzer,
		riskManager:   riskMgr,
		tradeExecutor: executor,
	}
}

// Run 运行量化系统
func (s *QuantSystem) Run(ctx context.Context) error {
	// 设置风险参数
	if err := s.riskManager.SetRiskParameters(ctx, &s.config.RiskParams); err != nil {
		return err
	}
	log.Debug("set risk parameters ok!")

	refreshInterval, err := time.ParseDuration(s.config.RefreshInterval)
	if err != nil {
		refreshInterval = time.Second * 10
	}

	// 订阅市场数据
	marketDataCh, err := s.dataCollector.SubscribeToMarketData(ctx, s.config.Symbols, refreshInterval)
	if err != nil {
		return err
	}

	log.Debug("subscribe to market data ok!")

	// 监控风险预警
	riskAlertCh, err := s.riskManager.MonitorPositions(ctx)
	if err != nil {
		return err
	}

	log.Debug("monitor positions ok!")

	// 主循环
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case marketData := <-marketDataCh:
			log.Debug("Received market data", "market", marketData)

			if err := s.handleMarketData(ctx, marketData); err != nil {
				log.Error("Error handling market data", "err", err)
			}

		case alert := <-riskAlertCh:
			log.Debug("Received risk alert: %+v\n", alert)

			if err := s.handleRiskAlert(ctx, alert); err != nil {
				log.Error("Error handling risk alert", "err", err)
			}
		}
	}
}

// handleMarketData 处理市场数据
func (s *QuantSystem) handleMarketData(ctx context.Context, data models.MarketData) error {
	// 1. 保存市场数据
	if err := s.dataStorage.SaveMarketData(ctx, &data); err != nil {
		return err
	}

	// 2. 收集token信息和社交指标
	tokenInfo, err := s.dataCollector.CollectTokenInfo(ctx, data.Symbol)
	if err != nil {
		return err
	}

	socialMetrics, err := s.dataCollector.CollectSocialMetrics(ctx, data.Symbol)
	if err != nil {
		return err
	}

	if len(socialMetrics) != 0 {
		// 3. 构建项目指标用于AI分析
		projectMetrics := &models.ProjectMetrics{
			TokenInfo: *tokenInfo,
			// 计算社交分数（可以根据需要调整计算方法）
			SocialScore: calculateSocialScore(socialMetrics),
			// 其他指标可以根据需要添加
			UpdatedAt: time.Now(),
		}

		// 4. 进行诈骗检测
		scamAnalysis, err := s.aiAnalyzer.DetectScam(ctx, projectMetrics)
		if err != nil {
			return err
		}

		// 如果诈骗可能性高于阈值，停止交易
		if scamAnalysis.ScamProbability > s.config.AIConfig.ScamThreshold {
			log.Warn("Warning: High scam probability detected for %s: %.2f", data.Symbol, scamAnalysis.ScamProbability)
			return nil
		}
	}

	// 5. 分析市场情绪
	sentiment, err := s.aiAnalyzer.AnalyzeSentiment(ctx, convertSocialMetricsToMap(socialMetrics))
	if err != nil {
		return err
	}

	// 如果市场情绪过于负面，可能需要调整策略
	if sentiment < -0.5 { // 假设-1到1的范围，-0.5表示相当负面
		log.Warn("Warning: Negative market sentiment for %s: %.2f\n", data.Symbol, sentiment)
		return nil
	}

	// 6. AI价格预测
	prediction, err := s.aiAnalyzer.PredictPrice(ctx, []models.MarketData{data})
	if err != nil {
		return err
	}

	// 检查预测置信度
	if prediction.Confidence < s.config.AIConfig.MinConfidence {
		return nil
	}

	// 7. 生成交易订单
	order := &trading.Order{
		Symbol:    data.Symbol,
		Amount:    s.calculateOrderAmount(prediction.PredictedPrice, data.Price),
		Price:     prediction.PredictedPrice,
		OrderType: s.config.TradingConfig.OrderType,
		Side:      s.determineOrderSide(prediction.PredictedPrice, data.Price),
	}

	// 8. 风险评估
	riskAssessment, err := s.riskManager.CheckTradeRisk(ctx, order)
	if err != nil {
		return err
	}

	// 如果风险可接受，执行交易
	if riskAssessment.IsAcceptable {
		log.Debug("Risk assessment for %s: acceptable", data.Symbol)
		return s.tradeExecutor.PlaceOrder(ctx, order)
	}

	log.Debug("AI预测结果: Symbol=%s, 价格=%.2f, 置信度=%.2f", data.Symbol, prediction.PredictedPrice, prediction.Confidence)

	return nil
}

// 辅助函数：计算社交分数
func calculateSocialScore(metrics map[string]float64) float64 {
	var score float64
	// 可以根据不同平台的指标权重计算综合分数
	weights := map[string]float64{
		"twitter_followers": 0.3,
		"telegram_members":  0.3,
		"github_stars":      0.2,
		"reddit_members":    0.2,
	}

	for platform, value := range metrics {
		if weight, exists := weights[platform]; exists {
			score += value * weight
		}
	}

	return score
}

// 辅助函数：转换社交指标为字符串映射
func convertSocialMetricsToMap(metrics map[string]float64) map[string]string {
	result := make(map[string]string)
	for k, v := range metrics {
		result[k] = fmt.Sprintf("%.2f", v)
	}
	return result
}

// handleRiskAlert 处理风险预警
func (s *QuantSystem) handleRiskAlert(ctx context.Context, alert risk.RiskAlert) error {
	// 根据风险预警类型和严重程度采取相应措施
	switch alert.Severity {
	case "high":
		// 可以选择清仓或降低仓位
		return s.emergencyClose(ctx, alert.Symbol)
	case "medium":
		// 可以选择减仓
		return s.reducePosition(ctx, alert.Symbol)
	default:
		// 记录警告信息
		log.Error("risk alert", "symbol", alert.Symbol, "description", alert.Description)
		return nil
	}
}

// 计算订单数量
func (s *QuantSystem) calculateOrderAmount(predictedPrice, currentPrice float64) float64 {
	// 这里可以实现更复杂的订单数量计算逻辑
	amount := s.config.TradingConfig.MaxOrderAmount
	if amount < s.config.TradingConfig.MinOrderAmount {
		amount = s.config.TradingConfig.MinOrderAmount
	}
	return amount
}

// 确定订单方向
func (s *QuantSystem) determineOrderSide(predictedPrice, currentPrice float64) string {
	if predictedPrice > currentPrice*(1+s.config.TradingConfig.PriceTolerance) {
		return "buy"
	}
	if predictedPrice < currentPrice*(1-s.config.TradingConfig.PriceTolerance) {
		return "sell"
	}
	return ""
}

// emergencyClose 紧急平仓
func (s *QuantSystem) emergencyClose(ctx context.Context, symbol string) error {
	// 获取当前持仓
	balance, err := s.tradeExecutor.GetBalance(ctx, symbol)
	if err != nil {
		return err
	}

	if balance > 0 {
		order := &trading.Order{
			Symbol:    symbol,
			Side:      "sell",
			Amount:    balance,
			OrderType: "market", // 紧急情况使用市价单
		}
		return s.tradeExecutor.PlaceOrder(ctx, order)
	}
	return nil
}

// reducePosition 降低仓位
func (s *QuantSystem) reducePosition(ctx context.Context, symbol string) error {
	balance, err := s.tradeExecutor.GetBalance(ctx, symbol)
	if err != nil {
		return err
	}

	if balance > 0 {
		// 减仓一半
		order := &trading.Order{
			Symbol:    symbol,
			Side:      "sell",
			Amount:    balance * 0.5,
			OrderType: "market",
		}
		return s.tradeExecutor.PlaceOrder(ctx, order)
	}
	return nil
}

var (
	flagconf string

	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
)

func init() {
	flag.StringVar(&flagconf, "conf", "../configs", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	// 加载配置
	config := &configs.Config{}
	configFile, err := os.ReadFile(flagconf)
	if err != nil {
		log.Error("Error reading config file", "err", err)
	}

	if err := json.Unmarshal(configFile, config); err != nil {
		log.Error("Error parsing config file", "err", err)
		return
	}

	log.Debug("Loaded config", "config", config)

	if config.Proxy != "" {
		_ = os.Setenv("HTTP_PROXY", config.Proxy)
		_ = os.Setenv("HTTPS_PROXY", config.Proxy)
		log.Debug("set proxy ok", "proxy", config.Proxy)
	}

	// 初始化各个组件
	collector := collectorData.NewMultiSourceCollector([]collectorData.DataSource{
		binance.NewBinanceDataSource(),
	}, log)

	log.Debug("init collector")

	storager, err := storage.NewPostgresStorage(config.Database.ConnStr)
	if err != nil {
		log.Error("Error creating storage", "err", err)
		return
	}

	log.Debug("init storager")

	analyzer := deepseek.NewDeepSeekAnalyzer(config.AIConfig.APIKey, config.AIConfig.ModelType)

	log.Debug("init analyzer")

	riskManager := risk.NewBasicRiskManager(config.RiskParams)

	log.Debug("init riskManager")

	executor := binanceTrading.NewBinanceExecutor(config.ExchangeConfig.APIKey, config.ExchangeConfig.SecretKey, config.ExchangeConfig.Debug)

	log.Debug("init executor")

	// 创建量化系统
	system := NewQuantSystem(
		config,
		collector,
		storager,
		analyzer,
		riskManager,
		executor,
	)

	// 运行系统
	ctx := context.Background()
	if err := system.Run(ctx); err != nil {
		log.Error("System error", "err", err)
	}
}
