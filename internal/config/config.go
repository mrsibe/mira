// Package config stores the local model provider settings for Mira.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	ProtocolResponses   = "responses"
	ProtocolCompletions = "completions"
	ProtocolMessages    = "messages"
	ProtocolGoogle      = "google"
	ProtocolAnthropic   = "anthropic"

	SearchProviderDuckDuckGo = "duckduckgo"
	SearchProviderTavily     = "tavily"

	ResponseStyleConcise  = "concise"
	ResponseStyleBalanced = "balanced"
	ResponseStyleDetailed = "detailed"

	WarmthReserved = "reserved"
	WarmthBalanced = "balanced"
	WarmthFriendly = "friendly"

	EnthusiasmCalm         = "calm"
	EnthusiasmBalanced     = "balanced"
	EnthusiasmEnthusiastic = "enthusiastic"

	EmojiNone     = "none"
	EmojiSparing  = "sparing"
	EmojiFrequent = "frequent"

	PersonalityProfessional = "professional"
	PersonalityFriendly     = "friendly"
	PersonalityCandid       = "candid"
	PersonalityQuirky       = "quirky"
	PersonalityEfficient    = "efficient"
	PersonalityCynical      = "cynical"

	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-5"

	DefaultSearchProvider = SearchProviderDuckDuckGo
	MaxCustomInstructions = 5000
	MaxSecretBytes        = 2560

	ModelSecretKey  = "model-api-key"
	TavilySecretKey = "tavily-api-key"

	SchemaVersion = 2

	KeyStorageKeychain    = "keychain"
	KeyStorageEnvironment = "environment"
	KeyStorageSession     = "session"
	KeyStorageNone        = "none"
)

// SecretStore keeps API keys out of the settings file.
type SecretStore interface {
	Get(key string) (value string, found bool, err error)
	Set(key, value string) error
	Delete(key string) error
}

// Settings is the runtime configuration. The active model's connection fields
// are resolved into the flat fields; they are never written to config.json.
type Settings struct {
	SchemaVersion int           `json:"-"`
	Models        []ModelConfig `json:"-"`
	ActiveModelID string        `json:"-"`

	Provider string `json:"-"`
	BaseURL  string `json:"-"`
	APIKey   string `json:"-"`
	Model    string `json:"-"`
	Protocol string `json:"-"`

	UserName             string `json:"userName,omitempty"`
	WebSearchEnabled     bool   `json:"webSearchEnabled"`
	SearchProvider       string `json:"searchProvider,omitempty"`
	TavilyAPIKey         string `json:"-"`
	ResponseStyle        string `json:"responseStyle,omitempty"`
	Personality          string `json:"personality,omitempty"`
	Warmth               string `json:"warmth,omitempty"`
	Enthusiasm           string `json:"enthusiasm,omitempty"`
	Emoji                string `json:"emoji,omitempty"`
	CustomInstructions   string `json:"customInstructions,omitempty"`
	MemoryEnabled        bool   `json:"memoryEnabled"`
	MemorySummaryEnabled bool   `json:"memorySummaryEnabled"`
	MemorySearchEnabled  bool   `json:"memorySearchEnabled"`
	IntelligentUIEnabled bool   `json:"intelligentUIEnabled"`
}

// Input is the frontend-safe preference update payload. Nil pointers preserve
// the existing value. Model connection settings use ModelInput instead.
type Input struct {
	UserName             *string `json:"userName,omitempty"`
	WebSearchEnabled     *bool   `json:"webSearchEnabled,omitempty"`
	SearchProvider       string  `json:"searchProvider"`
	TavilyAPIKey         string  `json:"tavilyApiKey"`
	ResponseStyle        string  `json:"responseStyle"`
	Personality          string  `json:"personality"`
	Warmth               string  `json:"warmth"`
	Enthusiasm           string  `json:"enthusiasm"`
	Emoji                string  `json:"emoji"`
	CustomInstructions   *string `json:"customInstructions,omitempty"`
	MemoryEnabled        *bool   `json:"memoryEnabled,omitempty"`
	MemorySummaryEnabled *bool   `json:"memorySummaryEnabled,omitempty"`
	MemorySearchEnabled  *bool   `json:"memorySearchEnabled,omitempty"`
	IntelligentUIEnabled *bool   `json:"intelligentUIEnabled,omitempty"`
}

