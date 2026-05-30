package client

import (
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

// ParseUserInfo extracts key-value metadata from the user-info table of a
// reg-sched.pl response. The fields live in a single <td class="bw80"> as
// <br>-separated text (e.g. "Name: Alex Quinn", "Banner ID: 800500002",
// "Advisor: Robin Vale"). We inspect text nodes individually so each
// "Label: value" pair is captured independently.
//
// Keys are the label lower-cased with spaces replaced by underscores
// (e.g. "Banner ID" -> "banner_id"), except "Advisor" which is stored as
// "advisor_name" for backward compatibility. Whitespace in values is collapsed
// and empty values (e.g. faculty Room/Phone rendered as &nbsp) are skipped.
func ParseUserInfo(r io.Reader) (map[string]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	meta := map[string]string{}
	cell := doc.Find("td.bw80").First()
	cell.Contents().Each(func(_ int, node *goquery.Selection) {
		if goquery.NodeName(node) != "#text" {
			return
		}
		text := strings.TrimSpace(node.Text())
		label, value, ok := strings.Cut(text, ":")
		if !ok {
			return
		}
		value = strings.Join(strings.Fields(value), " ")
		if value == "" {
			return
		}
		key := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(label)), " ", "_")
		if key == "advisor" {
			key = "advisor_name"
		}
		if _, exists := meta[key]; !exists {
			meta[key] = value
		}
	})
	return meta, nil
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

// ParseTermOptions extracts the 6-digit term codes from reg-sched.pl's
// <select name="termcode"> drop-down (present on the base form page). Codes are
// returned in document order; non-numeric or non-6-digit option values are
// skipped. Callers use models.LatestTerm to pick the furthest-future code.
func ParseTermOptions(r io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var codes []string
	doc.Find("select[name='termcode'] option").Each(func(_ int, opt *goquery.Selection) {
		val, ok := opt.Attr("value")
		if !ok {
			return
		}
		val = strings.TrimSpace(val)
		if len(val) != 6 {
			return
		}
		if _, err := strconv.Atoi(val); err != nil {
			return
		}
		codes = append(codes, val)
	})
	return codes, nil
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
