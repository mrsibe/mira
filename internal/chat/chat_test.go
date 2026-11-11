package chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mrsibe/goat/agent"
	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/search"
)

type stubSearchProvider struct {
	name    string
	results []search.Result
	err     error
	query   string
	limit   int
}

type stubMemory struct {
	summary string
}

func (m stubMemory) Prompt() string {
	return m.summary
}

func (m stubMemory) Search(
	context.Context,
	string,
	int,
	string,
) ([]memory.SearchResult, error) {
	return nil, nil
}

func (p *stubSearchProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "StubSearch"
}

func (p *stubSearchProvider) Search(_ context.Context, query string, limit int) ([]search.Result, error) {
	p.query = query
	p.limit = limit
	return p.results, p.err
}

func TestToolsForHonorsWebSearchSetting(t *testing.T) {
	engine := &Engine{
		Tools: []agent.Tool{currentTimeTool(), updatePlanTool()},
		NewSearchProvider: func(config.Settings) search.Provider {
			return &stubSearchProvider{}
		},
	}

	tools, prompt := engine.toolsFor(config.Settings{WebSearchEnabled: false}, "")
	if hasTool(tools, "web_search") {
		t.Fatal("web_search should be absent when disabled")
	}
	if !hasTool(tools, updatePlanToolName) {
		t.Fatal("update_plan should be present when search is disabled")
	}
	if !strings.Contains(prompt, "update_plan") {
		t.Fatal("plan instructions should be included")
	}
	if strings.Contains(prompt, "web_search") {
		t.Fatal("disabled search instructions were included")
	}

	tools, prompt = engine.toolsFor(config.Settings{WebSearchEnabled: true}, "")
	if !hasTool(tools, "web_search") {
		t.Fatal("web_search should be present when enabled")
	}
	if !strings.Contains(prompt, "web_search") || !strings.Contains(prompt, "Markdown 链接") {
		t.Fatalf("search prompt was not included: %q", prompt)
	}
}

func TestNewEngineIncludesPlanTool(t *testing.T) {
	engine := NewEngine()
	if !hasTool(engine.Tools, updatePlanToolName) {
		t.Fatal("NewEngine should include update_plan")
	}
}

func TestToolsForInjectsMemoryAccordingToSettings(t *testing.T) {
	engine := &Engine{
		Tools:  []agent.Tool{currentTimeTool(), updatePlanTool()},
		Memory: stubMemory{summary: "自动摘要内容"},
	}
	settings := config.Settings{
		MemoryEnabled:        true,
		MemorySummaryEnabled: true,
		MemorySearchEnabled:  true,
	}
	tools, prompt := engine.toolsFor(settings, "current")
	if !strings.Contains(prompt, "自动摘要内容") {
		t.Fatalf("memory was not injected: %q", prompt)
	}
	if !hasTool(tools, "search_memory") {
		t.Fatal("search_memory should be present when memory search is enabled")
	}

	settings.MemorySummaryEnabled = false
	_, prompt = engine.toolsFor(settings, "current")
	if strings.Contains(prompt, "自动摘要内容") {
		t.Fatal("automatic summary should be omitted when its switch is off")
	}

	settings.MemorySearchEnabled = false
	tools, prompt = engine.toolsFor(settings, "current")
	if hasTool(tools, "search_memory") || strings.Contains(prompt, "search_memory") {
		t.Fatal("memory search should be disabled")
	}

	settings.MemoryEnabled = false
	_, prompt = engine.toolsFor(settings, "current")
	if strings.Contains(prompt, "自动摘要内容") {
		t.Fatal("memory should not be injected when memory is disabled")
	}
}

