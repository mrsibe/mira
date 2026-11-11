// Package memory persists the automatic user-memory summary and searches past
// chats with a lightweight in-memory BM25 index.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/store"
)

const (
	SummaryThreshold   = 8
	initialTurnLimit   = 40
	maxTranscriptRunes = 60_000
	maxSummaryRunes    = 2000
	summaryTimeout     = 90 * time.Second

	autoStart = "<!-- MIRA:AUTO:START -->"
	autoEnd   = "<!-- MIRA:AUTO:END -->"
)

var errSummaryRunning = errors.New("记忆摘要正在生成")

// View is the frontend-safe memory snapshot.
type View struct {
	Summary          string `json:"summary"`
	Revision         int64  `json:"revision"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	PendingUserTurns int    `json:"pendingUserTurns"`
	SummaryThreshold int    `json:"summaryThreshold"`
	Running          bool   `json:"running"`
	LastError        string `json:"lastError,omitempty"`
}

// SaveInput updates the editable automatic summary.
type SaveInput struct {
	Summary  string `json:"summary"`
	Revision int64  `json:"revision"`
}

// CompleteFunc runs one non-streaming completion with the configured model.
type CompleteFunc func(context.Context, config.Settings, string, string) (string, error)

// Manager owns USER.md, summary state and the lazily rebuilt BM25 index.
type Manager struct {
	mu         sync.Mutex
	dir        string
	userPath   string
	statePath  string
	source     indexSource
	complete   CompleteFunc
	auto       string
	state      persistedState
	fileErr    error
	index      *bm25Index
	indexDirty bool
	running    bool
	generation int64
}

type persistedState struct {
	LastSummarizedAt time.Time `json:"lastSummarizedAt,omitempty"`
	SummaryUpdatedAt time.Time `json:"summaryUpdatedAt,omitempty"`
	PendingUserTurns int       `json:"pendingUserTurns,omitempty"`
	LastError        string    `json:"lastError,omitempty"`
	Revision         int64     `json:"revision,omitempty"`
}

// Open loads or creates the memory directory.
func Open(dir string, source indexSource, complete CompleteFunc) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("创建记忆目录: %w", err)
	}
	m := &Manager{
		dir:        dir,
		userPath:   filepath.Join(dir, "USER.md"),
		statePath:  filepath.Join(dir, "state.json"),
		source:     source,
		complete:   complete,
		indexDirty: true,
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.userPath)
	switch {
	case err == nil:
		summary, parseErr := parseUserFile(string(data))
		if parseErr != nil {
			m.fileErr = parseErr
		} else {
			m.auto = summary
		}
	case errors.Is(err, os.ErrNotExist):
		if err := m.writeUserFileLocked(m.auto); err != nil {
			return err
		}
	default:
		return fmt.Errorf("读取 USER.md: %w", err)
	}

	data, err = os.ReadFile(m.statePath)
	switch {
	case err == nil:
		if unmarshalErr := json.Unmarshal(data, &m.state); unmarshalErr != nil {
			m.state.LastError = "读取记忆状态失败：" + unmarshalErr.Error()
		}
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("读取记忆状态: %w", err)
	}
	return nil
}

// Get returns the current editable memory and task status.
func (m *Manager) Get() View {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.viewLocked()
}

// Prompt returns the automatic memory injected into the system prompt.
func (m *Manager) Prompt() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.auto
}

// Save updates the automatic summary when the caller's revision is current.
func (m *Manager) Save(input SaveInput) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if input.Revision != m.state.Revision {
		return m.viewLocked(), errors.New("记忆已在后台更新，请刷新后重试")
	}
	summary := sanitizeMarkerText(strings.TrimSpace(input.Summary))
	if len([]rune(summary)) > maxSummaryRunes {
		return m.viewLocked(), fmt.Errorf("自动摘要最多 %d 个字符", maxSummaryRunes)
	}
	if err := m.writeUserFileLocked(summary); err != nil {
		return m.viewLocked(), err
	}
	m.auto = summary
	m.state.Revision++
	m.state.SummaryUpdatedAt = time.Now().UTC()
	m.state.LastError = ""
	m.generation++
	if err := m.writeStateLocked(); err != nil {
		return m.viewLocked(), err
	}
	return m.viewLocked(), nil
}

// Regenerate synchronously rebuilds the automatic summary.
func (m *Manager) Regenerate(ctx context.Context, settings config.Settings) (View, error) {
	generation, summary, lastSummary, err := m.beginSummary()
	if err != nil {
		return m.Get(), err
	}
	result, latest, err := m.generate(ctx, settings, summary, lastSummary)
	m.finishSummary(generation, result, latest, err)
	if err != nil {
		return m.Get(), err
	}
	return m.Get(), nil
}

// Clear removes the automatic summary.
func (m *Manager) Clear() (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.writeUserFileLocked(""); err != nil {
		return m.viewLocked(), err
	}
	m.auto = ""
	m.generation++
	m.state.LastSummarizedAt = time.Now().UTC()
	m.state.SummaryUpdatedAt = time.Now().UTC()
	m.state.PendingUserTurns = 0
	m.state.LastError = ""
	m.state.Revision++
	if err := m.writeStateLocked(); err != nil {
		return m.viewLocked(), err
	}
	return m.viewLocked(), nil
}

// ObserveUserTurn records one completed user turn and starts a background
// summary when the configured threshold is reached.
func (m *Manager) ObserveUserTurn(settings config.Settings) {
	m.mu.Lock()
	m.state.PendingUserTurns++
	m.indexDirty = true
	stateErr := m.writeStateLocked()
	if stateErr != nil {
		m.state.LastError = "保存记忆状态失败：" + stateErr.Error()
	}
	shouldStart := m.shouldStartSummaryLocked(settings, false)
	m.mu.Unlock()
	if shouldStart {
		m.startSummary(settings, false)
	}
}

// MaybeStart runs the initial backfill after memory is enabled, or catches up
// when the pending threshold has already been crossed.
func (m *Manager) MaybeStart(settings config.Settings) {
	m.mu.Lock()
	initial := strings.TrimSpace(m.auto) == "" && m.state.LastSummarizedAt.IsZero()
	shouldStart := m.shouldStartSummaryLocked(settings, initial)
	m.mu.Unlock()
	if shouldStart {
		m.startSummary(settings, initial)
	}
}

// MarkDirty invalidates the BM25 index after a conversation changes.
func (m *Manager) MarkDirty() {
	m.mu.Lock()
	m.indexDirty = true
	m.mu.Unlock()
}

// Search returns BM25-ranked historical turns, excluding the current chat.
func (m *Manager) Search(
	_ context.Context,
	query string,
	limit int,
	excludeConversationID string,
) ([]SearchResult, error) {
	m.mu.Lock()
	if m.index == nil || m.indexDirty {
		index, err := buildBM25(m.source)
		if err != nil {
			m.mu.Unlock()
			return nil, err
		}
		m.index = index
		m.indexDirty = false
	}
	index := m.index
	m.mu.Unlock()
	return searchBM25(index, query, limit, excludeConversationID), nil
}

func (m *Manager) shouldStartSummaryLocked(settings config.Settings, initial bool) bool {
	if m.running || m.fileErr != nil ||
		!settings.MemoryEnabled || !settings.MemorySummaryEnabled {
		return false
	}
	if initial {
		return strings.TrimSpace(m.auto) == "" && m.state.LastSummarizedAt.IsZero()
	}
	return m.state.PendingUserTurns >= SummaryThreshold
}

func (m *Manager) startSummary(settings config.Settings, initial bool) {
	m.mu.Lock()
	if !m.shouldStartSummaryLocked(settings, initial) {
		m.mu.Unlock()
		return
	}
	m.running = true
	generation := m.generation
	m.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), summaryTimeout)
		defer cancel()
		summary := m.Prompt()
		lastSummary := m.lastSummarizedAt()
		result, latest, err := m.generate(ctx, settings, summary, lastSummary)
		m.finishSummary(generation, result, latest, err)
	}()
}

func (m *Manager) beginSummary() (int64, string, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return 0, "", time.Time{}, errSummaryRunning
	}
	m.running = true
	return m.generation, m.auto, m.state.LastSummarizedAt, nil
}

func (m *Manager) finishSummary(generation int64, summary string, latest time.Time, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	if generation != m.generation {
		return
	}
	if err != nil {
		m.state.LastError = err.Error()
		_ = m.writeStateLocked()
		return
	}
	if summary != "" {
		if writeErr := m.writeUserFileLocked(summary); writeErr != nil {
			m.state.LastError = writeErr.Error()
			_ = m.writeStateLocked()
			return
		}
		m.auto = summary
	}
	now := time.Now().UTC()
	if !latest.IsZero() {
		m.state.LastSummarizedAt = latest
	} else if m.state.LastSummarizedAt.IsZero() {
		m.state.LastSummarizedAt = now
	}
	m.state.SummaryUpdatedAt = now
	m.state.PendingUserTurns = 0
	m.state.LastError = ""
	m.state.Revision++
	if writeErr := m.writeStateLocked(); writeErr != nil {
		m.state.LastError = "保存记忆状态失败：" + writeErr.Error()
	}
}

func (m *Manager) generate(
	ctx context.Context,
	settings config.Settings,
	existingSummary string,
	lastSummary time.Time,
) (string, time.Time, error) {
	if m.complete == nil {
		return "", time.Time{}, errors.New("摘要模型未配置")
	}
	turns, latest, err := m.collectTurns(lastSummary, existingSummary == "")
	if err != nil {
		return "", time.Time{}, err
	}
	if len(turns) == 0 && strings.TrimSpace(existingSummary) == "" {
		return "", latest, nil
	}
	transcript := formatTurns(turns)
	systemPrompt := `你是 Mira 的记忆整理器。把已有记忆和新增聊天记录合并成一份简洁、稳定的用户长期记忆。

只记录在未来对话中仍然有用、且能从记录中得到支持的长期信息。不要记录临时任务、一次性问题、工具参数、内部思考、密钥或凭证。除非用户明确要求记住，否则不要记录健康、财务、身份等敏感信息。不要编造事实。

只输出 Markdown，不要代码围栏。使用以下分类，没有内容的分类可以省略：
## 用户画像
## 长期偏好
## 当前项目与目标
## 重要约束

合并重复信息，删除过时或相互矛盾的内容。总长度控制在 2000 个中文字符以内。`
	userPrompt := fmt.Sprintf(`<existing_memory>
%s
</existing_memory>

<new_transcript>
%s
</new_transcript>

请输出更新后的自动摘要。`, existingSummary, transcript)
	result, err := m.complete(ctx, settings, systemPrompt, userPrompt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成记忆摘要: %w", err)
	}
	result = cleanSummary(result)
	if result == "" {
		result = "暂无长期记忆。"
	}
	result = truncateRunes(result, maxSummaryRunes)
	return result, latest, nil
}

func (m *Manager) collectTurns(lastSummary time.Time, initial bool) ([]memoryTurn, time.Time, error) {
	var turns []memoryTurn
	var latest time.Time
	for _, conversation := range m.source.List() {
		_, records, err := m.source.GetRecords(conversation.ID)
		if err != nil {
			return nil, time.Time{}, err
		}
		for _, turn := range conversationTurns(conversation, records) {
			if !lastSummary.IsZero() && !turn.timestamp.After(lastSummary) {
				continue
			}
			turns = append(turns, turn)
			if turn.latest.After(latest) {
				latest = turn.latest
			}
		}
	}
	sort.Slice(turns, func(i, j int) bool {
		return turns[i].timestamp.Before(turns[j].timestamp)
	})
	if initial && len(turns) > initialTurnLimit {
		turns = turns[len(turns)-initialTurnLimit:]
	}
	return turns, latest, nil
}

type memoryTurn struct {
	timestamp time.Time
	latest    time.Time
	title     string
	user      string
	assistant string
}

func conversationTurns(conversation store.Conversation, records []store.MessageRecord) []memoryTurn {
	var turns []memoryTurn
	var current *memoryTurn
	flush := func() {
		if current == nil {
			return
		}
		if strings.TrimSpace(current.user) != "" || strings.TrimSpace(current.assistant) != "" {
			turns = append(turns, *current)
		}
		current = nil
	}
	for _, record := range records {
		message := record.Message
		text := strings.TrimSpace(message.Text())
		switch message.Role {
		case ai.RoleUser:
			flush()
			current = &memoryTurn{
				timestamp: record.Timestamp,
				latest:    record.Timestamp,
				title:     conversation.Title,
				user:      text,
			}
		case ai.RoleAssistant:
			if current == nil || len(message.ToolCalls()) > 0 || text == "" {
				continue
			}
			current.assistant = text
			if record.Timestamp.After(current.latest) {
				current.latest = record.Timestamp
			}
		}
	}
	flush()
	return turns
}

func formatTurns(turns []memoryTurn) string {
	var text strings.Builder
	for _, turn := range turns {
		if turn.title != "" {
			fmt.Fprintf(&text, "[%s] 对话：%s\n", turn.timestamp.UTC().Format("2006-01-02 15:04"), turn.title)
		}
		if turn.user != "" {
			fmt.Fprintf(&text, "用户：%s\n", turn.user)
		}
		if turn.assistant != "" {
			fmt.Fprintf(&text, "Mira：%s\n", turn.assistant)
		}
		text.WriteString("---\n")
	}
	value := text.String()
	runes := []rune(value)
	if len(runes) > maxTranscriptRunes {
		value = "（较早的聊天记录已省略）\n" + string(runes[len(runes)-maxTranscriptRunes:])
	}
	return strings.TrimSpace(value)
}

func (m *Manager) viewLocked() View {
	lastError := m.state.LastError
	if m.fileErr != nil {
		lastError = "USER.md 结构无效：" + m.fileErr.Error()
	}
	updatedAt := ""
	if !m.state.SummaryUpdatedAt.IsZero() {
		updatedAt = m.state.SummaryUpdatedAt.UTC().Format(time.RFC3339)
	}
	return View{
		Summary:          m.auto,
		Revision:         m.state.Revision,
		UpdatedAt:        updatedAt,
		PendingUserTurns: m.state.PendingUserTurns,
		SummaryThreshold: SummaryThreshold,
		Running:          m.running,
		LastError:        lastError,
	}
}

func (m *Manager) lastSummarizedAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.LastSummarizedAt
}

func (m *Manager) writeUserFileLocked(summary string) error {
	m.fileErr = nil
	return writeFileAtomic(m.userPath, renderUserFile(summary))
}

func (m *Manager) writeStateLocked() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("编码记忆状态: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(m.statePath, string(data))
}

func renderUserFile(summary string) string {
	return fmt.Sprintf(`# 用户记忆

## 自动摘要

%s
%s
%s
`, autoStart, summary, autoEnd)
}

func parseUserFile(content string) (string, error) {
	summary, err := section(content, autoStart, autoEnd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(summary), nil
}

func section(content, start, end string) (string, error) {
	startIndex := strings.Index(content, start)
	if startIndex < 0 {
		return "", fmt.Errorf("缺少标记 %s", start)
	}
	startIndex += len(start)
	endIndex := strings.Index(content[startIndex:], end)
	if endIndex < 0 {
		return "", fmt.Errorf("缺少标记 %s", end)
	}
	return content[startIndex : startIndex+endIndex], nil
}

func sanitizeMarkerText(value string) string {
	for _, marker := range []string{autoStart, autoEnd} {
		value = strings.ReplaceAll(value, marker, "")
	}
	return value
}

func cleanSummary(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "```") {
		lines := strings.Split(value, "\n")
		if len(lines) > 1 {
			lines = lines[1:]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			lines = lines[:len(lines)-1]
		}
		value = strings.Join(lines, "\n")
	}
	return strings.TrimSpace(value)
}

func writeFileAtomic(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("创建目录: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("写入文件: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("替换文件: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
