// Package store persists conversations as JSONL files and keeps a small index.
package store

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mrsibe/goat/ai"
)

const (
	fileVersion        = 1
	indexVersion       = 1
	projectFileVersion = 1
	defaultTitle       = "新对话"
	unfiledProjectName = "未分类"
	maxProjectName     = 48
)

// Conversation is the metadata shown in the sidebar.
type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Preview      string    `json:"preview"`
	ProjectID    string    `json:"projectId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MessageCount int       `json:"messageCount"`
}

// Project groups conversations in the sidebar.
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// SearchResult is one conversation matched by a full-text search.
type SearchResult struct {
	Conversation Conversation `json:"conversation"`
	MessageIndex int          `json:"messageIndex"`
	Snippet      string       `json:"snippet,omitempty"`
	Role         string       `json:"role,omitempty"`
	MatchCount   int          `json:"matchCount,omitempty"`
}

type indexFile struct {
	Version       int            `json:"version"`
	Conversations []Conversation `json:"conversations"`
}

type projectsFile struct {
	Version  int       `json:"version"`
	Projects []Project `json:"projects"`
}

type header struct {
	Type      string    `json:"type"`
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type entry struct {
	Type               string      `json:"type"`
	Timestamp          time.Time   `json:"timestamp"`
	ProjectID          string      `json:"projectId,omitempty"`
	Message            *ai.Message `json:"message,omitempty"`
	Usage              *ai.Usage   `json:"usage,omitempty"`
	ThinkingDurationMS int64       `json:"thinkingDurationMs,omitempty"`
	ContextKind        string      `json:"contextKind,omitempty"`
	ContextHash        string      `json:"contextHash,omitempty"`
}

type rawEntry struct {
	line  []byte
	value entry
}

type visibleMessage struct {
	rawIndex int
	role     ai.Role
}

// MessageRecord is a stored message together with the metadata needed to
// restore how long the assistant worked before it started answering. Context
// records are injected into model history but hidden from the conversation UI.
type MessageRecord struct {
	Message            ai.Message
	Timestamp          time.Time
	ThinkingDurationMS int64
	ContextKind        string
	ContextHash        string
	Usage              *ai.Usage
}

// Store serializes access to the index and conversation files.
type Store struct {
	mu           sync.Mutex
	dir          string
	index        string
	projectsPath string
	entries      []Conversation
	projects     []Project
}

// Open loads the index, rebuilding it from conversation files when missing or
// corrupt.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create conversation directory: %w", err)
	}
	s := &Store{
		dir:          dir,
		index:        filepath.Join(dir, "index.json"),
		projectsPath: filepath.Join(dir, "projects.json"),
	}
	projects, err := s.readProjects()
	if err != nil {
		return nil, err
	}
	s.projects = projects
	entries, err := s.readIndex()
	if err != nil || entries == nil {
		entries, err = s.rebuildIndex()
		if err != nil {
			return nil, err
		}
		if err := s.writeIndex(entries); err != nil {
			return nil, err
		}
	}
	s.entries = entries
	return s, nil
}

// List returns conversations sorted by most recently updated.
func (s *Store) List() []Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]Conversation(nil), s.entries...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

// Search finds conversations by title, preview and visible message text.
func (s *Store) Search(query string) []SearchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	needle := strings.ToLower(query)

	s.mu.Lock()
	conversations := append([]Conversation(nil), s.entries...)
	s.mu.Unlock()

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
	})

	results := make([]SearchResult, 0)
	for _, conversation := range conversations {
		_, records, err := s.readRecords(conversation.ID)
		if err != nil {
			continue
		}
		result := SearchResult{Conversation: conversation, MessageIndex: -1}
		titleMatch := strings.Contains(strings.ToLower(conversation.Title), needle)
		previewMatch := strings.Contains(strings.ToLower(conversation.Preview), needle)
		if titleMatch || previewMatch {
			result.Snippet = truncate(strings.Join(strings.Fields(conversation.Preview), " "), 140)
		}

		visibleIndex := 0
		for _, record := range records {
			if record.ContextKind != "" {
				continue
			}
			messageIndex := visibleIndex
			visibleIndex++
			if record.Message.Role != ai.RoleUser && record.Message.Role != ai.RoleAssistant {
				continue
			}
			text := record.Message.Text()
			if text == "" || !strings.Contains(strings.ToLower(text), needle) {
				continue
			}
			result.MatchCount++
			if result.MessageIndex >= 0 {
				continue
			}
			result.MessageIndex = messageIndex
			result.Role = string(record.Message.Role)
			result.Snippet = snippetAround(text, query, 48, 96)
		}

		if !titleMatch && !previewMatch && result.MessageIndex < 0 {
			continue
		}
		if result.MessageIndex < 0 && result.Snippet == "" {
			result.Snippet = truncate(strings.Join(strings.Fields(conversation.Preview), " "), 140)
		}
		results = append(results, result)
	}
	return results
}

// ListProjects returns projects in creation order.
func (s *Store) ListProjects() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]Project(nil), s.projects...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// CreateProject adds an empty project.
func (s *Store) CreateProject(name string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name, err := s.validateProjectName("", name)
	if err != nil {
		return Project{}, err
	}
	now := time.Now().UTC()
	id, err := newID(now)
	if err != nil {
		return Project{}, fmt.Errorf("generate project id: %w", err)
	}
	project := Project{ID: id, Name: name, CreatedAt: now}
	s.projects = append(s.projects, project)
	if err := s.writeProjects(s.projects); err != nil {
		s.projects = s.projects[:len(s.projects)-1]
		return Project{}, err
	}
	return project, nil
}

// RenameProject changes a project's display name.
func (s *Store) RenameProject(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findProject(id)
	if index < 0 {
		return fmt.Errorf("项目不存在")
	}
	name, err := s.validateProjectName(id, name)
	if err != nil {
		return err
	}
	previous := s.projects[index].Name
	s.projects[index].Name = name
	if err := s.writeProjects(s.projects); err != nil {
		s.projects[index].Name = previous
		return err
	}
	return nil
}

// DeleteProject removes a project and every conversation in it.
func (s *Store) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectIndex := s.findProject(id)
	if projectIndex < 0 {
		return fmt.Errorf("项目不存在")
	}
	deleted := make(map[string]bool)
	for _, conversation := range s.entries {
		if conversation.ProjectID != id {
			continue
		}
		if err := os.Remove(s.path(conversation.ID)); err != nil && !errors.Is(err, os.ErrNotExist) {
			if len(deleted) > 0 {
				s.removeConversations(deleted)
				_ = s.writeIndex(s.entries)
			}
			return fmt.Errorf("删除项目中的会话: %w", err)
		}
		deleted[conversation.ID] = true
	}
	s.removeConversations(deleted)
	s.projects = append(s.projects[:projectIndex], s.projects[projectIndex+1:]...)
	if err := s.writeIndex(s.entries); err != nil {
		return err
	}
	return s.writeProjects(s.projects)
}

// Create adds an empty conversation.
func (s *Store) Create() (Conversation, error) {
	return s.CreateInProject("")
}

// CreateInProject adds an empty conversation, optionally inside a project.
func (s *Store) CreateInProject(projectID string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if projectID != "" && s.findProject(projectID) < 0 {
		return Conversation{}, fmt.Errorf("项目不存在")
	}
	now := time.Now().UTC()
	id, err := newID(now)
	if err != nil {
		return Conversation{}, err
	}
	conversation := Conversation{
		ID:        id,
		Title:     defaultTitle,
		ProjectID: projectID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.writeHeader(header{
		Type:      "conversation",
		Version:   fileVersion,
		ID:        id,
		ProjectID: projectID,
		CreatedAt: now,
	}); err != nil {
		return Conversation{}, err
	}
	s.entries = append(s.entries, conversation)
	if err := s.writeIndex(s.entries); err != nil {
		s.entries = s.entries[:len(s.entries)-1]
		_ = os.Remove(s.path(id))
		return Conversation{}, err
	}
	return conversation, nil
}

// MoveConversation changes the project of a conversation. An empty project
// moves it to unfiled conversations.
func (s *Store) MoveConversation(id, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.find(id)
	if index < 0 {
		return fmt.Errorf("会话不存在")
	}
	if projectID != "" && s.findProject(projectID) < 0 {
		return fmt.Errorf("项目不存在")
	}
	if s.entries[index].ProjectID == projectID {
		return nil
	}
	item := entry{
		Type:      "project",
		Timestamp: time.Now().UTC(),
		ProjectID: projectID,
	}
	if err := s.appendEntry(id, item); err != nil {
		return err
	}
	s.entries[index].ProjectID = projectID
	return s.writeIndex(s.entries)
}

// Get returns conversation metadata and its model-visible messages.
func (s *Store) Get(id string) (Conversation, []ai.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conversation, records, err := s.getRecords(id)
	if err != nil {
		return Conversation{}, nil, err
	}
	messages := make([]ai.Message, len(records))
	for index, record := range records {
		messages[index] = record.Message
	}
	return conversation, messages, nil
}

// GetRecords returns messages with their persisted turn metadata.
func (s *Store) GetRecords(id string) (Conversation, []MessageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getRecords(id)
}

func (s *Store) getRecords(id string) (Conversation, []MessageRecord, error) {
	index := s.find(id)
	if index < 0 {
		return Conversation{}, nil, fmt.Errorf("会话不存在")
	}
	_, records, err := s.readRecords(id)
	if err != nil {
		return Conversation{}, nil, err
	}
	return s.entries[index], records, nil
}

// Append writes one completed message.
func (s *Store) Append(id string, message ai.Message) error {
	return s.AppendWithThinkingDuration(id, message, 0)
}

// AppendWithThinkingDuration writes one completed message and, for a final
// answer, records the time spent before answer text began streaming.
func (s *Store) AppendWithThinkingDuration(id string, message ai.Message, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.find(id)
	if index < 0 {
		return fmt.Errorf("会话不存在")
	}
	now := time.Now().UTC()
	item := entry{
		Type:               "message",
		Timestamp:          now,
		Message:            &message,
		ThinkingDurationMS: max(duration.Milliseconds(), 0),
	}
	if err := s.appendEntry(id, item); err != nil {
		return err
	}

	conversation := &s.entries[index]
	if message.Role == ai.RoleUser && conversation.Title == defaultTitle {
		if title := titleFromText(message.Text()); title != "" {
			conversation.Title = title
		}
	}
	if message.Role != ai.RoleTool && message.Role != ai.RoleSystem {
		if preview := previewFromMessage(message); preview != "" {
			conversation.Preview = preview
		}
	}
	conversation.UpdatedAt = now
	if message.Role != ai.RoleSystem {
		conversation.MessageCount++
	}
	return s.writeIndex(s.entries)
}

// AppendUsage records token usage without changing the sidebar metadata.
func (s *Store) AppendUsage(id string, usage ai.Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.find(id) < 0 {
		return fmt.Errorf("会话不存在")
	}
	item := entry{Type: "usage", Timestamp: time.Now().UTC(), Usage: &usage}
	return s.appendEntry(id, item)
}

// AppendContext stores an app-injected context message in model history
// without changing user-visible conversation metadata.
func (s *Store) AppendContext(id, kind, hash string, message ai.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.find(id) < 0 {
		return fmt.Errorf("会话不存在")
	}
	item := entry{
		Type:        "message",
		Timestamp:   time.Now().UTC(),
		Message:     &message,
		ContextKind: kind,
		ContextHash: hash,
	}
	return s.appendEntry(id, item)
}

// EditUserMessage replaces one visible user message with new text and removes
// everything after it. The rewrite is atomic so a failed write cannot leave a
// partially truncated transcript.
func (s *Store) EditUserMessage(id string, messageIndex int, content string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.find(id)
	if index < 0 {
		return Conversation{}, fmt.Errorf("会话不存在")
	}
	h, entries, visible, err := s.readRawConversation(id)
	if err != nil {
		return Conversation{}, err
	}
	target, err := visibleTarget(visible, messageIndex)
	if err != nil {
		return Conversation{}, err
	}
	if target.role != ai.RoleUser {
		return Conversation{}, errors.New("只能编辑用户消息")
	}

	old := s.entries[index]
	_, oldRecords, err := s.readRecords(id)
	if err != nil {
		return Conversation{}, err
	}
	oldDerived := derivedTitle(oldRecords)

	message := ai.TextMessage(ai.RoleUser, strings.TrimSpace(content))
	line, err := json.Marshal(entry{
		Type:      "message",
		Timestamp: time.Now().UTC(),
		Message:   &message,
	})
	if err != nil {
		return Conversation{}, fmt.Errorf("encode edited message: %w", err)
	}
	rewritten := rewriteEntries(entries[:target.rawIndex])
	rewritten = append(rewritten, rawEntry{line: line})
	h.ProjectID = old.ProjectID
	if err := s.replaceRawConversation(id, h, rewritten); err != nil {
		return Conversation{}, err
	}
	return s.refreshConversation(index, old, oldDerived)
}

// TruncateAssistantTurn removes one assistant turn and everything after it.
// The preceding user message is retained so the service can continue the run
// without duplicating it.
func (s *Store) TruncateAssistantTurn(id string, messageIndex int) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.find(id)
	if index < 0 {
		return Conversation{}, fmt.Errorf("会话不存在")
	}
	h, entries, visible, err := s.readRawConversation(id)
	if err != nil {
		return Conversation{}, err
	}
	target, err := visibleTarget(visible, messageIndex)
	if err != nil {
		return Conversation{}, err
	}
	if target.role != ai.RoleAssistant {
		return Conversation{}, errors.New("只能重新生成助手回答")
	}

	userPosition := -1
	for position := messageIndex - 1; position >= 0; position-- {
		if position < len(visible) && visible[position].role == ai.RoleUser {
			userPosition = position
			break
		}
	}
	if userPosition < 0 {
		return Conversation{}, errors.New("没有找到这条回答对应的用户消息")
	}
	turnPosition := -1
	for position := userPosition + 1; position < len(visible); position++ {
		if visible[position].role == ai.RoleAssistant {
			turnPosition = position
			break
		}
		if visible[position].role == ai.RoleUser {
			break
		}
	}
	if turnPosition < 0 || messageIndex < turnPosition {
		return Conversation{}, errors.New("没有找到可重新生成的助手回合")
	}

	old := s.entries[index]
	rewritten := rewriteEntries(entries[:visible[turnPosition].rawIndex])
	h.ProjectID = old.ProjectID
	if err := s.replaceRawConversation(id, h, rewritten); err != nil {
		return Conversation{}, err
	}
	return s.refreshConversation(index, old, "")
}

// ForkAssistantTurn copies the completed assistant turn into a new
// conversation. The source conversation is not modified.
func (s *Store) ForkAssistantTurn(id string, messageIndex int) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceIndex := s.find(id)
	if sourceIndex < 0 {
		return Conversation{}, fmt.Errorf("会话不存在")
	}
	source := s.entries[sourceIndex]
	_, entries, visible, err := s.readRawConversation(id)
	if err != nil {
		return Conversation{}, err
	}
	target, err := visibleTarget(visible, messageIndex)
	if err != nil {
		return Conversation{}, err
	}
	if target.role != ai.RoleAssistant {
		return Conversation{}, errors.New("只能从助手回答分叉")
	}

	userPosition := -1
	for position := messageIndex - 1; position >= 0; position-- {
		if position < len(visible) && visible[position].role == ai.RoleUser {
			userPosition = position
			break
		}
	}
	if userPosition < 0 {
		return Conversation{}, errors.New("没有找到这条回答对应的用户消息")
	}
	turnStart := -1
	for position := userPosition + 1; position < len(visible); position++ {
		if visible[position].role == ai.RoleAssistant {
			turnStart = position
			break
		}
		if visible[position].role == ai.RoleUser {
			break
		}
	}
	if turnStart < 0 || messageIndex < turnStart {
		return Conversation{}, errors.New("没有找到可分叉的助手回合")
	}
	turnEnd := turnStart
	for turnEnd+1 < len(visible) {
		role := visible[turnEnd+1].role
		if role != ai.RoleAssistant && role != ai.RoleTool {
			break
		}
		turnEnd++
	}
	if messageIndex > turnEnd {
		return Conversation{}, errors.New("只能从完整助手回合分叉")
	}

	cut := visible[turnEnd].rawIndex + 1
	for cut < len(entries) && entries[cut].value.Message == nil && entries[cut].value.Type == "usage" {
		cut++
	}
	copied := rewriteEntries(entries[:cut])
	if len(copied) == 0 {
		return Conversation{}, errors.New("这轮回答没有可复制的内容")
	}

	now := time.Now().UTC()
	newID, err := newID(now)
	if err != nil {
		return Conversation{}, fmt.Errorf("generate conversation id: %w", err)
	}
	header := header{
		Type:      "conversation",
		Version:   fileVersion,
		ID:        newID,
		ProjectID: source.ProjectID,
		CreatedAt: now,
	}
	if err := s.createRawConversation(newID, header, copied); err != nil {
		return Conversation{}, err
	}
	info, _, err := s.readRecords(newID)
	if err != nil {
		_ = os.Remove(s.path(newID))
		return Conversation{}, err
	}
	info.Title = truncate(source.Title+"（分支）", 80)
	s.entries = append(s.entries, info)
	if err := s.writeIndex(s.entries); err != nil {
		s.entries = s.entries[:len(s.entries)-1]
		_ = os.Remove(s.path(newID))
		return Conversation{}, err
	}
	return info, nil
}

// Rename updates a conversation title.
func (s *Store) Rename(id, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.find(id)
	if index < 0 {
		return fmt.Errorf("会话不存在")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = defaultTitle
	}
	s.entries[index].Title = truncate(title, 80)
	return s.writeIndex(s.entries)
}

// Delete removes a conversation and its file.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.find(id)
	if index < 0 {
		return fmt.Errorf("会话不存在")
	}
	if err := os.Remove(s.path(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除会话文件: %w", err)
	}
	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return s.writeIndex(s.entries)
}

func (s *Store) find(id string) int {
	for i := range s.entries {
		if s.entries[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) findProject(id string) int {
	for i := range s.projects {
		if s.projects[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) removeConversations(ids map[string]bool) {
	if len(ids) == 0 {
		return
	}
	kept := s.entries[:0]
	for _, conversation := range s.entries {
		if !ids[conversation.ID] {
			kept = append(kept, conversation)
		}
	}
	s.entries = kept
}

func (s *Store) validateProjectName(currentID, name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("项目名称不能为空")
	}
	if name == unfiledProjectName {
		return "", fmt.Errorf("“%s”是保留名称", unfiledProjectName)
	}
	name = truncate(name, maxProjectName)
	for i := range s.projects {
		if s.projects[i].ID == currentID {
			continue
		}
		if strings.EqualFold(s.projects[i].Name, name) {
			return "", fmt.Errorf("项目名称已存在")
		}
	}
	return name, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".jsonl")
}

func (s *Store) readProjects() ([]Project, error) {
	data, err := os.ReadFile(s.projectsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var doc projectsFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse projects: %w", err)
	}
	if doc.Version != projectFileVersion {
		return nil, fmt.Errorf("unsupported projects version %d", doc.Version)
	}
	return doc.Projects, nil
}

func (s *Store) readIndex() ([]Conversation, error) {
	data, err := os.ReadFile(s.index)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var doc indexFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Version != indexVersion {
		return nil, fmt.Errorf("unsupported index version %d", doc.Version)
	}
	return doc.Conversations, nil
}

func (s *Store) rebuildIndex() ([]Conversation, error) {
	dirEntries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read conversation directory: %w", err)
	}
	var conversations []Conversation
	for _, item := range dirEntries {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(item.Name(), ".jsonl")
		conversation, _, err := s.readRecords(id)
		if err != nil {
			continue
		}
		info, err := item.Info()
		if err == nil && info.ModTime().After(conversation.UpdatedAt) {
			conversation.UpdatedAt = info.ModTime()
		}
		conversations = append(conversations, conversation)
	}
	return conversations, nil
}

func (s *Store) writeIndex(entries []Conversation) error {
	doc := indexFile{Version: indexVersion, Conversations: entries}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(s.dir, ".index-*.tmp")
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write index: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.index); err != nil {
		return fmt.Errorf("replace index: %w", err)
	}
	return nil
}

func (s *Store) writeProjects(projects []Project) error {
	doc := projectsFile{Version: projectFileVersion, Projects: projects}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode projects: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(s.dir, ".projects-*.tmp")
	if err != nil {
		return fmt.Errorf("create projects: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write projects: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.projectsPath); err != nil {
		return fmt.Errorf("replace projects: %w", err)
	}
	return nil
}

func (s *Store) writeHeader(h header) error {
	file, err := os.OpenFile(s.path(h.ID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	defer file.Close()
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write conversation header: %w", err)
	}
	return file.Sync()
}

func (s *Store) appendEntry(id string, item entry) error {
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("encode entry: %w", err)
	}
	file, err := os.OpenFile(s.path(id), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open conversation: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append conversation: %w", err)
	}
	return file.Sync()
}

func (s *Store) readRawConversation(id string) (header, []rawEntry, []visibleMessage, error) {
	file, err := os.Open(s.path(id))
	if err != nil {
		return header{}, nil, nil, fmt.Errorf("open conversation: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var (
		h        header
		entries  []rawEntry
		visible  []visibleMessage
		seenHead bool
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !seenHead {
			if err := json.Unmarshal([]byte(line), &h); err != nil {
				return header{}, nil, nil, fmt.Errorf("parse conversation header: %w", err)
			}
			if h.Type != "conversation" || h.Version != fileVersion {
				return header{}, nil, nil, fmt.Errorf("unsupported conversation file")
			}
			seenHead = true
			continue
		}

		var item entry
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return header{}, nil, nil, fmt.Errorf("parse conversation entry: %w", err)
		}
		rawIndex := len(entries)
		entries = append(entries, rawEntry{line: []byte(line), value: item})
		if item.Message != nil && item.ContextKind == "" {
			visible = append(visible, visibleMessage{rawIndex: rawIndex, role: item.Message.Role})
		}
	}
	if err := scanner.Err(); err != nil {
		return header{}, nil, nil, fmt.Errorf("read conversation: %w", err)
	}
	if !seenHead {
		return header{}, nil, nil, fmt.Errorf("conversation file is empty")
	}
	return h, entries, visible, nil
}

func visibleTarget(visible []visibleMessage, index int) (visibleMessage, error) {
	if index < 0 || index >= len(visible) {
		return visibleMessage{}, errors.New("消息不存在")
	}
	return visible[index], nil
}

func rewriteEntries(entries []rawEntry) []rawEntry {
	result := make([]rawEntry, 0, len(entries))
	for _, item := range entries {
		if item.value.Type == "project" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func (s *Store) replaceRawConversation(id string, h header, entries []rawEntry) error {
	tmp, err := os.CreateTemp(s.dir, ".conversation-*.tmp")
	if err != nil {
		return fmt.Errorf("create conversation rewrite: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := writeRawConversation(tmp, h, entries); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path(id)); err != nil {
		return fmt.Errorf("replace conversation: %w", err)
	}
	return nil
}

func (s *Store) createRawConversation(id string, h header, entries []rawEntry) error {
	file, err := os.OpenFile(s.path(id), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}
	if err := writeRawConversation(file, h, entries); err != nil {
		file.Close()
		_ = os.Remove(s.path(id))
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		_ = os.Remove(s.path(id))
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(s.path(id))
		return err
	}
	return nil
}

func writeRawConversation(file *os.File, h header, entries []rawEntry) error {
	data, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("encode conversation header: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write conversation header: %w", err)
	}
	for _, item := range entries {
		line := item.line
		if len(line) == 0 {
			line, err = json.Marshal(item.value)
			if err != nil {
				return fmt.Errorf("encode conversation entry: %w", err)
			}
		}
		if _, err := file.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("write conversation entry: %w", err)
		}
	}
	return nil
}

func (s *Store) refreshConversation(index int, old Conversation, oldDerivedTitle string) (Conversation, error) {
	info, _, err := s.readRecords(old.ID)
	if err != nil {
		return Conversation{}, err
	}
	info.ID = old.ID
	info.ProjectID = old.ProjectID
	info.CreatedAt = old.CreatedAt
	if old.Title != defaultTitle && old.Title != oldDerivedTitle {
		info.Title = old.Title
	}

	previous := s.entries[index]
	s.entries[index] = info
	if err := s.writeIndex(s.entries); err != nil {
		s.entries[index] = previous
		return Conversation{}, err
	}
	return info, nil
}

func derivedTitle(records []MessageRecord) string {
	for _, record := range records {
		if record.ContextKind != "" || record.Message.Role != ai.RoleUser {
			continue
		}
		if title := titleFromText(record.Message.Text()); title != "" {
			return title
		}
	}
	return ""
}

func (s *Store) readRecords(id string) (Conversation, []MessageRecord, error) {
	file, err := os.Open(s.path(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Conversation{ID: id, Title: defaultTitle}, nil, nil
		}
		return Conversation{}, nil, fmt.Errorf("open conversation: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var (
		info     Conversation
		records  []MessageRecord
		seenHead bool
	)
	info.ID = id
	info.Title = defaultTitle
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !seenHead {
			var h header
			if err := json.Unmarshal([]byte(line), &h); err != nil {
				return Conversation{}, nil, fmt.Errorf("parse conversation header: %w", err)
			}
			if h.Type != "conversation" || h.Version != fileVersion {
				return Conversation{}, nil, fmt.Errorf("unsupported conversation file")
			}
			info.ID = h.ID
			info.ProjectID = h.ProjectID
			info.CreatedAt = h.CreatedAt
			info.UpdatedAt = h.CreatedAt
			seenHead = true
			continue
		}
		var item entry
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return Conversation{}, nil, fmt.Errorf("parse conversation entry: %w", err)
		}
		if item.Type == "project" {
			info.ProjectID = item.ProjectID
			continue
		}
		if item.ContextKind == "" && item.Timestamp.After(info.UpdatedAt) {
			info.UpdatedAt = item.Timestamp
		}
		if item.Message == nil {
			if item.Usage != nil {
				attachUsage(records, *item.Usage)
			}
			continue
		}
		records = append(records, MessageRecord{
			Message:            *item.Message,
			Timestamp:          item.Timestamp,
			ThinkingDurationMS: item.ThinkingDurationMS,
			ContextKind:        item.ContextKind,
			ContextHash:        item.ContextHash,
		})
		if item.ContextKind != "" {
			continue
		}
		if item.Message.Role != ai.RoleSystem {
			info.MessageCount++
		}
		if item.Message.Role == ai.RoleUser && info.Title == defaultTitle {
			if title := titleFromText(item.Message.Text()); title != "" {
				info.Title = title
			}
		}
		if item.Message.Role != ai.RoleTool && item.Message.Role != ai.RoleSystem {
			if preview := previewFromMessage(*item.Message); preview != "" {
				info.Preview = preview
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Conversation{}, nil, fmt.Errorf("read conversation: %w", err)
	}
	if !seenHead {
		return Conversation{}, nil, fmt.Errorf("conversation file is empty")
	}
	return info, records, nil
}

func attachUsage(records []MessageRecord, usage ai.Usage) {
	for index := len(records) - 1; index >= 0; index-- {
		record := &records[index]
		if record.ContextKind != "" || record.Message.Role != ai.RoleAssistant {
			continue
		}
		copied := usage
		if record.Usage == nil {
			record.Usage = &copied
		} else {
			record.Usage.Add(copied)
		}
		return
	}
}

func newID(now time.Time) (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate conversation id: %w", err)
	}
	return now.Format("20060102T150405Z") + "-" + hex.EncodeToString(random[:]), nil
}

func titleFromText(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimFunc(text, unicode.IsPunct)
	return truncate(text, 48)
}

func previewFromMessage(message ai.Message) string {
	text := strings.TrimSpace(message.Text())
	if text == "" && message.Role == ai.RoleTool {
		text = message.ToolName
	}
	return truncate(strings.Join(strings.Fields(text), " "), 90)
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}

func snippetAround(text, query string, before, after int) string {
	compact := strings.Join(strings.Fields(text), " ")
	lowerText := strings.ToLower(compact)
	byteIndex := strings.Index(lowerText, strings.ToLower(query))
	if byteIndex < 0 {
		return truncate(compact, before+after)
	}

	runes := []rune(compact)
	position := len([]rune(lowerText[:byteIndex]))
	start := max(0, position-before)
	end := min(len(runes), position+len([]rune(query))+after)
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet += "…"
	}
	return snippet
}
