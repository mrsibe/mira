package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	htmlstd "html"
	"io"
	"net/http"
	"net/url"
	"strings"

	xhtml "golang.org/x/net/html"
)

const (
	duckDuckGoHTMLURL    = "https://html.duckduckgo.com/html/"
	duckDuckGoInstantURL = "https://api.duckduckgo.com/"
	userAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36"
)

var (
	errDuckDuckGoBlocked = errors.New("DuckDuckGo 网页搜索可能触发了人机验证")
	errDuckDuckGoEmpty   = errors.New("DuckDuckGo 没有返回搜索结果")
)

// DuckDuckGoProvider uses the public HTML endpoint first and falls back to the
// Instant Answer API when the HTML endpoint is blocked or returns nothing.
type DuckDuckGoProvider struct {
	client       *http.Client
	htmlEndpoint string
	apiEndpoint  string
}

func NewDuckDuckGo(client *http.Client) *DuckDuckGoProvider {
	return &DuckDuckGoProvider{client: client}
}

func (p *DuckDuckGoProvider) Name() string { return "DuckDuckGo" }

func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}
	limit = normalizeLimit(limit)
	ctx, cancel := withRequestTimeout(ctx)
	defer cancel()

	htmlResults, htmlErr := p.searchHTML(ctx, query, limit)
	if len(htmlResults) > 0 {
		return htmlResults, nil
	}

	instantResults, instantErr := p.searchInstant(ctx, query, limit)
	if len(instantResults) > 0 {
		return instantResults, nil
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, providerError(p.Name(), ctx.Err())
	}
	if htmlErr == nil {
		htmlErr = errDuckDuckGoEmpty
	}
	if instantErr == nil {
		instantErr = errDuckDuckGoEmpty
	}
	if errors.Is(htmlErr, errDuckDuckGoBlocked) {
		return nil, fmt.Errorf(
			"DuckDuckGo 可能触发了人机验证，Instant Answer 也没有返回结果；可在设置中改用 Tavily（%v）",
			instantErr,
		)
	}
	return nil, fmt.Errorf(
		"DuckDuckGo 没有返回可用结果；可在设置中改用 Tavily（%v）",
		instantErr,
	)
}