// View is returned to the frontend. API keys are deliberately omitted.
type View struct {
	Models               []ModelView `json:"models"`
	ActiveModelID        string      `json:"activeModelId"`
	UserName             string      `json:"userName"`
	Configured           bool        `json:"configured"`
	ConfigPath           string      `json:"configPath"`
	WebSearchEnabled     bool        `json:"webSearchEnabled"`
	SearchProvider       string      `json:"searchProvider"`
	HasTavilyAPIKey      bool        `json:"hasTavilyApiKey"`
	TavilyAPIKeyStorage  string      `json:"tavilyApiKeyStorage"`
	KeychainAvailable    bool        `json:"keychainAvailable"`
	KeychainError        string      `json:"keychainError,omitempty"`
	ResponseStyle        string      `json:"responseStyle"`
	Personality          string      `json:"personality"`
	Warmth               string      `json:"warmth"`
	Enthusiasm           string      `json:"enthusiasm"`
	Emoji                string      `json:"emoji"`
	CustomInstructions   string      `json:"customInstructions"`
	MemoryEnabled        bool        `json:"memoryEnabled"`
	MemorySummaryEnabled bool        `json:"memorySummaryEnabled"`
	MemorySearchEnabled  bool        `json:"memorySearchEnabled"`
	IntelligentUIEnabled bool        `json:"intelligentUIEnabled"`
}

type storedSettings struct {
	SchemaVersion        int           `json:"schemaVersion"`
	Models               []ModelConfig `json:"models"`
	ActiveModelID        string        `json:"activeModelId,omitempty"`
	UserName             string        `json:"userName,omitempty"`
	WebSearchEnabled     bool          `json:"webSearchEnabled"`
	SearchProvider       string        `json:"searchProvider,omitempty"`
	ResponseStyle        string        `json:"responseStyle,omitempty"`
	Personality          string        `json:"personality,omitempty"`
	Warmth               string        `json:"warmth,omitempty"`
	Enthusiasm           string        `json:"enthusiasm,omitempty"`
	Emoji                string        `json:"emoji,omitempty"`
	CustomInstructions   string        `json:"customInstructions,omitempty"`
	MemoryEnabled        bool          `json:"memoryEnabled"`
	MemorySummaryEnabled bool          `json:"memorySummaryEnabled"`
	MemorySearchEnabled  bool          `json:"memorySearchEnabled"`
	IntelligentUIEnabled bool          `json:"intelligentUIEnabled"`
}

type legacyModelFields struct {
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
	Protocol string `json:"protocol"`
}

// Manager owns the settings file and keeps a validated snapshot in memory.
type Manager struct {
	mu                  sync.RWMutex
	path                string
	secrets             SecretStore
	settings            Settings
	loadErr             error
	modelKeys           map[string]string
	modelKeyStorage     map[string]string
	tavilyAPIKeyStorage string
	keychainAvailable   bool
	keychainError       string
}

