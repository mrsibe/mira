package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mrsibe/goat/agent"
	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/goat/ai/faux"
	"github.com/mrsibe/mira/internal/chat"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/prompt"
	"github.com/mrsibe/mira/internal/store"
)

type factoryFunc func(config.Settings, string) (*agent.Agent, error)

func (f factoryFunc) NewAgent(settings config.Settings, conversationID string) (*agent.Agent, error) {
	return f(settings, conversationID)
}

type recordingSink struct {
	events []chat.ChatEvent
	delta  chan struct{}
}

func (s *recordingSink) Send(event chat.ChatEvent) error {
	s.events = append(s.events, event)
	if event.Type == chat.EventMessageDelta && s.delta != nil {
		select {
		case s.delta <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestRunStreamsEventsAndPersistsHistory(t *testing.T) {
	model := faux.New(faux.Turn{Events: []ai.Event{
		{Type: ai.EventStart},
		{Type: ai.EventTextDelta, TextDelta: "你"},
		{Type: ai.EventTextDelta, TextDelta: "好"},
		{Type: ai.EventUsage, Usage: &ai.Usage{
			PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12,
		}},
		{Type: ai.EventDone, StopReason: ai.StopReasonStop},
	}})
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return agent.New(agent.Config{Model: model, SystemPrompt: "test"})
	}))

	sink := &recordingSink{}
	if err := svc.run(context.Background(), conversationID, "你好", SendOptions{}, sink); err != nil {
		t.Fatal(err)
	}

	if got := deltaText(sink.events); got != "你好" {
		t.Fatalf("streamed text = %q", got)
	}
	runEnd := findEvent(sink.events, chat.EventRunEnd)
	if runEnd == nil || runEnd.Usage == nil || runEnd.Usage.TotalTokens != 12 {
		t.Fatalf("run_end = %+v", runEnd)
	}
	if findEvent(sink.events, chat.EventRunStart) == nil {
		t.Fatal("missing run_start")
	}
	if findEvent(sink.events, chat.EventMessage) == nil {
		t.Fatal("missing persisted assistant message event")
	}

	conversation, history, err := svc.store.Get(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if conversation.Title != "你好" || conversation.MessageCount != 2 {
		t.Fatalf("conversation = %+v", conversation)
	}
	if len(history) != 2 || history[0].Role != ai.RoleUser ||
		history[1].Role != ai.RoleAssistant || history[1].Text() != "你好" {
		t.Fatalf("history = %+v", history)
	}

	detail, err := svc.GetConversation(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Messages) != 2 || detail.Messages[0].Role != "user" ||
		detail.Messages[1].Role != "assistant" {
		t.Fatalf("detail messages = %+v", detail.Messages)
	}
}

func TestRunPassesReasoningEffort(t *testing.T) {
	for _, test := range []struct {
		name   string
		effort string
		want   string
	}{
		{name: "automatic", effort: "", want: ""},
		{name: "low", effort: "low", want: "low"},
		{name: "medium", effort: "medium", want: "medium"},
		{name: "high", effort: "high", want: "high"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := faux.New(faux.Turn{Events: []ai.Event{
				{Type: ai.EventStart},
				{Type: ai.EventTextDelta, TextDelta: "ok"},
				{Type: ai.EventDone, StopReason: ai.StopReasonStop},
			}})
			svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
				return agent.New(agent.Config{Model: model})
			}))

			if err := svc.run(
				context.Background(),
				conversationID,
				"hi",
				SendOptions{ReasoningEffort: test.effort},
				&recordingSink{},
			); err != nil {
				t.Fatal(err)
			}
			if len(model.Requests) != 1 {
				t.Fatalf("model requests = %d", len(model.Requests))
			}
			if got := model.Requests[0].Options.ReasoningEffort; got != test.want {
				t.Fatalf("reasoning effort = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRunAppendsMemoryContextWithoutRewritingPrefix(t *testing.T) {
	model := faux.New(
		textTurn("第一次回答"),
		textTurn("第二次回答"),
	)
	svc, conversationID := newTestService(t, factoryFunc(
		func(_ config.Settings, id string) (*agent.Agent, error) {
			return agent.New(agent.Config{
				Model:          model,
				RequestOptions: ai.RequestOptions{SessionID: id},
			})
		},
	))
	memories, err := memory.Open(t.TempDir(), svc.store, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.memory = memories
	if _, err := memories.Save(memory.SaveInput{
		Summary:  "用户喜欢 Go。",
		Revision: 0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.run(
		context.Background(),
		conversationID,
		"第一条问题",
		SendOptions{},
		&recordingSink{},
	); err != nil {
		t.Fatal(err)
	}

	view := memories.Get()
	if _, err := memories.Save(memory.SaveInput{
		Summary:  "用户喜欢 Go 和 Rust。",
		Revision: view.Revision,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.run(
		context.Background(),
		conversationID,
		"第二条问题",
		SendOptions{},
		&recordingSink{},
	); err != nil {
		t.Fatal(err)
	}

	_, records, err := svc.store.GetRecords(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 6 {
		t.Fatalf("records = %d: %+v", len(records), records)
	}
	for _, index := range []int{0, 3} {
		if records[index].ContextKind != prompt.RuntimeMemoryKind {
			t.Fatalf("record %d is not memory context: %+v", index, records[index])
		}
	}
	if records[0].ContextHash == records[3].ContextHash {
		t.Fatal("changed memory should append a new version")
	}

	detail, err := svc.GetConversation(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Messages) != 4 {
		t.Fatalf("visible messages = %d", len(detail.Messages))
	}
	for index, message := range detail.Messages {
		if message.Index != index {
			t.Fatalf("visible indexes are not contiguous: %+v", detail.Messages)
		}
	}
	if detail.Conversation.Title != "第一条问题" || detail.Conversation.MessageCount != 4 {
		t.Fatalf("conversation metadata = %+v", detail.Conversation)
	}

	if len(model.Requests) != 2 {
		t.Fatalf("model requests = %d", len(model.Requests))
	}
	first := model.Requests[0].Messages
	second := model.Requests[1].Messages
	if len(first) >= len(second) {
		t.Fatalf("second request did not append: first=%d second=%d", len(first), len(second))
	}
	for index := range first {
		if first[index].Role != second[index].Role ||
			first[index].Text() != second[index].Text() {
			t.Fatalf(
				"cached prefix changed at message %d:\nfirst=%+v\nsecond=%+v",
				index,
				first[index],
				second[index],
			)
		}
	}
	if model.Requests[1].Options.SessionID != conversationID {
		t.Fatalf("prompt cache key = %q", model.Requests[1].Options.SessionID)
	}
}

func textTurn(text string) faux.Turn {
	return faux.Turn{Events: []ai.Event{
		{Type: ai.EventStart},
		{Type: ai.EventTextDelta, TextDelta: text},
		{Type: ai.EventDone, StopReason: ai.StopReasonStop},
	}}
}

func TestRunRejectsInvalidReasoningEffort(t *testing.T) {
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		t.Fatal("factory must not be called")
		return nil, nil
	}))

	err := svc.run(
		context.Background(),
		conversationID,
		"hi",
		SendOptions{ReasoningEffort: "max"},
		&recordingSink{},
	)
	if err == nil || !strings.Contains(err.Error(), "思维强度") {
		t.Fatalf("error = %v", err)
	}
	if _, history, getErr := svc.store.Get(conversationID); getErr != nil || len(history) != 0 {
		t.Fatalf("history = %d, err = %v", len(history), getErr)
	}
}

func TestRunRejectsUnconfiguredProvider(t *testing.T) {
	cfg, err := config.NewManager(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	conversations, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	svc := New(cfg, conversations, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return nil, errors.New("factory must not be called")
	}), nil)

	if err := svc.run(
		context.Background(),
		conversation.ID,
		"hello",
		SendOptions{},
		&recordingSink{},
	); !errors.Is(err, errNotConfigured) {
		t.Fatalf("error = %v", err)
	}
	if _, history, err := conversations.Get(conversation.ID); err != nil || len(history) != 0 {
		t.Fatalf("history = %d, err = %v", len(history), err)
	}
}

func TestRunExecutesToolAndPersistsToolHistory(t *testing.T) {
	model := faux.New(
		faux.Turn{Events: []ai.Event{
			{Type: ai.EventStart},
			{Type: ai.EventToolCallDelta, ContentIndex: 0, ToolCallDelta: &ai.ToolCallDelta{
				ID: "call_1", Name: "test_tool", ArgumentsDelta: `{}`,
			}},
			{Type: ai.EventDone, StopReason: ai.StopReasonToolCalls},
		}},
		faux.Turn{Events: []ai.Event{
			{Type: ai.EventStart},
			{Type: ai.EventTextDelta, TextDelta: "工具已完成"},
			{Type: ai.EventDone, StopReason: ai.StopReasonStop},
		}},
	)
	tool := agent.NewTextTool(ai.ToolSpec{Name: "test_tool"}, func(context.Context, ai.ToolCall) (string, error) {
		return "tool output", nil
	})
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return agent.New(agent.Config{
			Model: model,
			Tools: []agent.Tool{tool},
		})
	}))

	sink := &recordingSink{}
	if err := svc.run(context.Background(), conversationID, "use a tool", SendOptions{}, sink); err != nil {
		t.Fatal(err)
	}
	toolStart := findEvent(sink.events, chat.EventToolStart)
	if toolStart == nil || toolStart.ToolCall == nil || toolStart.ToolCall.Name != "test_tool" {
		t.Fatalf("tool_start = %+v", toolStart)
	}
	toolEnd := findEvent(sink.events, chat.EventToolEnd)
	if toolEnd == nil || toolEnd.ToolResult == nil || toolEnd.ToolResult.Text != "tool output" {
		t.Fatalf("tool_end = %+v", toolEnd)
	}

	_, history, err := svc.store.Get(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	roles := make([]ai.Role, 0, len(history))
	for _, message := range history {
		roles = append(roles, message.Role)
	}
	want := []ai.Role{ai.RoleUser, ai.RoleAssistant, ai.RoleTool, ai.RoleAssistant}
	if len(roles) != len(want) {
		t.Fatalf("history roles = %v", roles)
	}
	for index := range want {
		if roles[index] != want[index] {
			t.Fatalf("history roles = %v", roles)
		}
	}
	if history[2].ToolCallID != "call_1" || history[2].Text() != "tool output" {
		t.Fatalf("tool message = %+v", history[2])
	}
	if history[len(history)-1].Text() != "工具已完成" {
		t.Fatalf("final answer = %q", history[len(history)-1].Text())
	}
}

func TestRunRejectsConcurrentRun(t *testing.T) {
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return nil, errors.New("factory must not be called")
	}))

	handle, _, err := svc.beginRun(context.Background(), conversationID)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.endRun(handle)

	err = svc.run(context.Background(), conversationID, "hello", SendOptions{}, &recordingSink{})
	if !errors.Is(err, errBusy) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunReportsMaxStepsAsResumableError(t *testing.T) {
	model := faux.New(faux.Turn{Events: []ai.Event{
		{Type: ai.EventStart},
		{Type: ai.EventToolCallDelta, ContentIndex: 0, ToolCallDelta: &ai.ToolCallDelta{
			ID: "call_1", Name: "test_tool", ArgumentsDelta: `{}`,
		}},
		{Type: ai.EventDone, StopReason: ai.StopReasonToolCalls},
	}})
	tool := agent.NewTextTool(ai.ToolSpec{Name: "test_tool"}, func(context.Context, ai.ToolCall) (string, error) {
		return "tool output", nil
	})
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return agent.New(agent.Config{
			Model:    model,
			Tools:    []agent.Tool{tool},
			MaxSteps: 1,
		})
	}))

	sink := &recordingSink{}
	if err := svc.run(context.Background(), conversationID, "finish this", SendOptions{}, sink); err != nil {
		t.Fatal(err)
	}
	event := findEvent(sink.events, chat.EventError)
	if event == nil || event.Code != "max_steps" {
		t.Fatalf("error event = %+v", event)
	}
	if !strings.Contains(event.Error, "继续") {
		t.Fatalf("error should explain how to resume: %q", event.Error)
	}
}

