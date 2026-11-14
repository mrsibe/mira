package config

import (
	"encoding/json"
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

func TestModelLifecycleAndPerModelSecrets(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	secrets := newFakeSecretStore()

	manager, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	initial := manager.View()
	if len(initial.Models) != 0 || initial.Configured {
		t.Fatalf("new manager should have no models: %+v", initial)
	}

	view, err := manager.AddModel(ModelInput{
		Name:     "主模型",
		Provider: ProviderOpenAI,
		BaseURL:  "https://api.example.com/v1/",
		Model:    "test-model",
		Protocol: ProtocolCompletions,
		APIKey:   "sk-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Models) != 1 || view.ActiveModelID != view.Models[0].ID {
		t.Fatalf("model was not activated: %+v", view)
	}
	first := view.Models[0]
	if first.BaseURL != "https://api.example.com/v1" || first.Name != "主模型" ||
		first.ProviderLabel != "OpenAI" || !first.Configured ||
		first.APIKeyStorage != KeyStorageKeychain {
		t.Fatalf("model view = %+v", first)
	}
	if !view.Configured || !view.KeychainAvailable {
		t.Fatalf("view should be configured: %+v", view)
	}
	if secrets.values[modelSecretKey(first.ID)] != "sk-test" {
		t.Fatal("API key was not stored for the model")
	}
	if view.ConfigPath != filepath.Join(dir, "config.json") {
		t.Fatalf("config path = %q", view.ConfigPath)
	}
	if settings := manager.Settings(); settings.APIKey != "sk-test" ||
		settings.Model != "test-model" || settings.Protocol != ProtocolCompletions {
		t.Fatalf("active settings = %+v", settings)
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
	if !strings.Contains(string(data), `"schemaVersion": 2`) {
		t.Fatalf("config file is not schema v2: %s", data)
	}

	reloaded, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.View(); len(got.Models) != 1 || got.ActiveModelID != first.ID ||
		reloaded.Settings().APIKey != "sk-test" {
		t.Fatalf("reloaded view = %+v", got)
	}

	google, err := manager.AddModel(ModelInput{
		Provider: ProviderGoogle,
		APIKey:   "google-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(google.Models) != 2 || google.ActiveModelID != google.Models[1].ID {
		t.Fatalf("second model was not activated: %+v", google)
	}
	second := google.Models[1]
	if second.Model != "gemini-3-flash" || second.Protocol != ProtocolGoogle ||
		secrets.values[modelSecretKey(second.ID)] != "google-key" {
		t.Fatalf("google model = %+v", second)
	}

	activated, err := manager.ActivateModel(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if activated.ActiveModelID != first.ID || manager.Settings().APIKey != "sk-test" {
		t.Fatalf("activate model = %+v", activated)
	}

	updated, err := manager.UpdateModel(ModelInput{
		ID:       first.ID,
		Name:     "主模型",
		Provider: ProviderOpenAI,
		BaseURL:  first.BaseURL,
		Model:    "test-model-2",
		Protocol: ProtocolCompletions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Models[0].Model != "test-model-2" || !updated.Models[0].HasAPIKey {
		t.Fatalf("empty key did not preserve the saved key: %+v", updated.Models[0])
	}

	if _, err := manager.UpdateModel(ModelInput{
		ID:       first.ID,
		Provider: ProviderOpenAI,
		BaseURL:  first.BaseURL,
		Model:    "test-model-2",
		Protocol: ProtocolCompletions,
		APIKey:   "sk-new",
	}); err != nil {
		t.Fatal(err)
	}
	if secrets.values[modelSecretKey(first.ID)] != "sk-new" {
		t.Fatal("updated key was not stored")
	}

	cleared, err := manager.ClearModelAPIKey(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Models[0].HasAPIKey || cleared.Configured {
		t.Fatalf("cleared key should make the active model unconfigured: %+v", cleared)
	}
	if _, found := secrets.values[modelSecretKey(first.ID)]; found {
		t.Fatal("cleared key remains in the secret store")
	}
	if secrets.values[modelSecretKey(second.ID)] != "google-key" {
		t.Fatal("clearing one key changed another model")
	}

	deleted, err := manager.DeleteModel(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Models) != 1 || deleted.Models[0].ID != first.ID {
		t.Fatalf("delete model = %+v", deleted)
	}
	empty, err := manager.DeleteModel(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Models) != 0 || empty.Configured || empty.ActiveModelID != "" {
		t.Fatalf("deleting the last model = %+v", empty)
	}
}

func TestLegacyConfigMigratesAndWritesV2(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"baseUrl":"https://api.example.com/v1","model":"legacy-model","protocol":"completions","userName":"小明","webSearchEnabled":false}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	secrets := newFakeSecretStore()
	secrets.values[ModelSecretKey] = "legacy-key"

	manager, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	view := manager.View()
	if len(view.Models) != 1 || view.Models[0].ID != "default" ||
		view.Models[0].Provider != ProviderOpenAI ||
		view.Models[0].Model != "legacy-model" ||
		!view.Models[0].Configured || view.Models[0].APIKeyStorage != KeyStorageKeychain {
		t.Fatalf("migrated model = %+v", view)
	}
	if view.UserName != "小明" || view.WebSearchEnabled {
		t.Fatalf("legacy preferences = %+v", view)
	}
	if settings := manager.Settings(); settings.APIKey != "legacy-key" ||
		settings.Model != "legacy-model" || settings.Protocol != ProtocolCompletions {
		t.Fatalf("migrated settings = %+v", settings)
	}

	if _, err := manager.Save(Input{}); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `"schemaVersion": 2`) ||
		!strings.Contains(string(written), `"provider": "openai"`) {
		t.Fatalf("migrated config = %s", written)
	}
	var legacy legacyModelFields
	if err := json.Unmarshal(written, &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.BaseURL != "" || legacy.Model != "" || legacy.Protocol != "" {
		t.Fatalf("legacy top-level fields were written: %s", written)
	}
	reloaded, err := NewManager(dir, secrets)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.View(); len(got.Models) != 1 || got.Models[0].Model != "legacy-model" {
		t.Fatalf("reloaded migrated config = %+v", got)
	}
}

func TestEnvironmentFallbackAndLocalProvider(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "http://localhost:11434/v1")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "local-model")
	t.Setenv("MIRA_PROTOCOL", ProtocolCompletions)

	manager := newTestManager(t, t.TempDir())
	view := manager.View()
	if len(view.Models) != 1 {
		t.Fatalf("environment model = %+v", view)
	}
	model := view.Models[0]
	if model.Provider != ProviderOpenAI || model.BaseURL != "http://localhost:11434/v1" ||
		model.Model != "local-model" || model.Protocol != ProtocolCompletions ||
		!model.Configured || !view.Configured {
		t.Fatalf("environment fallback = %+v", view)
	}
}

func TestEnvironmentKeyOverridesAndCannotBeCleared(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("OPENAI_API_KEY", "env-key")
	manager := newTestManager(t, t.TempDir())
	view := manager.View()
	if len(view.Models) != 1 || view.Models[0].APIKeyStorage != KeyStorageEnvironment ||
		!view.Configured {
		t.Fatalf("environment source = %+v", view)
	}
	if _, err := manager.ClearModelAPIKey(view.Models[0].ID); err == nil ||
		!strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("environment key clear error = %v", err)
	}
	if !manager.View().Models[0].HasAPIKey {
		t.Fatal("failed environment clear removed the key")
	}
}

func TestProviderDefaultsAndProtocols(t *testing.T) {
	clearConfigEnv(t)
	tests := []struct {
		name         string
		input        ModelInput
		wantBase     string
		wantModel    string
		wantProtocol string
	}{
		{
			name:         "openai",
			input:        ModelInput{Provider: ProviderOpenAI, APIKey: "k"},
			wantBase:     DefaultBaseURL,
			wantModel:    DefaultModel,
			wantProtocol: ProtocolResponses,
		},
		{
			name: "azure",
			input: ModelInput{
				Provider: ProviderAzure,
				BaseURL:  "https://example.openai.azure.com/openai/v1",
				Model:    "deployment",
				APIKey:   "k",
			},
			wantBase:     "https://example.openai.azure.com/openai/v1",
			wantModel:    "deployment",
			wantProtocol: ProtocolResponses,
		},
		{
			name:         "google",
			input:        ModelInput{Provider: ProviderGoogle, APIKey: "k"},
			wantModel:    "gemini-3-flash",
			wantProtocol: ProtocolGoogle,
		},
		{
			name:         "anthropic",
			input:        ModelInput{Provider: ProviderAnthropic, APIKey: "k"},
			wantBase:     "https://api.anthropic.com",
			wantModel:    "claude-sonnet-5-5",
			wantProtocol: ProtocolAnthropic,
		},
		{
			name:         "deepseek",
			input:        ModelInput{Provider: ProviderDeepSeek, APIKey: "k"},
			wantBase:     "https://api.deepseek.com",
			wantModel:    "deepseek-reasoner",
			wantProtocol: ProtocolCompletions,
		},
		{
			name: "openrouter messages",
			input: ModelInput{
				Provider: ProviderOpenRouter,
				Protocol: ProtocolMessages,
				APIKey:   "k",
			},
			wantBase:     "https://openrouter.ai/api/v1",
			wantModel:    "anthropic/claude-sonnet-4",
			wantProtocol: ProtocolMessages,
		},
		{
			name:         "xai",
			input:        ModelInput{Provider: ProviderXAI, APIKey: "k"},
			wantBase:     "https://api.x.ai/v1",
			wantModel:    "grok-4",
			wantProtocol: ProtocolResponses,
		},
		{
			name: "custom",
			input: ModelInput{
				Provider: ProviderCustom,
				BaseURL:  "https://example.com/v1",
				Model:    "custom-model",
				APIKey:   "k",
			},
			wantBase:     "https://example.com/v1",
			wantModel:    "custom-model",
			wantProtocol: ProtocolCompletions,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := newTestManager(t, t.TempDir())
			view, err := manager.AddModel(test.input)
			if err != nil {
				t.Fatal(err)
			}
			model := view.Models[0]
			if model.BaseURL != test.wantBase || model.Model != test.wantModel ||
				model.Protocol != test.wantProtocol {
				t.Fatalf("model = %+v", model)
			}
		})
	}
}

func TestInvalidModelsAreRejected(t *testing.T) {
	clearConfigEnv(t)
	manager := newTestManager(t, t.TempDir())
	tests := []ModelInput{
		{Provider: "unknown", Model: "x", APIKey: "k"},
		{Provider: ProviderAzure, Model: "deployment", APIKey: "k"},
		{Provider: ProviderCustom, Model: "custom"},
		{Provider: ProviderCustom, BaseURL: "ftp://example.com", Model: "custom"},
		{Provider: ProviderCustom, BaseURL: "https://example.com", Model: "custom"},
	}
	for _, input := range tests {
		if _, err := manager.AddModel(input); err == nil {
			t.Fatalf("expected %+v to be rejected", input)
		}
	}
	if len(manager.View().Models) != 0 {
		t.Fatal("failed adds changed the model list")
	}
}

func TestResolveModelSettingsReusesStoredKey(t *testing.T) {
	clearConfigEnv(t)
	manager := newTestManager(t, t.TempDir())
	view, err := manager.AddModel(ModelInput{
		Provider: ProviderOpenAI,
		BaseURL:  "https://api.example.com/v1",
		Model:    "model",
		Protocol: ProtocolCompletions,
		APIKey:   "stored-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	model := view.Models[0]
	settings, err := manager.ResolveModelSettings(ModelInput{
		ID:       model.ID,
		Provider: model.Provider,
		BaseURL:  model.BaseURL,
		Model:    model.Model,
		Protocol: model.Protocol,
	})
	if err != nil {
		t.Fatal(err)
	}
	if settings.APIKey != "stored-key" {
		t.Fatalf("stored key was not reused: %+v", settings)
	}
	settings, err = manager.ResolveModelSettings(ModelInput{
		ID:       model.ID,
		Provider: model.Provider,
		BaseURL:  model.BaseURL,
		Model:    model.Model,
		Protocol: model.Protocol,
		APIKey:   "draft-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if settings.APIKey != "draft-key" {
		t.Fatalf("draft key was not used: %+v", settings)
	}
}

func TestGlobalPreferenceLifecycle(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	manager := newTestManager(t, dir)

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

	initial := manager.View()
	if !initial.WebSearchEnabled || initial.SearchProvider != SearchProviderDuckDuckGo ||
		initial.ResponseStyle != ResponseStyleConcise ||
		initial.Personality != PersonalityFriendly ||
		initial.Warmth != WarmthBalanced ||
		initial.Enthusiasm != EnthusiasmBalanced ||
		initial.Emoji != EmojiNone ||
		!initial.MemoryEnabled || !initial.MemorySummaryEnabled ||
		!initial.MemorySearchEnabled || !initial.IntelligentUIEnabled {
		t.Fatalf("defaults = %+v", initial)
	}

	if _, err := manager.Save(Input{SearchProvider: SearchProviderTavily}); err == nil {
		t.Fatal("Tavily without a key should be rejected")
	}
	disabled := false
	saved, err := manager.Save(Input{
		SearchProvider:       SearchProviderTavily,
		TavilyAPIKey:         "test-search-key",
		WebSearchEnabled:     &disabled,
		ResponseStyle:        ResponseStyleDetailed,
		Personality:          PersonalityCandid,
		Warmth:               WarmthFriendly,
		Enthusiasm:           EnthusiasmCalm,
		Emoji:                EmojiSparing,
		CustomInstructions:   stringPointer("  回答时先给结论。  "),
		MemorySummaryEnabled: &disabled,
		IntelligentUIEnabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.WebSearchEnabled || !saved.HasTavilyAPIKey ||
		saved.SearchProvider != SearchProviderTavily ||
		saved.ResponseStyle != ResponseStyleDetailed ||
		saved.Personality != PersonalityCandid ||
		saved.Warmth != WarmthFriendly ||
		saved.Enthusiasm != EnthusiasmCalm ||
		saved.Emoji != EmojiSparing ||
		saved.CustomInstructions != "回答时先给结论。" ||
		saved.MemorySummaryEnabled || saved.IntelligentUIEnabled {
		t.Fatalf("saved preferences = %+v", saved)
	}
	if saved.TavilyAPIKeyStorage != KeyStorageKeychain {
		t.Fatalf("Tavily key storage = %+v", saved)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "test-search-key") ||
		strings.Contains(string(data), "tavilyApiKey") {
		t.Fatalf("config file contains the Tavily key: %s", data)
	}

	reloaded, err := NewManager(dir, newFakeSecretStore())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.View().SearchProvider != SearchProviderDuckDuckGo ||
		reloaded.View().HasTavilyAPIKey {
		t.Fatalf("lost Tavily key did not fall back to DuckDuckGo: %+v", reloaded.View())
	}
	if _, err := manager.Save(Input{SearchProvider: "other"}); err == nil {
		t.Fatal("unsupported search provider should be rejected")
	}
	if _, err := manager.Save(Input{Personality: "dramatic"}); err == nil {
		t.Fatal("unsupported personality should be rejected")
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
	saved, err := manager.AddModel(ModelInput{
		Provider: ProviderOpenAI,
		BaseURL:  "https://api.example.com/v1",
		Model:    "test-model",
		APIKey:   "session-only-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Models[0].APIKeyStorage != KeyStorageSession ||
		!saved.Models[0].HasAPIKey || !saved.Configured {
		t.Fatalf("session-only view = %+v", saved)
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "session-only-key") {
		t.Fatalf("session-only key was written to config: %s", data)
	}

	reloaded, err := NewManager(dir, newFakeSecretStore())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.View().Models[0].HasAPIKey || reloaded.View().Configured {
		t.Fatal("session-only key survived a restart")
	}
}

func TestKeychainOverridesEnvironment(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("OPENAI_API_KEY", "env-key")
	secrets := newFakeSecretStore()
	secrets.values[ModelSecretKey] = "keychain-key"
	manager, err := NewManager(t.TempDir(), secrets)
	if err != nil {
		t.Fatal(err)
	}
	view := manager.View()
	if view.Models[0].APIKeyStorage != KeyStorageKeychain ||
		manager.Settings().APIKey != "keychain-key" {
		t.Fatalf("keychain did not override env: %+v", view)
	}
}

func TestSecretDeleteFailureKeepsKey(t *testing.T) {
	clearConfigEnv(t)
	secrets := newFakeSecretStore()
	manager, err := NewManager(t.TempDir(), secrets)
	if err != nil {
		t.Fatal(err)
	}
	view, err := manager.AddModel(ModelInput{
		Provider: ProviderOpenAI,
		BaseURL:  "https://api.example.com/v1",
		Model:    "test-model",
		APIKey:   "sk-delete",
	})
	if err != nil {
		t.Fatal(err)
	}
	secrets.deleteErr = errors.New("keychain is locked")
	if _, err := manager.ClearModelAPIKey(view.Models[0].ID); err == nil {
		t.Fatal("expected delete failure")
	}
	if !manager.View().Models[0].HasAPIKey ||
		secrets.values[modelSecretKey(view.Models[0].ID)] != "sk-delete" {
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
	if _, err := manager.AddModel(ModelInput{
		Provider: ProviderGoogle,
		APIKey:   tooLong,
	}); err == nil {
		t.Fatal("oversized API key should be rejected")
	}
	if len(secrets.values) != 0 {
		t.Fatal("rejected key reached the secret store")
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
	if len(manager.View().Models) != 0 {
		t.Fatalf("unexpected defaults: %+v", manager.View())
	}
	if _, err := manager.Save(Input{Personality: PersonalityCandid}); err != nil {
		t.Fatal(err)
	}
	if manager.LoadError() != nil {
		t.Fatal("a successful save should clear the load error")
	}
	if !strings.HasSuffix(manager.View().ConfigPath, "config.json") {
		t.Fatalf("config path = %q", manager.View().ConfigPath)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL", "MIRA_PROTOCOL",
		"AZURE_OPENAI_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY",
		"ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY", "XAI_API_KEY",
	} {
		t.Setenv(key, "")
	}
}

func stringPointer(value string) *string {
	return &value
}
