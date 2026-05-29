package client

import (
	"os"
	"testing"
)

func TestParseUserInfoAdvisor(t *testing.T) {
	f, err := os.Open("../../sample-responses/sample-student.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	meta, err := ParseUserInfo(f)
	if err != nil {
		t.Fatal(err)
	}

	got := meta["advisor_name"]
	const want = "Ian Ludden"
	if got != want {
		t.Errorf("advisor_name = %q, want %q", got, want)
	}
}
