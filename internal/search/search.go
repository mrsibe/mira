// Package search provides the web search providers used by Mira's agent tools.
package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultMaxResults = 5
	MaxResults        = 10

	maxResponseBytes = 2 << 20
	requestTimeout   = 15 * time.Second
)

var ErrEmptyQuery = errors.New("搜索词不能为空")

// Result is one normalized web search result.
type Result struct {
	Title   string
	URL     string
	Snippet string
}

// Provider searches the web and returns at most limit normalized results.
type Provider interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]Result, error)
}

// NewProvider creates a search provider. Unknown provider names fall back to
// DuckDuckGo so a manually edited configuration never blocks ordinary chat.
func NewProvider(name, tavilyAPIKey string, client *http.Client) Provider {
	if client == nil {
		client = &http.Client{Timeout: requestTimeout + 5*time.Second}
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "tavily":
		return &TavilyProvider{client: client, apiKey: strings.TrimSpace(tavilyAPIKey)}
	default:
		return &DuckDuckGoProvider{client: client}
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultMaxResults
	}
	if limit > MaxResults {
		return MaxResults
	}
	return limit
}

func withRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, requestTimeout)
}

func readLimited(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("搜索响应过大")
	}
	return data, nil
}

func cleanText(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

func validUTF8(value string) string {
	if utf8.ValidString(value) {
		return value
	}
	return strings.ToValidUTF8(value, "\uFFFD")
}

func providerError(provider string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s 搜索已取消或超时", provider)
	}
	return fmt.Errorf("%s 搜索失败: %w", provider, err)
}