// NewManager loads <dir>/config.json, falling back to environment variables
// and the system credential store.
func NewManager(dir string, secrets SecretStore) (*Manager, error) {
	path := filepath.Join(dir, "config.json")
	settings := defaultSettings()
	envModels := environmentModels()
	settings.Models = envModels
	settings.ActiveModelID = firstModelID(envModels)

	m := &Manager{
		path:                path,
		secrets:             secrets,
		settings:            settings,
		modelKeys:           make(map[string]string),
		modelKeyStorage:     make(map[string]string),
		tavilyAPIKeyStorage: KeyStorageNone,
	}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var saved storedSettings
		saved = defaultStoredSettings()
		if err := json.Unmarshal(data, &saved); err != nil {
			m.loadErr = fmt.Errorf("parse %s: %w", path, err)
		} else {
			applyStoredSettings(&settings, saved)
			if saved.SchemaVersion >= SchemaVersion {
				settings.Models = saved.Models
				settings.ActiveModelID = saved.ActiveModelID
			} else {
				var legacy legacyModelFields
				_ = json.Unmarshal(data, &legacy)
				if legacy.BaseURL != "" || legacy.Model != "" || legacy.Protocol != "" {
					settings.Models = []ModelConfig{legacyModel(legacy)}
					settings.ActiveModelID = "default"
				}
			}
		}
	case !errors.Is(err, os.ErrNotExist):
		m.loadErr = fmt.Errorf("read %s: %w", path, err)
	}

	m.settings = normalized(settings)
	m.loadSecrets()
	if m.settings.SearchProvider == SearchProviderTavily && m.settings.TavilyAPIKey == "" {
		m.settings.SearchProvider = DefaultSearchProvider
	}
	m.settings = normalized(m.settings)
	m.resolveActiveLocked()
	return m, nil
}

func defaultSettings() Settings {
	return Settings{
		SchemaVersion:        SchemaVersion,
		WebSearchEnabled:     true,
		SearchProvider:       DefaultSearchProvider,
		Personality:          PersonalityFriendly,
		MemoryEnabled:        true,
		MemorySummaryEnabled: true,
		MemorySearchEnabled:  true,
		IntelligentUIEnabled: true,
	}
}

func defaultStoredSettings() storedSettings {
	return storedSettings{
		WebSearchEnabled:     true,
		SearchProvider:       DefaultSearchProvider,
		Personality:          PersonalityFriendly,
		MemoryEnabled:        true,
		MemorySummaryEnabled: true,
		MemorySearchEnabled:  true,
		IntelligentUIEnabled: true,
	}
}

func applyStoredSettings(settings *Settings, saved storedSettings) {
	settings.SchemaVersion = SchemaVersion
	settings.UserName = saved.UserName
	settings.WebSearchEnabled = saved.WebSearchEnabled
	if saved.SearchProvider != "" {
		settings.SearchProvider = saved.SearchProvider
	}
	settings.ResponseStyle = saved.ResponseStyle
	settings.Personality = saved.Personality
	settings.Warmth = saved.Warmth
	settings.Enthusiasm = saved.Enthusiasm
	settings.Emoji = saved.Emoji
	settings.CustomInstructions = saved.CustomInstructions
	settings.MemoryEnabled = saved.MemoryEnabled
	settings.MemorySummaryEnabled = saved.MemorySummaryEnabled
	settings.MemorySearchEnabled = saved.MemorySearchEnabled
	settings.IntelligentUIEnabled = saved.IntelligentUIEnabled
}

func environmentModels() []ModelConfig {
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	protocol := strings.TrimSpace(os.Getenv("MIRA_PROTOCOL"))
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if baseURL == "" && model == "" && protocol == "" && apiKey == "" {
		return nil
	}
	return []ModelConfig{{
		ID:       "default",
		Name:     model,
		Provider: ProviderOpenAI,
		BaseURL:  firstNonEmpty(baseURL, DefaultBaseURL),
		Model:    firstNonEmpty(model, DefaultModel),
		Protocol: firstNonEmpty(protocol, ProtocolResponses),
	}}
}

func legacyModel(legacy legacyModelFields) ModelConfig {
	return ModelConfig{
		ID:       "default",
		Name:     legacy.Model,
		Provider: ProviderOpenAI,
		BaseURL:  firstNonEmpty(legacy.BaseURL, DefaultBaseURL),
		Model:    firstNonEmpty(legacy.Model, DefaultModel),
		Protocol: firstNonEmpty(legacy.Protocol, ProtocolResponses),
	}
}

