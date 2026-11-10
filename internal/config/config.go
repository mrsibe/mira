// Package config stores the local model provider settings for Mira.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	ProtocolResponses   = "responses"
	ProtocolCompletions = "completions"

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

	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-5"

	DefaultSearchProvider = SearchProviderDuckDuckGo
	MaxCustomInstructions = 5000
	MaxSecretBytes        = 2560

	ModelSecretKey  = "model-api-key"
	TavilySecretKey = "tavily-api-key"

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

// Settings is the persisted provider configuration.
type Settings struct {
	BaseURL              string `json:"baseUrl"`
	APIKey               string `json:"-"`
	Model                string `json:"model"`
	Protocol             string `json:"protocol"`
	UserName             string `json:"userName,omitempty"`
	WebSearchEnabled     bool   `json:"webSearchEnabled"`
	SearchProvider       string `json:"searchProvider,omitempty"`
	TavilyAPIKey         string `json:"-"`
	ResponseStyle        string `json:"responseStyle,omitempty"`
	Warmth               string `json:"warmth,omitempty"`
	Enthusiasm           string `json:"enthusiasm,omitempty"`
	Emoji                string `json:"emoji,omitempty"`
	CustomInstructions   string `json:"customInstructions,omitempty"`
	MemoryEnabled        bool   `json:"memoryEnabled"`
	MemorySummaryEnabled bool   `json:"memorySummaryEnabled"`
	MemorySearchEnabled  bool   `json:"memorySearchEnabled"`
}

// Input is the frontend-safe update payload. An empty APIKey keeps the
// existing key. A nil UserName keeps the existing display name. A nil
// WebSearchEnabled keeps the existing value. An empty TavilyAPIKey keeps the
// existing Tavily key.
type Input struct {
	BaseURL              string  `json:"baseUrl"`
	APIKey               string  `json:"apiKey"`
	Model                string  `json:"model"`
	Protocol             string  `json:"protocol"`
	UserName             *string `json:"userName,omitempty"`
	WebSearchEnabled     *bool   `json:"webSearchEnabled,omitempty"`
	SearchProvider       string  `json:"searchProvider"`
	TavilyAPIKey         string  `json:"tavilyApiKey"`
	ResponseStyle        string  `json:"responseStyle"`
	Warmth               string  `json:"warmth"`
	Enthusiasm           string  `json:"enthusiasm"`
	Emoji                string  `json:"emoji"`
	CustomInstructions   *string `json:"customInstructions,omitempty"`
	MemoryEnabled        *bool   `json:"memoryEnabled,omitempty"`
	MemorySummaryEnabled *bool   `json:"memorySummaryEnabled,omitempty"`
	MemorySearchEnabled  *bool   `json:"memorySearchEnabled,omitempty"`
}

// View is returned to the frontend. The API key is deliberately omitted.
type View struct {
	BaseURL              string `json:"baseUrl"`
	Model                string `json:"model"`
	Protocol             string `json:"protocol"`
	UserName             string `json:"userName"`
	HasAPIKey            bool   `json:"hasApiKey"`
	APIKeyStorage        string `json:"apiKeyStorage"`
	Configured           bool   `json:"configured"`
	ConfigPath           string `json:"configPath"`
	WebSearchEnabled     bool   `json:"webSearchEnabled"`
	SearchProvider       string `json:"searchProvider"`
	HasTavilyAPIKey      bool   `json:"hasTavilyApiKey"`
	TavilyAPIKeyStorage  string `json:"tavilyApiKeyStorage"`
	KeychainAvailable    bool   `json:"keychainAvailable"`
	KeychainError        string `json:"keychainError,omitempty"`
	ResponseStyle        string `json:"responseStyle"`
	Warmth               string `json:"warmth"`
	Enthusiasm           string `json:"enthusiasm"`
	Emoji                string `json:"emoji"`
	CustomInstructions   string `json:"customInstructions"`
	MemoryEnabled        bool   `json:"memoryEnabled"`
	MemorySummaryEnabled bool   `json:"memorySummaryEnabled"`
	MemorySearchEnabled  bool   `json:"memorySearchEnabled"`
}

