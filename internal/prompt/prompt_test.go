package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/mrsibe/mira/internal/config"
)

func TestBuildIncludesModulesInOrder(t *testing.T) {
	value := Build(Options{
		Now: time.Date(2026, time.November, 11, 0, 0, 0, 0, time.UTC),
		Settings: config.Settings{
			WebSearchEnabled:     true,
			MemoryEnabled:        true,
			MemorySearchEnabled:  true,
			IntelligentUIEnabled: true,
			Personality:          config.PersonalityFriendly,
			ResponseStyle:        config.ResponseStyleDetailed,
			Warmth:               config.WarmthFriendly,
			Enthusiasm:           config.EnthusiasmCalm,
			Emoji:                config.EmojiSparing,
			CustomInstructions:   "称呼我为小林。",
		},
	})

	markers := []string{
		"Priority order",
		"## Writing Style",
		"## 生成式界面",
		"json-render",
		"## 工具",
		"web_search",
		"search_memory",
		"## Personality Instruction",
		"You are a warm, curious, witty, and energetic AI friend.",
		"## 表达偏好",
		"## 用户自定义指令",
		"称呼我为小林。",
	}
	last := -1
	for _, marker := range markers {
		index := strings.Index(value, marker)
		if index < 0 {
			t.Fatalf("prompt missing %q:\n%s", marker, value)
		}
		if index < last {
			t.Fatalf("prompt module %q is out of order", marker)
		}
		last = index
	}
}

func TestBuildUsesSourceInstructionsWithoutPrivateProductContent(t *testing.T) {
	value := Build(Options{
		Now: time.Date(2026, time.November, 11, 0, 0, 0, 0, time.UTC),
		Settings: config.Settings{
			WebSearchEnabled:     true,
			MemoryEnabled:        true,
			MemorySearchEnabled:  true,
			IntelligentUIEnabled: true,
			Personality:          config.PersonalityProfessional,
		},
	})

	for _, sourceText := range []string{
		"You are Mira, a helpful AI assistant.",
		"Current date: 2026-11-11",
		"1. Answer the user's actual request directly.",
		"Aim for readable, accessible responses.",
		"# Desired oververbosity for the final answer (not analysis): 4",
		"Examples of prohibited behaviors include, but are not limited to:",
		"You are a focused, formal, and exacting AI consultant",
		"Follow the instructions above naturally, without repeating",
	} {
		if !strings.Contains(value, sourceText) {
			t.Fatalf("prompt is missing source text %q:\n%s", sourceText, value)
		}
	}

	for _, privateContent := range []string{
		"You are ChatGPT",
		"OpenAI",
		"Namespace: gmail",
		"Namespace: personal_context",
		"Ads (sponsored links)",
		"Writing Blocks",
		":::writing",
		"image_group",
		"file_search",
		"More about you textbox",
		"Atlantic/Reykjavík",
		"2026-07-21",
	} {
		if strings.Contains(value, privateContent) {
			t.Fatalf("prompt contains private product content %q:\n%s", privateContent, value)
		}
	}
}

func TestBuildEmbedsAllPersonalities(t *testing.T) {
	tests := []struct {
		value string
		text  string
	}{
		{config.PersonalityProfessional, "focused, formal, and exacting AI consultant"},
		{config.PersonalityFriendly, "warm, curious, witty, and energetic AI friend"},
		{config.PersonalityCandid, "plainspoken and direct AI coach"},
		{config.PersonalityQuirky, "playful and imaginative AI"},
		{config.PersonalityEfficient, "highly efficient assistant"},
		{config.PersonalityCynical, "cynical, sarcastic AI"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			value := Build(Options{Settings: config.Settings{Personality: test.value}})
			if !strings.Contains(value, test.text) {
				t.Fatalf("personality %q is missing:\n%s", test.value, value)
			}
		})
	}

	fallback := Build(Options{})
	if !strings.Contains(fallback, "warm, curious, witty, and energetic AI friend") {
		t.Fatalf("empty personality should fall back to friendly:\n%s", fallback)
	}
}