func (m *Manager) loadSecrets() {
	if m.secrets == nil {
		m.markKeychainUnavailable(errors.New("系统钥匙串不可用"))
	} else {
		for _, model := range m.settings.Models {
			key, storage := "", KeyStorageNone
			if value, found, err := m.secrets.Get(modelSecretKey(model.ID)); err != nil {
				m.markKeychainUnavailable(err)
			} else {
				m.keychainAvailable = true
				if found && strings.TrimSpace(value) != "" {
					key = strings.TrimSpace(value)
					storage = KeyStorageKeychain
				}
			}
			if key == "" && model.ID == "default" {
				if value, found, err := m.secrets.Get(ModelSecretKey); err != nil {
					m.markKeychainUnavailable(err)
				} else {
					m.keychainAvailable = true
					if found && strings.TrimSpace(value) != "" {
						key = strings.TrimSpace(value)
						storage = KeyStorageKeychain
					}
				}
			}
			if key == "" {
				if value := strings.TrimSpace(os.Getenv(modelEnvKey(model.Provider))); value != "" {
					key = value
					storage = KeyStorageEnvironment
				}
			}
			m.modelKeys[model.ID] = key
			m.modelKeyStorage[model.ID] = storage
		}
	}
	if m.secrets == nil {
		return
	}
	if value, found, err := m.secrets.Get(TavilySecretKey); err != nil {
		m.markKeychainUnavailable(err)
	} else {
		m.keychainAvailable = true
		if found && strings.TrimSpace(value) != "" {
			m.settings.TavilyAPIKey = strings.TrimSpace(value)
			m.tavilyAPIKeyStorage = KeyStorageKeychain
		}
	}
}

func (m *Manager) markKeychainUnavailable(err error) {
	m.keychainAvailable = false
	if m.keychainError == "" && err != nil {
		m.keychainError = err.Error()
	}
}

// LoadError reports a problem reading the existing settings file. The app can
// still run with defaults and overwrite the file from the settings dialog.
func (m *Manager) LoadError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadErr
}

// Settings returns the active model settings including its API key.
func (m *Manager) Settings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	settings := m.settings
	settings.Models = append([]ModelConfig(nil), m.settings.Models...)
	return settings
}

// View returns the frontend-safe configuration snapshot.
func (m *Manager) View() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viewLocked()
}

