package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const tavilySearchURL = "https://api.tavily.com/search"

// TavilyProvider calls Tavily's search API.
type TavilyProvider struct {
	client   *http.Client
	apiKey   string
	endpoint string
}

func NewTavily(client *http.Client, apiKey string) *TavilyProvider {
	return &TavilyProvider{client: client, apiKey: strings.TrimSpace(apiKey)}
}

func (p *TavilyProvider) Name() string { return "Tavily" }

func (p *TavilyProvider) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	if p.apiKey == "" {
		return nil, errors.New("未配置 Tavily API Key，请在设置中填写或改用 DuckDuckGo")
	}
	limit = normalizeLimit(limit)
	ctx, cancel := withRequestTimeout(ctx)
	defer cancel()

	endpoint := p.endpoint
	if endpoint == "" {
		endpoint = tavilySearchURL
	}
	payload, err := json.Marshal(map[string]any{
		"query":               query,
		"search_depth":        "basic",
		"max_results":         limit,
		"include_answer":      false,
		"include_raw_content": false,
	})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", userAgent)

	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, errors.New("Tavily API Key 无效或没有权限")
		case http.StatusTooManyRequests:
			return nil, errors.New("Tavily 请求过于频繁，请稍后再试")
		default:
			return nil, fmt.Errorf("Tavily HTTP %d", response.StatusCode)
		}
	}
	body, err := readLimited(response.Body)
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, errors.New("Tavily 返回了无法解析的响应")
	}

	results := make([]Result, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, item := range decoded.Results {
		if len(results) >= limit {
			break
		}
		normalized, ok := normalizeResultURL(item.URL)
		if !ok {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		title := cleanText(item.Title, 200)
		snippet := cleanText(item.Content, 500)
		if title == "" && snippet == "" {
			continue
		}
		if title == "" {
			title = normalized
		}
		seen[normalized] = struct{}{}
		results = append(results, Result{
			Title:   validUTF8(title),
			URL:     normalized,
			Snippet: validUTF8(snippet),
		})
	}
	if len(results) == 0 {
		return nil, errors.New("Tavily 没有返回可用结果")
	}
	return results, nil
}
