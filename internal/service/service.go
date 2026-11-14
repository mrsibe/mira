// Package service exposes the Go side of Mira to the MyGo frontend.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/mrsibe/goat/agent"
	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/mira/internal/chat"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/fonts"
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/prompt"
	"github.com/mrsibe/mira/internal/store"
)

const (
	maxInputRunes = 32_000
)

var (
	errBusy          = errors.New("正在生成回复，请先停止当前回复")
	errNotConfigured = errors.New("请先添加并启用模型")
)

// Service is bound to the page as `Mira`.
type Service struct {
	cfg    *config.Manager
	store  *store.Store
	memory *memory.Manager
	engine agentFactory
	test   func(context.Context, config.Settings) error

	mu     sync.Mutex
	active *activeRun
}

type agentFactory interface {
	NewAgent(config.Settings, string, []ai.Message) (*agent.Agent, error)
}

// New creates the bound application service.
func New(
	cfg *config.Manager,
	conversationStore *store.Store,
	engine agentFactory,
	memoryManager *memory.Manager,
) *Service {
	return &Service{
		cfg:    cfg,
		store:  conversationStore,
		engine: engine,
		memory: memoryManager,
		test:   chat.TestModel,
	}
}

// Bootstrap is the first call the page makes.
type Bootstrap struct {
	Settings             config.View           `json:"settings"`
	Providers            []config.ProviderInfo `json:"providers"`
	Conversations        []store.Conversation  `json:"conversations"`
	Projects             []store.Project       `json:"projects"`
	ActiveConversationID string                `json:"activeConversationId,omitempty"`
	Version              string                `json:"version"`
	ConfigError          string                `json:"configError,omitempty"`
}

// ConversationDetail restores one thread.
type ConversationDetail struct {
	Conversation store.Conversation `json:"conversation"`
	Messages     []chat.UIMessage   `json:"messages"`
	Usage        chat.UsageView     `json:"usage"`
}

// SendOptions contains per-message request controls.
type SendOptions struct {
	ReasoningEffort string `json:"reasoningEffort"`
}

type activeRun struct {
	conversationID string
	cancel         context.CancelFunc
}

// EventSink is implemented by *mygo.Channel[chat.ChatEvent] and by tests.
type EventSink interface {
	Send(chat.ChatEvent) error
}

// Bootstrap returns settings, conversation metadata and the app version.
func (s *Service) Bootstrap() Bootstrap {
	result := Bootstrap{
		Settings:      s.cfg.View(),
		Providers:     config.Providers(),
		Conversations: s.store.List(),
		Projects:      s.store.ListProjects(),
		Version:       mygo.App.Version(),
	}
	if err := s.cfg.LoadError(); err != nil {
		result.ConfigError = err.Error()
	}
	s.mu.Lock()
	if s.active != nil {
		result.ActiveConversationID = s.active.conversationID
	}
	s.mu.Unlock()
	return result
}

// ListConversations returns the sidebar entries.
func (s *Service) ListConversations() []store.Conversation {
	return s.store.List()
}

// SearchConversations searches titles, previews and visible message text.
func (s *Service) SearchConversations(query string) []store.SearchResult {
	return s.store.Search(query)
}

// ListProjects returns project categories in creation order.
func (s *Service) ListProjects() []store.Project {
	return s.store.ListProjects()
}

// ListFonts returns font families installed on this computer.
func (s *Service) ListFonts() []string {
	return fonts.List()
}

// CreateProject adds an empty project category.
func (s *Service) CreateProject(name string) (store.Project, error) {
	return s.store.CreateProject(name)
}

// RenameProject changes a project category's name.
func (s *Service) RenameProject(id, name string) error {
	return s.store.RenameProject(id, name)
}

