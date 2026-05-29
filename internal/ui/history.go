package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/luddenig/schedule-lookdown/internal/query"
)

const historyLimit = 20

type HistoryEntry struct {
	QueryType string            `json:"query_type"`
	Params    map[string]string `json:"params"`
	Result    query.Result      `json:"result"`
	FetchedAt time.Time         `json:"fetched_at"`
}

// entryIntentKey constructs a string key from a query type and parameters.
// This is used to detect whether the same query has already been run.
func entryIntentKey(queryType string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(queryType)
	for _, k := range keys {
		b.WriteByte('|')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

type queryHistory struct {
	entries []HistoryEntry
}

// add inserts a new entry or updates an existing one with the same intent.
// If at capacity, the oldest entry by FetchedAt is evicted first.
func (h *queryHistory) add(entry HistoryEntry) {
	key := entryIntentKey(entry.QueryType, entry.Params)
	for i, e := range h.entries {
		if entryIntentKey(e.QueryType, e.Params) == key {
			h.entries[i].Result = entry.Result
			h.entries[i].FetchedAt = entry.FetchedAt
			return
		}
	}
	if len(h.entries) >= historyLimit {
		oldest := h.findOldestIndex()
		h.entries = append(h.entries[:oldest], h.entries[oldest+1:]...)
	}
	h.entries = append(h.entries, entry)
}

// findOldestIndex finds the index of the oldest entry, by FetchedAt,
// in h.entries. If h.entries is empty, returns -1.
func (h *queryHistory) findOldestIndex() int {
	if h.entries == nil {
		return -1
	}

	oldest := 0
	for i, e := range h.entries {
		if e.FetchedAt.Before(h.entries[oldest].FetchedAt) {
			oldest = i
		}
	}

	return oldest
}

// clear removes all history entries.
func (h *queryHistory) clear() {
	h.entries = nil
}

// sorted returns entries ordered by FetchedAt descending (most recent first).
func (h *queryHistory) sorted() []HistoryEntry {
	out := make([]HistoryEntry, len(h.entries))
	copy(out, h.entries)
	sort.Slice(out, func(i, j int) bool {
		return out[i].FetchedAt.After(out[j].FetchedAt)
	})
	return out
}

func (h *queryHistory) marshal() ([]byte, error) {
	return json.Marshal(h.entries)
}

func (h *queryHistory) unmarshal(data []byte) error {
	return json.Unmarshal(data, &h.entries)
}

func historyEntryLabel(e HistoryEntry) string {
	// Reserve 2 chars for the "> " cursor prefix; each label prefix accounts
	// for the rest (e.g. "Course: " = 8 chars).
	availableWidth := panelInnerWidth - 2
	switch e.QueryType {
	case "course_search":
		code := e.Params["course_code"]
		if code == "" {
			code = e.Params["instructor"]
		}
		return fmt.Sprintf("Course: %s", truncateStr(code, availableWidth-8))
	case "schedule_lookup":
		return fmt.Sprintf("Sched: %s", truncateStr(e.Params["username"], availableWidth-7))
	case "roster_view":
		return fmt.Sprintf("Roster: %s", truncateStr(e.Params["course_id"], availableWidth-8))
	case "instructor_lookup":
		return fmt.Sprintf("Instr: %s", truncateStr(e.Params["username"], availableWidth-7))
	default:
		return e.QueryType
	}
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
