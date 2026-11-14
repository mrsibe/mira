package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	ProviderOpenAI     = "openai"
	ProviderAzure      = "azure"
	ProviderGoogle     = "google"
	ProviderAnthropic  = "anthropic"
	ProviderDeepSeek   = "deepseek"
	ProviderOpenRouter = "openrouter"
	ProviderXAI        = "xai"
	ProviderCustom     = "custom"
)

// ModelConfig is one persisted model connection.
type ModelConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Model    string `json:"model"`
	Protocol string `json:"protocol,omitempty"`
}

// ModelInput creates or updates a model. An empty APIKey preserves the
// existing key when ID matches a saved model.
type ModelInput struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
	Protocol string `json:"protocol"`
	APIKey   string `json:"apiKey"`
}

// ModelView is the frontend-safe model entry.
type ModelView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ProviderLabel string `json:"providerLabel"`
	BaseURL       string `json:"baseUrl"`
	Model         string `json:"model"`
	Protocol      string `json:"protocol"`
	Configured    bool   `json:"configured"`
	HasAPIKey     bool   `json:"hasApiKey"`
	APIKeyStorage string `json:"apiKeyStorage"`
}

// ProtocolOption is one selectable API protocol.
type ProtocolOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ProviderInfo describes the defaults and requirements of one provider.
type ProviderInfo struct {
	ID                string           `json:"id"`
	Label             string           `json:"label"`
	Description       string           `json:"description"`
	DefaultBaseURL    string           `json:"defaultBaseUrl"`
	BaseURLHint       string           `json:"baseUrlHint"`
	DefaultModel      string           `json:"defaultModel"`
	ModelLabel        string           `json:"modelLabel"`
	DefaultProtocol   string           `json:"defaultProtocol"`
	Protocols         []ProtocolOption `json:"protocols"`
	RequiresBaseURL   bool             `json:"requiresBaseUrl"`
	RequiresAPIKey    bool             `json:"requiresApiKey"`
	LocalAuthOptional bool             `json:"localAuthOptional"`
	APIKeyEnv         string           `json:"apiKeyEnv"`
}

var protocolOptions = map[string]ProtocolOption{
	ProtocolResponses:   {Value: ProtocolResponses, Label: "Responses"},
	ProtocolCompletions: {Value: ProtocolCompletions, Label: "Chat Completions"},
	ProtocolMessages:    {Value: ProtocolMessages, Label: "Anthropic Messages"},
	ProtocolGoogle:      {Value: ProtocolGoogle, Label: "Gemini"},
	ProtocolAnthropic:   {Value: ProtocolAnthropic, Label: "Anthropic Messages"},
}

