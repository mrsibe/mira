// Package chat adapts GOAT's agent loop to the small event protocol used by
// the Mira frontend.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mrsibe/goat/agent"
	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/goat/ai/anthropic"
	"github.com/mrsibe/goat/ai/azure"
	"github.com/mrsibe/goat/ai/deepseek"
	"github.com/mrsibe/goat/ai/google"
	"github.com/mrsibe/goat/ai/openai"
	"github.com/mrsibe/goat/ai/openrouter"
	"github.com/mrsibe/goat/ai/xai"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/prompt"
	"github.com/mrsibe/mira/internal/search"
	"github.com/mrsibe/mira/locales"
)

const (
	EventRunStart      = "run_start"
	EventMessageDelta  = "message_delta"
	EventThinkingStart = "thinking_start"
	EventThinking      = "thinking"
	EventThinkingEnd   = "thinking_end"
	EventToolStart     = "tool_start"
	EventToolEnd       = "tool_end"
	EventMessage       = "message"
	EventRunEnd        = "run_end"
	EventError         = "error"
)

const updatePlanToolName = "update_plan"

// MaxSteps is the per-run safety budget for model turns. GOAT counts every
// model turn, including turns that only update the plan, so a multi-step
// research task can legitimately consume far more than a dozen turns.
const MaxSteps = 64

// Memory supplies historical search to the agent.
type Memory interface {
	Search(context.Context, string, int, string) ([]memory.SearchResult, error)
}

// Engine creates GOAT agents from the saved provider settings.
type Engine struct {
	Tools             []agent.Tool
	NewSearchProvider func(config.Settings) search.Provider
	Memory            Memory
}

// NewEngine builds the shared tool set.
func NewEngine() *Engine {
	client := &http.Client{Timeout: 20 * time.Second}
	return &Engine{
		Tools: []agent.Tool{currentTimeTool(), updatePlanTool()},
		NewSearchProvider: func(settings config.Settings) search.Provider {
			return search.NewProvider(settings.SearchProvider, settings.TavilyAPIKey, client)
		},
	}
}

// NewAgent creates an agent initialized with one conversation transcript.
func (e *Engine) NewAgent(
	settings config.Settings,
	conversationID string,
	messages []ai.Message,
) (*agent.Agent, error) {
	model, err := newModel(settings)
	if err != nil {
		return nil, err
	}
	tools, systemPrompt := e.toolsFor(settings, conversationID)
	return agent.New(agent.Config{
		Model:          model,
		Messages:       messages,
		Tools:          tools,
		SystemPrompt:   systemPrompt,
		MaxSteps:       MaxSteps,
		RequestOptions: ai.RequestOptions{SessionID: conversationID},
	})
}

func newModel(settings config.Settings) (ai.Model, error) {
	var (
		model ai.Model
		err   error
	)
	switch settings.Provider {
	case config.ProviderAzure:
		cfg := azure.Config{
			APIKey:   settings.APIKey,
			Endpoint: settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		}
		if settings.Protocol == config.ProtocolCompletions {
			model, err = azure.NewCompletions(cfg)
		} else {
			model, err = azure.NewResponses(cfg)
		}
	case config.ProviderGoogle:
		model, err = google.New(google.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		})
	case config.ProviderAnthropic:
		model, err = anthropic.New(anthropic.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		})
	case config.ProviderDeepSeek:
		model, err = deepseek.New(deepseek.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		})
	case config.ProviderOpenRouter:
		cfg := openrouter.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		}
		if settings.Protocol == config.ProtocolMessages {
			model, err = openrouter.NewMessages(cfg)
		} else {
			model, err = openrouter.NewCompletions(cfg)
		}
	case config.ProviderXAI:
		model, err = xai.New(xai.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		})
	default:
		cfg := openai.Config{
			APIKey:   settings.APIKey,
			BaseURL:  settings.BaseURL,
			Model:    settings.Model,
			Provider: settings.Provider,
		}
		if settings.Protocol == config.ProtocolCompletions {
			model, err = openai.NewCompletions(cfg)
		} else {
			model, err = openai.NewResponses(cfg)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}
	return model, nil
}

func (e *Engine) toolsFor(settings config.Settings, conversationID string) ([]agent.Tool, string) {
	tools := append([]agent.Tool(nil), e.Tools...)
	systemPrompt := prompt.Build(prompt.Options{Settings: settings, Now: time.Now()})
	if settings.WebSearchEnabled {
		provider := search.NewProvider(settings.SearchProvider, settings.TavilyAPIKey, nil)
		if e.NewSearchProvider != nil {
			provider = e.NewSearchProvider(settings)
		}
		if provider == nil {
			provider = search.NewProvider(settings.SearchProvider, settings.TavilyAPIKey, nil)
		}
		tools = append(tools, webSearchTool(provider))
	}
	if settings.MemoryEnabled && settings.MemorySearchEnabled && e.Memory != nil {
		tools = append(tools, memorySearchTool(e.Memory, conversationID))
	}
	return tools, systemPrompt
}

