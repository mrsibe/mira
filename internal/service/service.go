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
	"github.com/mrsibe/mira/internal/memory"
	"github.com/mrsibe/mira/internal/prompt"
	"github.com/mrsibe/mira/internal/store"
)

const (
	maxInputRunes = 32_000
)

var (
	errBusy          = errors.New("正在生成回复，请先停止当前回复")
	errNotConfigured = errors.New("请先在设置中配置模型和 API Key")
)

// Service is bound to the page as `Mira`.
type Service struct {
	cfg    *config.Manager
	store  *store.Store
	memory *memory.Manager
	engine agentFactory

	mu     sync.Mutex
	active *activeRun
}

type agentFactory interface {
	NewAgent(config.Settings, string) (*agent.Agent, error)
}

// New creates the bound application service.
func New(
	cfg *config.Manager,
	conversationStore *store.Store,
	engine agentFactory,
	memoryManager *memory.Manager,
) *Service {
	return &Service{cfg: cfg, store: conversationStore, engine: engine, memory: memoryManager}
}

// Bootstrap is the first call the page makes.
type Bootstrap struct {
	Settings             config.View          `json:"settings"`
	Conversations        []store.Conversation `json:"conversations"`
	Projects             []store.Project      `json:"projects"`
	ActiveConversationID string               `json:"activeConversationId,omitempty"`
	Version              string               `json:"version"`
	ConfigError          string               `json:"configError,omitempty"`
}

// ConversationDetail restores one thread.
type ConversationDetail struct {
	Conversation store.Conversation `json:"conversation"`
	Messages     []chat.UIMessage   `json:"messages"`
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

// ListProjects returns project categories in creation order.
func (s *Service) ListProjects() []store.Project {
	return s.store.ListProjects()
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
	for _, record := range records {
		if record.ContextKind != "" {
			continue
		}
		message := chat.FromMessage(record.Message, len(detail.Messages))
		message.ThinkingDurationMS = record.ThinkingDurationMS
		detail.Messages = append(detail.Messages, message)
	}
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

// ClearAPIKey removes the saved key.
func (s *Service) ClearAPIKey() (config.View, error) {
	return s.cfg.ClearAPIKey()
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
	reasoningEffort := strings.TrimSpace(options.ReasoningEffort)
	switch reasoningEffort {
	case "", "low", "medium", "high":
	default:
		return fmt.Errorf("不支持的思维强度: %q", options.ReasoningEffort)
	}
	if !s.cfg.View().Configured {
		return errNotConfigured
	}

	conversation, records, err := s.store.GetRecords(conversationID)
	if err != nil {
		return err
	}
	if conversation.ID == "" {
		return errors.New("会话不存在")
	}
	handle, runCtx, err := s.beginRun(ctx, conversationID)
	if err != nil {
		return err
	}
	defer s.endRun(handle)
	runStarted := time.Now()

	settings := s.cfg.Settings()
	records, err = s.appendRuntimeMemory(conversationID, settings, records)
	if err != nil {
		return err
	}
	history := make([]ai.Message, len(records))
	for index, record := range records {
		history[index] = record.Message
	}
	assistant, err := s.engine.NewAgent(settings, conversationID)
	if err != nil {
		return err
	}
	session := assistant.NewSessionWithHistory(repairToolHistory(history))

	var (
		nextIndex  = len(history)
		sinkErr    error
		persistErr error
		answerAt   time.Time
	)
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
	if s.memory != nil {
		defer s.memory.ObserveUserTurn(settings)
	}

	result, runErr := session.Run(
		runCtx,
		content,
		agent.WithRequestOptions(ai.RequestOptions{ReasoningEffort: reasoningEffort}),
		agent.WithOnEvent(func(event agent.Event) {
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
		}),
	)

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
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			emit(chat.ChatEvent{Type: chat.EventRunEnd, Steps: result.Steps, Cancelled: true})
			return nil
		}
		if errors.Is(runErr, agent.ErrMaxSteps) {
			emit(chat.ChatEvent{
				Type: chat.EventError,
				Code: "max_steps",
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
			Error: runErr.Error(),
		})
		return nil
	}

	if err := s.store.AppendUsage(conversationID, result.Usage); err != nil {
		emit(chat.ChatEvent{
			Type:  chat.EventError,
			Code:  "usage_failed",
			Error: "保存用量失败：" + err.Error(),
		})
		return nil
	}
	emit(chat.ChatEvent{
		Type:  chat.EventRunEnd,
		Usage: chat.UsageViewFrom(result.Usage),
		Steps: result.Steps,
	})
	return nil
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