// Providers returns the supported model providers in display order.
func Providers() []ProviderInfo {
	return []ProviderInfo{
		{
			ID:                ProviderOpenAI,
			Label:             "OpenAI",
			Description:       "OpenAI 官方 Responses 或 Chat Completions 接口",
			DefaultBaseURL:    DefaultBaseURL,
			BaseURLHint:       DefaultBaseURL,
			DefaultModel:      DefaultModel,
			ModelLabel:        "模型",
			DefaultProtocol:   ProtocolResponses,
			Protocols:         protocolList(ProtocolResponses, ProtocolCompletions),
			RequiresAPIKey:    true,
			LocalAuthOptional: true,
			APIKeyEnv:         "OPENAI_API_KEY",
		},
		{
			ID:              ProviderAzure,
			Label:           "Azure OpenAI",
			Description:     "Azure OpenAI v1 Endpoint，模型字段填写部署名称",
			BaseURLHint:     "https://<resource>.openai.azure.com/openai/v1",
			ModelLabel:      "部署名称",
			DefaultProtocol: ProtocolResponses,
			Protocols:       protocolList(ProtocolResponses, ProtocolCompletions),
			RequiresBaseURL: true,
			RequiresAPIKey:  true,
			APIKeyEnv:       "AZURE_OPENAI_API_KEY",
		},
		{
			ID:              ProviderGoogle,
			Label:           "Google Gemini",
			Description:     "Google Gemini 原生接口",
			BaseURLHint:     "留空使用 Gemini 官方端点",
			DefaultModel:    "gemini-3-flash",
			ModelLabel:      "模型",
			DefaultProtocol: ProtocolGoogle,
			Protocols:       protocolList(ProtocolGoogle),
			RequiresAPIKey:  true,
			APIKeyEnv:       "GEMINI_API_KEY",
		},
		{
			ID:              ProviderAnthropic,
			Label:           "Anthropic",
			Description:     "Anthropic Messages 原生接口",
			DefaultBaseURL:  "https://api.anthropic.com",
			BaseURLHint:     "https://api.anthropic.com",
			DefaultModel:    "claude-sonnet-5-5",
			ModelLabel:      "模型",
			DefaultProtocol: ProtocolAnthropic,
			Protocols:       protocolList(ProtocolAnthropic),
			RequiresAPIKey:  true,
			APIKeyEnv:       "ANTHROPIC_API_KEY",
		},
		{
			ID:              ProviderDeepSeek,
			Label:           "DeepSeek",
			Description:     "DeepSeek Chat Completions，支持 reasoning 回放",
			DefaultBaseURL:  "https://api.deepseek.com",
			BaseURLHint:     "https://api.deepseek.com",
			DefaultModel:    "deepseek-reasoner",
			ModelLabel:      "模型",
			DefaultProtocol: ProtocolCompletions,
			Protocols:       protocolList(ProtocolCompletions),
			RequiresAPIKey:  true,
			APIKeyEnv:       "DEEPSEEK_API_KEY",
		},
		{
			ID:              ProviderOpenRouter,
			Label:           "OpenRouter",
			Description:     "OpenRouter Chat Completions 或 Anthropic Messages",
			DefaultBaseURL:  "https://openrouter.ai/api/v1",
			BaseURLHint:     "https://openrouter.ai/api/v1",
			DefaultModel:    "anthropic/claude-sonnet-4",
			ModelLabel:      "模型",
			DefaultProtocol: ProtocolCompletions,
			Protocols:       protocolList(ProtocolCompletions, ProtocolMessages),
			RequiresAPIKey:  true,
			APIKeyEnv:       "OPENROUTER_API_KEY",
		},
		{
			ID:              ProviderXAI,
			Label:           "xAI",
			Description:     "xAI Responses 接口",
			DefaultBaseURL:  "https://api.x.ai/v1",
			BaseURLHint:     "https://api.x.ai/v1",
			DefaultModel:    "grok-4",
			ModelLabel:      "模型",
			DefaultProtocol: ProtocolResponses,
			Protocols:       protocolList(ProtocolResponses),
			RequiresAPIKey:  true,
			APIKeyEnv:       "XAI_API_KEY",
		},
		{
			ID:                ProviderCustom,
			Label:             "自定义",
			Description:       "任意 OpenAI 兼容 Responses 或 Chat Completions 端点",
			BaseURLHint:       "https://example.com/v1",
			ModelLabel:        "模型",
			DefaultProtocol:   ProtocolCompletions,
			Protocols:         protocolList(ProtocolResponses, ProtocolCompletions),
			RequiresBaseURL:   true,
			RequiresAPIKey:    true,
			LocalAuthOptional: true,
			APIKeyEnv:         "OPENAI_API_KEY",
		},
	}
}

func protocolList(values ...string) []ProtocolOption {
	options := make([]ProtocolOption, 0, len(values))
	for _, value := range values {
		if option, ok := protocolOptions[value]; ok {
			options = append(options, option)
		}
	}
	return options
}

func providerInfo(id string) (ProviderInfo, bool) {
	for _, provider := range Providers() {
		if provider.ID == id {
			return provider, true
		}
	}
	return ProviderInfo{}, false
}

func prepareModel(input ModelInput, id string) (ModelConfig, string, error) {
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey != "" {
		if err := validateSecret(apiKey); err != nil {
			return ModelConfig{}, "", fmt.Errorf("API Key %w", err)
		}
	}
	model := normalizeModel(ModelConfig{
		ID:       strings.TrimSpace(firstNonEmpty(input.ID, id)),
		Name:     input.Name,
		Provider: input.Provider,
		BaseURL:  input.BaseURL,
		Model:    input.Model,
		Protocol: input.Protocol,
	})
	if err := validateModel(model); err != nil {
		return ModelConfig{}, "", err
	}
	return model, apiKey, nil
}

