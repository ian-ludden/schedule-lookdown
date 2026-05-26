package models

import (
	"fmt"
	"strconv"
)

// RHIT quarter sequence within an academic year (10=Fall, 20=Winter, 30=Spring, 40=Summer).
// The year digit in the term code is the *end* year of the academic year
// (e.g. 202630 = Spring 2025-26).
var termSequence = []int{10, 20, 30, 40}

func termIndex(quarter int) int {
	for i, q := range termSequence {
		if q == quarter {
			return i
		}
	}
	return -1
}

func parseTerm(code string) (year, quarter int, ok bool) {
	if len(code) != 6 {
		return 0, 0, false
	}
	y, err := strconv.Atoi(code[:4])
	if err != nil {
		return 0, 0, false
	}
	q, err := strconv.Atoi(code[4:])
	if err != nil {
		return 0, 0, false
	}
	return y, q, true
}

// NextTerm returns the term code following code in RHIT's quarter sequence.
// Unknown codes are returned unchanged.
func NextTerm(code string) string {
	year, quarter, ok := parseTerm(code)
	if !ok {
		return code
	}
	idx := termIndex(quarter)
	if idx < 0 {
		return code
	}
	next := idx + 1
	if next >= len(termSequence) {
		next = 0
		year++
	}
	return fmt.Sprintf("%04d%02d", year, termSequence[next])
}

// PrevTerm returns the term code preceding code in RHIT's quarter sequence.
// Unknown codes are returned unchanged.
func PrevTerm(code string) string {
	year, quarter, ok := parseTerm(code)
	if !ok {
		return code
	}
	idx := termIndex(quarter)
	if idx < 0 {
		return code
	}
	prev := idx - 1
	if prev < 0 {
		prev = len(termSequence) - 1
		year--
	}
	return fmt.Sprintf("%04d%02d", year, termSequence[prev])
}

// TermDisplayName converts a term code to a human-readable name.
// Examples: "202630" → "Spring 2025-26", "202640" → "Summer 2026".
// Unknown or malformed codes are returned as-is.
func TermDisplayName(code string) string {
	year, quarter, ok := parseTerm(code)
	if !ok {
		return code
	}
	switch quarter {
	case 10:
		return fmt.Sprintf("Fall %d-%02d", year-1, year%100)
	case 20:
		return fmt.Sprintf("Winter %d-%02d", year-1, year%100)
	case 30:
		return fmt.Sprintf("Spring %d-%02d", year-1, year%100)
	case 40:
		return fmt.Sprintf("Summer %d", year)
	}
	return code
}
