package memory

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mrsibe/goat/ai"
	"github.com/mrsibe/mira/internal/store"
	"golang.org/x/text/unicode/norm"
)

const (
	bm25K1         = 1.2
	bm25B          = 0.75
	defaultLimit   = 5
	maxResultRunes = 600
)

// SearchResult is one historical turn returned by the BM25 index.
type SearchResult struct {
	ConversationID string
	Title          string
	Timestamp      time.Time
	Text           string
}

type indexSource interface {
	List() []store.Conversation
	GetRecords(string) (store.Conversation, []store.MessageRecord, error)
}

type bm25Index struct {
	docs     []bm25Doc
	postings map[string][]posting
	df       map[string]int
	avgLen   float64
}

type bm25Doc struct {
	ConversationID string
	Title          string
	Timestamp      time.Time
	Text           string
	Length         int
	Tokens         map[string]int
}

type posting struct {
	Doc int
	TF  int
}

func buildBM25(source indexSource) (*bm25Index, error) {
	index := &bm25Index{
		postings: make(map[string][]posting),
		df:       make(map[string]int),
	}
	var totalLength int
	for _, conversation := range source.List() {
		_, records, err := source.GetRecords(conversation.ID)
		if err != nil {
			return nil, fmt.Errorf("读取会话 %s: %w", conversation.ID, err)
		}
		for _, doc := range turnDocs(conversation, records) {
			index.docs = append(index.docs, doc)
			totalLength += doc.Length
		}
	}
	if len(index.docs) > 0 {
		index.avgLen = float64(totalLength) / float64(len(index.docs))
		if index.avgLen == 0 {
			index.avgLen = 1
		}
	}
	for docIndex, doc := range index.docs {
		for token, count := range doc.Tokens {
			index.postings[token] = append(index.postings[token], posting{Doc: docIndex, TF: count})
			index.df[token]++
		}
	}
	return index, nil
}

func turnDocs(conversation store.Conversation, records []store.MessageRecord) []bm25Doc {
	docs := make([]bm25Doc, 0)
	var current *turnText
	flush := func() {
		if current == nil {
			return
		}
		var text strings.Builder
		if current.title != "" {
			fmt.Fprintf(&text, "对话：%s\n", current.title)
		}
		if current.user != "" {
			fmt.Fprintf(&text, "用户：%s\n", current.user)
		}
		if current.assistant != "" {
			fmt.Fprintf(&text, "Mira：%s", current.assistant)
		}
		value := strings.TrimSpace(text.String())
		if value != "" {
			tokens := tokenize(value)
			counts := make(map[string]int, len(tokens))
			for _, token := range tokens {
				counts[token]++
			}
			docs = append(docs, bm25Doc{
				ConversationID: conversation.ID,
				Title:          conversation.Title,
				Timestamp:      current.timestamp,
				Text:           value,
				Length:         len(tokens),
				Tokens:         counts,
			})
		}
		current = nil
	}

	for _, record := range records {
		if record.ContextKind != "" {
			continue
		}
		message := record.Message
		text := strings.TrimSpace(message.Text())
		switch message.Role {
		case ai.RoleUser:
			flush()
			current = &turnText{
				title:     conversation.Title,
				timestamp: record.Timestamp,
				user:      text,
			}
		case ai.RoleAssistant:
			if current == nil || len(message.ToolCalls()) > 0 || text == "" {
				continue
			}
			current.assistant = text
		}
	}
	flush()
	return docs
}

type turnText struct {
	title     string
	timestamp time.Time
	user      string
	assistant string
}

func searchBM25(index *bm25Index, query string, limit int, excludeConversationID string) []SearchResult {
	if index == nil || len(index.docs) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 10 {
		limit = 10
	}
	queryTerms := uniqueTokens(tokenize(query))
	if len(queryTerms) == 0 {
		return nil
	}

	scores := make(map[int]float64)
	totalDocs := float64(len(index.docs))
	for _, term := range queryTerms {
		postings := index.postings[term]
		if len(postings) == 0 {
			continue
		}
		df := float64(index.df[term])
		idf := math.Log(1 + (totalDocs-df+0.5)/(df+0.5))
		for _, item := range postings {
			doc := index.docs[item.Doc]
			if doc.ConversationID == excludeConversationID {
				continue
			}
			tf := float64(item.TF)
			denominator := tf + bm25K1*(1-bm25B+bm25B*float64(doc.Length)/index.avgLen)
			scores[item.Doc] += idf * tf * (bm25K1 + 1) / denominator
		}
	}

	type ranked struct {
		doc   int
		score float64
	}
	items := make([]ranked, 0, len(scores))
	for doc, score := range scores {
		items = append(items, ranked{doc: doc, score: score})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return index.docs[items[i].doc].Timestamp.After(index.docs[items[j].doc].Timestamp)
		}
		return items[i].score > items[j].score
	})
	if len(items) > limit {
		items = items[:limit]
	}

	results := make([]SearchResult, len(items))
	for i, item := range items {
		doc := index.docs[item.doc]
		results[i] = SearchResult{
			ConversationID: doc.ConversationID,
			Title:          doc.Title,
			Timestamp:      doc.Timestamp,
			Text:           snippet(doc.Text, query),
		}
	}
	return results
}

func tokenize(text string) []string {
	normalized := strings.ToLower(norm.NFKC.String(text))
	runes := []rune(normalized)
	tokens := make([]string, 0, len(runes)/2)
	for index := 0; index < len(runes); {
		r := runes[index]
		if isCJK(r) {
			end := index
			for end < len(runes) && isCJK(runes[end]) {
				end++
			}
			run := runes[index:end]
			for i, value := range run {
				tokens = append(tokens, string(value))
				if i+1 < len(run) {
					tokens = append(tokens, string([]rune{value, run[i+1]}))
				}
			}
			index = end
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			end := index + 1
			for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsNumber(runes[end])) {
				end++
			}
			tokens = append(tokens, string(runes[index:end]))
			index = end
			continue
		}
		index++
	}
	return tokens
}

func uniqueTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func snippet(text, query string) string {
	runes := []rune(text)
	if len(runes) <= maxResultRunes {
		return text
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return string(runes[:maxResultRunes]) + "…"
	}
	index := strings.Index(strings.ToLower(text), query)
	if index < 0 {
		return string(runes[:maxResultRunes]) + "…"
	}
	byteIndex := len([]rune(strings.ToLower(text)[:index]))
	start := byteIndex - maxResultRunes/3
	if start < 0 {
		start = 0
	}
	end := start + maxResultRunes
	if end > len(runes) {
		end = len(runes)
		start = max(0, end-maxResultRunes)
	}
	value := string(runes[start:end])
	if start > 0 {
		value = "…" + value
	}
	if end < len(runes) {
		value += "…"
	}
	return value
}