func normalizeModel(model ModelConfig) ModelConfig {
	model.ID = strings.TrimSpace(model.ID)
	model.Provider = strings.ToLower(strings.TrimSpace(model.Provider))
	if model.Provider == "" {
		model.Provider = ProviderOpenAI
	}
	model.Model = strings.TrimSpace(model.Model)
	model.Name = truncateRunes(strings.TrimSpace(model.Name), 64)
	model.BaseURL = strings.TrimRight(strings.TrimSpace(model.BaseURL), "/")
	model.Protocol = strings.ToLower(strings.TrimSpace(model.Protocol))

	info, known := providerInfo(model.Provider)
	if known {
		if model.BaseURL == "" && info.DefaultBaseURL != "" {
			model.BaseURL = strings.TrimRight(info.DefaultBaseURL, "/")
		}
		if model.Model == "" && info.DefaultModel != "" {
			model.Model = info.DefaultModel
		}
		if model.Protocol == "" {
			model.Protocol = info.DefaultProtocol
		}
		switch model.Provider {
		case ProviderOpenAI, ProviderAzure:
			if model.Protocol != ProtocolCompletions {
				model.Protocol = ProtocolResponses
			}
		case ProviderCustom:
			if model.Protocol != ProtocolResponses {
				model.Protocol = ProtocolCompletions
			}
		case ProviderOpenRouter:
			if model.Protocol != ProtocolMessages {
				model.Protocol = ProtocolCompletions
			}
		case ProviderDeepSeek:
			model.Protocol = ProtocolCompletions
		case ProviderXAI:
			model.Protocol = ProtocolResponses
		case ProviderGoogle:
			model.Protocol = ProtocolGoogle
		case ProviderAnthropic:
			model.Protocol = ProtocolAnthropic
		}
	}
	if model.Name == "" {
		model.Name = model.Model
	}
	return model
}

func validateModel(model ModelConfig) error {
	info, ok := providerInfo(model.Provider)
	if !ok {
		return fmt.Errorf("不支持的服务商: %q", model.Provider)
	}
	if strings.TrimSpace(model.Model) == "" {
		label := info.ModelLabel
		if label == "" {
			label = "模型名称"
		}
		return fmt.Errorf("%s不能为空", label)
	}
	if info.RequiresBaseURL && strings.TrimSpace(model.BaseURL) == "" {
		return errors.New("Base URL 不能为空")
	}
	if model.BaseURL != "" {
		parsed, err := url.Parse(model.BaseURL)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("Base URL 无效: %q", model.BaseURL)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("Base URL 必须使用 http 或 https")
		}
	}
	for _, option := range info.Protocols {
		if option.Value == model.Protocol {
			return nil
		}
	}
	return fmt.Errorf("不支持的 API 协议: %q", model.Protocol)
}

func modelConfigured(model ModelConfig, apiKey string) bool {
	if validateModel(model) != nil {
		return false
	}
	if !modelNeedsKey(model) {
		return true
	}
	return strings.TrimSpace(apiKey) != ""
}

func modelNeedsKey(model ModelConfig) bool {
	info, ok := providerInfo(model.Provider)
	if !ok || !info.RequiresAPIKey {
		return false
	}
	if info.LocalAuthOptional && isLocalURL(model.BaseURL) {
		return false
	}
	return true
}

func modelEnvKey(provider string) string {
	keys := modelEnvKeys(provider)
	for _, key := range keys {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return key
		}
	}
	return keys[0]
}

func modelEnvKeys(provider string) []string {
	switch provider {
	case ProviderAzure:
		return []string{"AZURE_OPENAI_API_KEY"}
	case ProviderGoogle:
		return []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}
	case ProviderAnthropic:
		return []string{"ANTHROPIC_API_KEY"}
	case ProviderDeepSeek:
		return []string{"DEEPSEEK_API_KEY"}
	case ProviderOpenRouter:
		return []string{"OPENROUTER_API_KEY"}
	case ProviderXAI:
		return []string{"XAI_API_KEY"}
	default:
		return []string{"OPENAI_API_KEY"}
	}
}

func modelSecretKey(id string) string {
	if id == "" {
		return ModelSecretKey
	}
	return ModelSecretKey + ":" + id
}

func modelIndex(models []ModelConfig, id string) int {
	for index, model := range models {
		if model.ID == id {
			return index
		}
	}
	return -1
}

func modelIDExists(models []ModelConfig, id string) bool {
	return modelIndex(models, id) >= 0
}

func firstModelID(models []ModelConfig) string {
	if len(models) == 0 {
		return ""
	}
	return models[0].ID
}

func normalizeModels(models []ModelConfig) []ModelConfig {
	result := make([]ModelConfig, 0, len(models))
	seen := make(map[string]bool, len(models))
	for _, model := range models {
		model = normalizeModel(model)
		if model.ID == "" || seen[model.ID] {
			model.ID = newModelID()
		}
		for seen[model.ID] {
			model.ID = newModelID()
		}
		seen[model.ID] = true
		result = append(result, model)
	}
	return result
}

func newModelID() string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err == nil {
		return hex.EncodeToString(data[:])
	}
	return fmt.Sprintf("model-%d", time.Now().UnixNano())
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
