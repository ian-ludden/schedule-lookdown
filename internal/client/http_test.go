package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadRoster(t *testing.T) {
	const wantCSV = "\"Username\",\"Last\"\n\"abbottm\",\"Abbott\"\n"
	var gotMethod, gotPath, gotID, gotDownload string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotID = r.PostFormValue("id")
		gotDownload = r.PostFormValue("download")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(wantCSV))
	}))
	defer srv.Close()

	orig := downloadURL
	downloadURL = srv.URL + "/regweb-cgi/reg-download.pl"
	defer func() { downloadURL = orig }()

	c, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	body, err := c.DownloadRoster(context.Background(), "CSSE220-01")
	if err != nil {
		t.Fatalf("DownloadRoster: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/regweb-cgi/reg-download.pl" {
		t.Errorf("path = %q, want /regweb-cgi/reg-download.pl", gotPath)
	}
	if gotID != "CSSE220-01" {
		t.Errorf("id = %q, want CSSE220-01", gotID)
	}
	if gotDownload != "Download Roster" {
		t.Errorf("download = %q, want \"Download Roster\"", gotDownload)
	}
	if string(body) != wantCSV {
		t.Errorf("body = %q, want %q", string(body), wantCSV)
	}
}

func TestDownloadRosterNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	orig := downloadURL
	downloadURL = srv.URL
	defer func() { downloadURL = orig }()

	c, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.DownloadRoster(context.Background(), "CSSE220-01"); err == nil {
		t.Error("expected error on non-200 response, got nil")
	}
}
