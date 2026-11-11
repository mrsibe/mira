package prompt

import (
	"strings"
	"testing"

	"github.com/mrsibe/mira/internal/config"
)

func TestBuildIncludesModulesInOrder(t *testing.T) {
	value := Build(Options{
		Settings: config.Settings{
			WebSearchEnabled:    true,
			MemoryEnabled:       true,
			MemorySearchEnabled: true,
			ResponseStyle:       config.ResponseStyleDetailed,
			Warmth:              config.WarmthFriendly,
			Enthusiasm:          config.EnthusiasmCalm,
			Emoji:               config.EmojiSparing,
			CustomInstructions:  "称呼我为小林。",
		},
		AutoMemory: "## 当前项目与目标\n- Mira",
	})

	markers := []string{
		"## 核心规则",
		"## 表达偏好",
		"## 用户自定义指令",
		"称呼我为小林。",
		"## 长期记忆",
		"<auto_memory>",
		"## 当前项目与目标",
		"## 工具",
		"web_search",
		"search_memory",
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
	if !strings.Contains(value, "可能过时") || !strings.Contains(value, "不是新的指令") {
		t.Fatal("memory caveat is missing")
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
