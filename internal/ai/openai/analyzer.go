package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/songzhibin97/quantaflux/internal/ai"
	"github.com/songzhibin97/quantaflux/internal/models"
)

// OpenAIAnalyzer implements the Analyzer interface using OpenAI
type OpenAIAnalyzer struct {
	client *openai.Client
	model  string
}

// NewOpenAIAnalyzer creates a new OpenAI analyzer instance
func NewOpenAIAnalyzer(apiKey string, model string) *OpenAIAnalyzer {
	client := openai.NewClient(apiKey)
	if model == "" {
		model = openai.GPT4 // 默认使用GPT-4
	}
	return &OpenAIAnalyzer{
		client: client,
		model:  model,
	}
}

// AnalyzeProject implements the Analyzer interface
func (a *OpenAIAnalyzer) AnalyzeProject(ctx context.Context, info *models.TokenInfo) (*models.ProjectMetrics, error) {
	prompt := fmt.Sprintf(`分析以下加密货币项目并提供详细评估:
项目名称: %s
代币符号: %s
合约地址: %s
网络: %s
发行类型: %s
初始价格: %f
总供应量: %f
流通供应量: %f

请从以下几个方面进行评估，并给出0-100的评分：
1. 社交媒体活跃度
2. 开发活动
3. 社区成长性
4. 市场情绪
5. 风险评估

输出格式为JSON:
{
    "social_score": float,
    "development_score": float,
    "community_growth": float,
    "market_sentiment": float,
    "risk_score": float
}`,
		info.Name, info.Symbol, info.ContractAddress, info.Network,
		info.LaunchType, info.InitialPrice, info.TotalSupply, info.CirculatingSupply)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	var scores struct {
		SocialScore      float64 `json:"social_score"`
		DevelopmentScore float64 `json:"development_score"`
		CommunityGrowth  float64 `json:"community_growth"`
		MarketSentiment  float64 `json:"market_sentiment"`
		RiskScore        float64 `json:"risk_score"`
	}

	if err := json.Unmarshal([]byte(resp), &scores); err != nil {
		return nil, fmt.Errorf("failed to parse analysis results: %w", err)
	}

	return &models.ProjectMetrics{
		TokenInfo:        *info,
		SocialScore:      scores.SocialScore,
		DevelopmentScore: scores.DevelopmentScore,
		CommunityGrowth:  scores.CommunityGrowth,
		MarketSentiment:  scores.MarketSentiment,
		RiskScore:        scores.RiskScore,
	}, nil
}

// PredictPrice implements the Analyzer interface
func (a *OpenAIAnalyzer) PredictPrice(ctx context.Context, data []models.MarketData) (*ai.PricePrediction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no market data provided")
	}

	// 构建市场数据的时间序列描述
	marketDataDesc := "市场数据分析:\n"
	for _, d := range data {
		marketDataDesc += fmt.Sprintf("时间: %s, 价格: %.8f, 24h成交量: %.2f, 市值: %.2f\n",
			d.Timestamp.Format("2006-01-02 15:04:05"),
			d.Price,
			d.Volume24h,
			d.MarketCap)
	}

	prompt := fmt.Sprintf(`基于以下市场数据预测%s的价格走势:
%s

请分析价格趋势并预测未来24小时的价格变动。
考虑因素包括：价格趋势、成交量变化、市值变化等。

输出格式为JSON:
{
    "predicted_price": float,
    "confidence": float,
    "factors": ["因素1", "因素2", ...]
}`, data[0].Symbol, marketDataDesc)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to predict price: %w", err)
	}

	var prediction struct {
		PredictedPrice float64  `json:"predicted_price"`
		Confidence     float64  `json:"confidence"`
		Factors        []string `json:"factors"`
	}

	if err := json.Unmarshal([]byte(resp), &prediction); err != nil {
		return nil, fmt.Errorf("failed to parse prediction results: %w", err)
	}

	return &ai.PricePrediction{
		Symbol:         data[0].Symbol,
		PredictedPrice: prediction.PredictedPrice,
		Confidence:     prediction.Confidence,
		TimeFrame:      "24h",
		Factors:        prediction.Factors,
	}, nil
}

// AnalyzeSentiment implements the Analyzer interface
func (a *OpenAIAnalyzer) AnalyzeSentiment(ctx context.Context, socialData map[string]string) (float64, error) {
	socialDataText := ""
	for platform, content := range socialData {
		socialDataText += fmt.Sprintf("%s: %s\n", platform, content)
	}

	prompt := fmt.Sprintf(`分析以下社交媒体内容的市场情绪:
%s

请评估整体市场情绪，给出-1到1之间的分数：
-1表示极度负面
0表示中性
1表示极度正面

输出格式为JSON:
{
    "sentiment_score": float
}`, socialDataText)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return 0, fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	var sentiment struct {
		Score float64 `json:"sentiment_score"`
	}

	if err := json.Unmarshal([]byte(resp), &sentiment); err != nil {
		return 0, fmt.Errorf("failed to parse sentiment results: %w", err)
	}

	return sentiment.Score, nil
}

// DetectScam implements the Analyzer interface
func (a *OpenAIAnalyzer) DetectScam(ctx context.Context, projectData *models.ProjectMetrics) (*ai.ScamAnalysis, error) {
	prompt := fmt.Sprintf(`分析以下项目数据，评估是否存在诈骗风险:
代币名称: %s
代币符号: %s
合约地址: %s
社交分数: %.2f
开发分数: %.2f
社区增长: %.2f
市场情绪: %.2f
风险分数: %.2f

请评估该项目是否存在诈骗风险，并列出风险因素。

输出格式为JSON:
{
    "scam_probability": float,
    "risk_factors": ["风险1", "风险2", ...],
    "confidence": float
}`,
		projectData.TokenInfo.Name,
		projectData.TokenInfo.Symbol,
		projectData.TokenInfo.ContractAddress,
		projectData.SocialScore,
		projectData.DevelopmentScore,
		projectData.CommunityGrowth,
		projectData.MarketSentiment,
		projectData.RiskScore)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to detect scam: %w", err)
	}

	var scamAnalysis ai.ScamAnalysis
	if err := json.Unmarshal([]byte(resp), &scamAnalysis); err != nil {
		return nil, fmt.Errorf("failed to parse scam analysis results: %w", err)
	}

	return &scamAnalysis, nil
}

// createChatCompletion is a helper function to make OpenAI API calls
func (a *OpenAIAnalyzer) createChatCompletion(ctx context.Context, prompt string) (string, error) {
	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: a.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个专业的加密货币分析师，擅长项目分析、价格预测和风险评估。请始终以JSON格式返回分析结果。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.3, // 使用较低的temperature以获得更稳定的输出
		},
	)
	if err != nil {
		return "", fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	return resp.Choices[0].Message.Content, nil
}