func TestCancellationPersistsPartialAnswer(t *testing.T) {
	svc, conversationID := newTestService(t, factoryFunc(func(config.Settings, string) (*agent.Agent, error) {
		return agent.New(agent.Config{Model: cancelModel{}, SystemPrompt: "test"})
	}))

	sink := &recordingSink{delta: make(chan struct{}, 1)}
	done := make(chan error, 1)
	go func() {
		done <- svc.run(context.Background(), conversationID, "write something", SendOptions{}, sink)
	}()

	select {
	case <-sink.delta:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a streamed delta")
	}
	svc.Stop(conversationID)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancelled run returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the cancelled run")
	}

	runEnd := findEvent(sink.events, chat.EventRunEnd)
	if runEnd == nil || !runEnd.Cancelled {
		t.Fatalf("run_end = %+v", runEnd)
	}
	_, history, err := svc.store.Get(conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Role != ai.RoleUser ||
		history[1].Role != ai.RoleAssistant || history[1].Text() != "partial" {
		t.Fatalf("partial history = %+v", history)
	}
}

func TestRepairToolHistoryAddsMissingResults(t *testing.T) {
	call := ai.ToolCall{ID: "call_1", Name: "get_current_time", Arguments: []byte(`{}`)}
	history := []ai.Message{
		ai.TextMessage(ai.RoleUser, "what time is it?"),
		{
			Role: ai.RoleAssistant,
			Content: []ai.ContentBlock{{
				Type:     ai.BlockToolCall,
				ToolCall: &call,
			}},
		},
		ai.TextMessage(ai.RoleUser, "hello?"),
	}

	repaired := repairToolHistory(history)
	if len(repaired) != 4 {
		t.Fatalf("repaired history length = %d", len(repaired))
	}
	if repaired[2].Role != ai.RoleTool || repaired[2].ToolCallID != call.ID ||
		!repaired[2].IsError {
		t.Fatalf("synthetic tool result = %+v", repaired[2])
	}
	if repaired[3].Role != ai.RoleUser {
		t.Fatalf("following message = %+v", repaired[3])
	}

	withResult := append([]ai.Message(nil), history[:2]...)
	withResult = append(withResult, ai.ToolTextMessage(call.ID, call.Name, "12:00", false))
	unchanged := repairToolHistory(withResult)
	if len(unchanged) != 3 || unchanged[2].IsError {
		t.Fatalf("existing result was changed: %+v", unchanged)
	}
}

