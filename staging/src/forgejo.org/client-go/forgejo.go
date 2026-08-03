// Package forgejo provides a Go client for the Forgejo REST API.
// It is generated from the Forgejo OpenAPI spec; see api/gen/ for the generator.
package forgejo

import (
	"context"
	"fmt"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client manages communication with the Forgejo REST API.
type Client struct {
	client *http.Client
	base   *url.URL

	// Services — one per resource group (ADR-0002).
	Repo        *RepoService
	Issue       *IssueService
	Org         *OrgService
	User        *UserService
	Admin       *AdminService
	Notify      *NotifyService
	ActivityPub *ActivitypubService
	Misc        *MiscService
}

// NewClient returns a new Forgejo API client for the given server URL.
// The provided http.Client is used as-is; token injection is the caller's
// responsibility (ADR-0004) — typically via a custom RoundTripper.
func NewClient(serverURL string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL %q: %w", serverURL, err)
	}
	if base.Scheme == "" {
		base.Scheme = "https"
	}
	// Ensure the base URL points at the API root (ADR-0001: parity with the Rust crate
	// which prepends /api/v1 internally on every request).
	if !strings.HasSuffix(base.Path, "/api/v1") {
		base = base.JoinPath("api", "v1")
	}
	c := &Client{client: httpClient, base: base}
	c.Repo = (*RepoService)(&service{client: c})
	c.Issue = (*IssueService)(&service{client: c})
	c.Org = (*OrgService)(&service{client: c})
	c.User = (*UserService)(&service{client: c})
	c.Admin = (*AdminService)(&service{client: c})
	c.Notify = (*NotifyService)(&service{client: c})
	c.ActivityPub = (*ActivitypubService)(&service{client: c})
	c.Misc = (*MiscService)(&service{client: c})
	return c, nil
}

// service is the embedded base for all resource services.
type service struct {
	client *Client
}

// do performs an HTTP request and returns the raw Response.
func (s *service) do(ctx context.Context, method, path string, body interface{}, opts interface{}) (*Response, error) {
	req, err := s.client.newRequest(ctx, method, path, body, opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.client.Do(req)
	if err != nil {
		return nil, err
	}
	return &Response{Response: resp}, nil
}

// newRequest builds an *http.Request against the client's base URL.
func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}, opts interface{}) (*http.Request, error) {
	rel := c.base.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, method, rel.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

var ErrNotImplemented = fmt.Errorf("not yet implemented — generated stub")


// Service types (used by generated methods).
type RepoService service
type IssueService service
type OrgService service
type UserService service
type AdminService service
type NotifyService service
type ActivitypubService service
type MiscService service

type ErrorResponse struct {
	*http.Response
	Message string `json:"message"`
	URL     string `json:"url"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %s", e.Response.Request.Method, e.Response.Request.URL, e.Response.StatusCode, e.Message)
}

func handleError(resp *http.Response) error {
	var er ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &er)
	if er.Message == "" { er.Message = string(body) }
	er.Response = resp
	return &er
}