// Save updates global preferences. Model connection settings are managed with
// the model methods.
func (m *Manager) Save(input Input) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.settings
	tavilyKey := ""
	if input.UserName != nil {
		next.UserName = *input.UserName
	}
	if input.WebSearchEnabled != nil {
		next.WebSearchEnabled = *input.WebSearchEnabled
	}
	if value := strings.TrimSpace(input.SearchProvider); value != "" {
		next.SearchProvider = value
	}
	if value := strings.TrimSpace(input.TavilyAPIKey); value != "" {
		if err := validateSecret(value); err != nil {
			return m.viewLocked(), fmt.Errorf("Tavily API Key %w", err)
		}
		next.TavilyAPIKey = value
		tavilyKey = value
	}
	if value := strings.TrimSpace(input.ResponseStyle); value != "" {
		next.ResponseStyle = value
	}
	if value := strings.TrimSpace(input.Personality); value != "" {
		next.Personality = value
	}
	if value := strings.TrimSpace(input.Warmth); value != "" {
		next.Warmth = value
	}
	if value := strings.TrimSpace(input.Enthusiasm); value != "" {
		next.Enthusiasm = value
	}
	if value := strings.TrimSpace(input.Emoji); value != "" {
		next.Emoji = value
	}
	if input.CustomInstructions != nil {
		next.CustomInstructions = *input.CustomInstructions
	}
	if input.MemoryEnabled != nil {
		next.MemoryEnabled = *input.MemoryEnabled
	}
	if input.MemorySummaryEnabled != nil {
		next.MemorySummaryEnabled = *input.MemorySummaryEnabled
	}
	if input.MemorySearchEnabled != nil {
		next.MemorySearchEnabled = *input.MemorySearchEnabled
	}
	if input.IntelligentUIEnabled != nil {
		next.IntelligentUIEnabled = *input.IntelligentUIEnabled
	}
	next = normalized(next)
	if err := validate(next); err != nil {
		return m.viewLocked(), err
	}
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	if tavilyKey != "" {
		if m.storeSecret(TavilySecretKey, tavilyKey) {
			m.tavilyAPIKeyStorage = KeyStorageKeychain
		} else {
			m.tavilyAPIKeyStorage = KeyStorageSession
		}
	}
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// AddModel appends a model and makes it the active model.
func (m *Manager) AddModel(input ModelInput) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, apiKey, err := prepareModel(input, "")
	if err != nil {
		return m.viewLocked(), err
	}
	if modelNeedsKey(model) && apiKey == "" {
		return m.viewLocked(), errors.New("请填写 API Key")
	}
	model.ID = newModelID()
	for modelIDExists(m.settings.Models, model.ID) {
		model.ID = newModelID()
	}
	next := m.settings
	next.Models = append(append([]ModelConfig(nil), m.settings.Models...), model)
	next.ActiveModelID = model.ID
	next = normalized(next)
	if err := validate(next); err != nil {
		return m.viewLocked(), err
	}
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	m.setModelKey(model.ID, apiKey)
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// UpdateModel replaces one model's connection fields. An empty API key keeps
// the existing key.
func (m *Manager) UpdateModel(input ModelInput) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := modelIndex(m.settings.Models, input.ID)
	if index < 0 {
		return m.viewLocked(), errors.New("模型不存在")
	}
	model, apiKey, err := prepareModel(input, input.ID)
	if err != nil {
		return m.viewLocked(), err
	}
	if modelNeedsKey(model) && apiKey == "" && m.modelKeys[model.ID] == "" {
		return m.viewLocked(), errors.New("请填写 API Key")
	}
	next := m.settings
	next.Models = append([]ModelConfig(nil), m.settings.Models...)
	next.Models[index] = model
	next = normalized(next)
	if err := validate(next); err != nil {
		return m.viewLocked(), err
	}
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	if apiKey != "" {
		m.setModelKey(model.ID, apiKey)
	}
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// DeleteModel removes one model and falls back to the first remaining model.
func (m *Manager) DeleteModel(id string) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := modelIndex(m.settings.Models, id)
	if index < 0 {
		return m.viewLocked(), errors.New("模型不存在")
	}
	next := m.settings
	next.Models = append([]ModelConfig(nil), m.settings.Models...)
	next.Models = append(next.Models[:index], next.Models[index+1:]...)
	if next.ActiveModelID == id {
		next.ActiveModelID = firstModelID(next.Models)
	}
	next = normalized(next)
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	_ = m.deleteSecret(modelSecretKey(id))
	if id == "default" {
		_ = m.deleteSecret(ModelSecretKey)
	}
	delete(m.modelKeys, id)
	delete(m.modelKeyStorage, id)
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// ActivateModel changes which model handles subsequent runs.
func (m *Manager) ActivateModel(id string) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if modelIndex(m.settings.Models, id) < 0 {
		return m.viewLocked(), errors.New("模型不存在")
	}
	next := m.settings
	next.ActiveModelID = id
	next = normalized(next)
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// ClearModelAPIKey removes one model's saved key.
func (m *Manager) ClearModelAPIKey(id string) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, ok := m.modelByIDLocked(id)
	if !ok {
		return m.viewLocked(), errors.New("模型不存在")
	}
	if storage := m.modelKeyStorage[id]; storage == KeyStorageEnvironment {
		env := modelEnvKey(model.Provider)
		return m.viewLocked(), fmt.Errorf("API Key 来自环境变量 %s，请在系统环境中修改", env)
	}
	if m.modelKeyStorage[id] == KeyStorageKeychain {
		if id == "default" {
			perModelErr := m.deleteSecret(modelSecretKey(id))
			legacyErr := m.deleteSecret(ModelSecretKey)
			if perModelErr != nil && legacyErr != nil {
				return m.viewLocked(), perModelErr
			}
		} else if err := m.deleteSecret(modelSecretKey(id)); err != nil {
			return m.viewLocked(), err
		}
	}
	delete(m.modelKeys, id)
	m.modelKeyStorage[id] = KeyStorageNone
	m.resolveActiveLocked()
	return m.viewLocked(), nil
}

