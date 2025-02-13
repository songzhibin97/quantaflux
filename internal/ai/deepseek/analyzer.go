package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/songzhibin97/quantaflux/internal/ai"
	"github.com/songzhibin97/quantaflux/internal/models"
)

const (
	defaultAPIEndpoint = "https://api.deepseek.com/v1"
	defaultModel       = "deepseek-chat"
)

// DeepSeekAnalyzer implements the Analyzer interface using DeepSeek
type DeepSeekAnalyzer struct {
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
}

// NewDeepSeekAnalyzer creates a new DeepSeek analyzer instance
func NewDeepSeekAnalyzer(apiKey string, model string) *DeepSeekAnalyzer {
	if model == "" {
		model = defaultModel
	}

	return &DeepSeekAnalyzer{
		apiKey:   apiKey,
		endpoint: defaultAPIEndpoint,
		model:    model,
		client:   &http.Client{},
	}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// AnalyzeProject implements the Analyzer interface
func (a *DeepSeekAnalyzer) AnalyzeProject(ctx context.Context, info *models.TokenInfo) (*models.ProjectMetrics, error) {
	prompt := fmt.Sprintf(`分析以下加密货币项目并提供详细评估:
项目名称: %s
代币符号: %s
合约地址: %s
网络: %s
发行类型: %s
初始价格: %f
总供应量: %f
流通供应量: %f

请根据以下几个维度进行评分（0-100）并给出具体理由：
1. 社交媒体活跃度 - 考虑Twitter、Telegram、Discord等平台的活跃度
2. 开发活动 - 评估代码提交、技术更新频率
3. 社区成长性 - 分析社区增长速度和参与度
4. 市场情绪 - 评估整体市场对项目的态度
5. 风险评估 - 综合评估项目风险因素

输出格式：
{
    "social_score": float,
    "development_score": float,
    "community_growth": float,
    "market_sentiment": float,
    "risk_score": float,
    "analysis": {
        "social": "评分理由",
        "development": "评分理由",
        "community": "评分理由",
        "sentiment": "评分理由",
        "risk": "评分理由"
    }
}`,
		info.Name, info.Symbol, info.ContractAddress, info.Network,
		info.LaunchType, info.InitialPrice, info.TotalSupply, info.CirculatingSupply)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	var analysis struct {
		SocialScore      float64 `json:"social_score"`
		DevelopmentScore float64 `json:"development_score"`
		CommunityGrowth  float64 `json:"community_growth"`
		MarketSentiment  float64 `json:"market_sentiment"`
		RiskScore        float64 `json:"risk_score"`
		Analysis         struct {
			Social      string `json:"social"`
			Development string `json:"development"`
			Community   string `json:"community"`
			Sentiment   string `json:"sentiment"`
			Risk        string `json:"risk"`
		} `json:"analysis"`
	}

	if err := json.Unmarshal([]byte(resp), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis results: %w", err)
	}

	return &models.ProjectMetrics{
		TokenInfo:        *info,
		SocialScore:      analysis.SocialScore,
		DevelopmentScore: analysis.DevelopmentScore,
		CommunityGrowth:  analysis.CommunityGrowth,
		MarketSentiment:  analysis.MarketSentiment,
		RiskScore:        analysis.RiskScore,
	}, nil
}

// PredictPrice implements the Analyzer interface
func (a *DeepSeekAnalyzer) PredictPrice(ctx context.Context, data []models.MarketData) (*ai.PricePrediction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no market data provided")
	}

	marketDataDesc := strings.Builder{}
	marketDataDesc.WriteString("市场数据分析：\n")
	for _, d := range data {
		marketDataDesc.WriteString(fmt.Sprintf("时间: %s\n价格: %.8f\n24h成交量: %.2f\n市值: %.2f\n\n",
			d.Timestamp.Format("2006-01-02 15:04:05"),
			d.Price,
			d.Volume24h,
			d.MarketCap))
	}

	prompt := fmt.Sprintf(`基于以下市场数据，对%s进行价格预测分析：

%s

请提供：
1. 24小时内的预测价格
2. 预测的可信度（0-1）
3. 影响价格的关键因素
4. 具体的分析理由

输出格式：
{
    "predicted_price": float,
    "confidence": float,
    "factors": ["因素1", "因素2", ...],
    "reasoning": "详细分析理由",
    "potential_risks": ["风险1", "风险2", ...]
}`, data[0].Symbol, marketDataDesc.String())

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to predict price: %w", err)
	}

	var prediction struct {
		PredictedPrice float64  `json:"predicted_price"`
		Confidence     float64  `json:"confidence"`
		Factors        []string `json:"factors"`
		Reasoning      string   `json:"reasoning"`
		PotentialRisks []string `json:"potential_risks"`
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

// DetectScam implements the Analyzer interface
func (a *DeepSeekAnalyzer) DetectScam(ctx context.Context, projectData *models.ProjectMetrics) (*ai.ScamAnalysis, error) {
	prompt := fmt.Sprintf(`请对以下项目进行深入的诈骗风险分析：

项目基本信息：
- 名称: %s
- 符号: %s
- 合约地址: %s
- 发行类型: %s

项目指标：
- 社交分数: %.2f
- 开发分数: %.2f
- 社区增长: %.2f
- 市场情绪: %.2f
- 风险分数: %.2f

请从以下角度分析：
1. 团队背景验证
2. 代码安全性
3. 资金流向分析
4. 社区真实性
5. 市场操纵迹象

输出格式：
{
    "scam_probability": float,
    "risk_factors": ["风险1", "风险2", ...],
    "confidence": float,
    "warnings": ["警告1", "警告2", ...],
    "recommendations": ["建议1", "建议2", ...]
}`,
		projectData.TokenInfo.Name,
		projectData.TokenInfo.Symbol,
		projectData.TokenInfo.ContractAddress,
		projectData.TokenInfo.LaunchType,
		projectData.SocialScore,
		projectData.DevelopmentScore,
		projectData.CommunityGrowth,
		projectData.MarketSentiment,
		projectData.RiskScore)

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to detect scam: %w", err)
	}

	var result struct {
		ScamProbability float64  `json:"scam_probability"`
		RiskFactors     []string `json:"risk_factors"`
		Confidence      float64  `json:"confidence"`
		Warnings        []string `json:"warnings"`
		Recommendations []string `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse scam analysis results: %w", err)
	}

	return &ai.ScamAnalysis{
		ScamProbability: result.ScamProbability,
		RiskFactors:     result.RiskFactors,
		Confidence:      result.Confidence,
	}, nil
}

// AnalyzeSentiment implements the Analyzer interface
func (a *DeepSeekAnalyzer) AnalyzeSentiment(ctx context.Context, socialData map[string]string) (float64, error) {
	var socialText strings.Builder
	for platform, content := range socialData {
		socialText.WriteString(fmt.Sprintf("== %s ==\n%s\n\n", platform, content))
	}

	prompt := fmt.Sprintf(`分析以下社交媒体数据的市场情绪：

%s

请提供：
1. 情绪评分（-1到1，-1表示极度负面，0表示中性，1表示极度正面）
2. 关键词提取
3. 情绪波动分析

输出格式：
{
    "sentiment_score": float,
    "keywords": ["关键词1", "关键词2", ...],
    "analysis": "详细分析",
    "trends": ["趋势1", "趋势2", ...]
}`, socialText.String())

	resp, err := a.createChatCompletion(ctx, prompt)
	if err != nil {
		return 0, fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	var result struct {
		SentimentScore float64  `json:"sentiment_score"`
		Keywords       []string `json:"keywords"`
		Analysis       string   `json:"analysis"`
		Trends         []string `json:"trends"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return 0, fmt.Errorf("failed to parse sentiment results: %w", err)
	}

	return result.SentimentScore, nil
}

// createChatCompletion sends a request to the DeepSeek API
func (a *DeepSeekAnalyzer) createChatCompletion(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: a.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "你是一个专业的加密货币分析师，擅长项目分析、价格预测和风险评估。请严格按照要求的JSON格式输出分析结果。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/chat/completions", a.endpoint),
		bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	if !json.Valid(body) {
		return "", fmt.Errorf("API 返回无效的 JSON 响应")
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("api error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from api")
	}

	return chatResp.Choices[0].Message.Content, nil
}