// DeleteProject removes a project and all conversations in it.
func (s *Service) DeleteProject(id string) error {
	s.mu.Lock()
	activeID := ""
	if s.active != nil {
		activeID = s.active.conversationID
	}
	s.mu.Unlock()
	if activeID != "" {
		for _, conversation := range s.store.List() {
			if conversation.ID == activeID && conversation.ProjectID == id {
				return errors.New("正在生成回复，无法删除该项目")
			}
		}
	}
	hadConversations := false
	for _, conversation := range s.store.List() {
		if conversation.ProjectID == id {
			hadConversations = true
			break
		}
	}
	if err := s.store.DeleteProject(id); err != nil {
		return err
	}
	if hadConversations && s.memory != nil {
		s.memory.MarkDirty()
	}
	return nil
}

// CreateConversation starts a new thread.
func (s *Service) CreateConversation(projectID string) (store.Conversation, error) {
	return s.store.CreateInProject(projectID)
}

// MoveConversation changes a conversation's project category.
func (s *Service) MoveConversation(id, projectID string) error {
	return s.store.MoveConversation(id, projectID)
}

// GetConversation loads a thread and its messages.
func (s *Service) GetConversation(id string) (ConversationDetail, error) {
	conversation, records, err := s.store.GetRecords(id)
	if err != nil {
		return ConversationDetail{}, err
	}
	detail := ConversationDetail{Conversation: conversation}
	detail.Messages = make([]chat.UIMessage, 0, len(records))
	var total ai.Usage
	for _, record := range records {
		if record.ContextKind != "" {
			continue
		}
		message := chat.FromMessage(record.Message, len(detail.Messages))
		message.ThinkingDurationMS = record.ThinkingDurationMS
		if record.Usage != nil {
			message.Usage = chat.UsageViewFrom(*record.Usage)
			total.Add(*record.Usage)
		}
		detail.Messages = append(detail.Messages, message)
	}
	detail.Usage = *chat.UsageViewFrom(total)
	return detail, nil
}

// RenameConversation changes a sidebar title.
func (s *Service) RenameConversation(id, title string) error {
	if err := s.store.Rename(id, title); err != nil {
		return err
	}
	if s.memory != nil {
		s.memory.MarkDirty()
	}
	return nil
}

// DeleteConversation removes a thread. It cannot delete the running thread.
func (s *Service) DeleteConversation(id string) error {
	s.mu.Lock()
	busy := s.active != nil && s.active.conversationID == id
	s.mu.Unlock()
	if busy {
		return errors.New("正在生成回复，无法删除该会话")
	}
	if err := s.store.Delete(id); err != nil {
		return err
	}
	if s.memory != nil {
		s.memory.MarkDirty()
	}
	return nil
}

// SaveSettings persists provider settings.
func (s *Service) SaveSettings(input config.Input) (config.View, error) {
	view, err := s.cfg.Save(input)
	if err != nil {
		return view, err
	}
	if s.memory != nil {
		s.memory.MaybeStart(s.cfg.Settings())
	}
	return view, nil
}

// GetMemory returns editable memory and summary status.
func (s *Service) GetMemory() memory.View {
	if s.memory == nil {
		return memory.View{SummaryThreshold: memory.SummaryThreshold}
	}
	return s.memory.Get()
}

// SaveMemory updates the automatic memory summary.
func (s *Service) SaveMemory(input memory.SaveInput) (memory.View, error) {
	if s.memory == nil {
		return memory.View{}, errors.New("记忆功能未初始化")
	}
	return s.memory.Save(input)
}

