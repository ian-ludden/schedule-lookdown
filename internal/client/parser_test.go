package client

import (
	"os"
	"testing"
)

func TestParseUserInfo(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    map[string]string // fields that must be present and equal
		absent  []string          // keys that must NOT be present
	}{
		{
			name:    "student schedule",
			fixture: "sample-student.html",
			want: map[string]string{
				"name":         "Alex Quinn",
				"banner_id":    "800500002",
				"username":     "quinna",
				"major":        "CS",
				"year":         "Y3",
				"advisor_name": "Robin Vale",
			},
			absent: []string{"dept"},
		},
		{
			name:    "faculty schedule",
			fixture: "sample-instructor-202630.html",
			want: map[string]string{
				"name":      "Robin Vale",
				"banner_id": "800500001",
				"username":  "valer",
				"dept":      "Comp Science and Software Eng",
			},
			// Major/Advisor are student-only; Room/Phone/Campus Mail are blank (&nbsp).
			absent: []string{"major", "year", "advisor_name", "room", "phone", "campus_mail"},
		},
		{
			name:    "course roster",
			fixture: "sample-roster.html",
			want: map[string]string{
				"course_id": "CSSE474-02",
			},
			absent: []string{"name", "banner_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open("../../sample-responses/" + tt.fixture)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			meta, err := ParseUserInfo(f)
			if err != nil {
				t.Fatal(err)
			}

			for k, want := range tt.want {
				if got := meta[k]; got != want {
					t.Errorf("meta[%q] = %q, want %q", k, got, want)
				}
			}
			for _, k := range tt.absent {
				if got, ok := meta[k]; ok {
					t.Errorf("meta[%q] = %q, want absent", k, got)
				}
			}
		})
	}
}

func TestParseSectionsRosterDetail(t *testing.T) {
	f, err := os.Open("../../sample-responses/sample-roster.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sections, err := ParseSections(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 {
		t.Fatal("expected at least one section in roster response")
	}

	s := sections[0]
	if s.CRN != "3096" {
		t.Errorf("CRN = %q, want %q", s.CRN, "3096")
	}
	if s.Title != "Theory of Computation" {
		t.Errorf("Title = %q, want %q", s.Title, "Theory of Computation")
	}
	if s.Capacity != 24 {
		t.Errorf("Capacity = %d, want %d", s.Capacity, 24)
	}
}
