package client

import (
	"errors"
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

// ParseUserOptions extracts the people listed in the ad-hoc group page's
// <select name="id6"> multi-select. Option text is "Last, First" optionally
// followed by a "(YR-MAJOR)" parenthetical for students, padded with spaces; the
// server truncates long names at a fixed width, which can cut the parenthetical
// mid-way (handled by dropping an unterminated "(..."). Options with an empty
// value are skipped.
func ParseUserOptions(r io.Reader) ([]models.Person, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	var people []models.Person
	doc.Find("select[name='id6'] option").Each(func(_ int, opt *goquery.Selection) {
		id, ok := opt.Attr("value")
		id = strings.TrimSpace(id)
		if !ok || id == "" {
			return
		}
		text := strings.Join(strings.Fields(opt.Text()), " ")
		if text == "" {
			return
		}

		p := models.Person{ID: id}
		if i := strings.LastIndex(text, " ("); i >= 0 {
			detail := text[i+2:]
			if strings.HasSuffix(detail, ")") {
				p.Detail = strings.TrimSuffix(detail, ")")
			}
			text = text[:i]
		}
		if last, first, found := strings.Cut(text, ", "); found {
			p.Last, p.First = last, first
		} else {
			p.Last = text
		}
		people = append(people, p)
	})
	return people, nil
}

// ParseGroupSchedule parses the page returned by the ad-hoc group grid request
// (type=gr_Grid). It holds two BORDER=1 tables: a roster of the group's members
// (USERNAME, NAME, BANNER ID, ..., EMAIL) and the merged term grid, whose cells
// are nested tables listing "username in ROOM" per occupant. The server shades
// (BGCOLOR) a nested table when two or more people share the slot.
func ParseGroupSchedule(r io.Reader) (models.GroupSchedule, error) {
	var g models.GroupSchedule
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return g, err
	}

	tables := doc.Find("table[border]")
	if tables.Length() < 2 {
		return g, errors.New("group page has no roster and grid tables (session expired or page changed?)")
	}

	// Rows belonging directly to a table (not to cell tables nested inside it).
	directRows := func(t *goquery.Selection) *goquery.Selection {
		if body := t.ChildrenFiltered("tbody"); body.Length() > 0 {
			return body.ChildrenFiltered("tr")
		}
		return t.ChildrenFiltered("tr")
	}

	// Roster: header row, then one row per member.
	cols := map[string]int{}
	directRows(tables.Eq(0)).Each(func(i int, tr *goquery.Selection) {
		if i == 0 {
			tr.ChildrenFiltered("th").Each(func(j int, th *goquery.Selection) {
				cols[strings.ToUpper(strings.TrimSpace(th.Text()))] = j
			})
			return
		}
		tds := tr.ChildrenFiltered("td")
		cell := func(name string) string {
			j, ok := cols[name]
			if !ok || j >= tds.Length() {
				return ""
			}
			return strings.Join(strings.Fields(tds.Eq(j).Text()), " ")
		}
		m := models.GroupMember{
			Username: cell("USERNAME"),
			Name:     cell("NAME"),
			BannerID: cell("BANNER ID"),
			Email:    cell("EMAIL"),
		}
		if m.Username != "" {
			g.Members = append(g.Members, m)
		}
	})

	// Grid: header row of hours, then one row per day.
	directRows(tables.Eq(1)).Each(func(i int, tr *goquery.Selection) {
		if i == 0 {
			tr.ChildrenFiltered("th").Each(func(j int, th *goquery.Selection) {
				if j > 0 { // first header is the "TERM GRID" corner label
					g.Hours = append(g.Hours, strings.TrimSpace(th.Text()))
				}
			})
			return
		}
		g.Days = append(g.Days, strings.TrimSpace(tr.ChildrenFiltered("th").First().Text()))
		var row []models.GroupCell
		tr.ChildrenFiltered("td").Each(func(_ int, td *goquery.Selection) {
			var c models.GroupCell
			if bg, ok := td.Find("table").First().Attr("bgcolor"); ok && strings.TrimSpace(bg) != "" {
				c.Conflict = true
			}
			td.Find("table td").Each(func(_ int, e *goquery.Selection) {
				text := strings.Join(strings.Fields(e.Text()), " ")
				if text == "" {
					return
				}
				user, room, _ := strings.Cut(text, " in ")
				c.Entries = append(c.Entries, models.GroupEntry{Username: user, Room: room})
			})
			row = append(row, c)
		})
		g.Cells = append(g.Cells, row)
	})

	if len(g.Days) == 0 {
		return g, errors.New("group page grid has no day rows")
	}
	return g, nil
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
