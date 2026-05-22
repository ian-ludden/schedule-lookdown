package client

import (
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/luddenig/schedule-lookdown/internal/models"
)

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

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
