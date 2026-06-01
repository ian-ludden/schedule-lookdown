package query

import (
	"context"
	"testing"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

const (
	combinedFixture = "../../sample-responses/sample-roster-combined.html"
	singleFixture   = "../../sample-responses/sample-roster.html"
)

func executeRoster(t *testing.T, fixture string, q *RosterView) Result {
	t.Helper()
	c, err := client.NewFixture(fixture)
	if err != nil {
		t.Fatalf("NewFixture(%s): %v", fixture, err)
	}
	res, err := q.Execute(context.Background(), c)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return res
}

// A bare course code with multiple sections lands on the section picker.
func TestRosterBareCodeYieldsSectionPicker(t *testing.T) {
	res := executeRoster(t, combinedFixture, &RosterView{Term: "202630", CourseID: "CSSE474"})

	if res.Mode != ModeSections {
		t.Fatalf("Mode = %q, want %q", res.Mode, ModeSections)
	}
	if len(res.Rows) != 3 {
		t.Errorf("got %d section rows, want 3", len(res.Rows))
	}
	// Columns should match the course-list table (first BORDER table).
	if len(res.Columns) == 0 || res.Columns[0] != "Course" {
		t.Errorf("Columns = %v, want a course-list table starting with Course", res.Columns)
	}
	// Rows are sections to drill into.
	if got := res.Rows[0][0]; got != "CSSE474-01" {
		t.Errorf("first section = %q, want CSSE474-01", got)
	}
	if res.Metadata["sections"] != "3" {
		t.Errorf("sections metadata = %q, want 3", res.Metadata["sections"])
	}
}

// Combined=1 on a bare code returns the merged roster with trimmed metadata.
func TestRosterCombinedTrimsSectionMetadata(t *testing.T) {
	res := executeRoster(t, combinedFixture, &RosterView{Term: "202630", CourseID: "CSSE474", Combined: true})

	if res.Mode != "" {
		t.Fatalf("Mode = %q, want roster (empty)", res.Mode)
	}
	if len(res.Rows) == 0 {
		t.Fatal("combined roster has no student rows")
	}
	// Course-level metadata is kept.
	if res.Metadata["title"] == "" {
		t.Error("title should be present on combined roster")
	}
	// Per-section metadata is trimmed.
	for _, k := range []string{"crn", "instructor", "schedule", "final_exam", "comments"} {
		if v := res.Metadata[k]; v != "" {
			t.Errorf("combined metadata %q = %q, want trimmed (empty)", k, v)
		}
	}
	// Enrl/Cap are summed across the three sections (0+23+24 / 0+24+24).
	if res.Metadata["enrolled"] != "47" {
		t.Errorf("enrolled = %q, want 47 (summed)", res.Metadata["enrolled"])
	}
	if res.Metadata["capacity"] != "48" {
		t.Errorf("capacity = %q, want 48 (summed)", res.Metadata["capacity"])
	}
}

// An explicit single section shows that section's roster with full metadata.
func TestRosterSingleSectionFullMetadata(t *testing.T) {
	res := executeRoster(t, singleFixture, &RosterView{Term: "202630", CourseID: "CSSE474-02"})

	if res.Mode != "" {
		t.Fatalf("Mode = %q, want roster (empty)", res.Mode)
	}
	if res.Metadata["crn"] == "" {
		t.Error("single-section roster should keep crn metadata")
	}
	if res.Metadata["instructor"] == "" {
		t.Error("single-section roster should keep instructor metadata")
	}
}
