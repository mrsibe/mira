package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/internal/store"
)

type fakeSource struct {
	mu            sync.Mutex
	conversations []store.Conversation
	records       map[string][]store.MessageRecord
}

func (f *fakeSource) List() []store.Conversation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Conversation(nil), f.conversations...)
}

func (f *fakeSource) GetRecords(id string) (store.Conversation, []store.MessageRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, conversation := range f.conversations {
		if conversation.ID == id {
			return conversation, append([]store.MessageRecord(nil), f.records[id]...), nil
		}
	}
	return store.Conversation{}, nil, fmt.Errorf("missing conversation %s", id)
}

func (f *fakeSource) add(conversation store.Conversation, records ...store.MessageRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.conversations = append(f.conversations, conversation)
	f.records[conversation.ID] = append([]store.MessageRecord(nil), records...)
}

func TestBM25SearchFindsChineseAndLatinTurns(t *testing.T) {
	now := time.Now().UTC()
	source := &fakeSource{records: make(map[string][]store.MessageRecord)}
	source.add(
		store.Conversation{ID: "go", Title: "后端项目"},
		turnRecords(now, "我喜欢用 Go 写后端服务，项目叫 Mira", "记住了。")...,
	)
	source.add(
		store.Conversation{ID: "cat", Title: "宠物"},
		turnRecords(now.Add(time.Minute), "我的猫叫橘子", "橘子很可爱。")...,
	)
	manager, err := Open(t.TempDir(), source, nil)
	if err != nil {
		t.Fatal(err)
	}

	results, err := manager.Search(context.Background(), "Go 后端", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ConversationID != "go" {
		t.Fatalf("latin search results = %+v", results)
	}

	results, err = manager.Search(context.Background(), "橘子", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ConversationID != "cat" {
		t.Fatalf("chinese search results = %+v", results)
	}
}

func TestSearchExcludesCurrentConversationAndRebuildsAfterMarkDirty(t *testing.T) {
	now := time.Now().UTC()
	source := &fakeSource{records: make(map[string][]store.MessageRecord)}
	source.add(
		store.Conversation{ID: "one", Title: "一"},
		turnRecords(now, "alpha beta", "ok")...,
	)
	source.add(
		store.Conversation{ID: "two", Title: "二"},
		turnRecords(now.Add(time.Minute), "alpha gamma", "ok")...,
	)
	manager, err := Open(t.TempDir(), source, nil)
	if err != nil {
		t.Fatal(err)
	}

	results, err := manager.Search(context.Background(), "alpha", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("search results = %+v", results)
	}

	results, err = manager.Search(context.Background(), "alpha", 5, "one")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ConversationID != "two" {
		t.Fatalf("excluded search results = %+v", results)
	}

	source.add(
		store.Conversation{ID: "three", Title: "三"},
		turnRecords(now.Add(2*time.Minute), "delta epsilon", "ok")...,
	)
	results, err = manager.Search(context.Background(), "delta", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("index should still be cached: %+v", results)
	}

	manager.MarkDirty()
	results, err = manager.Search(context.Background(), "delta", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ConversationID != "three" {
		t.Fatalf("rebuilt search results = %+v", results)
	}
}

func TestUserFileRoundTripAndRevisionConflict(t *testing.T) {
	dir := t.TempDir()
	manager, err := Open(dir, &fakeSource{records: map[string][]store.MessageRecord{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err := manager.Save(SaveInput{
		Manual:   "我喜欢简短回答。",
		Summary:  "## 用户画像\n- 后端工程师",
		Revision: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Revision != 1 || view.Manual != "我喜欢简短回答。" {
		t.Fatalf("saved view = %+v", view)
	}

	reloaded, err := Open(dir, &fakeSource{records: map[string][]store.MessageRecord{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	reloadedView := reloaded.Get()
	if reloadedView.Manual != view.Manual || reloadedView.Summary != view.Summary ||
		reloadedView.Revision != view.Revision {
		t.Fatalf("reloaded view = %+v", reloadedView)
	}

	if _, err := reloaded.Save(SaveInput{Revision: 0}); err == nil {
		t.Fatal("stale revision should be rejected")
	}
	refreshed := reloaded.Get()
	if refreshed.Manual != view.Manual || refreshed.Summary != view.Summary {
		t.Fatalf("conflict changed memory: %+v", refreshed)
	}
}

func TestUserFileRejectsMissingMarkers(t *testing.T) {
	if _, _, err := parseUserFile("# 用户记忆\n没有标记"); err == nil {
		t.Fatal("expected missing markers to be rejected")
	}
}

func TestObserveUserTurnStartsSummaryAtThreshold(t *testing.T) {
	now := time.Now().UTC()
	source := &fakeSource{records: make(map[string][]store.MessageRecord)}
	source.add(
		store.Conversation{ID: "chat", Title: "聊天"},
		turnRecords(now, "我在做一个桌面聊天客户端", "了解。")...,
	)
	var calls atomic.Int32
	manager, err := Open(t.TempDir(), source, func(
		context.Context,
		config.Settings,
		string,
		string,
	) (string, error) {
		calls.Add(1)
		return "## 当前项目与目标\n- 正在做桌面聊天客户端", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Settings{MemoryEnabled: true, MemorySummaryEnabled: true}

	for range SummaryThreshold - 1 {
		manager.ObserveUserTurn(settings)
	}
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("summary started after %d turns", SummaryThreshold-1)
	}
	if pending := manager.Get().PendingUserTurns; pending != SummaryThreshold-1 {
		t.Fatalf("pending turns = %d", pending)
	}

	manager.ObserveUserTurn(settings)
	waitFor(t, 2*time.Second, func() bool {
		view := manager.Get()
		return view.Summary != "" && view.PendingUserTurns == 0 && !view.Running
	})
	if calls.Load() != 1 {
		t.Fatalf("summary calls = %d", calls.Load())
	}
}

func TestInitialSummaryUsesLastFortyTurns(t *testing.T) {
	now := time.Now().UTC()
	source := &fakeSource{records: make(map[string][]store.MessageRecord)}
	var records []store.MessageRecord
	for index := range 50 {
		records = append(records,
			turnRecords(now.Add(time.Duration(index)*time.Minute),
				fmt.Sprintf("问题 %d", index+1),
				fmt.Sprintf("回答 %d", index+1),
			)...,
		)
	}
	source.add(store.Conversation{ID: "chat", Title: "长对话"}, records...)
	manager, err := Open(t.TempDir(), source, nil)
	if err != nil {
		t.Fatal(err)
	}

	turns, _, err := manager.collectTurns(time.Time{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != initialTurnLimit {
		t.Fatalf("initial turns = %d", len(turns))
	}
	if turns[0].user != "问题 11" || turns[len(turns)-1].user != "问题 50" {
		t.Fatalf("initial turn range = %q .. %q", turns[0].user, turns[len(turns)-1].user)
	}

	cutoff := now.Add(24 * time.Minute)
	turns, _, err = manager.collectTurns(cutoff, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 25 || turns[0].user != "问题 26" {
		t.Fatalf("incremental turns = %d, first = %q", len(turns), turns[0].user)
	}
}

func TestSummaryFailureKeepsOldSummary(t *testing.T) {
	manager, err := Open(t.TempDir(), &fakeSource{records: map[string][]store.MessageRecord{}}, func(
		context.Context,
		config.Settings,
		string,
		string,
	) (string, error) {
		return "", errors.New("模型暂时不可用")
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Save(SaveInput{Summary: "旧摘要", Revision: 0}); err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Regenerate(
		context.Background(),
		config.Settings{MemoryEnabled: true, MemorySummaryEnabled: true},
	); err == nil {
		t.Fatal("expected summary generation to fail")
	}
	view := manager.Get()
	if view.Summary != "旧摘要" {
		t.Fatalf("summary changed after failure: %q", view.Summary)
	}
	if !strings.Contains(view.LastError, "模型暂时不可用") {
		t.Fatalf("last error = %q", view.LastError)
	}
	if view.Running {
		t.Fatal("running flag was not cleared")
	}
}

func TestMaybeStartBackfillsEmptyMemoryOnce(t *testing.T) {
	now := time.Now().UTC()
	source := &fakeSource{records: make(map[string][]store.MessageRecord)}
	source.add(
		store.Conversation{ID: "chat", Title: "聊天"},
		turnRecords(now, "我住在上海", "记住了。")...,
	)
	var calls atomic.Int32
	manager, err := Open(t.TempDir(), source, func(
		context.Context,
		config.Settings,
		string,
		string,
	) (string, error) {
		calls.Add(1)
		return "## 用户画像\n- 住在上海", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	settings := config.Settings{MemoryEnabled: true, MemorySummaryEnabled: true}
	manager.MaybeStart(settings)
	waitFor(t, 2*time.Second, func() bool {
		return manager.Get().Summary != "" && !manager.Get().Running
	})
	manager.MaybeStart(settings)
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("initial backfill ran %d times", calls.Load())
	}
}

func turnRecords(timestamp time.Time, user, assistant string) []store.MessageRecord {
	return []store.MessageRecord{
		{Message: ai.TextMessage(ai.RoleUser, user), Timestamp: timestamp},
		{Message: ai.TextMessage(ai.RoleAssistant, assistant), Timestamp: timestamp.Add(time.Second)},
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}
