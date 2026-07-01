package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

var ErrAIDisabled = errors.New("ai service is not configured")

type AIService struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewAIService(cfg config.Config) *AIService {
	return &AIService{
		apiKey:  strings.TrimSpace(cfg.AIAPIKey),
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.AIBaseURL), "/"),
		model:   strings.TrimSpace(cfg.AIModel),
		client:  &http.Client{Timeout: 12 * time.Second},
	}
}

func (s *AIService) ChoiceAdvice(ctx context.Context, input domain.ChoiceAdviceInput) (domain.ChoiceAdvice, error) {
	if s.apiKey == "" || s.baseURL == "" || s.model == "" {
		return fallbackAdvice(input), ErrAIDisabled
	}

	profileJSON, _ := json.Marshal(input.Profile)
	prompt := fmt.Sprintf(`用户画像 JSON：%s
补充问题：%s

请作为中国新高考选科顾问，基于用户画像给出克制、可执行的建议。要求：
1. 不承诺录取结果，不替代官方政策与学校老师意见。
2. 优先提醒需要核对的省份政策、专业组选科要求、成绩稳定性和压力风险。
3. 只返回 JSON，不要 Markdown。
JSON 结构：
{"summary":"一句话判断，40字以内","risks":["风险或提醒1","风险或提醒2","风险或提醒3"],"actions":["下一步1","下一步2","下一步3"],"querySuggestions":["可搜索关键词1","可搜索关键词2","可搜索关键词3"]}`, string(profileJSON), input.Question)

	payload := chatCompletionRequest{
		Model: s.model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是严谨的中国高中选科决策助手，回答必须短、稳、可核对。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.ChoiceAdvice{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return domain.ChoiceAdvice{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fallbackAdvice(input), err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fallbackAdvice(input), fmt.Errorf("ai provider returned status %d", resp.StatusCode)
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fallbackAdvice(input), err
	}
	if len(result.Choices) == 0 {
		return fallbackAdvice(input), errors.New("ai provider returned no choices")
	}

	advice, err := parseChoiceAdvice(result.Choices[0].Message.Content)
	if err != nil {
		return fallbackAdvice(input), err
	}
	advice.Source = "ai"
	return limitAdvice(advice), nil
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func parseChoiceAdvice(content string) (domain.ChoiceAdvice, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var advice domain.ChoiceAdvice
	if err := json.Unmarshal([]byte(content), &advice); err != nil {
		return domain.ChoiceAdvice{}, err
	}
	return advice, nil
}

func fallbackAdvice(input domain.ChoiceAdviceInput) domain.ChoiceAdvice {
	track, _ := input.Profile["preferredTrack"].(string)
	subjects, _ := input.Profile["preferredSubjects"].([]any)
	combination := "当前组合"
	if track != "" && len(subjects) > 0 {
		combination = track + " + " + fmt.Sprint(subjects)
	}

	return domain.ChoiceAdvice{
		Summary: "先核对专业要求，再用成绩稳定性做取舍。",
		Risks: []string{
			combination + "不要只看覆盖率，需结合目标省份招生目录。",
			"若单科波动大，优先评估赋分和连续刷题压力。",
			"目标专业未明确时，保留可转向空间比追热门更重要。",
		},
		Actions: []string{
			"搜索目标专业近两年选科要求。",
			"整理三次大考六科排名和波动区间。",
			"收藏 3 篇同省同组合经验帖做对照。",
		},
		QuerySuggestions: []string{"临床医学 选科要求", "物化生 赋分压力", "同省 选科经验"},
		Source:           "fallback",
	}
}

func limitAdvice(advice domain.ChoiceAdvice) domain.ChoiceAdvice {
	advice.Summary = truncateRunes(advice.Summary, 48)
	advice.Risks = limitStrings(advice.Risks, 3, 60)
	advice.Actions = limitStrings(advice.Actions, 3, 60)
	advice.QuerySuggestions = limitStrings(advice.QuerySuggestions, 3, 20)
	return advice
}

func limitStrings(items []string, limit int, maxRunes int) []string {
	if len(items) > limit {
		items = items[:limit]
	}
	for i, item := range items {
		items[i] = truncateRunes(strings.TrimSpace(item), maxRunes)
	}
	return items
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