// RegenerateMemory rebuilds the memory summary now.
func (s *Service) RegenerateMemory() (memory.View, error) {
	if s.memory == nil {
		return memory.View{}, errors.New("记忆功能未初始化")
	}
	if !s.cfg.View().Configured {
		return memory.View{}, errNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return s.memory.Regenerate(ctx, s.cfg.Settings())
}

// ClearMemory removes the automatic memory summary.
func (s *Service) ClearMemory() (memory.View, error) {
	if s.memory == nil {
		return memory.View{}, errors.New("记忆功能未初始化")
	}
	return s.memory.Clear()
}

// AddModel creates a model and makes it active.
func (s *Service) AddModel(input config.ModelInput) (config.View, error) {
	return s.cfg.AddModel(input)
}

// UpdateModel updates one model's connection fields.
func (s *Service) UpdateModel(input config.ModelInput) (config.View, error) {
	return s.cfg.UpdateModel(input)
}

// DeleteModel removes one model.
func (s *Service) DeleteModel(id string) (config.View, error) {
	return s.cfg.DeleteModel(id)
}

// ActivateModel switches the current model.
func (s *Service) ActivateModel(id string) (config.View, error) {
	return s.cfg.ActivateModel(id)
}

// ClearModelAPIKey removes one model's saved key.
func (s *Service) ClearModelAPIKey(id string) (config.View, error) {
	return s.cfg.ClearModelAPIKey(id)
}

// TestModel verifies a draft model without persisting it.
func (s *Service) TestModel(input config.ModelInput) error {
	settings, err := s.cfg.ResolveModelSettings(input)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if s.test == nil {
		return chat.TestModel(ctx, settings)
	}
	return s.test(ctx, settings)
}

// ClearTavilyAPIKey removes the saved search key.
func (s *Service) ClearTavilyAPIKey() (config.View, error) {
	return s.cfg.ClearTavilyAPIKey()
}

// Stop cancels the active run for a conversation.
func (s *Service) Stop(conversationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil && s.active.conversationID == conversationID {
		s.active.cancel()
	}
}

// Send adds a user message and streams one agent run through a MyGo channel.
func (s *Service) Send(
	ctx context.Context,
	conversationID string,
	content string,
	options SendOptions,
	events *mygo.Channel[chat.ChatEvent],
) error {
	return s.run(ctx, conversationID, content, options, events)
}

// EditMessage replaces one user message, discards everything after it and
// streams a fresh assistant response.
func (s *Service) EditMessage(
	ctx context.Context,
	conversationID string,
	messageIndex int,
	content string,
	options SendOptions,
	events *mygo.Channel[chat.ChatEvent],
) error {
	return s.editMessage(ctx, conversationID, messageIndex, content, options, events)
}

func (s *Service) editMessage(
	ctx context.Context,
	conversationID string,
	messageIndex int,
	content string,
	options SendOptions,
	sink EventSink,
) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("消息不能为空")
	}
	if len([]rune(content)) > maxInputRunes {
		return fmt.Errorf("消息过长，最多 %d 个字符", maxInputRunes)
	}
	if err := validateSendOptions(options); err != nil {
		return err
	}
	if !s.cfg.View().Configured {
		return errNotConfigured
	}

	handle, runCtx, err := s.beginRun(ctx, conversationID)
	if err != nil {
		return err
	}
	defer s.endRun(handle)
	if _, err := s.store.EditUserMessage(conversationID, messageIndex, content); err != nil {
		return err
	}
	if s.memory != nil {
		s.memory.MarkDirty()
	}
	return s.executeRun(runCtx, handle, conversationID, "", options, sink, true, true)
}

// Regenerate reruns the assistant turn containing messageIndex. It keeps the
// preceding user message and discards that turn plus all later history.
func (s *Service) Regenerate(
	ctx context.Context,
	conversationID string,
	messageIndex int,
	options SendOptions,
	events *mygo.Channel[chat.ChatEvent],
) error {
	return s.regenerate(ctx, conversationID, messageIndex, options, events)
}

func (s *Service) regenerate(
	ctx context.Context,
	conversationID string,
	messageIndex int,
	options SendOptions,
	sink EventSink,
) error {
	if err := validateSendOptions(options); err != nil {
		return err
	}
	if !s.cfg.View().Configured {
		return errNotConfigured
	}

	handle, runCtx, err := s.beginRun(ctx, conversationID)
	if err != nil {
		return err
	}
	defer s.endRun(handle)
	if _, err := s.store.TruncateAssistantTurn(conversationID, messageIndex); err != nil {
		return err
	}
	if s.memory != nil {
		s.memory.MarkDirty()
	}
	return s.executeRun(runCtx, handle, conversationID, "", options, sink, true, false)
}