// ResolveModelSettings builds a runtime settings snapshot for a draft model
// without persisting it. An empty API key reuses the stored key for input.ID.
func (m *Manager) ResolveModelSettings(input ModelInput) (Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	model, apiKey, err := prepareModel(input, input.ID)
	if err != nil {
		return Settings{}, err
	}
	if apiKey == "" && input.ID != "" {
		apiKey = m.modelKeys[input.ID]
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv(modelEnvKey(model.Provider)))
	}
	settings := m.settings
	settings.Provider = model.Provider
	settings.BaseURL = model.BaseURL
	settings.Model = model.Model
	settings.Protocol = model.Protocol
	settings.APIKey = apiKey
	return settings, nil
}

// ClearTavilyAPIKey removes the saved Tavily key. If Tavily is selected, the
// search provider falls back to DuckDuckGo so the saved configuration remains
// valid.
func (m *Manager) ClearTavilyAPIKey() (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldKey := m.settings.TavilyAPIKey
	if m.tavilyAPIKeyStorage == KeyStorageKeychain {
		if err := m.deleteSecret(TavilySecretKey); err != nil {
			return m.viewLocked(), err
		}
	}
	next := m.settings
	next.TavilyAPIKey = ""
	if next.SearchProvider == SearchProviderTavily {
		next.SearchProvider = SearchProviderDuckDuckGo
	}
	if err := writeSettings(m.path, next); err != nil {
		if m.tavilyAPIKeyStorage == KeyStorageKeychain && oldKey != "" {
			_ = m.restoreSecret(TavilySecretKey, oldKey)
		}
		return m.viewLocked(), err
	}
	m.settings = next
	m.tavilyAPIKeyStorage = KeyStorageNone
	return m.viewLocked(), nil
}

func validateSecret(value string) error {
	if len([]byte(value)) > MaxSecretBytes {
		return fmt.Errorf("最多 %d 个字节", MaxSecretBytes)
	}
	return nil
}

func (m *Manager) storeSecret(key, value string) bool {
	if m.secrets == nil {
		m.markKeychainUnavailable(errors.New("系统钥匙串不可用"))
		return false
	}
	if err := m.secrets.Set(key, value); err != nil {
		m.markKeychainUnavailable(err)
		return false
	}
	m.keychainAvailable = true
	m.keychainError = ""
	return true
}

func (m *Manager) deleteSecret(key string) error {
	if m.secrets == nil {
		err := errors.New("系统钥匙串不可用")
		m.markKeychainUnavailable(err)
		return err
	}
	if err := m.secrets.Delete(key); err != nil {
		m.markKeychainUnavailable(err)
		return fmt.Errorf("删除系统钥匙串中的密钥: %w", err)
	}
	m.keychainAvailable = true
	m.keychainError = ""
	return nil
}

func (m *Manager) restoreSecret(key, value string) error {
	if m.secrets == nil {
		return errors.New("系统钥匙串不可用")
	}
	return m.secrets.Set(key, value)
}

func (m *Manager) setModelKey(id, key string) {
	key = strings.TrimSpace(key)
	if key == "" || id == "" {
		return
	}
	if m.storeSecret(modelSecretKey(id), key) {
		m.modelKeyStorage[id] = KeyStorageKeychain
		if id == "default" {
			_ = m.deleteSecret(ModelSecretKey)
		}
	} else {
		m.modelKeyStorage[id] = KeyStorageSession
	}
	m.modelKeys[id] = key
}

