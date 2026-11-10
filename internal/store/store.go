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
	fileVersion  = 1
	indexVersion = 1
	defaultTitle = "新对话"
)

// Conversation is the metadata shown in the sidebar.
type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Preview      string    `json:"preview"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MessageCount int       `json:"messageCount"`
}

type indexFile struct {
	Version       int            `json:"version"`
	Conversations []Conversation `json:"conversations"`
}

type header struct {
	Type      string    `json:"type"`
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type entry struct {
	Type               string      `json:"type"`
	Timestamp          time.Time   `json:"timestamp"`
	Message            *ai.Message `json:"message,omitempty"`
	Usage              *ai.Usage   `json:"usage,omitempty"`
	ThinkingDurationMS int64       `json:"thinkingDurationMs,omitempty"`
}

// MessageRecord is a stored message together with the metadata needed to
// restore how long the assistant worked before it started answering.
type MessageRecord struct {
	Message            ai.Message
	Timestamp          time.Time
	ThinkingDurationMS int64
}

// Store serializes access to the index and conversation files.
type Store struct {
	mu      sync.Mutex
	dir     string
	index   string
	entries []Conversation
}

// Open loads the index, rebuilding it from conversation files when missing or
// corrupt.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create conversation directory: %w", err)
	}
	s := &Store{dir: dir, index: filepath.Join(dir, "index.json")}
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

// Create adds an empty conversation.
func (s *Store) Create() (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	id, err := newID(now)
	if err != nil {
		return Conversation{}, err
	}
	conversation := Conversation{
		ID:        id,
		Title:     defaultTitle,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.writeHeader(header{
		Type:      "conversation",
		Version:   fileVersion,
		ID:        id,
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
	s.entries[index].UpdatedAt = time.Now().UTC()
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

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".jsonl")
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
			info.CreatedAt = h.CreatedAt
			info.UpdatedAt = h.CreatedAt
			seenHead = true
			continue
		}
		var item entry
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return Conversation{}, nil, fmt.Errorf("parse conversation entry: %w", err)
		}
		if item.Timestamp.After(info.UpdatedAt) {
			info.UpdatedAt = item.Timestamp
		}
		if item.Message == nil {
			continue
		}
		records = append(records, MessageRecord{
			Message:            *item.Message,
			Timestamp:          item.Timestamp,
			ThinkingDurationMS: item.ThinkingDurationMS,
		})
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
