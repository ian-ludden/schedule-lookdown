package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/luddenig/schedule-lookdown/internal/client"
)

func TestFetchDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dir.html")
	html := `<select name="id6" multiple>
	<option value="2000001">Doe, Jane (FR-ME)   </option>
	<option value="2000002">Roe, Richard        </option></select>`
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := client.NewFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	people, err := FetchDirectory(context.Background(), c, "202710")
	if err != nil {
		t.Fatalf("FetchDirectory: %v", err)
	}
	if len(people) != 2 || people[0].ID != "2000001" || people[0].Detail != "FR-ME" {
		t.Errorf("got %+v", people)
	}

	if _, err := FetchDirectory(context.Background(), c, ""); err == nil {
		t.Error("empty term: want error")
	}
}

func TestFetchDirectoryEmptyPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.html")
	if err := os.WriteFile(path, []byte("<html>login page</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _ := client.NewFixture(path)
	if _, err := FetchDirectory(context.Background(), c, "202710"); err == nil {
		t.Error("page without people: want error")
	}
}