func (m *Manager) resolveActiveLocked() {
	model, ok := m.activeModelLocked()
	if !ok {
		m.settings.Provider = ""
		m.settings.BaseURL = ""
		m.settings.Model = ""
		m.settings.Protocol = ""
		m.settings.APIKey = ""
		return
	}
	m.settings.Provider = model.Provider
	m.settings.BaseURL = model.BaseURL
	m.settings.Model = model.Model
	m.settings.Protocol = model.Protocol
	m.settings.APIKey = m.modelKeys[model.ID]
}

func (m *Manager) activeModelLocked() (ModelConfig, bool) {
	index := modelIndex(m.settings.Models, m.settings.ActiveModelID)
	if index < 0 {
		return ModelConfig{}, false
	}
	return m.settings.Models[index], true
}

func (m *Manager) modelByIDLocked(id string) (ModelConfig, bool) {
	index := modelIndex(m.settings.Models, id)
	if index < 0 {
		return ModelConfig{}, false
	}
	return m.settings.Models[index], true
}

func (m *Manager) viewLocked() View {
	models := make([]ModelView, 0, len(m.settings.Models))
	for _, model := range m.settings.Models {
		info, _ := providerInfo(model.Provider)
		key := m.modelKeys[model.ID]
		storage := m.modelKeyStorage[model.ID]
		if storage == "" {
			storage = KeyStorageNone
		}
		name := model.Name
		if name == "" {
			name = model.Model
		}
		models = append(models, ModelView{
			ID:            model.ID,
			Name:          name,
			Provider:      model.Provider,
			ProviderLabel: info.Label,
			BaseURL:       model.BaseURL,
			Model:         model.Model,
			Protocol:      model.Protocol,
			Configured:    modelConfigured(model, key),
			HasAPIKey:     key != "",
			APIKeyStorage: storage,
		})
	}
	active, ok := m.activeModelLocked()
	configured := ok && modelConfigured(active, m.modelKeys[active.ID])
	return View{
		Models:               models,
		ActiveModelID:        m.settings.ActiveModelID,
		UserName:             m.settings.UserName,
		Configured:           configured,
		ConfigPath:           m.path,
		WebSearchEnabled:     m.settings.WebSearchEnabled,
		SearchProvider:       m.settings.SearchProvider,
		HasTavilyAPIKey:      m.settings.TavilyAPIKey != "",
		TavilyAPIKeyStorage:  m.tavilyAPIKeyStorage,
		KeychainAvailable:    m.keychainAvailable,
		KeychainError:        m.keychainError,
		ResponseStyle:        m.settings.ResponseStyle,
		Personality:          m.settings.Personality,
		Warmth:               m.settings.Warmth,
		Enthusiasm:           m.settings.Enthusiasm,
		Emoji:                m.settings.Emoji,
		CustomInstructions:   m.settings.CustomInstructions,
		MemoryEnabled:        m.settings.MemoryEnabled,
		MemorySummaryEnabled: m.settings.MemorySummaryEnabled,
		MemorySearchEnabled:  m.settings.MemorySearchEnabled,
		IntelligentUIEnabled: m.settings.IntelligentUIEnabled,
	}
}

