package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDuckDuckGoHTMLParsesResults(t *testing.T) {
	fixture := `<!doctype html>
<html><body>
  <div class="result results_links">
    <h2><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fgo%3Fa%3D1">Go &amp; Agents</a></h2>
    <a class="result__snippet">Build <b>agents</b> with Go.</a>
  </div>
  <div class="result results_links">
    <h2><a class="result__a" href="https://example.org/second">Second result</a></h2>
    <div class="result__snippet">Another result.</div>
  </div>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "go agents" {
			t.Errorf("query = %q", got)
		}
		if got := r.Header.Get("User-Agent"); !strings.Contains(got, "Mozilla") {
			t.Errorf("user agent = %q", got)
		}
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	provider := &DuckDuckGoProvider{
		client:       server.Client(),
		htmlEndpoint: server.URL + "/html/",
	}
	results, err := provider.Search(context.Background(), "go agents", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Title != "Go & Agents" || results[0].URL != "https://example.com/go?a=1" {
		t.Fatalf("first result = %+v", results[0])
	}
	if results[0].Snippet != "Build agents with Go." {
		t.Fatalf("snippet = %q", results[0].Snippet)
	}

	all, err := provider.Search(context.Background(), "go agents", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[1].URL != "https://example.org/second" {
		t.Fatalf("all results = %+v", all)
	}
}

func TestDuckDuckGoChallengeFallsBackToInstantAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/html/":
			_, _ = w.Write([]byte(`<form id="challenge-form"></form><div class="anomaly-modal">Unfortunately, bots use DuckDuckGo too.</div>`))
		case "/ia/":
			if got := r.URL.Query().Get("format"); got != "json" {
				t.Errorf("format = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Heading":      "Go",
				"AbstractText": "Go is a programming language.",
				"AbstractURL":  "https://en.wikipedia.org/wiki/Go_(programming_language)",
				"RelatedTopics": []map[string]any{
					{"Text": "Go toolchain", "FirstURL": "https://go.dev/"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &DuckDuckGoProvider{
		client:       server.Client(),
		htmlEndpoint: server.URL + "/html/",
		apiEndpoint:  server.URL + "/ia/",
	}
	results, err := provider.Search(context.Background(), "golang", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].URL != "https://en.wikipedia.org/wiki/Go_(programming_language)" {
		t.Fatalf("abstract result = %+v", results[0])
	}
	if results[1].URL != "https://go.dev/" {
		t.Fatalf("related result = %+v", results[1])
	}
}

func TestDuckDuckGoEmptyFallbackIsActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/html/" {
			_, _ = w.Write([]byte(`<div class="anomaly-modal"></div>`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider := &DuckDuckGoProvider{
		client:       server.Client(),
		htmlEndpoint: server.URL + "/html/",
		apiEndpoint:  server.URL + "/ia/",
	}
	_, err := provider.Search(context.Background(), "no results", 5)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "人机验证") || !strings.Contains(err.Error(), "Tavily") {
		t.Fatalf("error = %v", err)
	}
}

func TestTavilySearchRequestAndResults(t *testing.T) {
	const testKey = "test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testKey {
			t.Errorf("authorization = %q", got)
		}
		var body struct {
			Query          string `json:"query"`
			SearchDepth    string `json:"search_depth"`
			MaxResults     int    `json:"max_results"`
			IncludeAnswer  bool   `json:"include_answer"`
			IncludeContent bool   `json:"include_raw_content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Query != "golang" || body.SearchDepth != "basic" || body.MaxResults != 2 ||
			body.IncludeAnswer || body.IncludeContent {
			t.Errorf("request body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"results":[
			{"title":"Go","url":"https://go.dev/","content":"The Go programming language."},
			{"title":"Go blog","url":"https://go.dev/blog/","content":"News from Go."}
		]}`))
	}))
	defer server.Close()

	provider := &TavilyProvider{client: server.Client(), apiKey: testKey, endpoint: server.URL}
	results, err := provider.Search(context.Background(), "golang", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Title != "Go" || results[1].URL != "https://go.dev/blog/" {
		t.Fatalf("results = %+v", results)
	}
}

func TestTavilyErrorsDoNotExposeKey(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		want   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: "API Key 无效"},
		{name: "rate limited", status: http.StatusTooManyRequests, want: "过于频繁"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			defer server.Close()
			provider := &TavilyProvider{
				client:   server.Client(),
				apiKey:   "tvly-secret",
				endpoint: server.URL,
			}
			_, err := provider.Search(context.Background(), "query", 5)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
			if strings.Contains(err.Error(), "tvly-secret") {
				t.Fatalf("error exposed key: %v", err)
			}
		})
	}
}

func TestNormalizeLimit(t *testing.T) {
	for _, test := range []struct {
		input int
		want  int
	}{
		{input: 0, want: DefaultMaxResults},
		{input: -1, want: DefaultMaxResults},
		{input: 1, want: 1},
		{input: 100, want: MaxResults},
	} {
		if got := normalizeLimit(test.input); got != test.want {
			t.Errorf("normalizeLimit(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}