func TestUpdatePlanToolNormalizesAndFormatsSteps(t *testing.T) {
	tool := updatePlanTool()
	result, err := tool.Execute(context.Background(), ai.ToolCall{
		Name: updatePlanToolName,
		Arguments: []byte(`{"steps":[
			{"title":"  搜索   最新发布说明 ","status":"completed"},
			{"title":"比较 X 与 Y 的差异","status":"in_progress"},
			{"title":"整理结论","status":"pending"}
		]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool result = %+v", result)
	}
	for _, want := range []string{
		"[完成] 搜索 最新发布说明",
		"[进行中] 比较 X 与 Y 的差异",
		"[待办] 整理结论",
	} {
		if !strings.Contains(result.Text(), want) {
			t.Fatalf("result missing %q:\n%s", want, result.Text())
		}
	}
}

func TestUpdatePlanToolRejectsInvalidSteps(t *testing.T) {
	longTitle := strings.Repeat("很长的步骤", 17)
	tests := []struct {
		name      string
		arguments string
		want      string
	}{
		{name: "empty", arguments: `{"steps":[]}`, want: "至少需要一个步骤"},
		{name: "too many", arguments: `{"steps":[
			{"title":"1","status":"pending"},{"title":"2","status":"pending"},
			{"title":"3","status":"pending"},{"title":"4","status":"pending"},
			{"title":"5","status":"pending"},{"title":"6","status":"pending"},
			{"title":"7","status":"pending"},{"title":"8","status":"pending"},
			{"title":"9","status":"pending"}]}`, want: "最多支持 8 个步骤"},
		{name: "empty title", arguments: `{"steps":[{"title":"  ","status":"pending"}]}`, want: "缺少标题"},
		{name: "long title", arguments: `{"steps":[{"title":"` + longTitle + `","status":"pending"}]}`, want: "超过 80 个字符"},
		{name: "bad status", arguments: `{"steps":[{"title":"测试","status":"doing"}]}`, want: "状态无效"},
		{name: "two in progress", arguments: `{"steps":[
			{"title":"一","status":"in_progress"},
			{"title":"二","status":"in_progress"}]}`, want: "最多只能有一个进行中的步骤"},
	}
	tool := updatePlanTool()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), ai.ToolCall{
				Name:      updatePlanToolName,
				Arguments: []byte(test.arguments),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Fatal("expected an error result")
			}
			if !strings.Contains(result.Text(), test.want) {
				t.Fatalf("result %q does not contain %q", result.Text(), test.want)
			}
		})
	}
}

func TestWebSearchToolFormatsResults(t *testing.T) {
	provider := &stubSearchProvider{results: []search.Result{
		{Title: "Go [docs]", URL: "https://go.dev/", Snippet: "The Go language."},
		{Title: "Blog", URL: "https://go.dev/blog/", Snippet: "Go news."},
	}}
	tool := webSearchTool(provider)
	result, err := tool.Execute(context.Background(), ai.ToolCall{
		Name:      "web_search",
		Arguments: []byte(`{"query":"golang","max_results":2}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool result = %+v", result)
	}
	if provider.query != "golang" || provider.limit != 2 {
		t.Fatalf("provider call = query %q, limit %d", provider.query, provider.limit)
	}
	text := result.Text()
	for _, want := range []string{
		"StubSearch",
		"[Go \\[docs\\]](<https://go.dev/>)",
		"[Blog](<https://go.dev/blog/>)",
		"Markdown 链接",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("result missing %q:\n%s", want, text)
		}
	}
}

func TestWebSearchToolReturnsErrorsAsToolResults(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments string
		provider  *stubSearchProvider
		want      string
	}{
		{
			name:      "empty query",
			arguments: `{"query":" "}`,
			provider:  &stubSearchProvider{},
			want:      "搜索词不能为空",
		},
		{
			name:      "provider error",
			arguments: `{"query":"golang"}`,
			provider:  &stubSearchProvider{err: errors.New("network down")},
			want:      "network down",
		},
		{
			name:      "bad arguments",
			arguments: `{`,
			provider:  &stubSearchProvider{},
			want:      "搜索参数无效",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tool := webSearchTool(test.provider)
			result, err := tool.Execute(context.Background(), ai.ToolCall{
				Name:      "web_search",
				Arguments: []byte(test.arguments),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError || !strings.Contains(result.Text(), test.want) {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func hasTool(tools []agent.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Spec().Name == name {
			return true
		}
	}
	return false
}
