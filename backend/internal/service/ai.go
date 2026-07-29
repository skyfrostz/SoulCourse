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

	content, err := s.complete(ctx, []chatMessage{
		{Role: "system", Content: "你是严谨的中国高中选科决策助手，回答必须短、稳、可核对。"},
		{Role: "user", Content: prompt},
	}, 0.3)
	if err != nil {
		return fallbackAdvice(input), err
	}

	advice, err := parseChoiceAdvice(content)
	if err != nil {
		return fallbackAdvice(input), err
	}
	advice.Source = "ai"
	return limitAdvice(advice), nil
}

func (s *AIService) TagPost(ctx context.Context, title string, content string) ([]string, error) {
	if s.apiKey == "" || s.baseURL == "" || s.model == "" {
		return nil, ErrAIDisabled
	}
	controlled := append(domain.TopicTags(), domain.SubjectTags()...)
	controlledJSON, _ := json.Marshal(controlled)
	prompt := fmt.Sprintf(`标题：%s
正文：%s
受控标签：%s

判断帖子明确匹配哪些受控标签。只返回 JSON：{"tags":["标签值"]}。
只能使用受控标签的 value；没有明确匹配时返回空数组，不要猜测或新增标签。`, title, content, controlledJSON)
	tagContext, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	result, err := s.complete(tagContext, []chatMessage{
		{Role: "system", Content: "你是中文新高考帖子分类器，只做保守的受控多标签分类。"},
		{Role: "user", Content: prompt},
	}, 0)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(stripJSONFences(result)), &parsed); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(parsed.Tags))
	for _, tag := range parsed.Tags {
		tag = strings.TrimSpace(tag)
		if domain.IsControlledTag(tag) && !containsTag(tags, tag) {
			tags = append(tags, tag)
		}
		if len(tags) == 5 {
			break
		}
	}
	return tags, nil
}

func (s *AIService) complete(ctx context.Context, messages []chatMessage, temperature float64) (string, error) {
	payload := chatCompletionRequest{Model: s.model, Messages: messages, Temperature: temperature}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ai provider returned status %d", resp.StatusCode)
	}
	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("ai provider returned no choices")
	}
	return result.Choices[0].Message.Content, nil
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
	var advice domain.ChoiceAdvice
	if err := json.Unmarshal([]byte(stripJSONFences(content)), &advice); err != nil {
		return domain.ChoiceAdvice{}, err
	}
	return advice, nil
}

func stripJSONFences(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
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