// Complete runs a single model completion without tools. It is used by the
// memory summarizer.
func Complete(
	ctx context.Context,
	settings config.Settings,
	systemPrompt string,
	userPrompt string,
) (string, error) {
	model, err := newModel(settings)
	if err != nil {
		return "", err
	}
	stream, err := model.Stream(ctx, ai.Request{
		Messages: []ai.Message{
			ai.TextMessage(ai.RoleSystem, systemPrompt),
			ai.TextMessage(ai.RoleUser, userPrompt),
		},
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var text strings.Builder
	for {
		event, nextErr := stream.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return "", nextErr
		}
		switch event.Type {
		case ai.EventTextDelta:
			text.WriteString(event.TextDelta)
		case ai.EventDone:
			if event.Message != nil && strings.TrimSpace(event.Message.Text()) != "" {
				return event.Message.Text(), nil
			}
		case ai.EventError:
			if event.Err != nil {
				return "", event.Err
			}
			if event.Error != "" {
				return "", errors.New(event.Error)
			}
			return "", errors.New("model stream reported an error")
		}
	}
	return text.String(), nil
}

// TestModel sends one minimal request through the same model factory used by
// chat runs. It does not persist anything.
func TestModel(ctx context.Context, settings config.Settings) error {
	model, err := newModel(settings)
	if err != nil {
		return err
	}
	stream, err := model.Stream(ctx, ai.Request{
		Messages: []ai.Message{ai.TextMessage(ai.RoleUser, "Reply with OK.")},
		Options:  ai.RequestOptions{MaxTokens: 16},
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		event, nextErr := stream.Next()
		if errors.Is(nextErr, io.EOF) {
			return errors.New(locales.ErrConnectionClosed())
		}
		if nextErr != nil {
			return nextErr
		}
		switch event.Type {
		case ai.EventTextDelta, ai.EventDone:
			return nil
		case ai.EventError:
			if event.Err != nil {
				return event.Err
			}
			if event.Error != "" {
				return errors.New(event.Error)
			}
			return errors.New("model stream reported an error")
		}
	}
}

// ChatEvent is one streamed update sent through a MyGo channel.
type ChatEvent struct {
	Type           string          `json:"type"`
	ConversationID string          `json:"conversationId,omitempty"`
	Text           string          `json:"text,omitempty"`
	Message        *UIMessage      `json:"message,omitempty"`
	ToolCall       *ToolCallView   `json:"toolCall,omitempty"`
	ToolResult     *ToolResultView `json:"toolResult,omitempty"`
	Usage          *UsageView      `json:"usage,omitempty"`
	Steps          int             `json:"steps,omitempty"`
	Cancelled      bool            `json:"cancelled,omitempty"`
	Code           string          `json:"code,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// UIMessage is the frontend-safe representation of an agent message.
type UIMessage struct {
	Index              int        `json:"index"`
	Role               string     `json:"role"`
	Blocks             []UIBlock  `json:"blocks"`
	ToolCallID         string     `json:"toolCallId,omitempty"`
	ToolName           string     `json:"toolName,omitempty"`
	IsError            bool       `json:"isError,omitempty"`
	ThinkingDurationMS int64      `json:"thinkingDurationMs,omitempty"`
	Usage              *UsageView `json:"usage,omitempty"`
}

// UIBlock is a text, thinking, or tool-call block.
type UIBlock struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ToolCall *ToolCallView `json:"toolCall,omitempty"`
}

// ToolCallView describes a model-requested tool call.
type ToolCallView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResultView describes a completed tool call.
type ToolResultView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Text    string `json:"text"`
	IsError bool   `json:"isError,omitempty"`
}

// UsageView contains the token counts of one run.
type UsageView struct {
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	CacheReadTokens  int64 `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64 `json:"cacheWriteTokens,omitempty"`
	ReasoningTokens  int64 `json:"reasoningTokens,omitempty"`
	TotalTokens      int64 `json:"totalTokens"`
}

// FromMessage converts a GOAT message to the frontend shape.
func FromMessage(message ai.Message, index int) UIMessage {
	result := UIMessage{
		Index:      index,
		Role:       string(message.Role),
		ToolCallID: message.ToolCallID,
		ToolName:   message.ToolName,
		IsError:    message.IsError,
	}
	for _, block := range message.Content {
		switch block.Type {
		case ai.BlockText:
			result.Blocks = append(result.Blocks, UIBlock{Type: "text", Text: block.Text})
		case ai.BlockThinking:
			result.Blocks = append(result.Blocks, UIBlock{Type: "thinking", Text: block.Thinking})
		case ai.BlockToolCall:
			if block.ToolCall == nil {
				continue
			}
			call := &ToolCallView{
				ID:        block.ToolCall.ID,
				Name:      block.ToolCall.Name,
				Arguments: strings.TrimSpace(string(block.ToolCall.Arguments)),
			}
			if call.Arguments == "" {
				call.Arguments = "{}"
			}
			result.Blocks = append(result.Blocks, UIBlock{Type: "tool_call", ToolCall: call})
		case ai.BlockImage:
			result.Blocks = append(result.Blocks, UIBlock{Type: "text", Text: locales.ImagePlaceholder()})
		}
	}
	return result
}

// ToolResultViewFrom converts a GOAT tool result.
func ToolResultViewFrom(call ai.ToolCall, result agent.ToolResult) *ToolResultView {
	return &ToolResultView{
		ID:      call.ID,
		Name:    call.Name,
		Text:    result.Text(),
		IsError: result.IsError,
	}
}

// UsageViewFrom converts GOAT usage.
func UsageViewFrom(usage ai.Usage) *UsageView {
	return &UsageView{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		ReasoningTokens:  usage.ReasoningTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func currentTimeTool() agent.Tool {
	return agent.NewTextTool(agent.ToolSpec{
		Name:        "get_current_time",
		Description: locales.ToolCurrentTimeDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": locales.ToolCurrentTimeZoneDescription(),
				},
			},
			"additionalProperties": false,
		},
	}, func(_ context.Context, call ai.ToolCall) (string, error) {
		var input struct {
			Timezone string `json:"timezone"`
		}
		if len(call.Arguments) > 0 {
			if err := json.Unmarshal(call.Arguments, &input); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
		}
		location := time.Local
		if strings.TrimSpace(input.Timezone) != "" {
			var err error
			location, err = time.LoadLocation(strings.TrimSpace(input.Timezone))
			if err != nil {
				return "", fmt.Errorf("unknown timezone %q", input.Timezone)
			}
		}
		now := time.Now().In(location)
		return now.Format(time.RFC3339) + "（" + location.String() + "）", nil
	})
}

type planStep struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

func updatePlanTool() agent.Tool {
	return agent.NewTool(agent.ToolSpec{
		Name:        updatePlanToolName,
		Description: locales.ToolUpdatePlanDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"steps": map[string]any{
					"type":     "array",
					"minItems": 1,
					"maxItems": 8,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"title": map[string]any{
								"type":        "string",
								"description": locales.ToolUpdatePlanStepTitleDescription(),
								"maxLength":   80,
							},
							"status": map[string]any{
								"type": "string",
								"enum": []string{"pending", "in_progress", "completed"},
							},
						},
						"required":             []string{"title", "status"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"steps"},
			"additionalProperties": false,
		},
	}, func(_ context.Context, call ai.ToolCall) (agent.ToolResult, error) {
		var input struct {
			Steps []planStep `json:"steps"`
		}
		if len(call.Arguments) > 0 {
			if err := json.Unmarshal(call.Arguments, &input); err != nil {
				return agent.ErrorResult(locales.ToolUpdatePlanInvalidArgs(err.Error())), nil
			}
		}
		steps, err := normalizePlanSteps(input.Steps)
		if err != nil {
			return agent.ErrorResult(locales.ToolUpdatePlanInvalid(err.Error())), nil
		}
		return agent.TextResult(formatPlan(steps)), nil
	})
}