// ForkConversation copies a completed assistant turn into a new conversation.
func (s *Service) ForkConversation(conversationID string, messageIndex int) (ConversationDetail, error) {
	s.mu.Lock()
	busy := s.active != nil
	s.mu.Unlock()
	if busy {
		return ConversationDetail{}, errBusy
	}
	conversation, err := s.store.ForkAssistantTurn(conversationID, messageIndex)
	if err != nil {
		return ConversationDetail{}, err
	}
	if s.memory != nil {
		s.memory.MarkDirty()
	}
	return s.GetConversation(conversation.ID)
}

func (s *Service) run(
	ctx context.Context,
	conversationID string,
	content string,
	options SendOptions,
	sink EventSink,
) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("消息不能为空")
	}
	if len([]rune(content)) > maxInputRunes {
		return fmt.Errorf("消息过长，最多 %d 个字符", maxInputRunes)
	}
	if err := validateSendOptions(options); err != nil {
		return err
	}
	if !s.cfg.View().Configured {
		return errNotConfigured
	}

	handle, runCtx, err := s.beginRun(ctx, conversationID)
	if err != nil {
		return err
	}
	defer s.endRun(handle)
	return s.executeRun(runCtx, handle, conversationID, content, options, sink, false, true)
}

func validateSendOptions(options SendOptions) error {
	switch strings.TrimSpace(options.ReasoningEffort) {
	case "", "low", "medium", "high":
		return nil
	default:
		return fmt.Errorf("不支持的思维强度: %q", options.ReasoningEffort)
	}
}

