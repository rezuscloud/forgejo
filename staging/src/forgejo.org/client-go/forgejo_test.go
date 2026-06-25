package forgejo

import (
	"net/url"
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("https://git.rezus.cloud", nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.base.Scheme != "https" {
		t.Errorf("expected scheme https, got %s", c.base.Scheme)
	}
	if c.base.Host != "git.rezus.cloud" {
		t.Errorf("expected host git.rezus.cloud, got %s", c.base.Host)
	}
}

func TestNewClientDefaultScheme(t *testing.T) {
	c, err := NewClient("git.rezus.cloud", nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c.base.Scheme != "https" {
		t.Errorf("expected default scheme https, got %s", c.base.Scheme)
	}
}

func TestNewClientInvalidURL(t *testing.T) {
	_, err := NewClient("://invalid", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestServicesInitialized(t *testing.T) {
	c, _ := NewClient("https://git.rezus.cloud", http.DefaultClient)
	if c.Repo == nil {
		t.Error("RepoService not initialized")
	}
	if c.Admin == nil {
		t.Error("AdminService not initialized")
	}
	if c.Issue == nil {
		t.Error("IssueService not initialized")
	}
	if c.Org == nil {
		t.Error("OrgService not initialized")
	}
	if c.User == nil {
		t.Error("UserService not initialized")
	}
}

func TestListOptionsApplyToURL(t *testing.T) {
	lo := ListOptions{Page: 3, PageSize: 50}
	u, _ := parseURL("https://example.com/path")
	lo.applyToURL(u)
	if u.Query().Get("page") != "3" {
		t.Errorf("expected page=3, got %s", u.Query().Get("page"))
	}
	if u.Query().Get("limit") != "50" {
		t.Errorf("expected limit=50, got %s", u.Query().Get("limit"))
	}
}

// helper for tests
func parseURL(s string) (*url.URL, error) {
	return url.Parse(s)
}
