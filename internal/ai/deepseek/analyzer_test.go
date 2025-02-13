package deepseek

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/songzhibin97/quantaflux/internal/models"
	"github.com/stretchr/testify/assert"
)

var apiKey = os.Getenv("DEEPSEEK_API_KEY")

func TestOpenAIAnalyzer_AnalyzeProject(t *testing.T) {
	analyzer := NewDeepSeekAnalyzer(apiKey, "")

	info := &models.TokenInfo{
		Name:              "Test Token",
		Symbol:            "TEST",
		ContractAddress:   "0x1234567890abcdef",
		Network:           "ethereum",
		LaunchType:        "IDO",
		InitialPrice:      1.0,
		TotalSupply:       1000000,
		CirculatingSupply: 500000,
	}

	ctx := context.Background()
	metrics, err := analyzer.AnalyzeProject(ctx, info)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.SocialScore, 0.0)
	assert.LessOrEqual(t, metrics.SocialScore, 100.0)
}

func TestOpenAIAnalyzer_PredictPrice(t *testing.T) {
	analyzer := NewDeepSeekAnalyzer(apiKey, "")

	data := []models.MarketData{
		{
			Symbol:    "TEST",
			Price:     100.0,
			Volume24h: 1000000,
			MarketCap: 10000000,
			Timestamp: time.Now().Add(-24 * time.Hour),
		},
		{
			Symbol:    "TEST",
			Price:     105.0,
			Volume24h: 1200000,
			MarketCap: 10500000,
			Timestamp: time.Now(),
		},
	}

	ctx := context.Background()
	prediction, err := analyzer.PredictPrice(ctx, data)

	assert.NoError(t, err)
	assert.NotNil(t, prediction)
	assert.GreaterOrEqual(t, prediction.Confidence, 0.0)
	assert.LessOrEqual(t, prediction.Confidence, 1.0)
}

func TestOpenAIAnalyzer_DetectScam(t *testing.T) {
	analyzer := NewDeepSeekAnalyzer(apiKey, "")

	projectData := &models.ProjectMetrics{
		TokenInfo: models.TokenInfo{
			Name:   "Test Token",
			Symbol: "TEST",
		},
		SocialScore:      80.0,
		DevelopmentScore: 75.0,
		CommunityGrowth:  70.0,
		MarketSentiment:  65.0,
		RiskScore:        30.0,
	}

	ctx := context.Background()
	analysis, err := analyzer.DetectScam(ctx, projectData)

	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.GreaterOrEqual(t, analysis.ScamProbability, 0.0)
	assert.LessOrEqual(t, analysis.ScamProbability, 1.0)
	assert.NotEmpty(t, analysis.RiskFactors)
}