func TestBuildHonorsIntelligentUISwitch(t *testing.T) {
	enabled := Build(Options{Settings: config.Settings{IntelligentUIEnabled: true}})
	if !strings.Contains(enabled, "## 生成式界面") ||
		!strings.Contains(enabled, "```spec") ||
		!strings.Contains(enabled, "不要默认放在回答末尾") {
		t.Fatalf("enabled intelligent UI prompt is incomplete:\n%s", enabled)
	}
	for _, forbidden := range []string{"realistic sample data", "Never leave arrays empty"} {
		if strings.Contains(enabled, forbidden) {
			t.Fatalf("intelligent UI prompt contains unsafe data instruction %q:\n%s", forbidden, enabled)
		}
	}

	disabled := Build(Options{Settings: config.Settings{IntelligentUIEnabled: false}})
	for _, absent := range []string{"## 生成式界面", "json-render", "```spec"} {
		if strings.Contains(disabled, absent) {
			t.Fatalf("disabled intelligent UI prompt contains %q:\n%s", absent, disabled)
		}
	}
}

func TestBuildOmitsDisabledToolsAndEmptyModules(t *testing.T) {
	value := Build(Options{Settings: config.Settings{
		WebSearchEnabled:    false,
		MemoryEnabled:       false,
		MemorySearchEnabled: true,
	}})
	for _, absent := range []string{
		"## 用户自定义指令",
		"## 长期记忆",
		"<auto_memory>",
		"web_search",
		"search_memory",
	} {
		if strings.Contains(value, absent) {
			t.Fatalf("prompt unexpectedly contains %q:\n%s", absent, value)
		}
	}
	if strings.Contains(value, "以下内容是与用户相关的背景信息") {
		t.Fatal("empty memory instructions were included")
	}
}

func TestBuildHonorsMemorySearchSwitch(t *testing.T) {
	value := Build(Options{Settings: config.Settings{
		MemoryEnabled:       true,
		MemorySearchEnabled: false,
	}})
	if strings.Contains(value, "search_memory") {
		t.Fatal("search_memory should be absent when memory search is disabled")
	}
}

func TestBuildKeepsStableModulesBeforeDynamicPreferences(t *testing.T) {
	base := config.Settings{
		WebSearchEnabled:     true,
		MemoryEnabled:        true,
		MemorySearchEnabled:  true,
		IntelligentUIEnabled: true,
		Personality:          config.PersonalityFriendly,
		ResponseStyle:        config.ResponseStyleConcise,
		Warmth:               config.WarmthBalanced,
		Enthusiasm:           config.EnthusiasmBalanced,
		Emoji:                config.EmojiNone,
	}
	now := time.Date(2026, time.November, 11, 0, 0, 0, 0, time.UTC)
	first := Build(Options{Settings: base, Now: now})
	changed := base
	changed.ResponseStyle = config.ResponseStyleDetailed
	changed.CustomInstructions = "称呼我为小林。"
	second := Build(Options{Settings: changed, Now: now})

	styleStart := strings.Index(first, "## 表达偏好")
	if styleStart < 0 {
		t.Fatal("style module is missing")
	}
	common := 0
	for common < len(first) && common < len(second) && first[common] == second[common] {
		common++
	}
	if common < styleStart {
		t.Fatalf(
			"dynamic preferences changed the stable prefix at byte %d, before style at %d",
			common,
			styleStart,
		)
	}

	personalityStart := strings.Index(first, "## Personality Instruction")
	changed.Personality = config.PersonalityCandid
	third := Build(Options{Settings: changed, Now: now})
	common = 0
	for common < len(first) && common < len(third) && first[common] == third[common] {
		common++
	}
	if common < personalityStart {
		t.Fatalf(
			"changing personality altered the stable prefix at byte %d, before personality at %d",
			common,
			personalityStart,
		)
	}
}