func (p *DuckDuckGoProvider) searchHTML(ctx context.Context, query string, limit int) ([]Result, error) {
	endpoint := p.htmlEndpoint
	if endpoint == "" {
		endpoint = duckDuckGoHTMLURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	values := parsed.Query()
	values.Set("q", query)
	parsed.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := readLimited(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	document := string(body)
	if looksBlocked(document) {
		return nil, errDuckDuckGoBlocked
	}

	results, err := parseDuckDuckGoHTML(body)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errDuckDuckGoEmpty
	}
	return results[:min(limit, len(results))], nil
}

func looksBlocked(document string) bool {
	lower := strings.ToLower(document)
	return strings.Contains(lower, "anomaly-modal") ||
		strings.Contains(lower, "challenge-form") ||
		strings.Contains(lower, "bots use duckduckgo")
}

func parseDuckDuckGoHTML(body []byte) ([]Result, error) {
	root, err := xhtml.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	var results []Result
	seen := make(map[string]struct{})
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && hasClass(node, "result") {
			link := findNode(node, func(item *xhtml.Node) bool {
				return item.Type == xhtml.ElementNode && hasClass(item, "result__a")
			})
			if link != nil {
				rawURL := attribute(link, "href")
				if normalized, ok := normalizeResultURL(rawURL); ok {
					title := cleanText(nodeText(link), 200)
					snippetNode := findNode(node, func(item *xhtml.Node) bool {
						return item.Type == xhtml.ElementNode && hasClass(item, "result__snippet")
					})
					snippet := ""
					if snippetNode != nil {
						snippet = cleanText(nodeText(snippetNode), 500)
					}
					if title != "" {
						if _, exists := seen[normalized]; !exists {
							seen[normalized] = struct{}{}
							results = append(results, Result{
								Title:   validUTF8(title),
								URL:     normalized,
								Snippet: validUTF8(snippet),
							})
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return results, nil
}

func hasClass(node *xhtml.Node, className string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key != "class" {
			continue
		}
		for _, value := range strings.Fields(attribute.Val) {
			if value == className {
				return true
			}
		}
	}
	return false
}

func findNode(node *xhtml.Node, match func(*xhtml.Node) bool) *xhtml.Node {
	if match(node) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findNode(child, match); found != nil {
			return found
		}
	}
	return nil
}

func attribute(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func nodeText(node *xhtml.Node) string {
	var text strings.Builder
	var walk func(*xhtml.Node)
	walk = func(current *xhtml.Node) {
		switch current.Type {
		case xhtml.TextNode:
			text.WriteString(current.Data)
			text.WriteByte(' ')
		case xhtml.ElementNode:
			switch current.Data {
			case "script", "style":
				return
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return htmlstd.UnescapeString(text.String())
}

func normalizeResultURL(raw string) (string, bool) {
	raw = htmlstd.UnescapeString(strings.TrimSpace(raw))
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if parsed.Hostname() == "duckduckgo.com" && strings.HasPrefix(parsed.Path, "/l/") {
		if target := parsed.Query().Get("uddg"); target != "" {
			return normalizeResultURL(target)
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if parsed.Host == "" {
		return "", false
	}
	return parsed.String(), true
}

type instantAnswer struct {
	Abstract      string         `json:"Abstract"`
	AbstractText  string         `json:"AbstractText"`
	AbstractURL   string         `json:"AbstractURL"`
	AbstractSrc   string         `json:"AbstractSource"`
	Answer        string         `json:"Answer"`
	AnswerType    string         `json:"AnswerType"`
	Definition    string         `json:"Definition"`
	DefinitionURL string         `json:"DefinitionURL"`
	Heading       string         `json:"Heading"`
	Redirect      string         `json:"Redirect"`
	Results       []instantTopic `json:"Results"`
	Related       []instantTopic `json:"RelatedTopics"`
}

type instantTopic struct {
	FirstURL string         `json:"FirstURL"`
	Text     string         `json:"Text"`
	Topics   []instantTopic `json:"Topics"`
}

func (p *DuckDuckGoProvider) searchInstant(ctx context.Context, query string, limit int) ([]Result, error) {
	endpoint := p.apiEndpoint
	if endpoint == "" {
		endpoint = duckDuckGoInstantURL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	values := parsed.Query()
	values.Set("q", query)
	values.Set("format", "json")
	values.Set("no_html", "1")
	values.Set("skip_disambig", "1")
	parsed.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	body, err := readLimited(response.Body)
	if err != nil {
		return nil, err
	}
	var answer instantAnswer
	if err := json.Unmarshal(body, &answer); err != nil {
		return nil, err
	}

	results := make([]Result, 0, limit)
	seen := make(map[string]struct{})
	add := func(title, rawURL, snippet string) {
		if len(results) >= limit {
			return
		}
		normalized, ok := normalizeResultURL(rawURL)
		if !ok {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		title = cleanText(title, 200)
		if title == "" {
			title = "DuckDuckGo Instant Answer"
		}
		snippet = cleanText(snippet, 500)
		if snippet == "" {
			return
		}
		seen[normalized] = struct{}{}
		results = append(results, Result{
			Title:   validUTF8(title),
			URL:     normalized,
			Snippet: validUTF8(snippet),
		})
	}

	title := strings.TrimSpace(answer.Heading)
	if title == "" {
		title = strings.TrimSpace(answer.AbstractSrc)
	}
	abstract := answer.AbstractText
	if abstract == "" {
		abstract = answer.Abstract
	}
	add(title, answer.AbstractURL, abstract)
	if answer.DefinitionURL != "" {
		add("DuckDuckGo Definition", answer.DefinitionURL, answer.Definition)
	}
	if answer.Answer != "" {
		add("DuckDuckGo Instant Answer", "https://duckduckgo.com/?q="+url.QueryEscape(query), answer.Answer)
	}
	for _, topic := range answer.Results {
		add(topic.Text, topic.FirstURL, topic.Text)
	}
	var addRelated func([]instantTopic)
	addRelated = func(topics []instantTopic) {
		for _, topic := range topics {
			if len(topic.Topics) > 0 {
				addRelated(topic.Topics)
				continue
			}
			add(topic.Text, topic.FirstURL, topic.Text)
		}
	}
	addRelated(answer.Related)
	if len(results) == 0 {
		return nil, errDuckDuckGoEmpty
	}
	return results, nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
