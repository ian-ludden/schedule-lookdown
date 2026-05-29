package client

import (
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// ParseUserInfo extracts key-value metadata from a reg-sched.pl student
// schedule response. The user-info table has a single <td> with all student
// fields as <br>-separated text (goquery's Text() concatenates them without
// newlines), e.g. "Name: Nandini  BregginAdvisor: Ian  Ludden".
//
// The advisor's full name is extracted from that cell, then matched against
// instructor links in the schedule table (format: "Ludden, Ian (luddenig)")
// to resolve the username. Returns an empty map when nothing is found.
func ParseUserInfo(r io.Reader) (map[string]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	meta := map[string]string{}

	// Step 1: find "Advisor:" in any <td> and extract the full name that follows.
	var advisorName string
	doc.Find("td").Each(func(_ int, cell *goquery.Selection) {
		if advisorName != "" {
			return
		}
		text := cell.Text()
		idx := strings.Index(strings.ToLower(text), "advisor:")
		if idx < 0 {
			return
		}
		name := strings.Join(strings.Fields(text[idx+len("advisor:"):]), " ")
		if name != "" {
			advisorName = name
		}
	})
	if advisorName == "" {
		return meta, nil
	}

	// Step 2: match the advisor's name against instructor links on the page.
	// Each instructor link sits in a <td> like "Ludden, Ian (luddenig)" with
	// href containing type=Instructor&...&id=luddenig.
	want := nameTokens(advisorName)
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		if meta["advisor"] != "" {
			return
		}
		href, _ := a.Attr("href")
		if !strings.Contains(href, "type=Instructor") {
			return
		}
		id := urlQueryParam(href, "id")
		if id == "" {
			return
		}
		td := a.Closest("td")
		if td.Length() == 0 {
			return
		}
		if nameTokensMatch(want, nameTokens(td.Text())) {
			meta["advisor"] = id
		}
	})

	return meta, nil
}

// nameTokens splits s into a set of lowercase tokens (≥2 chars) with
// punctuation stripped — normalises both "Ian Ludden" and "Ludden, Ian".
func nameTokens(s string) map[string]bool {
	tokens := make(map[string]bool)
	for _, f := range strings.Fields(strings.ToLower(s)) {
		f = strings.Trim(f, ".,():;")
		if len(f) >= 2 {
			tokens[f] = true
		}
	}
	return tokens
}

// nameTokensMatch returns true if every token in want is present in got.
func nameTokensMatch(want, got map[string]bool) bool {
	for t := range want {
		if !got[t] {
			return false
		}
	}
	return true
}

// urlQueryParam parses a URL and returns the value of the named query parameter.
func urlQueryParam(rawURL, param string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(param)
}

// ParseTable extracts headers and rows from the first table with a BORDER
// attribute — the schedule data table, not the header info table.
func ParseTable(r io.Reader) ([]string, [][]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, nil, err
	}

	var headers []string
	var rows [][]string

	// The header info table uses WIDTH=100%; the data table uses BORDER=1.
	doc.Find("table[border]").First().Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			s.Find("th").Each(func(_ int, th *goquery.Selection) {
				headers = append(headers, strings.TrimSpace(th.Text()))
			})
			return
		}
		var row []string
		s.Find("td").Each(func(_ int, td *goquery.Selection) {
			row = append(row, strings.TrimSpace(td.Text()))
		})
		if len(row) > 0 {
			rows = append(rows, row)
		}
	})

	return headers, rows, nil
}

// ParseSections parses the schedule table from a reg-sched.pl response into
// typed Section values. Column order matches the HTML:
// Course, CRN, Course Title, Instructor, CrHrs, Enrl, Cap, Term Schedule,
// Comments, Final Exam Schedule, Term Dates.
func ParseSections(r io.Reader) ([]models.Section, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var sections []models.Section

	doc.Find("table[border]").First().Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // skip header row
		}
		cells := s.Find("td")
		if cells.Length() < 7 {
			return
		}
		col := func(n int) string {
			return strings.TrimSpace(cells.Eq(n).Text())
		}
		sections = append(sections, models.Section{
			Course:     col(0),
			CRN:        col(1),
			Title:      col(2),
			Instructor: col(3),
			Credits:    atoi(col(4)),
			Enrolled:   atoi(col(5)),
			Capacity:   atoi(col(6)),
			Schedule:   col(7),
			Comments:   col(8),
			FinalExam:  col(9),
			TermDates:  col(10),
		})
	})

	return sections, nil
}

// ParseRoster extracts headers and rows from the roster student table in a
// reg-sched.pl roster response. The response contains two BORDER=1 tables:
// the first lists section details; the second lists enrolled students.
func ParseRoster(r io.Reader) ([]string, [][]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, nil, err
	}

	var headers []string
	var rows [][]string

	doc.Find("table[border]").Eq(1).Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			s.Find("th").Each(func(_ int, th *goquery.Selection) {
				headers = append(headers, strings.TrimSpace(th.Text()))
			})
			return
		}
		var row []string
		s.Find("td").Each(func(_ int, td *goquery.Selection) {
			row = append(row, strings.TrimSpace(td.Text()))
		})
		if len(row) > 0 {
			rows = append(rows, row)
		}
	})

	return headers, rows, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