func newTestService(t *testing.T, factory agentFactory) (*Service, string) {
	t.Helper()
	cfg, err := config.NewManager(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Save(config.Input{
		BaseURL:  "http://localhost:11434/v1",
		Model:    "test-model",
		Protocol: config.ProtocolResponses,
	}); err != nil {
		t.Fatal(err)
	}
	conversations, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	return New(cfg, conversations, factory, nil), conversation.ID
}

func deltaText(events []chat.ChatEvent) string {
	var builder strings.Builder
	for _, event := range events {
		if event.Type == chat.EventMessageDelta {
			builder.WriteString(event.Text)
		}
	}
	return builder.String()
}

func findEvent(events []chat.ChatEvent, eventType string) *chat.ChatEvent {
	for index := range events {
		if events[index].Type == eventType {
			return &events[index]
		}
	}
	return nil
}

type cancelModel struct{}

func (cancelModel) Info() ai.ModelInfo {
	return ai.ModelInfo{
		Provider:      "test",
		Model:         "cancel",
		API:           "test",
		SupportsTools: true,
	}
}

func (cancelModel) Stream(ctx context.Context, _ ai.Request) (ai.Stream, error) {
	return &cancelStream{ctx: ctx}, nil
}

type cancelStream struct {
	ctx  context.Context
	sent bool
}

func (s *cancelStream) Next() (ai.Event, error) {
	if !s.sent {
		s.sent = true
		return ai.Event{Type: ai.EventTextDelta, TextDelta: "partial"}, nil
	}
	<-s.ctx.Done()
	return ai.Event{}, s.ctx.Err()
}

func (*cancelStream) Close() error {
	return nil
}