func normalizePlanSteps(steps []planStep) ([]planStep, error) {
	if len(steps) == 0 {
		return nil, errors.New(locales.ToolUpdatePlanNeedStep())
	}
	if len(steps) > 8 {
		return nil, errors.New(locales.ToolUpdatePlanTooManySteps())
	}
	normalized := make([]planStep, len(steps))
	inProgress := 0
	for index, step := range steps {
		title := strings.Join(strings.Fields(step.Title), " ")
		if title == "" {
			return nil, errors.New(locales.ToolUpdatePlanStepMissingTitle(index + 1))
		}
		if len([]rune(title)) > 80 {
			return nil, errors.New(locales.ToolUpdatePlanStepTitleTooLong(index + 1))
		}
		switch step.Status {
		case "pending", "in_progress", "completed":
		default:
			return nil, errors.New(locales.ToolUpdatePlanStepInvalidStatus(index + 1))
		}
		if step.Status == "in_progress" {
			inProgress++
		}
		normalized[index] = planStep{Title: title, Status: step.Status}
	}
	if inProgress > 1 {
		return nil, errors.New(locales.ToolUpdatePlanMultipleInProgress())
	}
	return normalized, nil
}

func formatPlan(steps []planStep) string {
	var text strings.Builder
	text.WriteString(locales.ToolUpdatePlanUpdated() + "\n")
	hasInProgress := false
	allCompleted := true
	for index, step := range steps {
		if step.Status == "in_progress" {
			hasInProgress = true
		}
		if step.Status != "completed" {
			allCompleted = false
		}
		fmt.Fprintf(&text, "%d. [%s] %s\n", index+1, planStatusLabel(step.Status), step.Title)
	}
	switch {
	case allCompleted:
		text.WriteString(locales.ToolUpdatePlanAllDone())
	case hasInProgress:
		text.WriteString(locales.ToolUpdatePlanContinue())
	default:
		text.WriteString(locales.ToolUpdatePlanPickStep())
	}
	return text.String()
}

