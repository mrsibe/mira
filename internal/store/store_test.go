package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrsibe/goat/ai"
)

func TestConversationLifecycleAndReload(t *testing.T) {
	dir := t.TempDir()
	conversations, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	created, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != defaultTitle {
		t.Fatalf("new title = %q", created.Title)
	}

	user := ai.TextMessage(ai.RoleUser, "How do I learn Go?")
	assistant := ai.TextMessage(ai.RoleAssistant, "Start with the official tour.")
	if err := conversations.Append(created.ID, user); err != nil {
		t.Fatal(err)
	}
	if err := conversations.Append(created.ID, assistant); err != nil {
		t.Fatal(err)
	}
	if err := conversations.Append(created.ID, ai.ToolTextMessage(
		"call_1", "get_current_time", "2026-11-09T12:00:00+08:00", false,
	)); err != nil {
		t.Fatal(err)
	}
	if err := conversations.AppendUsage(created.ID, ai.Usage{
		PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15,
	}); err != nil {
		t.Fatal(err)
	}

	conversation, messages, err := conversations.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if conversation.Title != "How do I learn Go" {
		t.Fatalf("derived title = %q", conversation.Title)
	}
	if conversation.Preview != "Start with the official tour." {
		t.Fatalf("preview = %q", conversation.Preview)
	}
	if conversation.MessageCount != 3 {
		t.Fatalf("message count = %d", conversation.MessageCount)
	}
	if len(messages) != 3 || messages[0].Text() != user.Text() ||
		messages[1].Text() != assistant.Text() {
		t.Fatalf("messages = %+v", messages)
	}
	if err := conversations.Rename(created.ID, "  Go basics  "); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := reopened.List()
	if len(list) != 1 || list[0].Title != "Go basics" {
		t.Fatalf("reloaded list = %+v", list)
	}
	if _, loadedMessages, err := reopened.Get(created.ID); err != nil || len(loadedMessages) != 3 {
		t.Fatalf("reloaded messages = %d, err = %v", len(loadedMessages), err)
	}

	if err := reopened.Delete(created.ID); err != nil {
		t.Fatal(err)
	}
	if len(reopened.List()) != 0 {
		t.Fatal("deleted conversation is still listed")
	}
	if _, _, err := reopened.Get(created.ID); err == nil {
		t.Fatal("deleted conversation can still be loaded")
	}
	if _, err := os.Stat(filepath.Join(dir, created.ID+".jsonl")); !os.IsNotExist(err) {
		t.Fatalf("conversation file still exists: %v", err)
	}
}

func TestThinkingDurationRoundTrip(t *testing.T) {
	dir := t.TempDir()
	conversations, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := conversations.Append(created.ID, ai.TextMessage(ai.RoleUser, "调研一下")); err != nil {
		t.Fatal(err)
	}
	if err := conversations.AppendWithThinkingDuration(
		created.ID,
		ai.TextMessage(ai.RoleAssistant, "结论如下"),
		90*time.Second,
	); err != nil {
		t.Fatal(err)
	}

	for _, store := range []*Store{conversations, mustOpenStore(t, dir)} {
		_, records, err := store.GetRecords(created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 2 {
			t.Fatalf("records = %d", len(records))
		}
		if got := records[1].ThinkingDurationMS; got != 90_000 {
			t.Fatalf("thinking duration = %d ms, want 90000", got)
		}
	}
}

func mustOpenStore(t *testing.T, dir string) *Store {
	t.Helper()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestRebuildIndexFromConversationFiles(t *testing.T) {
	dir := t.TempDir()
	conversations, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := conversations.Append(created.ID, ai.TextMessage(ai.RoleUser, "Recover me")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "index.json")); err != nil {
		t.Fatal(err)
	}

	recovered, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := recovered.List()
	if len(list) != 1 {
		t.Fatalf("rebuilt list = %+v", list)
	}
	if list[0].ID != created.ID || list[0].Title != "Recover me" {
		t.Fatalf("rebuilt conversation = %+v", list[0])
	}
}

func TestUnknownConversation(t *testing.T) {
	conversations, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, call := range map[string]func() error{
		"append": func() error {
			return conversations.Append("missing", ai.TextMessage(ai.RoleUser, "hello"))
		},
		"usage":  func() error { return conversations.AppendUsage("missing", ai.Usage{}) },
		"rename": func() error { return conversations.Rename("missing", "title") },
		"delete": func() error { return conversations.Delete("missing") },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil || !strings.Contains(err.Error(), "会话不存在") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
