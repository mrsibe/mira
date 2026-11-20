package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mrsibe/mira/locales"
)

const (
	// RuntimeMemoryKind identifies hidden long-term-memory context records.
	RuntimeMemoryKind = "memory"
	runtimeVersion    = "1"
)

// RuntimeMemory is one versioned snapshot of the injected long-term memory.
// Hash changes only when the model-visible content changes.
type RuntimeMemory struct {
	Hash    string
	State   string
	Summary string
	Active  bool
}

// NewRuntimeMemory builds a versioned memory snapshot. When memory is disabled,
// the snapshot is used to invalidate an earlier injected version.
func NewRuntimeMemory(enabled bool, summary string) RuntimeMemory {
	summary = neutralizeContextTags(truncateRunes(strings.TrimSpace(summary), maxSummaryRunes))
	state := "off"
	if enabled {
		state = "empty"
		if summary != "" {
			state = "active"
		}
	}
	sum := sha256.Sum256([]byte(runtimeVersion + "\x00" + state + "\x00" + summary))
	return RuntimeMemory{
		Hash:    hex.EncodeToString(sum[:]),
		State:   state,
		Summary: summary,
		Active:  state == "active",
	}
}

// ShouldAppend reports whether this snapshot needs a new history event. Empty
// and disabled states are skipped when this conversation never had memory, so
// new chats do not grow an unnecessary context message.
func (m RuntimeMemory) ShouldAppend(lastHash string) bool {
	return m.Hash != lastHash && (lastHash != "" || m.Active)
}

// Text renders the hidden runtime context. User role is used because OpenAI's
// Responses adapter lifts system messages into request-level instructions,
// which would move this dynamic content back into the cached prefix.
func (m RuntimeMemory) Text() string {
	var text strings.Builder
	fmt.Fprintf(
		&text,
		"<mira_context kind=%q version=%q state=%q>\n",
		RuntimeMemoryKind,
		runtimeVersion,
		m.State,
	)
	if m.Active {
		text.WriteString(locales.PromptRuntimeIntro())
		text.WriteString("<auto_memory>\n")
		text.WriteString(m.Summary)
		text.WriteString("\n</auto_memory>\n")
	} else {
		if m.State == "empty" {
			text.WriteString(locales.PromptRuntimeNoMemory())
		} else {
			text.WriteString(locales.PromptRuntimeMemoryCleared())
		}
		text.WriteString(locales.PromptRuntimeMemoryInvalidated())
	}
	text.WriteString("</mira_context>")
	return text.String()
}

func neutralizeContextTags(value string) string {
	replacer := strings.NewReplacer(
		"<mira_context", "&lt;mira_context",
		"</mira_context", "&lt;/mira_context",
		"<auto_memory", "&lt;auto_memory",
		"</auto_memory", "&lt;/auto_memory",
	)
	return replacer.Replace(value)
}
