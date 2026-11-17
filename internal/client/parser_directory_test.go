package client

import (
	"strings"
	"testing"

	"github.com/luddenig/schedule-lookdown/internal/models"
)

func TestParseUserOptions(t *testing.T) {
	html := `<html><body><form>
	<select name="termcode"><option value="202710">Fall</option></select>
	<select name="id6" size="10" multiple="multiple">
	<option value="2000001">Doe, Jane (FR-ME)              </option>
	<option value="2000002">Roe, Richard                   </option>
	<option value="73473">Poe, Edgar Allan                </option>
	<option value="2000003">Lema O'Connor d'Arlach, Santiago (S</option>
	<option value="">-- none --</option>
	<option value="2000004">Mononym</option>
	</select></form></body></html>`

	got, err := ParseUserOptions(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseUserOptions: %v", err)
	}
	want := []models.Person{
		{ID: "2000001", Last: "Doe", First: "Jane", Detail: "FR-ME"},
		{ID: "2000002", Last: "Roe", First: "Richard"},
		{ID: "73473", Last: "Poe", First: "Edgar Allan"},
		{ID: "2000003", Last: "Lema O'Connor d'Arlach", First: "Santiago"},
		{ID: "2000004", Last: "Mononym"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d people %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseUserOptionsEmptyAndGarbled(t *testing.T) {
	for _, in := range []string{"", "<html><body>no select here", "<select name='id6'><option"} {
		got, err := ParseUserOptions(strings.NewReader(in))
		if err != nil || len(got) != 0 {
			t.Errorf("input %q: got %v, %v; want no people, nil error", in, got, err)
		}
	}
}