// Manager owns the settings file and keeps a validated snapshot in memory.
type Manager struct {
	mu                  sync.RWMutex
	path                string
	secrets             SecretStore
	settings            Settings
	loadErr             error
	apiKeyStorage       string
	tavilyAPIKeyStorage string
	keychainAvailable   bool
	keychainError       string
}

// NewManager loads <dir>/config.json, falling back to environment variables
// and the system credential store.
func NewManager(dir string, secrets SecretStore) (*Manager, error) {
	path := filepath.Join(dir, "config.json")
	envAPIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	settings := Settings{
		BaseURL:              firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), DefaultBaseURL),
		APIKey:               envAPIKey,
		Model:                firstNonEmpty(os.Getenv("OPENAI_MODEL"), DefaultModel),
		Protocol:             firstNonEmpty(os.Getenv("MIRA_PROTOCOL"), ProtocolResponses),
		WebSearchEnabled:     true,
		SearchProvider:       DefaultSearchProvider,
		MemoryEnabled:        true,
		MemorySummaryEnabled: true,
		MemorySearchEnabled:  true,
	}

	m := &Manager{
		path:                path,
		secrets:             secrets,
		apiKeyStorage:       KeyStorageNone,
		tavilyAPIKeyStorage: KeyStorageNone,
	}
	if envAPIKey != "" {
		m.apiKeyStorage = KeyStorageEnvironment
	}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		saved := Settings{
			WebSearchEnabled:     true,
			SearchProvider:       DefaultSearchProvider,
			MemoryEnabled:        true,
			MemorySummaryEnabled: true,
			MemorySearchEnabled:  true,
		}
		if err := json.Unmarshal(data, &saved); err != nil {
			m.loadErr = fmt.Errorf("parse %s: %w", path, err)
		} else {
			if saved.BaseURL != "" {
				settings.BaseURL = saved.BaseURL
			}
			if saved.Model != "" {
				settings.Model = saved.Model
			}
			if saved.Protocol != "" {
				settings.Protocol = saved.Protocol
			}
			settings.UserName = saved.UserName
			settings.WebSearchEnabled = saved.WebSearchEnabled
			if saved.SearchProvider != "" {
				settings.SearchProvider = saved.SearchProvider
			}
			settings.ResponseStyle = saved.ResponseStyle
			settings.Warmth = saved.Warmth
			settings.Enthusiasm = saved.Enthusiasm
			settings.Emoji = saved.Emoji
			settings.CustomInstructions = saved.CustomInstructions
			settings.MemoryEnabled = saved.MemoryEnabled
			settings.MemorySummaryEnabled = saved.MemorySummaryEnabled
			settings.MemorySearchEnabled = saved.MemorySearchEnabled
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
	return m, nil
}

func (m *Manager) loadSecrets() {
	if m.secrets == nil {
		m.markKeychainUnavailable(errors.New("系统钥匙串不可用"))
		return
	}
	if value, found, err := m.secrets.Get(ModelSecretKey); err != nil {
		m.markKeychainUnavailable(err)
	} else {
		m.keychainAvailable = true
		if found && strings.TrimSpace(value) != "" {
			m.settings.APIKey = strings.TrimSpace(value)
			m.apiKeyStorage = KeyStorageKeychain
		}
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

// Settings returns a copy of the current configuration, including the key.
func (m *Manager) Settings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// View returns the frontend-safe configuration snapshot.
func (m *Manager) View() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viewLocked()
}

// Save updates the configuration. An empty APIKey preserves the existing key.
func (m *Manager) Save(input Input) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.settings
	modelKey := ""
	tavilyKey := ""
	if value := strings.TrimSpace(input.BaseURL); value != "" {
		next.BaseURL = value
	}
	if value := strings.TrimSpace(input.Model); value != "" {
		next.Model = value
	}
	if value := strings.TrimSpace(input.Protocol); value != "" {
		next.Protocol = value
	}
	if value := strings.TrimSpace(input.APIKey); value != "" {
		if err := validateSecret(value); err != nil {
			return m.viewLocked(), fmt.Errorf("API Key %w", err)
		}
		next.APIKey = value
		modelKey = value
	}
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
	next = normalized(next)
	if err := validate(next); err != nil {
		return m.viewLocked(), err
	}
	if err := writeSettings(m.path, next); err != nil {
		return m.viewLocked(), err
	}
	m.settings = next
	m.loadErr = nil
	if modelKey != "" {
		if m.storeSecret(ModelSecretKey, modelKey) {
			m.apiKeyStorage = KeyStorageKeychain
		} else {
			m.apiKeyStorage = KeyStorageSession
		}
	}
	if tavilyKey != "" {
		if m.storeSecret(TavilySecretKey, tavilyKey) {
			m.tavilyAPIKeyStorage = KeyStorageKeychain
		} else {
			m.tavilyAPIKeyStorage = KeyStorageSession
		}
	}
	return m.viewLocked(), nil
}