func normalized(settings Settings) Settings {
	settings.SchemaVersion = SchemaVersion
	settings.UserName = truncateRunes(strings.TrimSpace(settings.UserName), 32)
	settings.SearchProvider = strings.ToLower(strings.TrimSpace(settings.SearchProvider))
	if settings.SearchProvider == "" {
		settings.SearchProvider = DefaultSearchProvider
	}
	settings.TavilyAPIKey = strings.TrimSpace(settings.TavilyAPIKey)
	settings.ResponseStyle = strings.ToLower(strings.TrimSpace(settings.ResponseStyle))
	if settings.ResponseStyle == "" {
		settings.ResponseStyle = ResponseStyleConcise
	}
	settings.Personality = strings.ToLower(strings.TrimSpace(settings.Personality))
	if settings.Personality == "" {
		settings.Personality = PersonalityFriendly
	}
	settings.Warmth = strings.ToLower(strings.TrimSpace(settings.Warmth))
	if settings.Warmth == "" {
		settings.Warmth = WarmthBalanced
	}
	settings.Enthusiasm = strings.ToLower(strings.TrimSpace(settings.Enthusiasm))
	if settings.Enthusiasm == "" {
		settings.Enthusiasm = EnthusiasmBalanced
	}
	settings.Emoji = strings.ToLower(strings.TrimSpace(settings.Emoji))
	if settings.Emoji == "" {
		settings.Emoji = EmojiNone
	}
	settings.CustomInstructions = strings.TrimSpace(settings.CustomInstructions)
	settings.Models = normalizeModels(settings.Models)
	settings.ActiveModelID = strings.TrimSpace(settings.ActiveModelID)
	if modelIndex(settings.Models, settings.ActiveModelID) < 0 {
		settings.ActiveModelID = firstModelID(settings.Models)
	}
	return settings
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func validate(settings Settings) error {
	if settings.SearchProvider != SearchProviderDuckDuckGo &&
		settings.SearchProvider != SearchProviderTavily {
		return fmt.Errorf("不支持的搜索供应商: %q", settings.SearchProvider)
	}
	if settings.SearchProvider == SearchProviderTavily && settings.TavilyAPIKey == "" {
		return errors.New("使用 Tavily 搜索时需要填写 API Key")
	}
	if !oneOf(settings.ResponseStyle, ResponseStyleConcise, ResponseStyleBalanced, ResponseStyleDetailed) {
		return fmt.Errorf("不支持的回答风格: %q", settings.ResponseStyle)
	}
	if !oneOf(
		settings.Personality,
		PersonalityProfessional,
		PersonalityFriendly,
		PersonalityCandid,
		PersonalityQuirky,
		PersonalityEfficient,
		PersonalityCynical,
	) {
		return fmt.Errorf("不支持的人格: %q", settings.Personality)
	}
	if !oneOf(settings.Warmth, WarmthReserved, WarmthBalanced, WarmthFriendly) {
		return fmt.Errorf("不支持的亲切程度: %q", settings.Warmth)
	}
	if !oneOf(settings.Enthusiasm, EnthusiasmCalm, EnthusiasmBalanced, EnthusiasmEnthusiastic) {
		return fmt.Errorf("不支持的热情程度: %q", settings.Enthusiasm)
	}
	if !oneOf(settings.Emoji, EmojiNone, EmojiSparing, EmojiFrequent) {
		return fmt.Errorf("不支持的 Emoji 设置: %q", settings.Emoji)
	}
	if len([]rune(settings.CustomInstructions)) > MaxCustomInstructions {
		return fmt.Errorf("自定义指令最多 %d 个字符", MaxCustomInstructions)
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func writeSettings(path string, settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	models := settings.Models
	if models == nil {
		models = []ModelConfig{}
	}
	stored := storedSettings{
		SchemaVersion:        SchemaVersion,
		Models:               models,
		ActiveModelID:        settings.ActiveModelID,
		UserName:             settings.UserName,
		WebSearchEnabled:     settings.WebSearchEnabled,
		SearchProvider:       settings.SearchProvider,
		ResponseStyle:        settings.ResponseStyle,
		Personality:          settings.Personality,
		Warmth:               settings.Warmth,
		Enthusiasm:           settings.Enthusiasm,
		Emoji:                settings.Emoji,
		CustomInstructions:   settings.CustomInstructions,
		MemoryEnabled:        settings.MemoryEnabled,
		MemorySummaryEnabled: settings.MemorySummaryEnabled,
		MemorySearchEnabled:  settings.MemorySearchEnabled,
		IntelligentUIEnabled: settings.IntelligentUIEnabled,
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create settings file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write settings: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
