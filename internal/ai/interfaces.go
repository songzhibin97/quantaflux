package ai

import (
	"context"

	"github.com/songzhibin97/quantaflux/internal/models"
)

// Analyzer defines methods for AI analysis
type Analyzer interface {
	// AnalyzeProject performs comprehensive project analysis
	AnalyzeProject(ctx context.Context, info *models.TokenInfo) (*models.ProjectMetrics, error)

	// PredictPrice predicts future price movements
	PredictPrice(ctx context.Context, data []models.MarketData) (*PricePrediction, error)

	// AnalyzeSentiment analyzes market sentiment from social data
	AnalyzeSentiment(ctx context.Context, socialData map[string]string) (float64, error)

	// DetectScam attempts to identify potential scam projects
	DetectScam(ctx context.Context, projectData *models.ProjectMetrics) (*ScamAnalysis, error)
}

// PricePrediction 价格预测结果
type PricePrediction struct {
	Symbol         string   `json:"symbol"`
	PredictedPrice float64  `json:"predicted_price"`
	Confidence     float64  `json:"confidence"`
	TimeFrame      string   `json:"time_frame"`
	Factors        []string `json:"factors"`
}

// ScamAnalysis 欺诈分析结果
type ScamAnalysis struct {
	ScamProbability float64  `json:"scam_probability"`
	RiskFactors     []string `json:"risk_factors"`
	Confidence      float64  `json:"confidence"`
}