func (s *Service) executeRun(
	runCtx context.Context,
	handle *activeRun,
	conversationID string,
	content string,
	options SendOptions,
	sink EventSink,
	continueRun bool,
	newUserTurn bool,
) error {
	conversation, records, err := s.store.GetRecords(conversationID)
	if err != nil {
		return err
	}
	if conversation.ID == "" {
		return errors.New("会话不存在")
	}
	reasoningEffort := strings.TrimSpace(options.ReasoningEffort)
	runStarted := time.Now()

	settings := s.cfg.Settings()
	visibleCount := visibleRecordCount(records)
	if !continueRun {
		// Edit and regenerate already end with the user message they are
		// rerunning. Injecting memory here would place context after that
		// message; the existing context record remains earlier in history.
		records, err = s.appendRuntimeMemory(conversationID, settings, records)
		if err != nil {
			return err
		}
	}
	history := make([]ai.Message, len(records))
	for index, record := range records {
		history[index] = record.Message
	}
	assistant, err := s.engine.NewAgent(settings, conversationID, repairToolHistory(history))
	if err != nil {
		return err
	}
	assistant.SetRequestOptions(ai.RequestOptions{
		SessionID:       conversationID,
		ReasoningEffort: reasoningEffort,
	})

	var (
		nextIndex  = visibleCount
		sinkErr    error
		persistErr error
		answerAt   time.Time
	)
	if !continueRun {
		nextIndex++
	}
	emit := func(event chat.ChatEvent) bool {
		if sinkErr != nil {
			return false
		}
		event.ConversationID = conversationID
		if err := sink.Send(event); err != nil {
			sinkErr = err
			handle.cancel()
			return false
		}
		return true
	}
	persist := func(message ai.Message, thinkingDuration time.Duration) {
		if persistErr != nil {
			return
		}
		if err := s.store.AppendWithThinkingDuration(conversationID, message, thinkingDuration); err != nil {
			persistErr = err
			handle.cancel()
		}
	}

	if !emit(chat.ChatEvent{Type: chat.EventRunStart}) {
		return sinkErr
	}
	if s.memory != nil && newUserTurn {
		defer s.memory.ObserveUserTurn(settings)
	}

	handleEvent := func(event agent.Event) {
		if persistErr != nil || sinkErr != nil {
			return
		}
		switch event.Type {
		case agent.EventMessageDelta:
			if event.Text != "" {
				if answerAt.IsZero() {
					answerAt = time.Now()
				}
				emit(chat.ChatEvent{Type: chat.EventMessageDelta, Text: event.Text})
			}
		case agent.EventThinkingStart:
			answerAt = time.Time{}
			emit(chat.ChatEvent{Type: chat.EventThinkingStart})
		case agent.EventThinkingDelta:
			if event.Thinking != "" {
				answerAt = time.Time{}
				emit(chat.ChatEvent{Type: chat.EventThinking, Text: event.Thinking})
			}
		case agent.EventThinkingEnd:
			emit(chat.ChatEvent{Type: chat.EventThinkingEnd})
		case agent.EventToolStart:
			if event.ToolCall != nil {
				answerAt = time.Time{}
				emit(chat.ChatEvent{
					Type: chat.EventToolStart,
					ToolCall: &chat.ToolCallView{
						ID:        event.ToolCall.ID,
						Name:      event.ToolCall.Name,
						Arguments: strings.TrimSpace(string(event.ToolCall.Arguments)),
					},
				})
			}
		case agent.EventToolEnd:
			if event.ToolCall != nil && event.ToolResult != nil {
				emit(chat.ChatEvent{
					Type:       chat.EventToolEnd,
					ToolResult: chat.ToolResultViewFrom(*event.ToolCall, *event.ToolResult),
				})
			}
		case agent.EventMessageEnd:
			if event.Message == nil || event.Message.Role == ai.RoleSystem {
				return
			}
			message := ai.CloneMessage(*event.Message)
			var thinkingDuration time.Duration
			if message.Role == ai.RoleAssistant && len(message.ToolCalls()) == 0 {
				endedAt := answerAt
				if endedAt.IsZero() {
					endedAt = time.Now()
				}
				if elapsed := endedAt.Sub(runStarted); elapsed > 0 {
					thinkingDuration = elapsed
				}
			}
			persist(message, thinkingDuration)
			if message.Role != ai.RoleUser {
				emit(chat.ChatEvent{
					Type:    chat.EventMessage,
					Message: pointer(chat.FromMessage(message, nextIndex)),
				})
				nextIndex++
			}
		}
	}
	unsubscribe := assistant.Subscribe(handleEvent)
	defer unsubscribe()

	var (
		result agent.Result
		runErr error
	)
	if continueRun {
		result, runErr = assistant.Continue(runCtx)
	} else {
		result, runErr = assistant.Prompt(runCtx, content)
	}

	if sinkErr != nil {
		return sinkErr
	}
	if persistErr != nil {
		emit(chat.ChatEvent{
			Type:  chat.EventError,
			Code:  "persist_failed",
			Error: "保存会话失败：" + persistErr.Error(),
		})
		return nil
	}
	var usage *chat.UsageView
	if hasUsage(result.Usage) {
		usage = chat.UsageViewFrom(result.Usage)
		if err := s.store.AppendUsage(conversationID, result.Usage); err != nil {
			emit(chat.ChatEvent{
				Type:  chat.EventError,
				Code:  "usage_failed",
				Error: "保存用量失败：" + err.Error(),
			})
			return nil
		}
	}
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			emit(chat.ChatEvent{
				Type:      chat.EventRunEnd,
				Steps:     result.Steps,
				Usage:     usage,
				Cancelled: true,
			})
			return nil
		}
		if errors.Is(runErr, agent.ErrMaxSteps) {
			emit(chat.ChatEvent{
				Type:  chat.EventError,
				Code:  "max_steps",
				Usage: usage,
				Error: fmt.Sprintf(
					"这次任务已达到单次 %d 个模型轮次的上限。当前进度已保存，回复“继续”可以接着完成。",
					chat.MaxSteps,
				),
			})
			return nil
		}
		emit(chat.ChatEvent{
			Type:  chat.EventError,
			Code:  "agent_error",
			Usage: usage,
			Error: runErr.Error(),
		})
		return nil
	}

	emit(chat.ChatEvent{
		Type:  chat.EventRunEnd,
		Usage: usage,
		Steps: result.Steps,
	})
	return nil
}

