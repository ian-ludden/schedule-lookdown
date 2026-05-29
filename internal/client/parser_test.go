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

	got := meta["advisor"]
	const want = "luddenig"
	if got != want {
		t.Errorf("advisor = %q, want %q", got, want)
	}
}