// ClearAPIKey removes the saved key.
func (m *Manager) ClearAPIKey() (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.apiKeyStorage == KeyStorageEnvironment {
		return m.viewLocked(), errors.New("API Key 来自环境变量 OPENAI_API_KEY，请在系统环境中修改")
	}
	oldKey := m.settings.APIKey
	if m.apiKeyStorage == KeyStorageKeychain {
		if err := m.deleteSecret(ModelSecretKey); err != nil {
			return m.viewLocked(), err
		}
	}
	next := m.settings
	next.APIKey = ""
	if err := writeSettings(m.path, next); err != nil {
		if m.apiKeyStorage == KeyStorageKeychain && oldKey != "" {
			_ = m.restoreSecret(ModelSecretKey, oldKey)
		}
		return m.viewLocked(), err
	}
	m.settings = next
	m.apiKeyStorage = KeyStorageNone
	return m.viewLocked(), nil
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

func (m *Manager) viewLocked() View {
	configured := m.settings.Model != "" && m.settings.BaseURL != "" &&
		(m.settings.APIKey != "" || isLocalURL(m.settings.BaseURL))
	return View{
		BaseURL:              m.settings.BaseURL,
		Model:                m.settings.Model,
		Protocol:             m.settings.Protocol,
		UserName:             m.settings.UserName,
		HasAPIKey:            m.settings.APIKey != "",
		APIKeyStorage:        m.apiKeyStorage,
		Configured:           configured,
		ConfigPath:           m.path,
		WebSearchEnabled:     m.settings.WebSearchEnabled,
		SearchProvider:       m.settings.SearchProvider,
		HasTavilyAPIKey:      m.settings.TavilyAPIKey != "",
		TavilyAPIKeyStorage:  m.tavilyAPIKeyStorage,
		KeychainAvailable:    m.keychainAvailable,
		KeychainError:        m.keychainError,
		ResponseStyle:        m.settings.ResponseStyle,
		Warmth:               m.settings.Warmth,
		Enthusiasm:           m.settings.Enthusiasm,
		Emoji:                m.settings.Emoji,
		CustomInstructions:   m.settings.CustomInstructions,
		MemoryEnabled:        m.settings.MemoryEnabled,
		MemorySummaryEnabled: m.settings.MemorySummaryEnabled,
		MemorySearchEnabled:  m.settings.MemorySearchEnabled,
	}
}

func normalized(settings Settings) Settings {
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	if settings.BaseURL == "" {
		settings.BaseURL = DefaultBaseURL
	}
	settings.Model = strings.TrimSpace(settings.Model)
	if settings.Model == "" {
		settings.Model = DefaultModel
	}
	settings.Protocol = strings.TrimSpace(settings.Protocol)
	if settings.Protocol != ProtocolCompletions {
		settings.Protocol = ProtocolResponses
	}
	settings.APIKey = strings.TrimSpace(settings.APIKey)
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
	if strings.TrimSpace(settings.Model) == "" {
		return errors.New("模型名称不能为空")
	}
	parsed, err := url.Parse(settings.BaseURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("Base URL 无效: %q", settings.BaseURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("Base URL 必须使用 http 或 https")
	}
	if settings.Protocol != ProtocolResponses && settings.Protocol != ProtocolCompletions {
		return fmt.Errorf("不支持的 API 协议: %q", settings.Protocol)
	}
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

func isLocalURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1" ||
		strings.HasSuffix(host, ".local")
}

func writeSettings(path string, settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
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