func planStatusLabel(status string) string {
	switch status {
	case "pending":
		return locales.PlanStatusPending()
	case "in_progress":
		return locales.PlanStatusInProgress()
	case "completed":
		return locales.PlanStatusCompleted()
	default:
		return status
	}
}

func memorySearchTool(memory Memory, conversationID string) agent.Tool {
	return agent.NewTool(agent.ToolSpec{
		Name:        "search_memory",
		Description: locales.ToolSearchMemoryDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": locales.ToolSearchMemoryQueryDescription(),
					"maxLength":   200,
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": locales.ToolSearchMemoryLimitDescription(),
					"minimum":     1,
					"maximum":     10,
				},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
	}, func(ctx context.Context, call ai.ToolCall) (agent.ToolResult, error) {
		var input struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if len(call.Arguments) > 0 {
			if err := json.Unmarshal(call.Arguments, &input); err != nil {
				return agent.ErrorResult(locales.ToolSearchMemoryInvalidArgs(err.Error())), nil
			}
		}
		query := strings.TrimSpace(input.Query)
		if query == "" {
			return agent.ErrorResult(locales.ToolSearchMemoryEmptyQuery()), nil
		}
		if len([]rune(query)) > 200 {
			return agent.ErrorResult(locales.ToolSearchMemoryQueryTooLong()), nil
		}
		results, err := memory.Search(ctx, query, input.Limit, conversationID)
		if err != nil {
			return agent.ErrorResult(locales.ToolSearchMemoryFailed(err.Error())), nil
		}
		if len(results) == 0 {
			return agent.TextResult(locales.ToolSearchMemoryNoResults()), nil
		}
		var text strings.Builder
		text.WriteString(locales.ToolSearchMemoryResultsIntro(query))
		for index, result := range results {
			fmt.Fprintf(&text, "%d. [%s] %s\n%s\n\n",
				index+1,
				result.Timestamp.Local().Format("2006-01-02 15:04"),
				result.Title,
				result.Text,
			)
		}
		text.WriteString(locales.ToolSearchMemoryNote())
		return agent.TextResult(text.String()), nil
	})
}

func webSearchTool(provider search.Provider) agent.Tool {
	return agent.NewTool(agent.ToolSpec{
		Name:        "web_search",
		Description: locales.ToolWebSearchDescription(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": locales.ToolWebSearchQueryDescription(),
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": locales.ToolSearchMemoryLimitDescription(),
					"minimum":     1,
					"maximum":     search.MaxResults,
				},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
	}, func(ctx context.Context, call ai.ToolCall) (agent.ToolResult, error) {
		var input struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if len(call.Arguments) > 0 {
			if err := json.Unmarshal(call.Arguments, &input); err != nil {
				return agent.ErrorResult(locales.ToolWebSearchInvalidArgs(err.Error())), nil
			}
		}
		query := strings.TrimSpace(input.Query)
		if query == "" {
			return agent.ErrorResult(locales.ToolWebSearchEmptyQuery()), nil
		}
		results, err := provider.Search(ctx, query, input.MaxResults)
		if err != nil {
			return agent.ErrorResult(locales.ToolWebSearchFailed(err.Error())), nil
		}
		if len(results) == 0 {
			return agent.ErrorResult(locales.ToolWebSearchNoResults()), nil
		}
		return agent.TextResult(formatSearchResults(provider.Name(), query, results)), nil
	})
}

func formatSearchResults(provider, query string, results []search.Result) string {
	var text strings.Builder
	text.WriteString(locales.ToolWebSearchResultsIntro(provider, query))
	for index, result := range results {
		title := strings.NewReplacer("[", "\\[", "]", "\\]").Replace(result.Title)
		fmt.Fprintf(&text, "%d. [%s](<%s>)\n", index+1, title, result.URL)
		if result.Snippet != "" {
			fmt.Fprintf(&text, "   %s\n", result.Snippet)
		}
	}
	text.WriteString(locales.ToolWebSearchNote())
	return text.String()
}