func hasUsage(usage ai.Usage) bool {
	return usage.PromptTokens != 0 ||
		usage.CompletionTokens != 0 ||
		usage.CacheReadTokens != 0 ||
		usage.CacheWriteTokens != 0 ||
		usage.ReasoningTokens != 0 ||
		usage.TotalTokens != 0
}

// appendRuntimeMemory keeps dynamic memory in an append-only history event.
// Rebuilding the stable system prompt each run is safe; rewriting a memory block
// inside it would invalidate every cached token that follows.
func (s *Service) appendRuntimeMemory(
	conversationID string,
	settings config.Settings,
	records []store.MessageRecord,
) ([]store.MessageRecord, error) {
	enabled := settings.MemoryEnabled && settings.MemorySummaryEnabled
	summary := ""
	if enabled && s.memory != nil {
		summary = s.memory.Prompt()
	}
	context := prompt.NewRuntimeMemory(enabled, summary)

	lastHash := ""
	for index := len(records) - 1; index >= 0; index-- {
		if records[index].ContextKind == prompt.RuntimeMemoryKind {
			lastHash = records[index].ContextHash
			break
		}
	}
	if !context.ShouldAppend(lastHash) {
		return records, nil
	}

	message := ai.TextMessage(ai.RoleUser, context.Text())
	if err := s.store.AppendContext(
		conversationID,
		prompt.RuntimeMemoryKind,
		context.Hash,
		message,
	); err != nil {
		return nil, err
	}
	return append(records, store.MessageRecord{
		Message:     message,
		Timestamp:   time.Now().UTC(),
		ContextKind: prompt.RuntimeMemoryKind,
		ContextHash: context.Hash,
	}), nil
}

func visibleRecordCount(records []store.MessageRecord) int {
	count := 0
	for _, record := range records {
		if record.ContextKind == "" {
			count++
		}
	}
	return count
}

func (s *Service) beginRun(ctx context.Context, conversationID string) (*activeRun, context.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		return nil, nil, errBusy
	}
	runCtx, cancel := context.WithCancel(ctx)
	handle := &activeRun{conversationID: conversationID, cancel: cancel}
	s.active = handle
	return handle, runCtx, nil
}

func (s *Service) endRun(handle *activeRun) {
	s.mu.Lock()
	if s.active == handle {
		s.active = nil
	}
	s.mu.Unlock()
	handle.cancel()
}

func pointer[T any](value T) *T {
	return &value
}

// repairToolHistory makes restored history safe to send after a cancelled run.
// A cancelled model stream can leave an assistant tool call without its result;
// providers reject that shape, so add an explicit error observation.
func repairToolHistory(history []ai.Message) []ai.Message {
	if len(history) == 0 {
		return history
	}
	repaired := make([]ai.Message, 0, len(history))
	for index := 0; index < len(history); index++ {
		message := history[index]
		repaired = append(repaired, message)
		if message.Role != ai.RoleAssistant {
			continue
		}
		calls := message.ToolCalls()
		if len(calls) == 0 {
			continue
		}

		answered := make(map[string]bool, len(calls))
		next := index + 1
		for next < len(history) && history[next].Role == ai.RoleTool {
			if history[next].ToolCallID != "" {
				answered[history[next].ToolCallID] = true
			}
			repaired = append(repaired, history[next])
			next++
		}
		for _, call := range calls {
			if answered[call.ID] {
				continue
			}
			repaired = append(repaired, ai.ToolTextMessage(
				call.ID,
				call.Name,
				"工具调用未完成（上一条回复被中止）。",
				true,
			))
		}
		index = next - 1
	}
	return repaired
}
