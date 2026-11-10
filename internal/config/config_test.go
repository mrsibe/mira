package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSecretStore struct {
	values    map[string]string
	getErr    error
	setErr    error
	deleteErr error
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{values: make(map[string]string)}
}

func (s *fakeSecretStore) Get(key string) (string, bool, error) {
	if s.getErr != nil {
		return "", false, s.getErr
	}
	value, found := s.values[key]
	return value, found, nil
}

func (s *fakeSecretStore) Set(key, value string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = value
	return nil
}

func (s *fakeSecretStore) Delete(key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.values, key)
	return nil
}

func newTestManager(t *testing.T, dir string) *Manager {
	t.Helper()
	manager, err := NewManager(dir, newFakeSecretStore())
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestSaveReloadAndKeyLifecycle(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	secrets := newFakeSecretStore()

	manager, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	initial := manager.View()
	if initial.BaseURL != DefaultBaseURL || initial.Model != DefaultModel {
		t.Fatalf("unexpected defaults: %+v", initial)
	}
	if initial.Configured || initial.HasAPIKey {
		t.Fatalf("new manager should be unconfigured: %+v", initial)
	}

	view, err := manager.Save(Input{
		BaseURL:  "https://api.example.com/v1/",
		APIKey:   "sk-test",
		Model:    "test-model",
		Protocol: ProtocolCompletions,
		UserName: stringPointer("小明"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.BaseURL != "https://api.example.com/v1" {
		t.Fatalf("base URL was not normalized: %q", view.BaseURL)
	}
	if !view.HasAPIKey || !view.Configured {
		t.Fatalf("saved settings should be configured: %+v", view)
	}
	if view.APIKeyStorage != KeyStorageKeychain || !view.KeychainAvailable {
		t.Fatalf("key storage = %+v", view)
	}
	if secrets.values[ModelSecretKey] != "sk-test" {
		t.Fatal("API key was not stored in the secret store")
	}
	if view.ConfigPath != filepath.Join(dir, "config.json") {
		t.Fatalf("config path = %q", view.ConfigPath)
	}
	if view.UserName != "小明" {
		t.Fatalf("user name = %q", view.UserName)
	}

	info, err := os.Stat(view.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if permission := info.Mode().Perm(); permission != 0o600 {
		t.Fatalf("config permission = %o, want 600", permission)
	}
	data, err := os.ReadFile(view.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-test") || strings.Contains(string(data), "apiKey") {
		t.Fatalf("config file contains an API key: %s", data)
	}

	reloaded, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	settings := reloaded.Settings()
	if settings.APIKey != "sk-test" || settings.Model != "test-model" ||
		settings.Protocol != ProtocolCompletions || settings.UserName != "小明" {
		t.Fatalf("reloaded settings = %+v", settings)
	}

	kept, err := reloaded.Save(Input{
		BaseURL:  "https://api.example.com/v1",
		Model:    "test-model",
		Protocol: ProtocolCompletions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !kept.HasAPIKey {
		t.Fatal("an empty API key input should preserve the saved key")
	}
	if kept.UserName != "小明" {
		t.Fatal("omitting the user name should preserve it")
	}

	cleared, err := reloaded.ClearAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if cleared.HasAPIKey || cleared.Configured {
		t.Fatalf("cleared key should make the remote provider unconfigured: %+v", cleared)
	}

	afterClear, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if afterClear.Settings().APIKey != "" {
		t.Fatal("cleared API key was persisted")
	}
}

func TestUserNameCanBeCleared(t *testing.T) {
	clearConfigEnv(t)
	manager := newTestManager(t, t.TempDir())

	if _, err := manager.Save(Input{UserName: stringPointer("小明")}); err != nil {
		t.Fatal(err)
	}
	if view := manager.View(); view.UserName != "小明" {
		t.Fatalf("user name = %q", view.UserName)
	}

	if _, err := manager.Save(Input{UserName: stringPointer("")}); err != nil {
		t.Fatal(err)
	}
	if view := manager.View(); view.UserName != "" {
		t.Fatalf("cleared user name = %q", view.UserName)
	}
}

func TestEnvironmentFallbackAndLocalProvider(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "http://localhost:11434/v1")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "local-model")
	t.Setenv("MIRA_PROTOCOL", ProtocolCompletions)

	manager := newTestManager(t, t.TempDir())
	view := manager.View()
	if view.BaseURL != "http://localhost:11434/v1" || view.Model != "local-model" {
		t.Fatalf("environment fallback = %+v", view)
	}
	if view.Protocol != ProtocolCompletions {
		t.Fatalf("protocol = %q", view.Protocol)
	}
	if !view.Configured {
		t.Fatal("a local provider should be configured without an API key")
	}
}

func TestInvalidSettingsAreRejected(t *testing.T) {
	clearConfigEnv(t)
	manager := newTestManager(t, t.TempDir())

	for _, baseURL := range []string{"not-a-url", "ftp://example.com/v1", "://missing-scheme"} {
		if _, err := manager.Save(Input{BaseURL: baseURL}); err == nil {
			t.Fatalf("expected %q to be rejected", baseURL)
		}
	}

	if len(manager.Settings().BaseURL) == 0 {
		t.Fatal("failed saves must not clear the current settings")
	}
}

func TestLoadErrorKeepsDefaults(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := newTestManager(t, dir)
	if manager.LoadError() == nil {
		t.Fatal("expected a load error for malformed JSON")
	}
	if manager.View().Model != DefaultModel {
		t.Fatalf("defaults were not applied: %+v", manager.View())
	}

	if _, err := manager.Save(Input{Model: "recovered"}); err != nil {
		t.Fatal(err)
	}
	if manager.LoadError() != nil {
		t.Fatal("a successful save should clear the load error")
	}
	if !strings.HasSuffix(manager.View().ConfigPath, "config.json") {
		t.Fatalf("config path = %q", manager.View().ConfigPath)
	}
}

func TestSearchSettingsDefaultsAndLifecycle(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	secrets := newFakeSecretStore()
	manager, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}

	initial := manager.View()
	if !initial.WebSearchEnabled || initial.SearchProvider != SearchProviderDuckDuckGo {
		t.Fatalf("search defaults = %+v", initial)
	}
	if initial.HasTavilyAPIKey {
		t.Fatal("new settings should not have a Tavily key")
	}

	if _, err := manager.Save(Input{SearchProvider: SearchProviderTavily}); err == nil {
		t.Fatal("Tavily without a key should be rejected")
	}
	if manager.View().SearchProvider != SearchProviderDuckDuckGo {
		t.Fatal("a failed save changed the search provider")
	}

	disabled := false
	saved, err := manager.Save(Input{
		SearchProvider:   SearchProviderTavily,
		TavilyAPIKey:     "test-search-key",
		WebSearchEnabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.WebSearchEnabled || !saved.HasTavilyAPIKey ||
		saved.SearchProvider != SearchProviderTavily {
		t.Fatalf("saved search settings = %+v", saved)
	}
	if saved.TavilyAPIKeyStorage != KeyStorageKeychain ||
		secrets.values[TavilySecretKey] != "test-search-key" {
		t.Fatalf("Tavily key was not stored in the keychain: %+v", saved)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "test-search-key") || strings.Contains(string(data), "tavilyApiKey") {
		t.Fatalf("config file contains the Tavily key: %s", data)
	}

	kept, err := manager.Save(Input{SearchProvider: SearchProviderTavily})
	if err != nil {
		t.Fatal(err)
	}
	if !kept.HasTavilyAPIKey || kept.WebSearchEnabled {
		t.Fatalf("empty key or nil toggle changed settings: %+v", kept)
	}

	reloaded, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if settings := reloaded.Settings(); settings.TavilyAPIKey != "test-search-key" ||
		settings.WebSearchEnabled || settings.SearchProvider != SearchProviderTavily {
		t.Fatalf("reloaded search settings = %+v", settings)
	}

	cleared, err := reloaded.ClearTavilyAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if cleared.HasTavilyAPIKey || cleared.SearchProvider != SearchProviderDuckDuckGo {
		t.Fatalf("clear Tavily key = %+v", cleared)
	}
	if reloaded.Settings().TavilyAPIKey != "" {
		t.Fatal("cleared Tavily key was persisted")
	}
	if _, found := secrets.values[TavilySecretKey]; found {
		t.Fatal("cleared Tavily key remains in the secret store")
	}
}

func TestLegacyConfigDefaultsWebSearchOn(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"baseUrl":"https://api.example.com/v1","model":"test","protocol":"responses"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	manager := newTestManager(t, dir)
	view := manager.View()
	if !view.WebSearchEnabled || view.SearchProvider != SearchProviderDuckDuckGo {
		t.Fatalf("legacy config search defaults = %+v", view)
	}

	falseValue := false
	if _, err := manager.Save(Input{WebSearchEnabled: &falseValue}); err != nil {
		t.Fatal(err)
	}
	reloaded := newTestManager(t, dir)
	if reloaded.View().WebSearchEnabled {
		t.Fatal("explicitly disabled web search was not persisted")
	}
}

func TestInvalidSearchProviderIsRejected(t *testing.T) {
	clearConfigEnv(t)
	manager := newTestManager(t, t.TempDir())
	if _, err := manager.Save(Input{SearchProvider: "other"}); err == nil {
		t.Fatal("expected unsupported provider to be rejected")
	}
}

func TestPersonalizationAndMemoryDefaultsAndLifecycle(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	manager := newTestManager(t, dir)

	initial := manager.View()
	if initial.ResponseStyle != ResponseStyleConcise ||
		initial.Warmth != WarmthBalanced ||
		initial.Enthusiasm != EnthusiasmBalanced ||
		initial.Emoji != EmojiNone {
		t.Fatalf("personalization defaults = %+v", initial)
	}
	if !initial.MemoryEnabled || !initial.MemorySummaryEnabled || !initial.MemorySearchEnabled {
		t.Fatalf("memory defaults = %+v", initial)
	}

	summaryDisabled := false
	saved, err := manager.Save(Input{
		ResponseStyle:        ResponseStyleDetailed,
		Warmth:               WarmthFriendly,
		Enthusiasm:           EnthusiasmCalm,
		Emoji:                EmojiSparing,
		CustomInstructions:   stringPointer("  回答时先给结论。  "),
		MemorySummaryEnabled: &summaryDisabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ResponseStyle != ResponseStyleDetailed ||
		saved.Warmth != WarmthFriendly ||
		saved.Enthusiasm != EnthusiasmCalm ||
		saved.Emoji != EmojiSparing ||
		saved.CustomInstructions != "回答时先给结论。" ||
		saved.MemorySummaryEnabled {
		t.Fatalf("saved personalization = %+v", saved)
	}

	reloaded := newTestManager(t, dir)
	if reloaded.View().ResponseStyle != ResponseStyleDetailed ||
		reloaded.View().MemorySummaryEnabled {
		t.Fatalf("reloaded personalization = %+v", reloaded.View())
	}

	if _, err := manager.Save(Input{ResponseStyle: "verbose"}); err == nil {
		t.Fatal("unsupported response style should be rejected")
	}
	if _, err := manager.Save(Input{Warmth: "clingy"}); err == nil {
		t.Fatal("unsupported warmth should be rejected")
	}
	if _, err := manager.Save(Input{Emoji: "always"}); err == nil {
		t.Fatal("unsupported emoji level should be rejected")
	}
	tooLong := strings.Repeat("a", MaxCustomInstructions+1)
	if _, err := manager.Save(Input{CustomInstructions: &tooLong}); err == nil {
		t.Fatal("oversized custom instructions should be rejected")
	}
}

func TestKeychainUnavailableFallsBackToSessionOnly(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	secrets := newFakeSecretStore()
	keychainErr := errors.New("secret service is unavailable")
	secrets.getErr = keychainErr
	secrets.setErr = keychainErr

	manager, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if view := manager.View(); view.KeychainAvailable ||
		!strings.Contains(view.KeychainError, keychainErr.Error()) {
		t.Fatalf("keychain status = %+v", view)
	}

	saved, err := manager.Save(Input{
		BaseURL:        "https://api.example.com/v1",
		Model:          "test-model",
		APIKey:         "session-only-key",
		SearchProvider: SearchProviderTavily,
		TavilyAPIKey:   "session-only-search-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.APIKeyStorage != KeyStorageSession || !saved.HasAPIKey || !saved.Configured {
		t.Fatalf("session-only view = %+v", saved)
	}
	if saved.TavilyAPIKeyStorage != KeyStorageSession || !saved.HasTavilyAPIKey {
		t.Fatalf("session-only search key = %+v", saved)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "session-only-key") ||
		strings.Contains(string(data), "session-only-search-key") {
		t.Fatalf("session-only key was written to config: %s", data)
	}

	reloaded, err := NewManager(dir, newFakeSecretStore())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.View().HasAPIKey {
		t.Fatal("session-only key survived a restart")
	}
	if reloaded.View().SearchProvider != SearchProviderDuckDuckGo ||
		reloaded.View().HasTavilyAPIKey {
		t.Fatalf("lost Tavily key did not fall back to DuckDuckGo: %+v", reloaded.View())
	}
}

func TestKeychainOverridesEnvironmentAndEnvironmentCannotBeCleared(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("OPENAI_API_KEY", "env-key")
	secrets := newFakeSecretStore()
	secrets.values[ModelSecretKey] = "keychain-key"
	manager, err := NewManager(t.TempDir(), secrets)
	if err != nil {
		t.Fatal(err)
	}
	view := manager.View()
	if view.APIKeyStorage != KeyStorageKeychain || manager.Settings().APIKey != "keychain-key" {
		t.Fatalf("keychain did not override env: %+v", view)
	}

	envOnly := newFakeSecretStore()
	envManager, err := NewManager(t.TempDir(), envOnly)
	if err != nil {
		t.Fatal(err)
	}
	if view := envManager.View(); view.APIKeyStorage != KeyStorageEnvironment {
		t.Fatalf("environment source = %+v", view)
	}
	if _, err := envManager.ClearAPIKey(); err == nil ||
		!strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("environment key clear error = %v", err)
	}
	if !envManager.View().HasAPIKey {
		t.Fatal("failed environment clear removed the key")
	}
}

func TestSecretDeleteFailureKeepsKey(t *testing.T) {
	clearConfigEnv(t)
	secrets := newFakeSecretStore()
	manager, err := NewManager(t.TempDir(), secrets)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Save(Input{APIKey: "sk-delete"}); err != nil {
		t.Fatal(err)
	}
	secrets.deleteErr = errors.New("keychain is locked")
	if _, err := manager.ClearAPIKey(); err == nil {
		t.Fatal("expected delete failure")
	}
	if !manager.View().HasAPIKey || secrets.values[ModelSecretKey] != "sk-delete" {
		t.Fatal("delete failure changed key state")
	}
}

func TestOversizedSecretsAreRejected(t *testing.T) {
	clearConfigEnv(t)
	secrets := newFakeSecretStore()
	manager, err := NewManager(t.TempDir(), secrets)
	if err != nil {
		t.Fatal(err)
	}
	tooLong := strings.Repeat("a", MaxSecretBytes+1)
	if _, err := manager.Save(Input{APIKey: tooLong}); err == nil {
		t.Fatal("oversized API key should be rejected")
	}
	if len(secrets.values) != 0 {
		t.Fatal("rejected key reached the secret store")
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL", "MIRA_PROTOCOL"} {
		t.Setenv(key, "")
	}
}

func stringPointer(value string) *string {
	return &value
}
