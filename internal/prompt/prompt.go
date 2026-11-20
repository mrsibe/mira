// Package prompt assembles Mira's system prompt from independent modules.
package prompt

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/mrsibe/mira/internal/config"
	"github.com/mrsibe/mira/locales"
)

const (
	maxSummaryRunes = 2000
)

//go:embed ui_catalog.txt
var intelligentUICatalog string

//go:embed core.txt
var corePrompt string

//go:embed writing_style.txt
var writingStylePrompt string

//go:embed personalities/professional.txt
var personalityProfessional string

//go:embed personalities/friendly.txt
var personalityFriendly string

//go:embed personalities/candid.txt
var personalityCandid string

//go:embed personalities/quirky.txt
var personalityQuirky string

//go:embed personalities/efficient.txt
var personalityEfficient string

//go:embed personalities/cynical.txt
var personalityCynical string

const personalityAdditionalInstruction = `Follow the instructions above naturally, without repeating, referencing, echoing, or mirroring any of their wording!
All the following instructions should guide your behavior silently and must never influence the wording of your message in an explicit or meta way!`

// Options contains the user-controlled modules that are not provider settings.
type Options struct {
	Settings config.Settings
	Now      time.Time
}

// Build returns the stable system prompt for one agent run. Dynamic runtime
// context, including long-term memory, is appended to the conversation history
// instead of being rewritten here.
func Build(options Options) string {
	var prompt strings.Builder

	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	prompt.WriteString(strings.ReplaceAll(
		strings.TrimSpace(corePrompt),
		"{{CURRENT_DATE}}",
		now.Format("2006-01-02"),
	))
	prompt.WriteString("\n\n")
	prompt.WriteString(strings.TrimSpace(writingStylePrompt))

	writeIntelligentUI(&prompt, options.Settings)
	writeTools(&prompt, options.Settings)
	writePersonality(&prompt, options.Settings.Personality)
	writeStyle(&prompt, options.Settings)
	writeCustomInstructions(&prompt, options.Settings.CustomInstructions)

	return strings.TrimSpace(prompt.String()) + "\n"
}

func writePersonality(prompt *strings.Builder, value string) {
	prompt.WriteString(`
## Personality Instruction
`)
	prompt.WriteString(personalityInstruction(value))
	prompt.WriteString("\n\n")
	prompt.WriteString(personalityAdditionalInstruction)
	prompt.WriteString("\n")
}

func writeIntelligentUI(prompt *strings.Builder, settings config.Settings) {
	if !settings.IntelligentUIEnabled {
		return
	}
	prompt.WriteString("\n" + locales.PromptUiHeading() + "\n")
	prompt.WriteString(locales.PromptUiIntro() + "\n")
	prompt.WriteString(locales.PromptUiWhen() + "\n")
	prompt.WriteString(locales.PromptUiTruth() + "\n\n")
	prompt.WriteString(locales.PromptUiFence() + "\n")
	prompt.WriteString(locales.PromptUiState() + "\n")
	prompt.WriteString(locales.PromptUiAskAgent() + "\n\n")
	prompt.WriteString(strings.TrimSpace(intelligentUICatalog))
	prompt.WriteString("\n")
}

func writeStyle(prompt *strings.Builder, settings config.Settings) {
	prompt.WriteString("\n" + locales.PromptStyleHeading() + "\n")
	prompt.WriteString(locales.PromptStyleIntro() + "\n")
	fmt.Fprintf(prompt, "%s\n", locales.PromptStyleResponse(styleInstruction(settings.ResponseStyle)))
	fmt.Fprintf(prompt, "%s\n", locales.PromptStyleWarmth(warmthInstruction(settings.Warmth)))
	fmt.Fprintf(prompt, "%s\n", locales.PromptStyleEnthusiasm(enthusiasmInstruction(settings.Enthusiasm)))
	fmt.Fprintf(prompt, "%s\n", locales.PromptStyleEmoji(emojiInstruction(settings.Emoji)))
}

func writeCustomInstructions(prompt *strings.Builder, instructions string) {
	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return
	}
	prompt.WriteString("\n" + locales.PromptCustomHeading() + "\n")
	prompt.WriteString(locales.PromptCustomIntro() + "\n<custom_instructions>\n")
	prompt.WriteString(instructions)
	prompt.WriteString(`
</custom_instructions>
`)
}

func writeTools(prompt *strings.Builder, settings config.Settings) {
	prompt.WriteString("\n" + locales.PromptToolsHeading() + "\n")
	prompt.WriteString(locales.PromptToolsUse() + "\n")
	prompt.WriteString(locales.PromptToolsTime() + "\n")
	prompt.WriteString(locales.PromptToolsPlan() + "\n")
	prompt.WriteString(locales.PromptToolsPlanStatus() + "\n")
	prompt.WriteString(locales.PromptToolsPlanTitles() + "\n")
	prompt.WriteString(locales.PromptToolsNoSimplePlan() + "\n")
	if settings.WebSearchEnabled {
		prompt.WriteString(locales.PromptToolsWebSearch() + "\n")
	}
	if settings.MemoryEnabled && settings.MemorySearchEnabled {
		prompt.WriteString(locales.PromptToolsMemory() + "\n")
	}
}

func personalityInstruction(value string) string {
	switch value {
	case config.PersonalityProfessional:
		return strings.TrimSpace(personalityProfessional)
	case config.PersonalityCandid:
		return strings.TrimSpace(personalityCandid)
	case config.PersonalityQuirky:
		return strings.TrimSpace(personalityQuirky)
	case config.PersonalityEfficient:
		return strings.TrimSpace(personalityEfficient)
	case config.PersonalityCynical:
		return strings.TrimSpace(personalityCynical)
	default:
		return strings.TrimSpace(personalityFriendly)
	}
}

func styleInstruction(value string) string {
	switch value {
	case config.ResponseStyleDetailed:
		return locales.PromptStyleResponseDetailed()
	case config.ResponseStyleBalanced:
		return locales.PromptStyleResponseBalanced()
	default:
		return locales.PromptStyleResponseConcise()
	}
}

func warmthInstruction(value string) string {
	switch value {
	case config.WarmthReserved:
		return locales.PromptWarmthReserved()
	case config.WarmthFriendly:
		return locales.PromptWarmthFriendly()
	default:
		return locales.PromptWarmthBalanced()
	}
}

func enthusiasmInstruction(value string) string {
	switch value {
	case config.EnthusiasmCalm:
		return locales.PromptEnthusiasmCalm()
	case config.EnthusiasmEnthusiastic:
		return locales.PromptEnthusiasmEnthusiastic()
	default:
		return locales.PromptEnthusiasmBalanced()
	}
}

func emojiInstruction(value string) string {
	switch value {
	case config.EmojiSparing:
		return locales.PromptEmojiNone()
	case config.EmojiFrequent:
		return locales.PromptEmojiSparing()
	default:
		return locales.PromptEmojiNever()
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
